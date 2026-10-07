// Package mcpscope narrows the host MCP inventory to one role's servers for a
// native launch. See docs/claude-launch-identity.md.
package mcpscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/roleslug"
)

// Server is one inventory entry in mcporter's shape. Unknown keys are ignored.
type Server struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd"`
	URL     string            `json:"url"`
	BaseURL string            `json:"baseUrl"`
	Headers map[string]string `json:"headers"`
	AOS     struct {
		Roles []string `json:"roles"`
	} `json:"x-aos"`
}

// Inventory is the parsed host inventory.
type Inventory struct {
	Servers map[string]Server
}

// Selection is one role's servers and the ones it leaves out.
type Selection struct {
	Role     string
	Selected []string
	Omitted  []string
	Scoped   int // selected servers that carry a role tag
	servers  map[string]Server
}

// ErrNoInventory marks an absent inventory, which leaves the launch unscoped.
var ErrNoInventory = errors.New("mcpscope: no MCP inventory")

// Load reads an mcporter inventory and checks every role tag against roles,
// resolving retired slugs, so a renamed role cannot silently lose its servers.
func Load(path string, roles []string) (*Inventory, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoInventory
	}
	if err != nil {
		return nil, fmt.Errorf("mcpscope: read %s: %w", path, err)
	}
	var top struct {
		Servers map[string]Server `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("mcpscope: parse %s: %w", path, err)
	}
	known := map[string]bool{}
	for _, r := range roles {
		known[r] = true
	}
	for _, name := range sortedNames(top.Servers) {
		for _, tag := range top.Servers[name].AOS.Roles {
			if !known[roleslug.Canonical(tag)] {
				return nil, fmt.Errorf("mcpscope: server %q is tagged for role %q, which is neither a roster role nor a retired alias", name, tag)
			}
		}
	}
	return &Inventory{Servers: top.Servers}, nil
}

// Select keeps every untagged server and every server tagged for role.
func (inv *Inventory) Select(role string) Selection {
	role = roleslug.Canonical(role)
	sel := Selection{Role: role, servers: map[string]Server{}}
	for _, name := range sortedNames(inv.Servers) {
		s := inv.Servers[name]
		if len(s.AOS.Roles) == 0 {
			sel.Selected = append(sel.Selected, name)
			sel.servers[name] = s
			continue
		}
		if tagged(s, role) {
			sel.Selected = append(sel.Selected, name)
			sel.servers[name] = s
			sel.Scoped++
			continue
		}
		sel.Omitted = append(sel.Omitted, name)
	}
	return sel
}

func tagged(s Server, role string) bool {
	for _, tag := range s.AOS.Roles {
		if roleslug.Canonical(tag) == role {
			return true
		}
	}
	return false
}

// skillPrefix and skillSlug mirror agentic-os-kai scripts/sync_mcp_skills.py,
// which names the generated skill for each inventory server.
const skillPrefix = "mcp-tools-"

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// SkillName is the generated tool-reference skill for one inventory server.
func SkillName(server string) string {
	return skillPrefix + strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(server), "-"), "-")
}

// OmittedSkills names the tool-reference skill of each server the role leaves
// out, so a session home does not list tools its harness never mounts.
func (sel Selection) OmittedSkills() []string {
	skills := make([]string, 0, len(sel.Omitted))
	for _, name := range sel.Omitted {
		skills = append(skills, SkillName(name))
	}
	return skills
}

// Summary is the one line a launch prints so the narrowing is never silent.
func (sel Selection) Summary() string {
	return fmt.Sprintf("agent-compose: MCP for %s: %d servers (%d role-scoped), %d omitted",
		sel.Role, len(sel.Selected), sel.Scoped, len(sel.Omitted))
}

// WriteClaude renders a --mcp-config file under dir, named by its content so seats
// of one role share it. ${HOME} expands against home, as the host projection does.
func (sel Selection) WriteClaude(dir, home string) (string, error) {
	out := map[string]any{}
	for name, s := range sel.servers {
		out[name] = claudeServer(s, home)
	}
	payload, err := json.MarshalIndent(map[string]any{"mcpServers": out}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("mcpscope: render: %w", err)
	}
	payload = append(payload, '\n')
	sum := sha256.Sum256(payload)
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.json", sel.Role, hex.EncodeToString(sum[:])[:12]))
	if existing, err := os.ReadFile(path); err == nil && string(existing) == string(payload) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("mcpscope: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".mcp-*.json")
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

// CodexOverrides disables each omitted server for one Codex launch, since the
// Codex registry is a shared host file that cannot hold a per-seat list.
func (sel Selection) CodexOverrides() []string {
	var args []string
	for _, name := range sel.Omitted {
		args = append(args, "-c", "mcp_servers."+name+".enabled=false")
	}
	return args
}

func claudeServer(s Server, home string) map[string]any {
	expand := func(v string) string { return strings.ReplaceAll(v, "${HOME}", home) }
	expandPath := func(v string) string {
		if strings.Contains(v, "${HOME}") {
			return filepath.Clean(filepath.FromSlash(expand(v)))
		}
		return v
	}
	expandMap := func(m map[string]string) map[string]string {
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[k] = expand(v)
		}
		return out
	}
	endpoint := s.URL
	if strings.TrimSpace(endpoint) == "" {
		endpoint = s.BaseURL
	}
	if strings.TrimSpace(endpoint) != "" {
		out := map[string]any{"type": "http", "url": expand(endpoint)}
		if len(s.Headers) > 0 {
			out["headers"] = expandMap(s.Headers)
		}
		return out
	}
	out := map[string]any{"command": expandPath(s.Command)}
	if len(s.Args) > 0 {
		args := make([]string, len(s.Args))
		for i, a := range s.Args {
			args[i] = expand(a)
		}
		out["args"] = args
	}
	if len(s.Env) > 0 {
		out["env"] = expandMap(s.Env)
	}
	if strings.TrimSpace(s.Cwd) != "" {
		out["cwd"] = expandPath(s.Cwd)
	}
	return out
}

func sortedNames(servers map[string]Server) []string {
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GooseArgs replaces goose's configured extensions for one session with the
// role's servers, keeping the builtin and platform extensions named in keep.
func (sel Selection) GooseArgs(keep []string, home string) ([]string, error) {
	args := []string{"--no-profile"}
	if len(keep) > 0 {
		args = append(args, "--with-builtin", strings.Join(keep, ","))
	}
	for _, name := range sel.Selected {
		s := sel.servers[name]
		if endpoint := serverEndpoint(s); endpoint != "" {
			// The flag takes a URL alone, so a header would be dropped silently.
			if len(s.Headers) > 0 {
				return nil, fmt.Errorf("mcpscope: goose cannot pass headers for %q", name)
			}
			args = append(args, "--with-streamable-http-extension", homeExpander(home)(endpoint))
			continue
		}
		spec, err := gooseStdio(name, s, home)
		if err != nil {
			return nil, err
		}
		args = append(args, "--with-extension", spec)
	}
	return args, nil
}

// gooseStdio renders `name:ENV=v command args` in the grammar goose splits it
// with: whitespace-separated, quotes group, and no backslash escapes.
func gooseStdio(name string, s Server, home string) (string, error) {
	if strings.TrimSpace(s.Cwd) != "" {
		return "", fmt.Errorf("mcpscope: goose cannot set a working directory for %q", name)
	}
	server := claudeServer(s, home)
	parts := make([]string, 0, len(s.Env)+len(s.Args)+1)
	env, _ := server["env"].(map[string]string)
	for _, key := range sortedKeys(env) {
		parts = append(parts, key+"="+env[key])
	}
	parts = append(parts, server["command"].(string))
	if args, ok := server["args"].([]string); ok {
		parts = append(parts, args...)
	}
	quoted := make([]string, len(parts))
	for i, part := range parts {
		q, err := gooseQuote(part)
		if err != nil {
			return "", fmt.Errorf("mcpscope: goose cannot express an argument of %q: %w", name, err)
		}
		quoted[i] = q
	}
	return name + ":" + strings.Join(quoted, " "), nil
}

func gooseQuote(part string) (string, error) {
	switch {
	case part == "":
		return "", errors.New("an empty argument splits to nothing")
	case !strings.ContainsAny(part, " \t\n\"'"):
		return part, nil
	case !strings.Contains(part, `"`):
		return `"` + part + `"`, nil
	case !strings.Contains(part, "'"):
		return "'" + part + "'", nil
	}
	return "", errors.New("it holds both quote characters")
}

