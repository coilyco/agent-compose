package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/mcpscope"
	"github.com/coilyco/agent-compose/v2/internal/person"
)

func TestNativeHarnessCommandScopesClaudeMCP(t *testing.T) {
	t.Parallel()
	got := nativeHarnessCommand("claude", []string{"--model", "opus"}, nativeIdentity{
		MCP: mcpLaunch{ClaudeConfig: "/state/mcp/eng-platform-abc.json"},
	})
	want := []string{
		"claude",
		"--strict-mcp-config", "--mcp-config", "/state/mcp/eng-platform-abc.json",
		"--model", "opus",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestNativeHarnessCommandDisablesOmittedCodexServers(t *testing.T) {
	t.Parallel()
	overrides := []string{"-c", "mcp_servers.pw_x.enabled=false"}
	got := nativeHarnessCommand("codex", []string{"resume"}, nativeIdentity{MCP: mcpLaunch{CodexArgs: overrides}})
	want := []string{"codex", "-c", "mcp_servers.pw_x.enabled=false", "resume"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

// A seat with no scope would load the whole user-level MCP set, so every
// harness with a scope step refuses rather than launch unscoped.
func TestNativeMCPScopeFailsClosed(t *testing.T) {
	t.Setenv("AOS_NATIVE_CANONICAL_HOME", t.TempDir())
	t.Setenv(openCodeConfigEnv, "")
	noRoster := func() (*person.Person, error) { return nil, errors.New("roster unreadable") }
	for _, harness := range []string{"claude", "codex", "goose", "opencode"} {
		if _, err := nativeMCPScope(io.Discard, noRoster, "eng-platform", harness, t.TempDir(), nil); err == nil {
			t.Fatalf("%s: an unreadable roster should refuse", harness)
		}
		if _, err := nativeMCPScope(io.Discard, person.Load, "eng-platform", harness, t.TempDir(), nil); !errors.Is(err, mcpscope.ErrNoInventory) {
			t.Fatalf("%s: a host with no inventory should refuse, got %v", harness, err)
		}
	}
}

// The scope parses only after a session verb, so bare goose gains one.
func TestGooseScopeFollowsTheSessionVerb(t *testing.T) {
	t.Parallel()
	scope := []string{"--no-profile", "--with-builtin", "developer"}
	cases := map[string]struct{ args, want []string }{
		"bare":    {nil, []string{"goose", "session", "--no-profile", "--with-builtin", "developer"}},
		"session": {[]string{"session", "--name", "x"}, []string{"goose", "session", "--no-profile", "--with-builtin", "developer", "--name", "x"}},
		"run":     {[]string{"run", "-t", "hi"}, []string{"goose", "run", "--no-profile", "--with-builtin", "developer", "-t", "hi"}},
	}
	for name, c := range cases {
		got := nativeHarnessCommand("goose", c.args, nativeIdentity{MCP: mcpLaunch{GooseArgs: scope}})
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: command = %#v, want %#v", name, got, c.want)
		}
	}
}

// A verb that loads no extensions, a caller's own profile choice, and a resume
// each leave goose as it is, and read no roster.
func TestGooseScopeStandsAsideWhereNothingLoads(t *testing.T) {
	t.Parallel()
	unread := func() (*person.Person, error) { t.Fatal("the roster should not be read"); return nil, nil }
	for _, args := range [][]string{{"configure"}, {"--version"}, {"session", "--no-profile"}, {"session", "--resume"}, {"s", "-r"}} {
		got, err := nativeMCPScope(io.Discard, unread, "eng-platform", "goose", t.TempDir(), args)
		if err != nil || !reflect.DeepEqual(got, mcpLaunch{}) {
			t.Fatalf("%v: scope = %#v, %v; want none", args, got, err)
		}
	}
}

func TestGooseKeepsItsBuiltinAndPlatformExtensions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := filepath.Join(home, ".config", "goose")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	config := "extensions:\n" +
		"  developer: {enabled: true, type: builtin}\n" +
		"  memory: {enabled: false, type: builtin}\n" +
		"  todo: {enabled: true, type: platform}\n" +
		"  teable: {enabled: true, type: streamable_http, uri: http://x/mcp}\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	keep, err := gooseKeptExtensions(home)
	if err != nil || strings.Join(keep, ",") != "developer,todo" {
		t.Fatalf("kept = %v, %v; want the enabled builtin and platform ones", keep, err)
	}
	if keep, err := gooseKeptExtensions(t.TempDir()); err != nil || keep != nil {
		t.Fatalf("no config: kept = %v, %v; want nothing and no error", keep, err)
	}
}

// OpenCode's inline config is the caller's own scope when already set.
func TestOpenCodeKeepsACallerInlineConfig(t *testing.T) {
	t.Setenv(openCodeConfigEnv, `{"mcp":{}}`)
	unread := func() (*person.Person, error) { t.Fatal("the roster should not be read"); return nil, nil }
	got, err := nativeMCPScope(io.Discard, unread, "eng-platform", "opencode", t.TempDir(), nil)
	if err != nil || !reflect.DeepEqual(got, mcpLaunch{}) {
		t.Fatalf("scope = %#v, %v; want the caller's own", got, err)
	}
}

// A caller's own MCP config is a scope of its own, so no roster is read.
func TestNativeMCPScopeKeepsACallerScope(t *testing.T) {
	t.Parallel()
	unread := func() (*person.Person, error) { t.Fatal("the roster should not be read"); return nil, nil }
	got, err := nativeMCPScope(io.Discard, unread, "eng-platform", "claude", t.TempDir(), []string{"--mcp-config", "mine.json"})
	if err != nil || !reflect.DeepEqual(got, mcpLaunch{}) {
		t.Fatalf("scope = %#v, %v; want the caller's own", got, err)
	}
}

// The bundle is composed before the scope step, so the omitted servers' skills
// are named from the same inventory and role. COI-2121
func TestOmittedMCPSkillsNameTheOtherRolesServers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AOS_NATIVE_CANONICAL_HOME", home)
	t.Setenv(openCodeConfigEnv, "")
	if err := os.MkdirAll(filepath.Join(home, ".mcporter"), 0o755); err != nil {
		t.Fatal(err)
	}
	inventory := `{"mcpServers": {
		"shared": {"command": "npx"},
		"local_x_playwright_eng-platform": {"command": "npx", "x-aos": {"roles": ["eng-platform"]}},
		"local_x_playwright_scientist": {"command": "npx", "x-aos": {"roles": ["scientist"]}}
	}}`
	if err := os.WriteFile(filepath.Join(home, ".mcporter", "mcporter.json"), []byte(inventory), 0o600); err != nil {
		t.Fatal(err)
	}
	got := omittedMCPSkills(io.Discard, person.Load, "eng-platform", "claude", nil)
	if want := []string{"mcp-tools-local-x-playwright-scientist"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("omitted skills = %v, want %v", got, want)
	}
	if got := omittedMCPSkills(io.Discard, person.Load, "eng-platform", "claude", []string{"--mcp-config", "mine.json"}); got != nil {
		t.Fatalf("a caller's own scope should omit nothing, got %v", got)
	}
}

// An absent inventory leaves every skill in place and lets the scope step refuse.
func TestOmittedMCPSkillsNeverBlockALaunch(t *testing.T) {
	t.Setenv("AOS_NATIVE_CANONICAL_HOME", t.TempDir())
	t.Setenv(openCodeConfigEnv, "")
	var warned strings.Builder
	if got := omittedMCPSkills(&warned, person.Load, "eng-platform", "claude", nil); got != nil || warned.Len() != 0 {
		t.Fatalf("no inventory: got %v, warned %q; want nothing", got, warned.String())
	}
	unreadable := func() (*person.Person, error) { return nil, errors.New("roster unreadable") }
	if got := omittedMCPSkills(&warned, unreadable, "eng-platform", "claude", nil); got != nil || !strings.Contains(warned.String(), "skipped") {
		t.Fatalf("unreadable roster: got %v, warned %q; want a warning and no omission", got, warned.String())
	}
}

func TestNativeHarnessCommandDropsStrictForAConnectorGrant(t *testing.T) {
	t.Parallel()
	got := nativeHarnessCommand("claude", nil, nativeIdentity{
		Settings: "/state/mcp/prod-director-abc-settings.json",
		MCP:      mcpLaunch{ClaudeConfig: "/state/mcp/prod-director-abc.json", Allow: []map[string]any{{"serverUrl": "https://x.invalid"}}},
	})
	want := []string{
		"claude",
		"--mcp-config", "/state/mcp/prod-director-abc.json",
		"--settings", "/state/mcp/prod-director-abc-settings.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestApplyConnectorSettingsMergesIntoTheBundleFragment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	base := filepath.Join(dir, "claude-settings.json")
	if err := os.WriteFile(base, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := nativeIdentity{Settings: base, MCP: mcpLaunch{Allow: []map[string]any{{"serverUrl": "https://x.invalid"}}}}
	if err := applyConnectorSettings(io.Discard, "prod-director", dir, nil, &identity); err != nil {
		t.Fatal(err)
	}
	if identity.Settings == base || !strings.HasSuffix(identity.Settings, "-settings.json") {
		t.Fatalf("settings = %q, want a derived file", identity.Settings)
	}
	raw, _ := os.ReadFile(identity.Settings)
	if !strings.Contains(string(raw), `"theme": "dark"`) || !strings.Contains(string(raw), "allowedMcpServers") {
		t.Errorf("derived = %s", raw)
	}
}

func TestApplyConnectorSettingsFallsBackToStrictWhenTheCallerOwnsSettings(t *testing.T) {
	t.Parallel()
	identity := nativeIdentity{
		Settings: "/bundle/claude-settings.json",
		MCP:      mcpLaunch{ClaudeConfig: "/c.json", Allow: []map[string]any{{"serverUrl": "https://x.invalid"}}, Deny: []string{"t"}},
	}
	if err := applyConnectorSettings(io.Discard, "prod-director", t.TempDir(), []string{"--settings", "/mine.json"}, &identity); err != nil {
		t.Fatal(err)
	}
	if len(identity.MCP.Allow) != 0 || len(identity.MCP.Deny) != 0 || identity.Settings != "/bundle/claude-settings.json" {
		t.Fatalf("identity = %#v, want the grant dropped and settings untouched", identity)
	}
	got := nativeHarnessCommand("claude", []string{"--settings", "/mine.json"}, identity)
	if got[1] != "--strict-mcp-config" {
		t.Fatalf("command = %#v, want strict restored", got)
	}
}

func TestApplyConnectorSettingsIsANoOpWithoutAGrant(t *testing.T) {
	t.Parallel()
	identity := nativeIdentity{Settings: "/bundle/claude-settings.json", MCP: mcpLaunch{ClaudeConfig: "/c.json"}}
	if err := applyConnectorSettings(io.Discard, "scientist", t.TempDir(), nil, &identity); err != nil {
		t.Fatal(err)
	}
	if identity.Settings != "/bundle/claude-settings.json" {
		t.Fatalf("settings = %q, want untouched", identity.Settings)
	}
}

// A goose seat whose inventory carries a header server launches without it, and the
// launch says which. COI-2639
func TestGooseSeatLaunchesWithoutAHeaderServerAndNamesIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AOS_NATIVE_CANONICAL_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	if err := os.MkdirAll(filepath.Join(home, ".mcporter"), 0o755); err != nil {
		t.Fatal(err)
	}
	inventory := `{"mcpServers": {
		"shared": {"command": "npx"},
		"authed_linear": {"baseUrl": "https://x.invalid/mcp", "headers": {"Authorization": "t"}}
	}}`
	if err := os.WriteFile(filepath.Join(home, ".mcporter", "mcporter.json"), []byte(inventory), 0o600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	got, err := nativeMCPScope(&out, person.Load, "eng-platform", "goose", t.TempDir(), nil)
	if err != nil {
		t.Fatalf("a header server must not refuse a goose launch: %v", err)
	}
	if joined := strings.Join(got.GooseArgs, " "); strings.Contains(joined, "x.invalid") || !strings.Contains(joined, "npx") {
		t.Fatalf("goose args = %v, want the stdio server and not the header one", got.GooseArgs)
	}
	if !strings.Contains(out.String(), "goose cannot pass headers, so not mounted: authed_linear") {
		t.Fatalf("the launch should name what it left out:\n%s", out.String())
	}
}
