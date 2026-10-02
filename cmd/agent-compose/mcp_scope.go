package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/mcpscope"
	"github.com/coilyco/agent-compose/v2/internal/person"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// mcpLaunch is the per-harness form of one role's MCP selection.
type mcpLaunch struct {
	ClaudeConfig string
	CodexArgs    []string
	// GooseArgs follow goose's session verb, which is the only place they parse.
	GooseArgs []string
	// Env is set on the harness process, for OpenCode's inline config.
	Env []string
}

// openCodeConfigEnv is OpenCode's inline config, merged over every other layer.
const openCodeConfigEnv = "OPENCODE_CONFIG_CONTENT"

// mcpInventoryHome is the home the host projection expands ${HOME} against, so a
// scoped server launches exactly as the host registry would launch it.
func mcpInventoryHome() (string, error) {
	if h := strings.TrimSpace(os.Getenv("AOS_NATIVE_CANONICAL_HOME")); h != "" {
		return h, nil
	}
	return os.UserHomeDir()
}

func loadMCPInventory(p *person.Person, path string) (*mcpscope.Inventory, error) {
	entries, err := p.RoleCatalog()
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(entries))
	for _, e := range entries {
		roles = append(roles, e.Slug)
	}
	return mcpscope.Load(path, roles)
}

// nativeMCPScope narrows the host MCP inventory to role for one launch, and
// fails closed rather than load the user-level set. See docs/launch.md.
func nativeMCPScope(w io.Writer, loadPerson func() (*person.Person, error), role, harness, stateDir string, args []string) (mcpLaunch, error) {
	switch harness {
	case "claude":
		if nativeArgsCarry(args, "--mcp-config") || nativeArgsCarry(args, "--strict-mcp-config") {
			return mcpLaunch{}, nil
		}
	case "codex":
	case "goose":
		if !gooseScopeApplies(args) {
			return mcpLaunch{}, nil
		}
	case "opencode":
		if strings.TrimSpace(os.Getenv(openCodeConfigEnv)) != "" {
			return mcpLaunch{}, nil
		}
	default:
		return mcpLaunch{}, nil
	}
	p, err := loadPerson()
	if err != nil {
		return mcpLaunch{}, fmt.Errorf("load the roster the MCP scope reads: %w", err)
	}
	if p == nil {
		return mcpLaunch{}, errors.New("no roster to scope MCP servers against")
	}
	home, err := mcpInventoryHome()
	if err != nil {
		return mcpLaunch{}, fmt.Errorf("resolve MCP inventory home: %w", err)
	}
	inv, err := loadMCPInventory(p, filepath.Join(home, ".mcporter", "mcporter.json"))
	if errors.Is(err, mcpscope.ErrNoInventory) {
		return mcpLaunch{}, fmt.Errorf("%w; pass --mcp-config to launch with a scope of your own", err)
	}
	if err != nil {
		return mcpLaunch{}, err
	}
	sel := inv.Select(role)
	fmt.Fprintln(w, sel.Summary())
	switch harness {
	case "codex":
		return mcpLaunch{CodexArgs: sel.CodexOverrides()}, nil
	case "goose":
		keep, err := gooseKeptExtensions(home)
		if err != nil {
			return mcpLaunch{}, err
		}
		gooseArgs, err := sel.GooseArgs(keep, home)
		return mcpLaunch{GooseArgs: gooseArgs}, err
	case "opencode":
		content, err := sel.OpenCodeConfig(home)
		return mcpLaunch{Env: []string{openCodeConfigEnv + "=" + content}}, err
	}
	path, err := sel.WriteClaude(filepath.Join(stateDir, "mcp"), home)
	if err != nil {
		return mcpLaunch{}, err
	}
	return mcpLaunch{ClaudeConfig: path}, nil
}

// gooseSessionVerbs are the subcommands that load extensions. Bare `goose`
// starts a session too, but its root command takes none of the scope flags.
var gooseSessionVerbs = map[string]bool{"session": true, "s": true, "run": true}

// gooseScopeApplies is false for a verb that loads no extensions, a caller's
// own --no-profile, and a resume, which restores the set the session recorded.
func gooseScopeApplies(args []string) bool {
	if len(args) > 0 && !gooseSessionVerbs[args[0]] {
		return false
	}
	return !nativeArgsCarry(args, "--no-profile") && !nativeArgsCarry(args, "--resume") &&
		!nativeArgsCarry(args, "-r")
}

// gooseCommand puts the scope after the session verb, adding `session` to a
// bare launch.
func gooseCommand(args, scope []string) []string {
	verb, rest := "session", args
	if len(args) > 0 {
		verb, rest = args[0], args[1:]
	}
	command := append([]string{"goose", verb}, scope...)
	return append(command, rest...)
}

// gooseKeptExtensions lists the enabled builtin and platform extensions in the
// user's goose config, which --no-profile would otherwise drop with the rest.
func gooseKeptExtensions(home string) ([]string, error) {
	dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if dir == "" {
		dir = filepath.Join(home, ".config")
	}
	path := filepath.Join(dir, "goose", "config.yaml")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read goose config: %w", err)
	}
	var config struct {
		Extensions map[string]struct {
			Enabled bool   `yaml:"enabled"`
			Type    string `yaml:"type"`
		} `yaml:"extensions"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("parse goose config %s: %w", path, err)
	}
	var keep []string
	for name, extension := range config.Extensions {
		if extension.Enabled && (extension.Type == "builtin" || extension.Type == "platform") {
			keep = append(keep, name)
		}
	}
	sort.Strings(keep)
	return keep, nil
}

// runMCP reports one role's resolved MCP set without launching anything.
func runMCP(_ context.Context, cmd *cli.Command) error {
	role := strings.TrimSpace(cmd.String("role"))
	p, _, err := loadSelectedPersonWithLibraries(cmd.String("person-source"), nil)
	if err != nil {
		return err
	}
	path := cmd.String("inventory")
	if path == "" {
		home, err := mcpInventoryHome()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".mcporter", "mcporter.json")
	}
	inv, err := loadMCPInventory(p, path)
	if err != nil {
		return err
	}
	sel := inv.Select(role)
	w := cmd.Root().Writer
	fmt.Fprintln(w, strings.TrimPrefix(sel.Summary(), "agent-compose: "))
	for _, name := range sel.Selected {
		fmt.Fprintf(w, "+ %s\n", name)
	}
	for _, name := range sel.Omitted {
		fmt.Fprintf(w, "- %s\n", name)
	}
	return nil
}
