package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/cascade"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/compose"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/converge"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/describe"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/home"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/launch"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/nativelaunch"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/nativeui"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/overlay"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/palette"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/roster"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/statusline"
)

// version is stamped by the release build via -ldflags; dev builds say dev.
var version = "dev"

// nativeCodexIntroductionPrompt trails the composed introduction, which names
// the seat first so a session list does not open on identical text (#233).
const nativeCodexIntroductionPrompt = "Introduce yourself now as the active Codex seat, drawing on your loaded identity card and personality meld. Keep it warm and concise, then ask what the human would like to work on."

// nestedLaunchFlag is accepted and ignored. A nested launch no longer needs
// announcing, and dropping the flag outright would break callers. #403
const nestedLaunchFlag = "--nested"

// splitNativeLaunchFlags peels agent-compose's own launch flags off the head.
// `launch` skips flag parsing, so this is the only place one is read.
func splitNativeLaunchFlags(args []string) []string {
	if len(args) > 0 && args[0] == nestedLaunchFlag {
		return args[1:]
	}
	return args
}

// nestedLaunchSkipsConverge reports a launch inheriting a converged host from
// its parent. agent-compose#348
func nestedLaunchSkipsConverge(childDepth int) bool { return childDepth > 0 }

// nestedLaunchNotice keeps both sentinel call sites saying one thing.
func nestedLaunchNotice(step string) string {
	return "agent-compose: nested launch detected; skipping " + step
}

// dispatchArgs makes the acompose install name behave as the compose verb,
// so the daily command is dash-free and stutter-free.
func dispatchArgs(args []string) []string {
	base := args[0]
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	if strings.TrimSuffix(base, ".exe") != "acompose" {
		return args
	}
	if len(args) >= 2 && args[1] == "statusline" {
		return args
	}
	if rest := splitNativeLaunchFlags(args[1:]); len(rest) >= 2 &&
		!strings.HasPrefix(rest[0], "-") && nativeHarness(rest[1]) {
		return append([]string{args[0], "launch"}, args[1:]...)
	}
	return append([]string{args[0], "compose"}, args[1:]...)
}

func nativeHarness(value string) bool {
	switch value {
	case "claude", "codex", "goose", "opencode":
		return true
	default:
		return false
	}
}