// OpenCodeConfig is an inline config that defines the role's servers and turns
// off each omitted one, for OPENCODE_CONFIG_CONTENT. OpenCode merges it last.
func (sel Selection) OpenCodeConfig(home string) (string, error) {
	mcp := map[string]any{}
	for _, name := range sel.Selected {
		s := sel.servers[name]
		server := claudeServer(s, home)
		if endpoint := serverEndpoint(s); endpoint != "" {
			entry := map[string]any{"type": "remote", "url": server["url"], "enabled": true}
			if headers, ok := server["headers"]; ok {
				entry["headers"] = headers
			}
			mcp[name] = entry
			continue
		}
		if strings.TrimSpace(s.Cwd) != "" {
			return "", fmt.Errorf("mcpscope: opencode cannot set a working directory for %q", name)
		}
		command := []string{server["command"].(string)}
		if args, ok := server["args"].([]string); ok {
			command = append(command, args...)
		}
		entry := map[string]any{"type": "local", "command": command, "enabled": true}
		if env, ok := server["env"]; ok {
			entry["environment"] = env
		}
		mcp[name] = entry
	}
	for _, name := range sel.Omitted {
		mcp[name] = map[string]any{"enabled": false}
	}
	payload, err := json.Marshal(map[string]any{"mcp": mcp})
	if err != nil {
		return "", fmt.Errorf("mcpscope: render: %w", err)
	}
	return string(payload), nil
}

func serverEndpoint(s Server) string {
	if strings.TrimSpace(s.URL) != "" {
		return s.URL
	}
	return strings.TrimSpace(s.BaseURL)
}

func homeExpander(home string) func(string) string {
	return func(v string) string { return strings.ReplaceAll(v, "${HOME}", home) }
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
