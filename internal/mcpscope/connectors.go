package mcpscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/roleslug"
)

// Connector is a claude.ai connector that strict mode hides, so a granted role runs
// behind an allowlist. Deny holds tool names a grant keeps out, empty for full access.
type Connector struct {
	URL   string
	Roles []string
	Deny  []string
}

// prod-director holds full read and write on every row below, so no row carries a Deny.
// Kai, 2026-10-28, superseding the read-only decision on COI-2991.
var claudeAIConnectors = []Connector{
	{URL: "https://mcp.slack.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.stripe.com", Roles: []string{"prod-director"}},
	{URL: "https://mcp.honeycomb.io/mcp", Roles: []string{"prod-director"}},
	{URL: "https://comment.io/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.getdex.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://api.wisprflow.ai/connect/mcp", Roles: []string{"prod-director"}},
	{URL: "https://agenttools.wolfram.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://api.anthropic.com/v1/pages/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.airtable.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://sprites.dev/mcp", Roles: []string{"prod-director"}},
	{URL: "https://docsmcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	{URL: "https://slidesmcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	{URL: "https://sheetsmcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	{URL: "https://api.elevenlabs.io/v1/mcp-claude-marketplace", Roles: []string{"prod-director"}},
	{URL: "https://ai.todoist.net/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.audible.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.exa.ai/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.calendly.com", Roles: []string{"prod-director"}},
	{URL: "https://mcp.trello.com/v1", Roles: []string{"prod-director"}},
	{URL: "https://mcp-gateway-external-pilot.spotify.net/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.sentry.dev/mcp", Roles: []string{"prod-director"}},
	{URL: "https://drivemcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	{URL: "https://calendarmcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	{URL: "https://gmailmcp.googleapis.com/mcp/v1", Roles: []string{"prod-director"}},
	// These need auth on the account. The row stays so each works once Kai authenticates.
	{URL: "https://exe.dev/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp2.readwise.io/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.ws.sonos.com/mcp", Roles: []string{"prod-director"}},
	{URL: "https://mcp.axiom.co/mcp", Roles: []string{"prod-director"}},
	{URL: "https://teable.coilysiren.me/mcp", Roles: []string{"prod-director"}},
}

// SelectWith is Select with an explicit connector table, so a test needs no shipped one.
func (inv *Inventory) SelectWith(role string, table []Connector) Selection {
	sel := inv.selectServers(role)
	sel.connectors = grantedFrom(sel.Role, table)
	return sel
}

// GrantedConnectors are the shipped connectors a role holds, with no inventory read. The
// launch spec hands them to a launcher that renders the allowlist itself.
func GrantedConnectors(role string) []Connector {
	return grantedFrom(roleslug.Canonical(role), claudeAIConnectors)
}

func grantedFrom(role string, table []Connector) []Connector {
	var out []Connector
	for _, c := range table {
		for _, tag := range c.Roles {
			if roleslug.Canonical(tag) == role {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// Connectors are the claude.ai connectors the role is granted.
func (sel Selection) Connectors() []Connector { return sel.connectors }

// Allowlist is the allowedMcpServers entry list that stands in for --strict-mcp-config:
// each selected server by URL or exact command, then each granted connector by URL.
func (sel Selection) Allowlist(home string) []map[string]any {
	var out []map[string]any
	for _, name := range sortedNames(sel.servers) {
		server := claudeServer(sel.servers[name], home)
		if url, ok := server["url"].(string); ok {
			out = append(out, map[string]any{"serverUrl": url})
			continue
		}
		command := []string{server["command"].(string)}
		if args, ok := server["args"].([]string); ok {
			command = append(command, args...)
		}
		out = append(out, map[string]any{"serverCommand": command})
	}
	for _, c := range sel.connectors {
		out = append(out, map[string]any{"serverUrl": c.URL})
	}
	return out
}

// DeniedTools are the tool names the granted connectors keep out of the session.
func (sel Selection) DeniedTools() []string {
	var out []string
	for _, c := range sel.connectors {
		out = append(out, c.Deny...)
	}
	return out
}

// WriteSettings merges allow and deny into the bundle's settings fragment, since Claude
// takes one --settings file. A missing base starts empty. Named by content.
func WriteSettings(dir, role, base string, allow []map[string]any, deny []string) (string, error) {
	doc := map[string]any{}
	if strings.TrimSpace(base) != "" {
		raw, err := os.ReadFile(base)
		if err != nil {
			return "", fmt.Errorf("mcpscope: read bundle settings: %w", err)
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", fmt.Errorf("mcpscope: parse bundle settings %s: %w", base, err)
		}
	}
	doc["allowedMcpServers"] = allow
	if len(deny) > 0 {
		permissions, _ := doc["permissions"].(map[string]any)
		if permissions == nil {
			permissions = map[string]any{}
		}
		existing, _ := permissions["deny"].([]any)
		for _, rule := range deny {
			existing = append(existing, rule)
		}
		permissions["deny"] = existing
		doc["permissions"] = permissions
	}
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("mcpscope: render settings: %w", err)
	}
	payload = append(payload, '\n')
	sum := sha256.Sum256(payload)
	path := filepath.Join(dir, fmt.Sprintf("%s-%s-settings.json", roleslug.Canonical(role), hex.EncodeToString(sum[:])[:12]))
	if existing, err := os.ReadFile(path); err == nil && string(existing) == string(payload) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.json")
	if err != nil {
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	return path, nil
}