func main() {
	cmd := &cli.Command{
		Name:    "agent-compose",
		Usage:   "compose personality context into an immutable bundle",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "print the build version",
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Println(version)
					return nil
				},
			},
			{
				Name:  "config",
				Usage: "inspect host configuration",
				Commands: []*cli.Command{{
					Name:      "validate",
					Usage:     "strictly validate one agent-compose.yaml without converging",
					ArgsUsage: "<agent-compose.yaml>",
					Action:    runConfigValidate,
				}},
			},
			{
				Name:      "compose",
				Usage:     "converge the host, compose a bundle, or refresh then exec after --",
				ArgsUsage: "[request.kdl] [-- <command> [args...]]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Usage: "bundle output directory (defaults to ~/.agent-compose/bundles)",
					},
					&cli.BoolFlag{
						Name:  "explain",
						Usage: "add the role briefing, the expressions, and the decision tree",
					},
					&cli.BoolFlag{
						Name:  "reapply",
						Usage: "rewrite the host compose layout even when it is current",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "report the work a successful run did, which is otherwise silent",
					},
					&cli.StringFlag{
						Name:  "layout",
						Usage: "load-point layout for the exec path (with a request)",
					},
					&cli.StringFlag{
						Name:  "target",
						Value: ".",
						Usage: "directory receiving the load points on the exec path",
					},
				},
				Action: runCompose,
			},
			{
				Name:      "statusline",
				Usage:     "render the nearest projected role identity and composition health",
				ArgsUsage: "",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "target",
						Value: ".",
						Usage: "file or directory used to find the nearest projection",
					},
					&cli.BoolFlag{
						Name:  "color",
						Usage: "emit ANSI identity colors even when stdout is not a terminal",
					},
					&cli.BoolFlag{
						Name:  "subagent",
						Usage: "read a Claude Code subagent tick from stdin and emit one JSON row per agent",
					},
				},
				Action: runStatusline,
			},
			{
				Name:      "whoami",
				Usage:     "print what this session calls itself, for a SessionStart hook",
				ArgsUsage: "",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "target",
						Value: ".",
						Usage: "file or directory used to find the nearest projection",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "emit seat, role, and bundle fingerprint for a transcript join",
					},
				},
				Action: runWhoami,
			},
			{
				Name:            "launch",
				Usage:           "launch one native harness with a caller-assigned role bundle",
				ArgsUsage:       "<role> <harness> [harness arguments...]",
				Description:     "A launch from inside a launched session starts a second seat, one hop deep. See docs/native-role-launch.md.",
				SkipFlagParsing: true,
				Action:          runNativeLaunch,
			},
			{
				Name: "bundle", Usage: "inspect or export verified bundles",
				Commands: []*cli.Command{
					{
						Name: "export", Usage: "write a deterministic verified .tar.gz archive", ArgsUsage: "<bundle-dir>",
						Flags:  []cli.Flag{&cli.StringFlag{Name: "out", Required: true, Usage: "archive output path"}},
						Action: runBundleExport,
					},
					{
						Name: "materialize", Usage: "materialize one verified host role bundle without launching a harness",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "role", Required: true, Usage: "caller-assigned role"},
							&cli.StringFlag{Name: "harness", Required: true, Usage: "harness layout: claude, codex, goose, or opencode"},
							&cli.StringFlag{Name: "out", Usage: "bundle cache directory"},
						},
						Action: runBundleMaterialize,
					},
				},
			},
			{
				Name:  "catalog",
				Usage: "inspect the selected local profile catalogue",
				Commands: []*cli.Command{
					{
						Name: "personalities", Usage: "list personalities or resolve one cue",
						Description: "JSON items: slug, skill, description, aliases, identity primitives, source_library, digest, and affinities.",
						Flags:       personCatalogFlags(true), Action: runCatalogPersonalities,
					},
					{
						Name: "roles", Usage: "list profile roles",
						Description: "JSON items: slug, purpose, role skill provenance, seats, ordered personalities, and favorite_color.",
						Flags:       personCatalogFlags(false), Action: runCatalogRoles,
					},
					{
						Name: "seats", Usage: "list profile seats",
						Description: "JSON items: role plus the complete stable seat key, name, pronouns, channel, and tier.",
						Flags:       append(personCatalogFlags(false), &cli.StringFlag{Name: "role", Usage: "limit to one role"}), Action: runCatalogSeats,
					},
					{
						Name: "boundaries", Usage: "one-token-per-cell boundary assignment across roles",
						Description: "JSON items: boundary, owner, and a verb per role. Text is a fixed-order matrix narrow enough to sit beside a second copy.",
						Flags:       personCatalogFlags(false), Action: runCatalogBoundaries,
					},
					{
						Name: "expressions", Usage: "list stable expression vocabulary",
						Description: "JSON items are the stable expression strings.",
						Flags:       []cli.Flag{&cli.BoolFlag{Name: "json", Usage: "emit agent-compose.catalog.v1 JSON"}}, Action: runCatalogExpressions,
					},
				},
			},
			{
				Name:      "describe",
				Usage:     "render a bundle's stored decision tree",
				ArgsUsage: "<bundle-dir>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "why",
						Usage: "follow one item (e.g. skill:personality-tenacious) to its outcome",
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: "expand collapsed exclusion groups",
					},
				},
				Action: runDescribe,
			},
			{
				Name:      "diff",
				Usage:     "report semantic decision changes between two bundles",
				ArgsUsage: "<left-bundle> <right-bundle>",
				Action:    runDiff,
			},
			{
				Name:      "verify",
				Usage:     "verify that a bundle is complete and safe to consume",
				ArgsUsage: "<bundle-dir>",
				Action:    runVerify,
			},
			{
				Name:  "overlay",
				Usage: "project one member identity and caller-supplied expression",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "role",
						Usage:    "canonical role",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "seat",
						Usage:    "harness seat within the role",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "expression",
						Value: "available",
						Usage: "renderer expression supplied by the caller",
					},
					&cli.IntFlag{
						Name:  "width",
						Value: 40,
						Usage: "maximum text width",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "emit the versioned renderer document",
					},
					&cli.StringFlag{
						Name:  "person-source",
						Usage: "external roster-package root (defaults to embedded roster:core)",
					},
					&cli.StringSliceFlag{
						Name:  "personality-library",
						Usage: "additional local personality-library root (repeatable)",
					},
				},
				Action: runOverlay,
			},
			{
				Name:   "cascade",
				Hidden: true,
				Usage:  "compose doctrine sources into harness global load points",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "print the plan, change nothing",
					},
					&cli.BoolFlag{
						Name:  "check",
						Usage: "verify composed outputs are in sync; exit 1 on drift",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Usage: "report the work a successful run did, which is otherwise silent",
					},
				},
				Action: runCascade,
			},
			{
				Name:      "roster",
				Hidden:    true,
				Usage:     "render the seat dispatch table as a v1-cascade source",
				ArgsUsage: "[source-path...]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Usage: "artifact directory (defaults to ~/.agent-compose/sources/personality)",
					},
					&cli.StringFlag{
						Name:  "person-source",
						Usage: "external roster-package root (defaults to embedded roster:core)",
					},
					&cli.StringSliceFlag{
						Name:  "personality-library",
						Usage: "additional local personality-library root (repeatable)",
					},
				},
				Action: runRoster,
			},
			{
				Name:   "palette-snapshot",
				Hidden: true,
				Usage:  "regenerate the committed role palette page, or verify it is current",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Usage: "write the page to this path",
						Value: palette.SnapshotPath,
					},
					&cli.BoolFlag{
						Name:  "check",
						Usage: "fail when the committed page is stale instead of rewriting it",
					},
				},
				Action: runPaletteSnapshot,
			},
			{
				Name:   "palette-data",
				Hidden: true,
				Usage:  "render canonical personality and role colors for the local palette explorer",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "out",
						Usage:    "write JSON to this path instead of stdout",
						Required: false,
					},
					&cli.StringFlag{
						Name:  "person-source",
						Usage: "external roster-package root (defaults to embedded roster:core)",
					},
					&cli.StringSliceFlag{
						Name:  "personality-library",
						Usage: "additional local personality-library root (repeatable)",
					},
				},
				Action: runPaletteData,
			},
			{
				Name:  "native-ui",
				Usage: "emit Claude Code themes and settings fragments for every role",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Usage: "write theme and settings files under this directory instead of stdout",
					},
					&cli.StringFlag{
						Name:  "role",
						Usage: "emit one role instead of the whole catalogue",
					},
					&cli.StringFlag{
						Name:  "spinner-mode",
						Value: "replace",
						Usage: "replace the harness spinner verbs or append to them",
					},
					&cli.StringFlag{
						Name:  "person-source",
						Usage: "external roster-package root (defaults to embedded roster:core)",
					},
					&cli.StringSliceFlag{
						Name:  "personality-library",
						Usage: "additional local personality-library root (repeatable)",
					},
				},
				Action: runNativeUI,
			},
			{
				Name:      "project",
				Usage:     "transactionally place a verified bundle at harness load points",
				ArgsUsage: "<bundle-dir>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "layout",
						Usage:    "load-point layout name",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "target",
						Value: ".",
						Usage: "directory receiving the load points",
					},
					&cli.StringFlag{
						Name:  "scope",
						Value: project.ScopeRepo,
						Usage: "load-point scope: repo (project files) or home (global files)",
					},
				},
				Action: runProject,
			},
		},
	}
	if err := cmd.Run(context.Background(), dispatchArgs(os.Args)); err != nil {
		fmt.Fprintf(os.Stderr, "agent-compose: %v\n", err)
		os.Exit(1)
	}
}

