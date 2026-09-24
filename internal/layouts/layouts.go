// Package layouts loads the harness load-point table, so no harness name lives
// in Go. The default is layouts.yaml, and AGENT_COMPOSE_LAYOUTS names a file
// that replaces it whole. See docs/projection.md.
package layouts

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Env names the replacement table, which a deployment mounts to own the harness set.
const Env = "AGENT_COMPOSE_LAYOUTS"

//go:embed layouts.yaml
var defaultTable []byte

type LoadPoints struct {
	Instructions string `yaml:"instructions"`
	SkillsDir    string `yaml:"skills"`
}

// Layout declares load points per delivery mode. A nil mode is unsupported.
type Layout struct {
	Native   *LoadPoints `yaml:"native"`
	Compiled *LoadPoints `yaml:"compiled"`
}

type Harness struct {
	Cascade bool   `yaml:"cascade"`
	Repo    Layout `yaml:"repo"`
	Home    Layout `yaml:"home"`
}

// Load reads the replacement table when Env names one, else the default.
func Load() (map[string]Harness, error) {
	if override := strings.TrimSpace(os.Getenv(Env)); override != "" {
		raw, err := os.ReadFile(override)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", Env, err)
		}
		return Parse(raw, override)
	}
	return Parse(defaultTable, "layouts.yaml")
}

// Parse decodes a table strictly and rejects a load point that escapes its root.
func Parse(raw []byte, source string) (map[string]Harness, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	var table map[string]Harness
	if err := decoder.Decode(&table); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("%s: trailing YAML document", source)
	}
	if len(table) == 0 {
		return nil, fmt.Errorf("%s: declares no harness", source)
	}
	for _, name := range Names(table) {
		harness := table[name]
		for scope, layout := range map[string]Layout{"repo": harness.Repo, "home": harness.Home} {
			if layout.Native == nil && layout.Compiled == nil {
				return nil, fmt.Errorf("%s: %s.%s declares no delivery mode", source, name, scope)
			}
			for mode, points := range map[string]*LoadPoints{"native": layout.Native, "compiled": layout.Compiled} {
				if points == nil {
					continue
				}
				where := fmt.Sprintf("%s: %s.%s.%s", source, name, scope, mode)
				if err := checkRelative(where+".instructions", points.Instructions, true); err != nil {
					return nil, err
				}
				if err := checkRelative(where+".skills", points.SkillsDir, false); err != nil {
					return nil, err
				}
			}
		}
		if harness.Cascade && (harness.Home.Native == nil || harness.Home.Native.SkillsDir == "") {
			return nil, fmt.Errorf("%s: %s sets cascade without home native instructions and skills", source, name)
		}
	}
	return table, nil
}

func checkRelative(where, value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", where)
		}
		return nil
	}
	clean := path.Clean(value)
	if path.IsAbs(value) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%s %q must stay relative to its root", where, value)
	}
	return nil
}

// Names returns the harness names sorted, so diagnostics are stable.
func Names(table map[string]Harness) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
