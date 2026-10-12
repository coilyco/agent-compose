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
// behind an allowlist. Deny holds the tool names that keep the grant read-first.
type Connector struct {
	URL   string
	Roles []string
	Deny  []string
}

const (
	slackTool     = "mcp__claude_ai_Slack__"
	stripeTool    = "mcp__claude_ai_Stripe__"
	honeycombTool = "mcp__claude_ai_Honeycomb__"
	dexTool       = "mcp__claude_ai_Dex__"
	wolframTool   = "mcp__claude_ai_Wolfram__"
	// The connector's display name is not confirmed on a seat, so deny under both spellings.
	wolframAlphaTool = "mcp__claude_ai_Wolfram_Alpha__"
)

var claudeAIConnectors = []Connector{
	{URL: "https://mcp.slack.com/mcp", Roles: []string{"prod-director"}, Deny: prefixed(slackTool,
		"slack_add_list_record", "slack_add_reaction", "slack_complete_file_upload", "slack_create_canvas",
		"slack_create_conversation", "slack_create_list", "slack_get_file_upload_url", "slack_schedule_message",
		"slack_send_message", "slack_send_message_draft", "slack_update_canvas", "slack_update_list",
		"slack_update_list_record")},
	{URL: "https://mcp.stripe.com", Roles: []string{"prod-director"}, Deny: prefixed(stripeTool,
		"stripe_api_write", "convert_currency", "send_money", "setup_recipient", "transfer_money")},
	{URL: "https://mcp.honeycomb.io/mcp", Roles: []string{"prod-director"}, Deny: prefixed(honeycombTool,
		"create_board", "create_marker", "create_recipient", "create_trigger", "update_board", "update_trigger",
		"record_onboarding_state")},
	{URL: "https://comment.io/mcp", Roles: []string{"prod-director"}},
	// Dex admits the COI-2788 cleanup: merge, update, archive and delete contacts, add tags
	// and groups. Removing tags or groups, notes, reminders and calendar writes stay out.
	{URL: "https://mcp.getdex.com/mcp", Roles: []string{"prod-director"}, Deny: prefixed(dexTool,
		"dex_create_contact", "dex_complete_keep_in_touch", "dex_create_note", "dex_update_note",
		"dex_delete_note", "dex_create_reminder", "dex_update_reminder", "dex_delete_reminder",
		"dex_update_tag", "dex_delete_tag", "dex_remove_tags_from_contacts", "dex_update_group",
		"dex_delete_group", "dex_remove_contacts_from_group", "dex_create_custom_field",
		"dex_update_custom_field", "dex_delete_custom_field", "dex_set_custom_field_values",
		"dex_create_calendar_event", "dex_update_calendar_event", "dex_delete_calendar_event")},
	// Wispr Flow's MCP is read-only per its docs, so nothing is denied. COI-2772
	{URL: "https://api.wisprflow.ai/connect/mcp", Roles: []string{"prod-director"}},
	// Wolfram: the notebook and paclet writes stay out, the evaluators stay in. COI-2772
	{URL: "https://agenttools.wolfram.com/mcp", Roles: []string{"prod-director"}, Deny: append(
		prefixed(wolframTool, "WriteNotebook", "SubmitPaclet"),
		prefixed(wolframAlphaTool, "WriteNotebook", "SubmitPaclet")...)},
}

func prefixed(prefix string, names ...string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = prefix + name
	}
	return out
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