func personCatalogFlags(includeQuery bool) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "person-source", Usage: "external roster-package root (defaults to embedded roster:core)"},
		&cli.StringSliceFlag{Name: "personality-library", Usage: "additional local personality-library root (repeatable)"},
		&cli.BoolFlag{Name: "json", Usage: "emit agent-compose.catalog.v1 JSON"},
	}
	if includeQuery {
		flags = append(flags, &cli.StringFlag{Name: "query", Usage: "personality slug or declared cue"})
	}
	return flags
}

func runConfigValidate(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("config validate needs exactly one agent-compose.yaml path")
	}
	_, err := cascade.LoadConfig(cmd.Args().First())
	return err
}

func runBundleExport(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("bundle export needs exactly one bundle directory")
	}
	return bundle.Export(cmd.Args().First(), cmd.String("out"))
}

func runBundleMaterialize(_ context.Context, cmd *cli.Command) error {
	role := strings.TrimSpace(cmd.String("role"))
	harness := strings.TrimSpace(cmd.String("harness"))
	if !nativeHarness(harness) {
		return fmt.Errorf("unsupported native harness %q: want claude, codex, goose, or opencode", harness)
	}
	paths := cascade.DefaultPaths()
	if code := converge.Run(paths, converge.Options{}, os.Stderr, os.Stderr); code != 0 {
		return cli.Exit("", code)
	}
	stateDir, err := home.Dir()
	if err != nil {
		return fmt.Errorf("resolve agent-compose state: %w", err)
	}
	outDir := strings.TrimSpace(cmd.String("out"))
	if outDir == "" {
		outDir = filepath.Join(stateDir, "bundles")
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("resolve bundle materialization directory: %w", err)
	}
	personSelection, err := loadHostPersonOptions(paths)
	if err != nil {
		return err
	}
	result, err := nativelaunch.Refresh(nativelaunch.Options{
		Role: role, Harness: harness, CWD: cwd, TargetDir: cwd,
		PlanPath: filepath.Join(filepath.Dir(paths.Composed), "repository-plan.yaml"),
		OutDir:   outDir, PersonSelection: personSelection, SkipProjection: true,
	})
	if err != nil {
		return err
	}
	printCompositionWarnings(os.Stderr, result.Composition.Resolution.Warnings)
	payload := struct {
		Format string `json:"format"`
		Bundle string `json:"bundle"`
	}{Format: "agent-compose.bundle-selection.v1", Bundle: result.BundleDir}
	return json.NewEncoder(os.Stdout).Encode(payload)
}

func writeCatalog(value any, asJSON bool, text string) error {
	if !asJSON {
		_, err := fmt.Fprint(os.Stdout, text)
		return err
	}
	raw, err := json.MarshalIndent(map[string]any{"format": "agent-compose.catalog.v1", "items": value}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(raw))
	return err
}

func runCatalogPersonalities(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	var names []string
	if query := cmd.String("query"); query != "" {
		names, err = p.LookupCue(query)
		if err != nil {
			return err
		}
		if names == nil {
			names = []string{}
		}
	}
	entries, err := p.PersonalityCatalog(names)
	if err != nil {
		return err
	}
	var text strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(
			&text,
			"%s // %s // %s // aliases: %s // roles: %s\n",
			entry.Slug,
			entry.Skill,
			entry.Description,
			strings.Join(entry.Aliases, ", "),
			catalogAffinityRoles(entry.Affinities),
		)
	}
	return writeCatalog(entries, cmd.Bool("json"), text.String())
}

func runCatalogRoles(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	entries, err := p.RoleCatalog()
	if err != nil {
		return err
	}
	var text strings.Builder
	for _, entry := range entries {
		text.WriteString(catalogRoleLine(entry))
	}
	return writeCatalog(entries, cmd.Bool("json"), text.String())
}

// The label tracks the `personalities` field rather than the meld heading, so
// this line and `catalog personalities` name the same thing.
func catalogRoleLine(entry person.RoleCatalogEntry) string {
	return fmt.Sprintf(
		"%s // %s // %s // personalities: %s // color: %s\n",
		entry.Slug,
		entry.Skill,
		entry.Purpose,
		strings.Join(entry.Personalities, ", "),
		entry.FavoriteColor,
	)
}

func runCatalogBoundaries(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	roles, entries := p.BoundaryMatrix()
	return writeCatalog(entries, cmd.Bool("json"), boundaryMatrixText(roles, entries))
}

