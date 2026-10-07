package mcpscope

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var roster = []string{"eng-platform", "sysadmin-senior", "scientist"}

const inventory = `{
  "mcpServers": {
    "shared": {"command": "npx", "args": ["-y", "tool", "--config", "${HOME}/x.json"]},
    "remote": {"baseUrl": "https://example.invalid/mcp"},
    "pw_platform": {"command": "npx", "args": ["pw"], "x-aos": {"roles": ["eng-platform"]}},
    "pw_sysadmin": {"command": "npx", "args": ["pw"], "x-aos": {"roles": ["senior-sysadmin"]}}
  }
}`

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcporter.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSelectKeepsUntaggedAndOwnRole(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	sel := inv.Select("eng-platform")
	if got := strings.Join(sel.Selected, ","); got != "pw_platform,remote,shared" {
		t.Errorf("selected = %s", got)
	}
	if got := strings.Join(sel.Omitted, ","); got != "pw_sysadmin" {
		t.Errorf("omitted = %s", got)
	}
	if sel.Scoped != 1 {
		t.Errorf("scoped = %d, want 1", sel.Scoped)
	}
}

// A retired slug in a tag, or at launch, resolves to the current role.
func TestRetiredAliasesResolve(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"sysadmin-senior", "sysadmin", "senior-sysadmin"} {
		sel := inv.Select(role)
		if !contains(sel.Selected, "pw_sysadmin") || contains(sel.Selected, "pw_platform") {
			t.Errorf("%s selected %v", role, sel.Selected)
		}
	}
}

func TestUnknownTagFailsLoadNamingIt(t *testing.T) {
	_, err := Load(write(t, `{"mcpServers": {"pw": {"command": "npx", "x-aos": {"roles": ["wizard"]}}}}`), roster)
	if err == nil || !strings.Contains(err.Error(), `"wizard"`) {
		t.Fatalf("err = %v, want the unknown slug named", err)
	}
}

func TestMissingInventoryIsNotAnError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "absent.json"), roster)
	if !errors.Is(err, ErrNoInventory) {
		t.Fatalf("err = %v, want ErrNoInventory", err)
	}
}

func TestWriteClaudeRendersHostShape(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	sel := inv.Select("eng-platform")
	path, err := sel.WriteClaude(dir, "/home/kai")
	if err != nil {
		t.Fatal(err)
	}
	again, err := sel.WriteClaude(dir, "/home/kai")
	if err != nil || again != path {
		t.Fatalf("second write = %s, %v; want the same content-addressed path %s", again, err, path)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "x-aos") || strings.Contains(string(raw), "${HOME}") {
		t.Errorf("rendered file leaks inventory-only keys or an unexpanded home:\n%s", raw)
	}
	var got struct {
		Servers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 3 || got.Servers["remote"]["type"] != "http" || got.Servers["remote"]["url"] != "https://example.invalid/mcp" {
		t.Errorf("servers = %v", got.Servers)
	}
	if args := got.Servers["shared"]["args"].([]any); args[3] != "/home/kai/x.json" {
		t.Errorf("shared args = %v", args)
	}
}

func TestCodexOverridesDisableOmitted(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(inv.Select("scientist").CodexOverrides(), " ")
	want := "-c mcp_servers.pw_platform.enabled=false -c mcp_servers.pw_sysadmin.enabled=false"
	if got != want {
		t.Errorf("overrides = %q, want %q", got, want)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func TestGooseArgsRenderInTheGrammarGooseSplits(t *testing.T) {
	inv, err := Load(write(t, `{"mcpServers": {
  "remote": {"url": "https://example.invalid/mcp"},
  "script": {"command": "sh", "args": ["-c", "PATH=${HOME}/bin:$PATH exec tool"], "env": {"B": "2", "A": "1"}},
  "pw_sysadmin": {"command": "npx", "x-aos": {"roles": ["senior-sysadmin"]}}
}}`), roster)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inv.Select("eng-platform").GooseArgs([]string{"developer", "todo"}, "/home/k")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--no-profile", "--with-builtin", "developer,todo",
		"--with-streamable-http-extension", "https://example.invalid/mcp",
		"--with-extension", `script:A=1 B=2 sh -c "PATH=/home/k/bin:$PATH exec tool"`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("args =\n%q\nwant\n%q", got, want)
	}
}

// A header or a working directory has no goose flag, and dropping it silently
// would launch a server without its auth or in the wrong place.
func TestGooseArgsRefuseWhatGooseCannotCarry(t *testing.T) {
	for _, body := range []string{
		`{"mcpServers": {"r": {"url": "https://x/mcp", "headers": {"Authorization": "t"}}}}`,
		`{"mcpServers": {"c": {"command": "tool", "cwd": "/srv"}}}`,
		`{"mcpServers": {"q": {"command": "tool", "args": ["it's \"both\""]}}}`,
	} {
		inv, err := Load(write(t, body), roster)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := inv.Select("eng-platform").GooseArgs(nil, "/home/k"); err == nil {
			t.Errorf("%s: want a refusal", body)
		}
	}
}

func TestOpenCodeConfigDefinesSelectedAndDisablesOmitted(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := inv.Select("eng-platform").OpenCodeConfig("/home/k")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		MCP map[string]map[string]any `json:"mcp"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.MCP["remote"]["type"] != "remote" || got.MCP["remote"]["url"] != "https://example.invalid/mcp" {
		t.Errorf("remote = %v", got.MCP["remote"])
	}
	shared := got.MCP["shared"]
	command, _ := shared["command"].([]any)
	if shared["type"] != "local" || len(command) != 5 || command[4] != "/home/k/x.json" {
		t.Errorf("shared = %v, want a local command with ${HOME} expanded", shared)
	}
	if off := got.MCP["pw_sysadmin"]; len(off) != 1 || off["enabled"] != false {
		t.Errorf("omitted = %v, want only enabled=false so any definition elsewhere is turned off", off)
	}
}

// The skill name follows agentic-os-kai's generator: lowercase, runs of other
// characters collapsed to one hyphen. COI-2121
func TestSkillNameMatchesTheGeneratedSkill(t *testing.T) {
	for server, want := range map[string]string{
		"local_coilyco_playwright_prod-director": "mcp-tools-local-coilyco-playwright-prod-director",
		"tailnet_coilyco_lunch_money":            "mcp-tools-tailnet-coilyco-lunch-money",
		"public_coilyco_eco":                     "mcp-tools-public-coilyco-eco",
	} {
		if got := SkillName(server); got != want {
			t.Errorf("SkillName(%q) = %q, want %q", server, got, want)
		}
	}
}

func TestOmittedSkillsFollowTheOmittedServers(t *testing.T) {
	inv, err := Load(write(t, inventory), roster)
	if err != nil {
		t.Fatal(err)
	}
	got := inv.Select("eng-platform").OmittedSkills()
	if len(got) != 1 || got[0] != "mcp-tools-pw-sysadmin" {
		t.Fatalf("omitted skills = %v, want [mcp-tools-pw-sysadmin]", got)
	}
}
