package cascade

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/home"
)

// Paths carries the injectable filesystem anchors so tests never touch the
// real home; DefaultPaths resolves the live locations.
type Paths struct {
	Config       string
	Composed     string
	ProjectsRoot string
	Home         string
}

// RunOptions controls preview, forced application, and layout reporting.
type RunOptions struct {
	DryRun  bool
	Reapply bool
	Verbose bool
}

func DefaultPaths() Paths {
	root, _ := os.UserHomeDir()
	projects := os.Getenv("PROJECTS_ROOT")
	if projects == "" {
		projects = filepath.Join(root, "projects")
	}
	stateDir, err := home.Dir()
	if err != nil {
		stateDir = filepath.Join(root, ".agent-compose")
	}
	return Paths{
		Config:       filepath.Join(stateDir, "agent-compose.yaml"),
		Composed:     filepath.Join(stateDir, "COMPOSED.md"),
		ProjectsRoot: projects,
		Home:         root,
	}
}

type planned struct {
	sources   []string
	overrides map[string]string
}

func buildPlan(cfg *Config, paths Paths, stderr io.Writer, strict bool) (map[string]planned, plan, map[string]string, string, int) {
	if err := ValidateSources(cfg); err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return nil, plan{}, nil, "", 1
	}
	gathered, errs := GatherSources(cfg)
	appendix, appendixErrs := GatherAppendix(cfg)
	errs = append(errs, appendixErrs...)
	if strict && len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(stderr, "agent-compose: %s\n", err)
		}
		return nil, plan{}, nil, "", 1
	}
	for _, err := range errs {
		fmt.Fprintf(stderr, "agent-compose: warning: %s (skipped)\n", err)
	}

	filtering := cfg.Scopes != nil
	var machineScopes []string
	if filtering {
		machineScopes = *cfg.Scopes
	}
	sources := SelectByScope(gathered, machineScopes, filtering)
	excluded := len(gathered) - len(sources)
	if len(sources) == 0 {
		reason := "no sources resolved (check `sources` / `roots`)"
		if filtering {
			reason = fmt.Sprintf("no sources matched this machine's scopes %v", machineScopes)
		}
		fmt.Fprintf(stderr, "agent-compose: config present but %s; refusing to write an empty COMPOSED.md\n", reason)
		return nil, plan{}, nil, "", 1
	}

	loadPoints, err := ResolveLoadPoints(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %s\n", err)
		return nil, plan{}, nil, "", 1
	}
	p := planOutputs(sources, loadPoints, paths.Composed)
	p.appendix = appendix
	p.delivery = cfg.SourceDelivery
	p.importable = ConfiguredSources(cfg)
	if len(p.errors) > 0 {
		for _, err := range p.errors {
			fmt.Fprintf(stderr, "agent-compose: %s\n", err)
		}
		return nil, plan{}, nil, "", 1
	}

	byTarget := map[string]planned{}
	for harness, target := range p.outputs {
		byTarget[target] = planned{sources: p.slices[harness], overrides: p.overrides[harness]}
	}
	if len(loadPoints) == 0 {
		byTarget[paths.Composed] = planned{sources: sources, overrides: map[string]string{}}
	}
	tail := ""
	if excluded > 0 {
		tail = fmt.Sprintf(" (%d excluded by scope)", excluded)
	}
	return byTarget, p, loadPoints, tail, 0
}