// Columns are the widest cell so two runs line up, which is the whole point:
// the one-owner one-scope invariant is invisible in prose. agent-compose#325
func boundaryMatrixText(roles []string, entries []person.BoundaryMatrixEntry) string {
	nameWidth := len("boundary")
	for _, entry := range entries {
		nameWidth = max(nameWidth, len(entry.Boundary))
	}
	widths := make([]int, len(roles))
	for i, role := range roles {
		widths[i] = len(role)
		for _, entry := range entries {
			widths[i] = max(widths[i], len(entry.Verbs[role]))
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%-*s", nameWidth, "boundary")
	for i, role := range roles {
		fmt.Fprintf(&out, "  %-*s", widths[i], role)
	}
	out.WriteString("\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "%-*s", nameWidth, entry.Boundary)
		for i, role := range roles {
			fmt.Fprintf(&out, "  %-*s", widths[i], entry.Verbs[role])
		}
		out.WriteString("\n")
	}
	return out.String()
}

func runCatalogSeats(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	seats, err := p.SeatCatalog(cmd.String("role"))
	if err != nil {
		return err
	}
	var text strings.Builder
	for _, entry := range seats {
		fmt.Fprintf(
			&text,
			"%s // %s // %s // %s // channel: %s // tier: %s\n",
			entry.Role,
			entry.Seat.Selector(),
			entry.Seat.Name,
			entry.Seat.Pronouns,
			entry.Seat.Channel,
			entry.Seat.Tier,
		)
	}
	return writeCatalog(seats, cmd.Bool("json"), text.String())
}

func catalogAffinityRoles(affinities []person.PersonalityMeld) string {
	roles := make([]string, 0, len(affinities))
	for _, affinity := range affinities {
		roles = append(roles, affinity.Role)
	}
	return strings.Join(roles, ", ")
}

func runCatalogExpressions(_ context.Context, cmd *cli.Command) error {
	expressions := person.ExpressionVocabulary()
	return writeCatalog(expressions, cmd.Bool("json"), strings.Join(expressions, "\n")+"\n")
}

func runOverlay(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	doc, err := overlay.Build(
		p,
		cmd.String("role"),
		cmd.String("seat"),
		cmd.String("expression"),
	)
	if err != nil {
		return err
	}
	if cmd.Bool("json") {
		raw, err := overlay.Marshal(doc)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(raw)
		return err
	}
	rendered, err := overlay.RenderText(doc, cmd.Int("width"))
	if err != nil {
		return err
	}
	fmt.Print(rendered)
	return nil
}

func runCompose(_ context.Context, cmd *cli.Command) error {
	command := argvAfterDash()
	args := cmd.Args().Slice()
	if len(command) > 0 && len(args) >= len(command) {
		args = args[:len(args)-len(command)]
	}
	requestPath := ""
	if len(args) > 0 {
		requestPath = args[0]
	}
	if requestPath != "" && (cmd.Bool("reapply") || cmd.Bool("verbose")) {
		return fmt.Errorf("--reapply and --verbose apply only to host convergence without a request")
	}

	if len(command) > 0 {
		return refreshThenExec(cmd, requestPath, command)
	}
	if cmd.String("layout") != "" {
		return fmt.Errorf("--layout drives the exec path; add `-- <command>` (or use the project verb)")
	}
	if requestPath == "" {
		if code := converge.Run(cascade.DefaultPaths(), converge.Options{
			Reapply: cmd.Bool("reapply"),
			Verbose: cmd.Bool("verbose"),
		}, os.Stdout, os.Stderr); code != 0 {
			return cli.Exit("", code)
		}
		return nil
	}
	outDir := cmd.String("out")
	if outDir == "" {
		stateDir, err := home.Dir()
		if err != nil {
			return fmt.Errorf("no --out given and no state dir: %w", err)
		}
		outDir = filepath.Join(stateDir, "bundles")
	}
	hostPerson, err := loadHostPersonOptions(cascade.DefaultPaths())
	if err != nil {
		return err
	}
	result, err := compose.RunWithOptions(requestPath, outDir, hostPerson)
	if err != nil {
		return err
	}
	printCompositionWarnings(os.Stderr, result.Resolution.Warnings)
	summaryOpts := person.RoleTranscriptOptions{
		Color:     colorEnabled(),
		TrueColor: trueColorTerminal(),
		Expanded:  cmd.Bool("explain"),
	}
	if err := printSummary(os.Stdout, result, summaryOpts); err != nil {
		return err
	}
	if cmd.Bool("explain") {
		rendered, err := describe.Bundle(result.Bundle.Dir, describe.Options{All: true, Color: colorEnabled(), TrueColor: trueColorTerminal()})
		if err != nil {
			return err
		}
		fmt.Print(rendered)
	}
	return nil
}

func runNativeLaunch(_ context.Context, cmd *cli.Command) error {
	args := splitNativeLaunchFlags(cmd.Args().Slice())
	if len(args) < 2 {
		return fmt.Errorf("launch needs <role> <harness> [harness arguments...]")
	}
	role := strings.TrimSpace(args[0])
	harness := strings.TrimSpace(args[1])
	if !nativeHarness(harness) {
		return fmt.Errorf(
			"unsupported native harness %q: want claude, codex, goose, or opencode",
			harness,
		)
	}
	childDepth, err := launch.NestedDepth(
		os.Getenv(launch.EnvSentinel),
		os.Getenv(launch.EnvDepth),
	)
	if err != nil {
		return err
	}
	paths := cascade.DefaultPaths()
	verbose := os.Getenv(nativelaunch.EnvVerbose) != ""
	// Convergence status is routine bookkeeping. Its failures still reach
	// stderr, so dropping the running commentary loses no signal.
	convergeStatus := io.Discard
	if verbose {
		convergeStatus = os.Stdout
	}
	// The recursion the sentinel guards is the converge, not the launch. A
	// nested seat skips it exactly as refreshThenExec does. agent-compose#348
	if nestedLaunchSkipsConverge(childDepth) {
		fmt.Fprintln(os.Stderr, nestedLaunchNotice("converge"))
	} else if code := converge.Run(
		paths,
		converge.Options{Verbose: verbose},
		convergeStatus,
		os.Stderr,
	); code != 0 {
		return cli.Exit("", code)
	}
	stateDir, err := home.Dir()
	if err != nil {
		return fmt.Errorf("resolve agent-compose state: %w", err)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("resolve native launch directory: %w", err)
	}
	personSelection, err := loadHostPersonOptions(paths)
	if err != nil {
		return err
	}
	runtimeHome := strings.TrimSpace(os.Getenv(nativelaunch.EnvRuntimeHome))
	// Only a session home replaces the host load point. A repo-scope launch
	// still reads the host file, where repeating the base would double it.
	operatingBase, operatingAppendix := "", ""
	if runtimeHome != "" {
		cfg, err := cascade.LoadConfig(paths.Config)
		if err != nil {
			return fmt.Errorf("load host configuration for the operating base: %w", err)
		}
		operatingBase, operatingAppendix, err = cascade.OperatingBaseParts(cfg, harness, role)
		if err != nil {
			return err
		}
	}
	// Keyed to the depth rather than to a flag, so it covers every nested
	// launch instead of the announced ones. #403
	if childDepth > 0 {
		target := cwd
		if runtimeHome != "" {
			target = runtimeHome
		}
		if err := refuseOwnProjectionReplacement(target); err != nil {
			return err
		}
	}
	result, err := nativelaunch.Refresh(nativelaunch.Options{
		Role:              role,
		Harness:           harness,
		ModelTier:         os.Getenv(nativelaunch.EnvModelTier),
		CWD:               cwd,
		TargetDir:         cwd,
		RuntimeHome:       runtimeHome,
		OperatingBase:     operatingBase,
		OperatingAppendix: operatingAppendix,
		PlanPath:          filepath.Join(filepath.Dir(paths.Composed), "repository-plan.yaml"),
		OutDir:            filepath.Join(stateDir, "bundles"),
		PersonSelection:   personSelection,
	})
	if err != nil {
		return err
	}
	// Selector provenance is routine on a launch and already carried by the
	// decision trace. `agent-compose compose` remains the verb that says it.
	if verbose {
		printCompositionWarnings(os.Stderr, result.Composition.Resolution.Warnings)
	}
	interactive := nativeLaunchInteractive(os.Stdin, os.Stdout)
	summaryOpts := person.RoleTranscriptOptions{
		Color:     colorEnabled(),
		TrueColor: trueColorTerminal(),
	}
	state := "new"
	if result.BundleReused {
		state = "reused"
	}
	if verbose && interactive {
		printNativeLaunchStatus(os.Stderr, role, harness, result, state)
	}
	if err := printNativeLaunchSummary(
		os.Stdout,
		result.Composition,
		summaryOpts,
		summaryLayout{RoleLast: interactive, Audit: verbose},
	); err != nil {
		return err
	}
	if err := acknowledgeNativeLaunch(
		os.Stdin,
		os.Stdout,
		interactive,
	); err != nil {
		return err
	}
	if err := clearNativeLaunchEnvironment(); err != nil {
		return err
	}
	if runtimeHome != "" {
		if err := activateNativeRuntimeHome(runtimeHome, harness); err != nil {
			return err
		}
	}
	if verbose && !interactive {
		printNativeLaunchStatus(os.Stderr, role, harness, result, state)
	}
	return execReal(
		nativeHarnessCommand(harness, args[2:], nativeIdentity{
			SeatName:     result.SeatName,
			Settings:     result.HarnessSettings,
			Introduction: result.Introduction,
		}),
		append(
			append(
				roleAttributionEnv(role),
				launch.DepthEnv(childDepth),
			),
			sessionBundleEnv(result.BundleDir, harness)...,
		)...,
	)
}

func runStatusline(_ context.Context, cmd *cli.Command) error {
	opts := statusline.Options{
		Target:    cmd.String("target"),
		Color:     cmd.Bool("color") || colorEnabled(),
		TrueColor: trueColorTerminal(),
	}
	if cmd.Bool("subagent") {
		// The subagent mode already emits newline-terminated JSON rows.
		rendered, err := statusline.RenderSubagents(os.Stdin, opts)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.Root().Writer, rendered)
		return nil
	}
	rendered, err := statusline.Render(opts)
	if err != nil {
		return err
	}
	if rendered != "" {
		fmt.Fprintln(cmd.Root().Writer, rendered)
	}
	return nil
}

// runWhoami prints the composed name and nothing else, so a shell hook can use
// it without parsing. Silence means no projection. See docs/whoami.md.
func runWhoami(_ context.Context, cmd *cli.Command) error {
	opts := statusline.Options{Target: cmd.String("target")}
	render := statusline.Whoami
	if cmd.Bool("json") {
		render = statusline.WhoamiJSON
	}
	name, err := render(opts)
	if err != nil {
		return err
	}
	if name != "" {
		fmt.Fprintln(cmd.Root().Writer, name)
	}
	return nil
}

func nativeLaunchInteractive(input, output *os.File) bool {
	return term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd()))
}

