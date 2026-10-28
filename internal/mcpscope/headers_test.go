package mcpscope

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A header whose ${VAR} is unset stays off the generated config. With the variable
// set the entry is unchanged and the value never reaches the file.
func TestWriteClaudeLeavesOffAHeaderWhoseVariableIsUnset(t *testing.T) {
	inv := &Inventory{Servers: map[string]Server{
		"public_coilyco_linear": {URL: "https://example.invalid/mcp", Headers: map[string]string{
			"Authorization": "${LINEAR_MCP_AUTHORIZATION:-}", "X-Static": "fixed",
		}},
		"defaulted": {URL: "https://example.invalid/d", Headers: map[string]string{"X-Mode": "${MODE:-ro}"}},
	}}
	write := func(env map[string]string) (map[string]map[string]any, string) {
		t.Helper()
		sel := inv.selectServers("eng-platform")
		sel.lookup = func(name string) (string, bool) { value, ok := env[name]; return value, ok }
		path, err := sel.WriteClaude(t.TempDir(), t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var config struct {
			Servers map[string]map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(raw, &config); err != nil {
			t.Fatal(err)
		}
		return config.Servers, strings.Join(sel.HeaderOmissions(), "\n")
	}

	for name, env := range map[string]map[string]string{"unset": {}, "blank": {"LINEAR_MCP_AUTHORIZATION": " \n"}} {
		got, omitted := write(env)
		headers, _ := got["public_coilyco_linear"]["headers"].(map[string]any)
		if _, present := headers["Authorization"]; present {
			t.Fatalf("%s: Authorization still in the config: %v", name, headers)
		}
		if headers["X-Static"] != "fixed" {
			t.Fatalf("%s: a static header was dropped: %v", name, headers)
		}
		for _, want := range []string{"public_coilyco_linear", "LINEAR_MCP_AUTHORIZATION", "aws sso login"} {
			if !strings.Contains(omitted, want) {
				t.Fatalf("%s: the launch line lacks %q: %q", name, want, omitted)
			}
		}
		if strings.Contains(omitted, "X-Mode") || got["defaulted"]["headers"] == nil {
			t.Fatalf("%s: a header with a default was treated as unreadable: %v %q", name, got["defaulted"], omitted)
		}
	}

	got, omitted := write(map[string]string{"LINEAR_MCP_AUTHORIZATION": "Bearer s3cret"})
	headers, _ := got["public_coilyco_linear"]["headers"].(map[string]any)
	if headers["Authorization"] != "${LINEAR_MCP_AUTHORIZATION:-}" || omitted != "" {
		t.Fatalf("a readable secret changed the entry: %v %q", headers, omitted)
	}
}

// OpenCode takes the same filtered headers as the Claude config.
func TestOpenCodeConfigLeavesOffAnUnreadableHeader(t *testing.T) {
	inv := &Inventory{Servers: map[string]Server{
		"remote": {URL: "https://example.invalid/mcp", Headers: map[string]string{"Authorization": "${X_AUTH:-}"}},
	}}
	sel := inv.selectServers("eng-platform")
	sel.lookup = func(string) (string, bool) { return "", false }
	content, err := sel.OpenCodeConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "Authorization") {
		t.Fatalf("Authorization still in the OpenCode config: %s", content)
	}
}
