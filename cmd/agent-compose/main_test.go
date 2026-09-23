package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/cascade"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/compose"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/describe"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/nativelaunch"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/overlay"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/palette"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/roster"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func TestConfigValidateRejectsRemovedRoleProviderKeys(t *testing.T) {
	valid := filepath.Join(t.TempDir(), "agent-compose.yaml")
	if err := os.WriteFile(valid, []byte(
		"person_policy: external-only\nperson_source: ./person\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := cascade.LoadConfig(valid); err != nil {
		t.Fatalf("valid strict config: %v", err)
	}

	for name, body := range map[string]string{
		"inline": "role_providers: {}\n",
		"file":   "role_providers_file: role-providers.yaml\n",
	} {
		t.Run(name, func(t *testing.T) {
			invalid := filepath.Join(t.TempDir(), "agent-compose.yaml")
			if err := os.WriteFile(invalid, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := cascade.LoadConfig(invalid); err == nil || !strings.Contains(err.Error(), "field role_providers") {
				t.Fatalf("removed role-provider config passed: %v", err)
			}
		})
	}
}

func TestExternalPersonProfileExampleExercisesEveryPersonSurface(t *testing.T) {
	profile := filepath.Join("..", "..", "examples", "person-profile")
	library := filepath.Join("..", "..", "examples", "shared-personality-library")
	request := filepath.Join(profile, "request.kdl")
	p, err := person.LoadDirectoryWithLibraries(profile, library)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compose.Run(request, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Verify(result.Bundle.Dir); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "example.tar.gz")
	if err := bundle.Export(result.Bundle.Dir, archive); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(archive); err != nil || info.Size() == 0 {
		t.Fatalf("example export is missing: info=%v err=%v", info, err)
	}
	if _, err := describe.Bundle(result.Bundle.Dir, describe.Options{}); err != nil {
		t.Fatal(err)
	}
	projected, err := overlay.Build(p, "bulk-captioner", "chatbot-sonnet-low", "available")
	if err != nil || projected.Seat.Name != "example content guide" {
		t.Fatalf("example overlay failed: doc=%v err=%v", projected, err)
	}
	if _, err := palette.Build(p); err != nil {
		t.Fatal(err)
	}
	files, err := roster.Render(p, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"person.json",
		"person.v4.json",
		"personality-index.md",
		".agents/skills/role-bulk-captioner/SKILL.md",
	} {
		if len(files[path]) == 0 {
			t.Errorf("example roster omitted %q", path)
		}
	}
	if entries, err := p.PersonalityCatalog(nil); err != nil || len(entries) != 3 {
		t.Fatalf("example personality catalogue failed: entries=%v err=%v", entries, err)
	}
	if entries, err := p.RoleCatalog(); err != nil || len(entries) != 2 {
		t.Fatalf("example role catalogue failed: entries=%v err=%v", entries, err)
	}
	if entries, err := p.SeatCatalog("bulk-captioner"); err != nil || len(entries) != 1 {
		t.Fatalf("example seat catalogue failed: entries=%v err=%v", entries, err)
	}
}

func TestDirectPersonSelectionInheritsExternalOnlyHost(t *testing.T) {
	dir := t.TempDir()
	personRoot, err := filepath.Abs(filepath.Join(
		"..", "..", "testdata", "contracts", "person-independent",
	))
	if err != nil {
		t.Fatal(err)
	}
	paths := cascade.Paths{
		Config: filepath.Join(dir, "agent-compose.yaml"),
		Home:   filepath.Join(dir, "home"),
	}
	config := "person_policy: " + personpolicy.ExternalOnly + "\n" +
		"person_source: " + personRoot + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	p, external, err := loadSelectedPersonAt("", paths)
	if err != nil {
		t.Fatal(err)
	}
	if !external || p.Name != "workbench" {
		t.Fatalf("direct selection = external:%t person:%q", external, p.Name)
	}
}

func TestDirectPersonSelectionFailsClosedWithBrokenHostGuard(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config: filepath.Join(dir, "agent-compose.yaml"),
		Home:   filepath.Join(dir, "home"),
	}
	if err := os.WriteFile(
		paths.Config,
		[]byte("person_policy: "+personpolicy.ExternalOnly+"\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadSelectedPersonAt("", paths); err == nil ||
		!strings.Contains(err.Error(), "requires person_source") {
		t.Fatalf("broken host guard error = %v", err)
	}
}

func TestProjectPersonPolicyRejectsEmbeddedBundle(t *testing.T) {
	result, err := compose.Run(
		filepath.Join("..", "..", "testdata", "contracts", "native.kdl"),
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = validateProjectPersonPolicy(result.Bundle.Dir, compose.Options{
		PersonPolicy: personpolicy.ExternalOnly,
		PersonSource: "/person",
	})
	if err == nil || !strings.Contains(err.Error(), "roster:core") {
		t.Fatalf("embedded bundle policy error = %v", err)
	}
}

func TestProjectPersonPolicyAcceptsExternalBundle(t *testing.T) {
	result, err := compose.Run(
		filepath.Join("..", "..", "testdata", "contracts", "custom-person.kdl"),
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProjectPersonPolicy(result.Bundle.Dir, compose.Options{
		PersonPolicy: personpolicy.ExternalOnly,
		PersonSource: "/person",
	}); err != nil {
		t.Fatalf("external bundle rejected: %v", err)
	}
}

func TestDispatchArgs(t *testing.T) {
	cases := map[string]struct{ in, want []string }{
		"canonical name untouched": {
			[]string{"/usr/local/bin/agent-compose", "describe", "x"},
			[]string{"/usr/local/bin/agent-compose", "describe", "x"},
		},
		"acompose injects compose": {
			[]string{"/opt/homebrew/bin/acompose"},
			[]string{"/opt/homebrew/bin/acompose", "compose"},
		},
		"acompose keeps trailing args": {
			[]string{"acompose", "--", "claude"},
			[]string{"acompose", "compose", "--", "claude"},
		},
		"acompose role and harness inject launch": {
			[]string{"acompose", "frontend-eng", "codex", "--model", "gpt"},
			[]string{"acompose", "launch", "frontend-eng", "codex", "--model", "gpt"},
		},
		"acompose request and layout remain compose": {
			[]string{"acompose", "--layout", "codex", "request.kdl", "--", "codex"},
			[]string{"acompose", "compose", "--layout", "codex", "request.kdl", "--", "codex"},
		},
		"acompose nested role and harness inject launch": {
			[]string{"acompose", "--nested", "scientist", "claude", "-p", "measure it"},
			[]string{"acompose", "launch", "--nested", "scientist", "claude", "-p", "measure it"},
		},
		"acompose nested without a harness remains compose": {
			[]string{"acompose", "--nested", "request.kdl"},
			[]string{"acompose", "compose", "--nested", "request.kdl"},
		},
		"acompose exposes statusline directly": {
			[]string{"acompose", "statusline", "--target", "/workspace"},
			[]string{"acompose", "statusline", "--target", "/workspace"},
		},
		"windows exe suffix": {
			[]string{`C:\shims\acompose.exe`, "req.kdl"},
			[]string{`C:\shims\acompose.exe`, "compose", "req.kdl"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := dispatchArgs(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPrintSummaryUsesSlashSeparators(t *testing.T) {
	t.Parallel()
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	result := summaryFixture(t, p)
	wantPersonalities := strings.Join(p.Roles["platform-eng"].Personalities, " // ")
	wantColor := result.Resolution.FavoriteColor
	wantIdentity := "agent identity: " + p.Roles["platform-eng"].Identity.Name

	var output strings.Builder
	if err := printSummary(&output, result, person.RoleTranscriptOptions{Expanded: true}); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"request: model tier frontier // delivery native-skills",
		"roster: core // provided by: roster:core",
		"role: platform-eng",
		"personalities: " + wantPersonalities,
		"melded color: " + wantColor,
		"personality: tenacious",
		wantIdentity,
		"renderer expressions: available // listening // thinking",
		"decisions: 1 selected // 1 excluded // 1 shadowed // 1 delivered",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "·") {
		t.Fatalf("summary retained middle-dot separator:\n%s", got)
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if len(line) > 0 && line[0] == ' ' {
			t.Fatalf("summary line is not flush-left: %q", line)
		}
	}

	var colored strings.Builder
	if err := printSummary(&colored, result, person.RoleTranscriptOptions{
		Color: true, TrueColor: true,
	}); err != nil {
		t.Fatal(err)
	}
	wantColoredBundle := strings.TrimSuffix(color.ANSI(wantColor, "bundle", true), "\x1b[0m")
	if !strings.Contains(colored.String(), wantColoredBundle) {
		t.Fatalf("summary intro did not use melded truecolor:\n%q", colored.String())
	}
}

func TestPrintCompositionWarningsUsesExplicitWarningPrefix(t *testing.T) {
	var output strings.Builder
	printCompositionWarnings(&output, []string{
		`source "aos" provider role "dev-advocate" matched composed skill "writing-kai-voice" through selectors "*writing*", "*voice*", selected once`,
	})
	want := `agent-compose: warning: source "aos" provider role "dev-advocate" matched composed skill "writing-kai-voice" through selectors "*writing*", "*voice*", selected once` + "\n"
	if output.String() != want {
		t.Fatalf("warning output = %q, want %q", output.String(), want)
	}
}

func summaryFixture(t *testing.T, p *person.Person) *compose.Result {
	t.Helper()
	personalities := p.Roles["platform-eng"].Personalities
	favoriteColor := p.Roles["platform-eng"].FavoriteColor
	return &compose.Result{
		Bundle: &bundle.Result{Key: "abc123", Dir: "/tmp/bundle", Reused: true},
		Resolution: &resolver.Resolution{
			Request: &schema.Request{
				Role:      "platform-eng",
				ModelTier: schema.ModelTierFrontier,
				Delivery:  "native-skills",
			},
			Personalities: personalities,
			FavoriteColor: favoriteColor,
			SourceIDs:     []string{"roster:core", "aos-public"},
			Person:        p,
			Decisions: []resolver.Decision{
				{Outcome: resolver.OutcomeSelected},
				{Outcome: resolver.OutcomeExcluded},
				{Outcome: resolver.OutcomeShadowed},
				{Outcome: resolver.OutcomeDelivered},
			},
		},
	}
}

func TestNativeHarnessCommandPromptsFreshCodexSession(t *testing.T) {
	t.Parallel()
	for name, args := range map[string][]string{
		"bare": nil,
		"AOS trust override": {
			"--config",
			`projects={"/tmp/aos-native/session/projects"={trust_level="trusted"}}`,
		},
		"model selection": {"--model", "gpt-5.6"},
	} {
		t.Run(name, func(t *testing.T) {
			got := nativeHarnessCommand("codex", args, nativeIdentity{})
			if got[len(got)-1] != nativeCodexIntroductionPrompt {
				t.Fatalf("command does not end with the introduction prompt: %#v", got)
			}
		})
	}
}

func TestNativeHarnessCommandPreservesExplicitCodexWork(t *testing.T) {
	t.Parallel()
	for name, args := range map[string][]string{
		"prompt":         {"help me debug this"},
		"subcommand":     {"prod-director", "run the tests"},
		"unknown option": {"--future-option"},
	} {
		t.Run(name, func(t *testing.T) {
			want := append([]string{"codex"}, args...)
			if got := nativeHarnessCommand("codex", args, nativeIdentity{}); !reflect.DeepEqual(got, want) {
				t.Fatalf("command = %#v, want unchanged %#v", got, want)
			}
		})
	}
}

func TestNativeHarnessCommandDoesNotPromptOtherHarnesses(t *testing.T) {
	t.Parallel()
	if got, want := nativeHarnessCommand("claude", nil, nativeIdentity{}), []string{"claude"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestNativeHarnessCommandCarriesClaudeIdentity(t *testing.T) {
	t.Parallel()
	got := nativeHarnessCommand("claude", nil, nativeIdentity{
		SeatName: "Angie",
		Settings: "/bundles/abc/claude-settings.json",
	})
	want := []string{
		"claude",
		"--name", "Angie",
		"--settings", "/bundles/abc/claude-settings.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestNativeHarnessCommandKeepsIdentityFlagsAheadOfCallerArgs(t *testing.T) {
	t.Parallel()
	got := nativeHarnessCommand("claude", []string{"--model", "opus"}, nativeIdentity{
		SeatName: "Angie",
		Settings: "/bundles/abc/claude-settings.json",
	})
	want := []string{
		"claude",
		"--name", "Angie",
		"--settings", "/bundles/abc/claude-settings.json",
		"--model", "opus",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestNativeHarnessCommandYieldsToCallerIdentityFlags(t *testing.T) {
	t.Parallel()
	for name, args := range map[string][]string{
		"separate value": {"--name", "Scout", "--settings", "/tmp/mine.json"},
		"inline value":   {"--name=Scout", "--settings=/tmp/mine.json"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			want := append([]string{"claude"}, args...)
			got := nativeHarnessCommand("claude", args, nativeIdentity{
				SeatName: "Angie",
				Settings: "/bundles/abc/claude-settings.json",
			})
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("command = %#v, want unchanged %#v", got, want)
			}
		})
	}
}

func TestNativeHarnessCommandTreatsPostTerminatorFlagsAsHarnessInput(t *testing.T) {
	t.Parallel()
	args := []string{"--", "--name", "Scout"}
	got := nativeHarnessCommand("claude", args, nativeIdentity{SeatName: "Angie"})
	want := append([]string{"claude", "--name", "Angie"}, args...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestNativeHarnessCommandLeavesOtherHarnessesUnnamed(t *testing.T) {
	t.Parallel()
	for _, harness := range []string{"codex", "goose", "opencode"} {
		t.Run(harness, func(t *testing.T) {
			t.Parallel()
			got := nativeHarnessCommand(harness, []string{"--flag"}, nativeIdentity{
				SeatName: "Angie",
				Settings: "/bundles/abc/claude-settings.json",
			})
			for _, arg := range got {
				if arg == "--name" || arg == "--settings" {
					t.Fatalf("harness %q command carries a Claude flag: %#v", harness, got)
				}
			}
		})
	}
}

func TestNativeLaunchSummaryPutsRoleTranscriptLast(t *testing.T) {
	t.Parallel()
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	result := summaryFixture(t, p)
	var output strings.Builder
	if err := printNativeLaunchSummary(
		&output,
		result,
		person.RoleTranscriptOptions{},
		summaryLayout{RoleLast: true, Audit: true},
	); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	audit := strings.Index(got, "sources:")
	role := strings.Index(got, "role metadata")
	personality := strings.Index(got, "personality metadata")
	if audit < 0 || personality < audit || role < personality {
		t.Fatalf("native launch summary order is wrong:\n%s", got)
	}
	if strings.Contains(got[role:], "sources:") {
		t.Fatalf("routine audit followed the role metadata:\n%s", got)
	}
}

func TestNativeLaunchSummaryWithoutAuditPrintsOnlyTheTranscript(t *testing.T) {
	t.Parallel()
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	result := summaryFixture(t, p)
	var output strings.Builder
	if err := printNativeLaunchSummary(
		&output,
		result,
		person.RoleTranscriptOptions{},
		summaryLayout{RoleLast: true},
	); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, unwanted := range []string{"bundle ", "request:", "sources:", "decisions:", "path:", "trace:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("quiet launch summary kept %q:\n%s", unwanted, got)
		}
	}
	for _, want := range []string{"personality metadata", "role metadata", "role: platform-eng"} {
		if !strings.Contains(got, want) {
			t.Fatalf("quiet launch summary dropped %q:\n%s", want, got)
		}
	}
	if !strings.HasPrefix(got, "personality metadata") {
		t.Fatalf("quiet launch summary did not lead with the transcript:\n%s", got)
	}
}

func TestAcknowledgeNativeLaunchWaitsForEnter(t *testing.T) {
	t.Parallel()
	input := &enterGateReader{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	var output strings.Builder
	done := make(chan error, 1)
	go func() {
		done <- acknowledgeNativeLaunch(input, &output, true)
	}()
	<-input.started
	if got, want := output.String(), "Press Enter to continue"; got != want {
		t.Fatalf("visible acknowledgement = %q, want %q", got, want)
	}
	select {
	case err := <-done:
		t.Fatalf("acknowledgement returned before Enter: %v", err)
	default:
	}
	close(input.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Press Enter to continue\n"; got != want {
		t.Fatalf("acknowledgement output = %q, want %q", got, want)
	}
}

func TestAcknowledgeNativeLaunchSkipsNonInteractiveInput(t *testing.T) {
	t.Parallel()
	input := &failReader{err: errors.New("stdin must not be read")}
	var output strings.Builder
	if err := acknowledgeNativeLaunch(input, &output, false); err != nil {
		t.Fatal(err)
	}
	if input.reads != 0 || output.Len() != 0 {
		t.Fatalf("non-interactive acknowledgement touched I/O: reads=%d output=%q",
			input.reads, output.String())
	}
}

func TestAcknowledgeNativeLaunchStopsBeforeExecOnInputFailure(t *testing.T) {
	t.Parallel()
	want := errors.New("interrupted")
	err := acknowledgeNativeLaunch(&failReader{err: want}, io.Discard, true)
	if !errors.Is(err, want) {
		t.Fatalf("acknowledgement error = %v, want wrapped interruption", err)
	}
}

func TestNativeLaunchInteractiveRejectsPipes(t *testing.T) {
	t.Parallel()
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer inputWriter.Close()
	output, outputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	defer outputWriter.Close()
	if nativeLaunchInteractive(input, outputWriter) {
		t.Fatal("piped input and output were treated as an interactive terminal")
	}
}

type failReader struct {
	err   error
	reads int
}

func (r *failReader) Read(_ []byte) (int, error) {
	r.reads++
	return 0, r.err
}

type enterGateReader struct {
	started chan struct{}
	release chan struct{}
}

func (r *enterGateReader) Read(p []byte) (int, error) {
	close(r.started)
	<-r.release
	p[0] = '\n'
	return 1, nil
}

func TestPrintVerificationUsesBoundedCounts(t *testing.T) {
	t.Parallel()
	verification := &bundle.Verification{
		Files: 128,
		Identities: []bundle.Identity{
			{Source: "roster:core", Skill: "personality-tenacious"},
			{Source: "aos-public", Skill: "coding-go"},
		},
	}

	var output strings.Builder
	printVerification(&output, verification)
	if got, want := output.String(), "bundle verified: 2 skills // 128 files\n"; got != want {
		t.Fatalf("verification output = %q, want %q", got, want)
	}
}

func TestActivateNativeRuntimeHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", "old-home")
	t.Setenv("USERPROFILE", "old-profile")
	t.Setenv("CODEX_HOME", "old-codex")
	t.Setenv("XDG_CONFIG_HOME", "old-config")
	t.Setenv("CLAUDE_CONFIG_DIR", "old-claude")

	if err := activateNativeRuntimeHome(home, "claude"); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"HOME":              home,
		"USERPROFILE":       home,
		"CODEX_HOME":        filepath.Join(home, ".codex"),
		"XDG_CONFIG_HOME":   filepath.Join(home, ".config"),
		"CLAUDE_CONFIG_DIR": filepath.Join(home, ".claude"),
	} {
		if got := os.Getenv(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestClearNativeLaunchEnvironmentRemovesSelectionFacts(t *testing.T) {
	names := []string{
		nativelaunch.EnvModelTier,
		"AGENT_COMPOSE_MODEL_CLASS",
		nativelaunch.EnvRuntimeHome,
	}
	for _, name := range names {
		t.Setenv(name, "fixture")
	}
	if err := clearNativeLaunchEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if _, ok := os.LookupEnv(name); ok {
			t.Errorf("%s remains in the harness environment", name)
		}
	}
}

func TestActivateNativeRuntimeHomePreservesCanonicalCodexState(t *testing.T) {
	hostCodexHome := filepath.Join(t.TempDir(), ".codex")
	if err := os.MkdirAll(hostCodexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	hostCodexHome, err := filepath.EvalSymlinks(hostCodexHome)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err := os.Symlink(hostCodexHome, filepath.Join(home, ".codex")); err != nil {
		t.Fatal(err)
	}

	if err := activateNativeRuntimeHome(home, "codex"); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("CODEX_HOME"); got != hostCodexHome {
		t.Fatalf("CODEX_HOME = %q, want canonical host state %q", got, hostCodexHome)
	}
	if got := os.Getenv("HOME"); got != home {
		t.Fatalf("HOME = %q, want isolated runtime home %q", got, home)
	}
}

func TestCatalogRoleLineLabelsThePersonalityMeld(t *testing.T) {
	line := catalogRoleLine(person.RoleCatalogEntry{
		Slug:          "platform-eng",
		Skill:         "role-platform-eng",
		Purpose:       "Build and land the foundational software.",
		Personalities: []string{"tenacious", "grounded"},
		FavoriteColor: "#9c8b31",
	})
	want := "platform-eng // role-platform-eng // Build and land the foundational software. " +
		"// personalities: tenacious, grounded // color: #9c8b31\n"
	if line != want {
		t.Fatalf("catalog role line =\n%q\nwant\n%q", line, want)
	}
	if strings.Contains(line, "boundary") {
		t.Errorf("catalog role line still labels the meld as a boundary: %q", line)
	}
}

func TestCatalogVerbsRejectAPositionalTheyDoNotRead(t *testing.T) {
	// Calls the real actions, so a verb that stops consulting
	// rejectCatalogArgs fails here rather than only the helper regressing.
	for name, action := range map[string]func(context.Context, *cli.Command) error{
		"personalities": runCatalogPersonalities,
		"roles":         runCatalogRoles,
		"seats":         runCatalogSeats,
		"boundaries":    runCatalogBoundaries,
		"expressions":   runCatalogExpressions,
		"snapshot":      runCatalogSnapshot,
	} {
		t.Run(name, func(t *testing.T) {
			cmd := &cli.Command{Name: name, Action: action}
			err := cmd.Run(context.Background(), []string{name, "zzz"})
			if err == nil {
				t.Fatal("a stray positional must fail rather than be discarded")
			}
			if !strings.Contains(err.Error(), "zzz") {
				t.Errorf("error must name the discarded argument, got %q", err)
			}
		})
	}
	clean := &cli.Command{
		Name:   "roles",
		Action: func(_ context.Context, c *cli.Command) error { return rejectCatalogArgs(c) },
	}
	if err := clean.Run(context.Background(), []string{"roles"}); err != nil {
		t.Fatalf("no positional must stay allowed: %v", err)
	}
}

// housecast#8041: evalkit.roster projects an annotator view off exactly
// these fields, so a regression here is a regression there too.
func TestCatalogSnapshotCarriesWhatEvalkitRosterReads(t *testing.T) {
	out := filepath.Join(t.TempDir(), "snapshot.json")
	cmd := &cli.Command{
		Name:   "snapshot",
		Flags:  append(personCatalogFlags(false), &cli.StringFlag{Name: "out"}),
		Action: runCatalogSnapshot,
	}
	if err := cmd.Run(context.Background(), []string{"snapshot", "--out", out}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("snapshot is not valid JSON: %v", err)
	}
	if got := snapshot["format"]; got != person.SnapshotFormat {
		t.Errorf("format = %v, want %v", got, person.SnapshotFormat)
	}

	order, ok := snapshot["role_order"].([]any)
	if !ok || len(order) == 0 {
		t.Fatalf("role_order missing or empty: %v", snapshot["role_order"])
	}
	roles, ok := snapshot["roles"].(map[string]any)
	if !ok {
		t.Fatalf("roles missing or not an object: %v", snapshot["roles"])
	}
	role, ok := roles["prod-manager"].(map[string]any)
	if !ok {
		t.Fatalf("roles[prod-manager] missing: %v", roles)
	}
	for _, key := range []string{"display_name", "purpose", "boundaries", "acts", "adjacents", "personalities"} {
		if _, ok := role[key]; !ok {
			t.Errorf("roles[prod-manager] is missing %q, which evalkit/roster.py reads", key)
		}
	}

	boundaries, ok := snapshot["boundaries"].(map[string]any)
	if !ok {
		t.Fatalf("boundaries missing or not an object: %v", snapshot["boundaries"])
	}
	modify, ok := boundaries["modify-live-backend"].(map[string]any)
	if !ok {
		t.Fatalf("boundaries[modify-live-backend] missing: %v", boundaries)
	}
	if owner, _ := modify["owner"].(string); owner != "sysadmin-senior" {
		t.Errorf("boundaries[modify-live-backend].owner = %q, want sysadmin-senior", owner)
	}
}

func TestApplyTelemetryEnvironmentLabelsTheSeatAndDropsInheritedLogging(t *testing.T) {
	config := filepath.Join(t.TempDir(), "agent-compose.yaml")
	body := "sources: []\ntelemetry:\n  otlp_metrics_endpoint: http://collector.example:4318/v1/metrics\n"
	if err := os.WriteFile(config, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AOS_NATIVE_SESSION", "yx46")
	t.Setenv("AGENT_COMPOSE_TELEMETRY", "")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "seat=prod-director,shadow=ru78")
	t.Setenv("OTEL_LOG_USER_PROMPTS", "1")

	if err := applyTelemetryEnvironment(config, "claude", "platform-eng"); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); got != "seat=platform-eng,shadow=yx46" {
		t.Errorf("OTEL_RESOURCE_ATTRIBUTES = %q", got)
	}
	if _, ok := os.LookupEnv("OTEL_LOG_USER_PROMPTS"); ok {
		t.Error("an inherited prompt-logging flag reaches the seat")
	}

	t.Setenv("AGENT_COMPOSE_TELEMETRY", "off")
	if err := applyTelemetryEnvironment(config, "claude", "platform-eng"); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv("CLAUDE_CODE_ENABLE_TELEMETRY"); ok {
		t.Error("the kill switch left telemetry enabled")
	}
}

func TestApplyTelemetryEnvironmentWithoutHostConfig(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENABLE_TELEMETRY", "1")
	missing := filepath.Join(t.TempDir(), "absent.yaml")
	if err := applyTelemetryEnvironment(missing, "claude", "platform-eng"); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv("CLAUDE_CODE_ENABLE_TELEMETRY"); ok {
		t.Error("a host without telemetry config still exports")
	}
}