func printNativeLaunchStatus(
	w io.Writer,
	role, harness string,
	result *nativelaunch.Result,
	state string,
) {
	fmt.Fprintf(
		w,
		"agent-compose: assigned %s to %s (%s tier, %s bundle, %d files)\n",
		role,
		harness,
		result.ModelTier,
		state,
		result.Projected,
	)
}

func acknowledgeNativeLaunch(
	input io.Reader,
	output io.Writer,
	interactive bool,
) error {
	if !interactive {
		return nil
	}
	if _, err := fmt.Fprint(output, "Press Enter to continue"); err != nil {
		return fmt.Errorf("write native launch acknowledgement: %w", err)
	}
	var next [1]byte
	for {
		count, err := input.Read(next[:])
		if count > 0 && next[0] == '\n' {
			if _, writeErr := fmt.Fprintln(output); writeErr != nil {
				return fmt.Errorf("finish native launch acknowledgement: %w", writeErr)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read native launch acknowledgement: %w", err)
		}
	}
}

// nativeIdentity carries the composed surfaces a harness can accept as launch
// arguments rather than as installed host state.
type nativeIdentity struct {
	SeatName     string
	Settings     string
	Introduction string
}

func nativeHarnessCommand(harness string, args []string, identity nativeIdentity) []string {
	command := append([]string{harness}, nativeIdentityArgs(harness, args, identity)...)
	command = append(command, args...)
	if harness == "codex" && codexAcceptsInitialPrompt(args) {
		prompt := nativeCodexIntroductionPrompt
		if identity.Introduction != "" {
			prompt = identity.Introduction + " " + prompt
		}
		command = append(command, prompt)
	}
	return command
}

// nativeIdentityArgs hands Claude Code its identity as flags, per
// docs/claude-launch-identity.md. A caller-supplied flag always wins.
func nativeIdentityArgs(harness string, args []string, identity nativeIdentity) []string {
	if harness != "claude" {
		return nil
	}
	var flags []string
	if identity.SeatName != "" && !nativeArgsCarry(args, "--name") {
		flags = append(flags, "--name", identity.SeatName)
	}
	if identity.Settings != "" && !nativeArgsCarry(args, "--settings") {
		flags = append(flags, "--settings", identity.Settings)
	}
	return flags
}

// nativeArgsCarry reports whether the caller already supplied a flag, in either
// the separate-value or the inline `--flag=value` spelling.
func nativeArgsCarry(args []string, flag string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

func codexAcceptsInitialPrompt(args []string) bool {
	valueOptions := map[string]bool{
		"-a": true, "--add-dir": true, "--ask-for-approval": true,
		"-c": true, "--cd": true, "--config": true,
		"-i": true, "--image": true,
		"-m": true, "--model": true,
		"-p": true, "--profile": true,
		"-s": true, "--sandbox": true,
		"--disable": true, "--enable": true,
		"--local-provider": true, "--remote": true, "--remote-auth-token-env": true,
	}
	booleanOptions := map[string]bool{
		"--dangerously-bypass-approvals-and-sandbox": true,
		"--dangerously-bypass-hook-trust":            true,
		"--no-alt-screen":                            true,
		"--oss":                                      true,
		"--search":                                   true,
		"--strict-config":                            true,
	}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			return false
		}
		name := arg
		if before, _, found := strings.Cut(arg, "="); found {
			name = before
		}
		if valueOptions[name] {
			if strings.Contains(arg, "=") {
				continue
			}
			index++
			if index >= len(args) {
				return false
			}
			continue
		}
		if !booleanOptions[name] {
			return false
		}
	}
	return true
}