// Run composes and wires symlinks; absent config is a documented no-op.
func Run(paths Paths, opts RunOptions, stdout, stderr io.Writer) int {
	if _, err := os.Stat(paths.Config); err != nil {
		fmt.Fprintf(stdout, "agent-compose: no config at %s; nothing to do (opt-in)\n", paths.Config)
		return 0
	}
	cfg, err := LoadConfig(paths.Config)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	byTarget, p, loadPoints, tail, code := buildPlan(cfg, paths, stderr, false)
	if code != 0 {
		return code
	}
	active := map[string]bool{}
	for target := range byTarget {
		active[target] = true
	}
	stale := staleGeneratedOutputs(paths.Composed, active)
	manifestPath := planPath(filepath.Dir(paths.Composed))
	legacyPlanPath := filepath.Join(filepath.Dir(paths.Composed), "repository-plan.json")
	repositoryPlan, err := RenderRepositoryPlan(cfg, paths.ProjectsRoot)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}

	if opts.Verbose {
		printLayout(stdout, paths.Config, manifestPath, byTarget, p, loadPoints)
	}

	if opts.DryRun {
		for _, target := range sortedKeys(byTarget) {
			entry := byTarget[target]
			body, err := composeTarget(p, entry.sources, entry.overrides)
			if err != nil {
				fmt.Fprintf(stderr, "agent-compose: %v\n", err)
				return 1
			}
			if opts.Reapply || readOr(target, "\x00") != body {
				fmt.Fprintf(stdout, "would write %s (%d source(s))%s\n", target, len(entry.sources), tail)
			}
		}
		if opts.Reapply || readOr(manifestPath, "\x00") != repositoryPlan {
			fmt.Fprintf(stdout, "would write %s (repository plan)\n", manifestPath)
		}
		if _, err := os.Stat(legacyPlanPath); err == nil {
			fmt.Fprintf(stdout, "would remove %s (obsolete repository plan)\n", legacyPlanPath)
		}
		for _, target := range stale {
			fmt.Fprintf(stdout, "would remove %s (obsolete generated output)\n", target)
		}
		for _, harness := range sortedKeys2(loadPoints) {
			if opts.Reapply || !symlinkUpToDate(loadPoints[harness], p.outputs[harness]) {
				fmt.Fprintf(stdout, "would link  %s -> %s  [%s]\n", loadPoints[harness], p.outputs[harness], harness)
			}
		}
		return 0
	}

	if err := os.MkdirAll(filepath.Dir(paths.Composed), 0o755); err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	changed := 0
	for _, target := range stale {
		os.Remove(target)
		fmt.Fprintf(stdout, "removed %s (obsolete generated output)\n", target)
		changed++
	}
	if _, err := os.Stat(legacyPlanPath); err == nil {
		if err := os.Remove(legacyPlanPath); err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "removed %s (obsolete repository plan)\n", legacyPlanPath)
		changed++
	}
	for _, target := range sortedKeys(byTarget) {
		entry := byTarget[target]
		body, err := composeTarget(p, entry.sources, entry.overrides)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		if !opts.Reapply && readOr(target, "\x00") == body {
			continue
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		if opts.Verbose {
			fmt.Fprintf(stdout, "wrote   %s (%d source(s))%s\n", target, len(entry.sources), tail)
		}
		changed++
	}
	if opts.Reapply || readOr(manifestPath, "\x00") != repositoryPlan {
		if err := writeFileAtomic(manifestPath, []byte(repositoryPlan)); err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		if opts.Verbose {
			fmt.Fprintf(stdout, "wrote   %s (repository plan)\n", manifestPath)
		}
		changed++
	}
	for _, harness := range sortedKeys2(loadPoints) {
		line, err := installSymlink(loadPoints[harness], p.outputs[harness], opts.Reapply)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		if line != "" {
			if opts.Verbose {
				fmt.Fprintf(stdout, "%s  [%s]\n", line, harness)
			}
			changed++
		}
	}
	if opts.Verbose {
		fmt.Fprintf(stdout, "cascade outputs=%d load-points=%d repository-plan=1 changed=%d\n",
			len(byTarget), len(loadPoints), changed)
	}
	return 0
}

// printLayout emits deterministic file flow through composed outputs and
// harness load points, including per-harness override inputs.
func printLayout(
	w io.Writer,
	configPath string,
	manifestPath string,
	byTarget map[string]planned,
	p plan,
	loadPoints map[string]string,
) {
	for _, target := range sortedKeys(byTarget) {
		entry := byTarget[target]
		for _, source := range entry.sources {
			fmt.Fprintf(w, "layout  %s => %s\n", source, target)
			if override := entry.overrides[source]; override != "" {
				fmt.Fprintf(w, "layout  %s => %s\n", override, target)
			}
		}
	}
	// Listed by destination, not target: a role slug that names nothing is
	// otherwise invisible until a seat launches without its block.
	for _, block := range p.appendix {
		destination := "every composed output"
		if len(block.Roles) > 0 {
			destination = "role bundles: " + strings.Join(block.Roles, ", ")
		}
		fmt.Fprintf(w, "layout  %s => %s\n", block.Fence, destination)
	}
	fmt.Fprintf(w, "layout  %s => %s\n", configPath, manifestPath)
	for _, harness := range sortedKeys2(loadPoints) {
		fmt.Fprintf(w, "layout  %s => %s\n", p.outputs[harness], loadPoints[harness])
	}
}

