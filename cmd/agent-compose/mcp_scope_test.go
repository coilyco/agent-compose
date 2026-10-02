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