func clearNativeLaunchEnvironment() error {
	for _, name := range []string{
		nativelaunch.EnvModelTier,
		"AGENT_COMPOSE_MODEL_CLASS",
		nativelaunch.EnvRuntimeHome,
	} {
		if err := os.Unsetenv(name); err != nil {
			return fmt.Errorf("clear native launch environment %s: %w", name, err)
		}
	}
	return nil
}

func activateNativeRuntimeHome(home, harness string) error {
	absolute, err := filepath.Abs(home)
	if err != nil {
		return fmt.Errorf("resolve native runtime home: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return fmt.Errorf("inspect native runtime home %s: %w", absolute, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("native runtime home %s is not a directory", absolute)
	}
	codexHome := filepath.Join(absolute, ".codex")
	if harness == "codex" {
		if resolved, err := filepath.EvalSymlinks(codexHome); err == nil {
			codexHome = resolved
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("resolve native Codex home %s: %w", codexHome, err)
		}
	}
	environment := map[string]string{
		"HOME":            absolute,
		"USERPROFILE":     absolute,
		"CODEX_HOME":      codexHome,
		"XDG_CONFIG_HOME": filepath.Join(absolute, ".config"),
	}
	if harness == "claude" {
		environment["CLAUDE_CONFIG_DIR"] = filepath.Join(absolute, ".claude")
	}
	for name, value := range environment {
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("set native runtime environment %s: %w", name, err)
		}
	}
	return nil
}

func runDescribe(_ context.Context, cmd *cli.Command) error {
	dir := cmd.Args().First()
	if dir == "" {
		return fmt.Errorf("describe needs a bundle directory")
	}
	opts := describe.Options{All: cmd.Bool("all"), Color: colorEnabled(), TrueColor: trueColorTerminal()}
	var rendered string
	var err error
	if why := cmd.String("why"); why != "" {
		rendered, err = describe.Why(dir, why, opts)
	} else {
		rendered, err = describe.Bundle(dir, opts)
	}
	if err != nil {
		return err
	}
	fmt.Print(rendered)
	return nil
}

func runDiff(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return fmt.Errorf("diff needs exactly two bundle directories")
	}
	rendered, err := describe.Diff(cmd.Args().Get(0), cmd.Args().Get(1))
	if err != nil {
		return err
	}
	fmt.Print(rendered)
	return nil
}

func runVerify(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("verify needs exactly one bundle directory")
	}
	verification, err := bundle.Verify(cmd.Args().First())
	if err != nil {
		return err
	}
	printVerification(os.Stdout, verification)
	return nil
}

func printVerification(w io.Writer, verification *bundle.Verification) {
	fmt.Fprintf(w, "bundle verified: %d skills // %d files\n",
		len(verification.Identities), verification.Files)
}

// refuseOwnProjectionReplacement stops a nested launch from replacing the load
// points its own session runs on. See docs/native-role-launch.md.
func refuseOwnProjectionReplacement(target string) error {
	existing := project.ReadProjection(target)
	if existing.Bundle == "" {
		return nil
	}
	if session := strings.TrimSpace(os.Getenv(launch.SessionBundleEnv)); session != "" &&
		session != existing.Bundle {
		return nil
	}
	return fmt.Errorf(
		"a nested launch in %s would replace the load points this session runs on; launch from another directory or stage %s",
		target,
		nativelaunch.EnvRuntimeHome,
	)
}

// roleAttributionEnv attributes the composed role. Empty yields nothing,
// because a wrong role is worse than none.
func roleAttributionEnv(role string) []string {
	if strings.TrimSpace(role) == "" {
		return nil
	}
	return []string{launch.AttributionRoleEnv + "=" + role}
}

// sessionBundleEnv binds this session to its own composition rather than to
// whatever projection sits nearest the cwd. See docs/whoami.md.
func sessionBundleEnv(bundleDir, layout string) []string {
	if strings.TrimSpace(bundleDir) == "" || strings.TrimSpace(layout) == "" {
		return nil
	}
	return []string{
		launch.SessionBundleEnv + "=" + bundleDir,
		launch.SessionLayoutEnv + "=" + layout,
	}
}

// execRealWithRole execs the target with the role attributed, if there is one.
func execRealWithRole(role string, argv []string) error {
	return execReal(argv, roleAttributionEnv(role)...)
}