// Check verifies on-disk outputs match a fresh compose; drift prints a
// unified diff and fails, mirroring the v1 pre-commit hook.
func Check(paths Paths, stdout, stderr io.Writer) int {
	if _, err := os.Stat(paths.Config); err != nil {
		fmt.Fprintf(stdout, "agent-compose: no config at %s; nothing to check (opt-in)\n", paths.Config)
		return 0
	}
	cfg, err := LoadConfig(paths.Config)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	byTarget, p, _, _, code := buildPlan(cfg, paths, stderr, true)
	if code != 0 {
		return code
	}
	active := map[string]bool{}
	for target := range byTarget {
		active[target] = true
	}
	drifted := false
	for _, target := range staleGeneratedOutputs(paths.Composed, active) {
		drifted = true
		fmt.Fprintf(stderr, "agent-compose: drift - obsolete generated output remains at %s. Run `agent-compose cascade` to remove it.\n", target)
	}
	for _, target := range sortedKeys(byTarget) {
		entry := byTarget[target]
		expected, err := composeTarget(p, entry.sources, entry.overrides)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		actual := readOr(target, "")
		if actual == expected {
			fmt.Fprintf(stdout, "agent-compose: %s in sync\n", target)
			continue
		}
		drifted = true
		fmt.Fprintf(stderr, "agent-compose: drift - %s is missing, stale, or hand-edited. Run `agent-compose cascade` to regenerate.\n", target)
		writeDiff(stderr, actual, expected, target)
	}
	manifestPath := planPath(filepath.Dir(paths.Composed))
	legacyPlanPath := filepath.Join(filepath.Dir(paths.Composed), "repository-plan.json")
	expectedPlan, err := RenderRepositoryPlan(cfg, paths.ProjectsRoot)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	if actual := readOr(manifestPath, ""); actual == expectedPlan {
		fmt.Fprintf(stdout, "agent-compose: %s in sync\n", manifestPath)
	} else {
		drifted = true
		fmt.Fprintf(stderr, "agent-compose: drift - %s (repository plan) is missing, stale, or hand-edited. Run `agent-compose compose` to regenerate.\n", manifestPath)
		writeDiff(stderr, actual, expectedPlan, manifestPath)
	}
	if _, err := os.Stat(legacyPlanPath); err == nil {
		drifted = true
		fmt.Fprintf(stderr, "agent-compose: drift - obsolete repository plan remains at %s. Run `agent-compose cascade` to remove it.\n", legacyPlanPath)
	}
	if drifted {
		return 1
	}
	return 0
}

// writeDiff prints a bounded minus/plus diff of the first divergent region;
// simpler than v1's difflib but the same job: make staleness legible.
func writeDiff(w io.Writer, actual, expected, name string) {
	fmt.Fprintf(w, "--- %s (on disk)\n+++ expected (fresh compose)\n", name)
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")
	emitted := 0
	for i := 0; emitted < 40 && (i < len(actualLines) || i < len(expectedLines)); i++ {
		var a, e string
		if i < len(actualLines) {
			a = actualLines[i]
		}
		if i < len(expectedLines) {
			e = expectedLines[i]
		}
		if a == e {
			continue
		}
		if i < len(actualLines) {
			fmt.Fprintf(w, "-%s\n", a)
			emitted++
		}
		if i < len(expectedLines) && emitted < 40 {
			fmt.Fprintf(w, "+%s\n", e)
			emitted++
		}
	}
}

func readOr(path, fallback string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(raw)
}

func writeFileAtomic(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".repository-plan-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(raw); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func sortedKeys(m map[string]planned) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeys2(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