// refreshThenExec is the absorbed launch verb: refresh context, then hand
// the process to the real command, sentinel-guarded against recursion.
func refreshThenExec(cmd *cli.Command, requestPath string, command []string) error {
	if os.Getenv(launch.EnvSentinel) != "" {
		fmt.Fprintln(os.Stderr, nestedLaunchNotice("refresh"))
		return execReal(command)
	}
	if requestPath == "" {
		if code := converge.Run(cascade.DefaultPaths(), converge.Options{
			Reapply: cmd.Bool("reapply"),
			Verbose: cmd.Bool("verbose"),
		}, os.Stdout, os.Stderr); code != 0 {
			return cli.Exit("", code)
		}
		return execReal(command)
	}
	layout := cmd.String("layout")
	if layout == "" {
		return fmt.Errorf("a request with `--` needs --layout to place its load points")
	}
	outDir := cmd.String("out")
	if outDir == "" {
		stateDir, err := home.Dir()
		if err != nil {
			return fmt.Errorf("no --out given and no state dir: %w", err)
		}
		outDir = filepath.Join(stateDir, "bundles")
	}
	hostPerson, err := loadHostPersonOptions(cascade.DefaultPaths())
	if err != nil {
		return err
	}
	result, err := launch.Refresh(launch.Options{
		RequestPath:  requestPath,
		Layout:       layout,
		TargetDir:    cmd.String("target"),
		OutDir:       outDir,
		PersonPolicy: hostPerson.PersonPolicy,
		PersonSource: hostPerson.PersonSource,
	})
	if err != nil {
		return err
	}
	// A fallback launch reuses a projection this run did not compose, so the
	// role it would name is not known to be this session's.
	composedRole := ""
	if !result.Fallback && result.BundleDir != "" {
		if manifest, err := bundle.ReadManifest(result.BundleDir); err == nil {
			composedRole = manifest.Role
		}
	}
	if result.Fallback {
		fmt.Fprintf(os.Stderr, "agent-compose: WARNING: refresh failed (%s); launching with the last-known-good projection\n", result.Warning)
	} else {
		printCompositionWarnings(os.Stderr, result.Warnings)
		state := "new"
		if result.BundleReused {
			state = "reused"
		}
		fmt.Fprintf(os.Stderr, "agent-compose: context refreshed (%s bundle, %d files) for %s\n",
			state, result.Projected, cmd.String("target"))
	}
	return execRealWithRole(composedRole, command)
}

func printCompositionWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "agent-compose: warning: %s\n", warning)
	}
}

// argvAfterDash finds the raw command after the `--` terminator, independent
// of how the flag parser folds those tokens into Args.
func argvAfterDash() []string {
	for i, arg := range os.Args {
		if arg == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

// colorEnabled keeps redirected output plain and deterministic; color is a
// TTY-only affordance and NO_COLOR always wins.
func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func trueColorTerminal() bool {
	ct := os.Getenv("COLORTERM")
	return strings.Contains(ct, "truecolor") || strings.Contains(ct, "24bit")
}

func runCascade(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("dry-run") && cmd.Bool("check") {
		return fmt.Errorf("--dry-run and --check are mutually exclusive")
	}
	paths := cascade.DefaultPaths()
	var code int
	if cmd.Bool("check") {
		code = cascade.Check(paths, os.Stdout, os.Stderr)
	} else {
		code = cascade.Run(paths, cascade.RunOptions{
			DryRun:  cmd.Bool("dry-run"),
			Verbose: cmd.Bool("verbose"),
		}, os.Stdout, os.Stderr)
	}
	if code != 0 {
		return cli.Exit("", code)
	}
	return nil
}

func runRoster(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	var sources []*schema.Source
	for _, sourcePath := range cmd.Args().Slice() {
		src, err := schema.LoadSource(sourcePath)
		if err != nil {
			return err
		}
		sources = append(sources, src)
	}
	outFlag := cmd.String("out")
	if outFlag == "" {
		stateDir, err := home.Dir()
		if err != nil {
			return fmt.Errorf("no --out given and no state dir: %w", err)
		}
		outFlag = filepath.Join(stateDir, "sources", "personality")
	}
	outDir, err := filepath.Abs(outFlag)
	if err != nil {
		return err
	}
	files, err := roster.Render(p, sources, outDir)
	if err != nil {
		return err
	}
	result, err := project.ApplyOwned(outDir, files, "roster", p.ProviderID())
	if err != nil {
		return err
	}
	fmt.Printf("roster artifact: %d files under %s\n", len(result.Files), outDir)
	return nil
}

func runPaletteSnapshot(_ context.Context, cmd *cli.Command) error {
	p, err := person.Load()
	if err != nil {
		return err
	}
	rendered, err := palette.RenderSnapshot(p)
	if err != nil {
		return err
	}
	out := cmd.String("out")
	if cmd.Bool("check") {
		return palette.CheckSnapshot(out, rendered)
	}
	return palette.WriteSnapshot(out, rendered)
}

func runPaletteData(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	raw, err := palette.Marshal(p)
	if err != nil {
		return err
	}
	if out := cmd.String("out"); out != "" {
		return palette.Write(out, raw)
	}
	_, err = os.Stdout.Write(raw)
	return err
}

func runNativeUI(_ context.Context, cmd *cli.Command) error {
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), cmd.StringSlice("personality-library"))
	if err != nil {
		return err
	}
	opts := nativeui.Options{SpinnerMode: cmd.String("spinner-mode")}
	if opts.SpinnerMode != "replace" && opts.SpinnerMode != "append" {
		return fmt.Errorf("spinner-mode must be replace or append, got %q", opts.SpinnerMode)
	}

	var bundles []nativeui.Bundle
	if role := cmd.String("role"); role != "" {
		bundle, err := nativeui.BuildRole(p, role, opts)
		if err != nil {
			return err
		}
		bundles = []nativeui.Bundle{bundle}
	} else if bundles, err = nativeui.Build(p, opts); err != nil {
		return err
	}

	out := cmd.String("out")
	if out == "" {
		raw, err := json.MarshalIndent(bundles, "", "  ")
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(append(raw, '\n'))
		return err
	}

	written, err := nativeui.WriteAll(out, bundles)
	if err != nil {
		return err
	}
	fmt.Printf("native-ui artifact: %d files under %s\n", written, out)
	return nil
}

func loadSelectedPerson(source string) (*person.Person, bool, error) {
	return loadSelectedPersonAt(source, cascade.DefaultPaths())
}

func loadSelectedPersonAt(source string, paths cascade.Paths) (*person.Person, bool, error) {
	return loadSelectedPersonWithLibrariesAt(source, nil, paths)
}

func loadSelectedPersonWithLibraries(source string, libraries []string) (*person.Person, bool, error) {
	return loadSelectedPersonWithLibrariesAt(source, libraries, cascade.DefaultPaths())
}

func loadSelectedPersonWithLibrariesAt(source string, libraries []string, paths cascade.Paths) (*person.Person, bool, error) {
	if source == "" {
		hostPerson, err := loadHostPersonOptions(paths)
		if err != nil {
			return nil, false, err
		}
		source = hostPerson.PersonSource
		if source == "" {
			p, err := person.Load()
			return p, false, err
		}
	}
	p, err := person.LoadDirectoryWithLibraries(source, libraries...)
	return p, true, err
}

func loadHostPersonOptions(paths cascade.Paths) (compose.Options, error) {
	if _, err := os.Stat(paths.Config); err != nil {
		if os.IsNotExist(err) {
			return compose.Options{}, nil
		}
		return compose.Options{}, fmt.Errorf("inspect host config %s: %w", paths.Config, err)
	}
	cfg, err := cascade.LoadConfig(paths.Config)
	if err != nil {
		return compose.Options{}, err
	}
	if cfg.PersonPolicy == "" {
		return compose.Options{}, nil
	}
	return compose.Options{
		PersonPolicy: cfg.PersonPolicy,
		PersonSource: cascade.ResolveConfiguredPath(
			cfg.PersonSource,
			paths.Config,
			paths.Home,
		),
		PersonalityLibraries: resolveConfiguredPaths(cfg.PersonalityLibraries, paths.Config, paths.Home),
	}, nil
}

func resolveConfiguredPaths(values []string, configPath, home string) []string {
	paths := make([]string, 0, len(values))
	for _, value := range values {
		if person.IsCoreLibrary(value) {
			paths = append(paths, person.CoreLibraryRoot)
			continue
		}
		paths = append(paths, cascade.ResolveConfiguredPath(value, configPath, home))
	}
	return paths
}

func runProject(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("project needs exactly one bundle directory")
	}
	bundleDir := cmd.Args().First()
	hostPerson, err := loadHostPersonOptions(cascade.DefaultPaths())
	if err != nil {
		return err
	}
	if err := validateProjectPersonPolicy(bundleDir, hostPerson); err != nil {
		return err
	}
	result, err := project.ProjectScoped(bundleDir, cmd.String("layout"), cmd.String("target"), cmd.String("scope"))
	if err != nil {
		return err
	}
	fmt.Printf("projected %d files into layout %s (%s scope) under %s\n",
		len(result.Files), result.Layout, cmd.String("scope"), cmd.String("target"))
	return nil
}

func validateProjectPersonPolicy(bundleDir string, opts compose.Options) error {
	if opts.PersonPolicy != personpolicy.ExternalOnly {
		return nil
	}
	verification, err := bundle.Verify(bundleDir)
	if err != nil {
		return err
	}
	external := false
	for _, identity := range verification.Identities {
		if identity.Source == "roster:core" {
			return fmt.Errorf("external-only person policy rejects bundle source %q", identity.Source)
		}
		if strings.HasPrefix(identity.Source, "person:") {
			external = true
		}
	}
	if external {
		return nil
	}
	return fmt.Errorf("external-only person policy requires an external person bundle")
}

// summaryLayout selects what a composition summary prints and where.
type summaryLayout struct {
	// RoleLast leaves the identity transcript as the last block on screen,
	// which a launch wants and a plain compose does not.
	RoleLast bool
	// Audit adds the bundle intro and the routine source, decision, and path
	// counts around the transcript.
	Audit bool
}

// printSummary renders the complete selected-role identity transcript followed
// by bounded composition audit counts.
func printSummary(
	w io.Writer,
	r *compose.Result,
	opts person.RoleTranscriptOptions,
) error {
	return printNativeLaunchSummary(w, r, opts, summaryLayout{Audit: true})
}

func printNativeLaunchSummary(
	w io.Writer,
	r *compose.Result,
	opts person.RoleTranscriptOptions,
	layout summaryLayout,
) error {
	req := r.Resolution.Request
	metadata, err := r.Resolution.Person.RenderRoleTranscript(
		req.Role,
		r.Resolution.FavoriteColor,
		opts,
	)
	if err != nil {
		return err
	}
	if !layout.Audit {
		fmt.Fprint(w, metadata)
		return nil
	}
	state := "new"
	if r.Bundle.Reused {
		state = "reused"
	}
	counts := map[string]int{}
	for _, d := range r.Resolution.Decisions {
		counts[d.Outcome]++
	}
	intro := fmt.Sprintf(
		"bundle %s (%s)\nrequest: model tier %s // delivery %s\n",
		r.Bundle.Key, state, req.ModelTier, req.Delivery,
	)
	if opts.Color {
		intro = color.ANSI(r.Resolution.FavoriteColor, intro, opts.TrueColor)
	}
	fmt.Fprintln(w, intro)
	var audit strings.Builder
	fmt.Fprintf(&audit, "sources: %s\n", strings.Join(r.Resolution.SourceIDs, ", "))
	fmt.Fprintf(&audit, "decisions: %d selected // %d excluded // %d shadowed // %d delivered\n",
		counts[resolver.OutcomeSelected], counts[resolver.OutcomeExcluded],
		counts[resolver.OutcomeShadowed], counts[resolver.OutcomeDelivered])
	fmt.Fprintf(&audit, "path: %s\n", r.Bundle.Dir)
	fmt.Fprintf(&audit, "trace: %s\n", filepath.Join(r.Bundle.Dir, "trace.json"))
	if layout.RoleLast {
		fmt.Fprint(w, audit.String())
		fmt.Fprintln(w)
		fmt.Fprint(w, metadata)
		return nil
	}
	fmt.Fprint(w, metadata)
	fmt.Fprintln(w)
	fmt.Fprint(w, audit.String())
	return nil
}
