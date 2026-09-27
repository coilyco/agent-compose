package person

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// actsOverlayFile appends estate acts to a mounted roster, with no count
// guideline. See docs/roster-composition.md.
const actsOverlayFile = "acts" + yamlFragmentExt

type actsOverlay struct {
	Overlay       string                             `yaml:"overlay"`
	Roles         map[string][]overlayAct            `yaml:"roles"`
	Personalities map[string][]overlayAct            `yaml:"personalities"`
	Boundaries    map[string]map[string][]overlayAct `yaml:"boundaries"`
}

type overlayAct struct {
	Tool string `yaml:"tool"`
	Text string `yaml:"text"`
}

// applyActsOverlay appends the root's acts.yaml, checking every name, because an
// act on a misspelled role reaches nobody and looks like an act nobody wrote.
func applyActsOverlay(source fs.FS, p *Person) error {
	raw, err := fs.ReadFile(source, actsOverlayFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	var overlay actsOverlay
	if err := decoder.Decode(&overlay); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: %w", actsOverlayFile, err)
	}
	if strings.TrimSpace(overlay.Overlay) == "" {
		return fmt.Errorf("%s: overlay must name itself", actsOverlayFile)
	}
	where := func(kind, name string) string {
		return fmt.Sprintf("%s %s %s %q", actsOverlayFile, overlay.Overlay, kind, name)
	}
	for _, name := range sortedKeys(overlay.Roles) {
		role, ok := p.Roles[name]
		if !ok {
			return fmt.Errorf("%s names no role in this roster", where("role", name))
		}
		appended, err := appendActs(where("role", name), role.Acts, overlay.Roles[name], "")
		if err != nil {
			return err
		}
		role.Acts = appended
		p.Roles[name] = role
	}
	for _, name := range sortedKeys(overlay.Personalities) {
		personality, ok := p.Personalities[name]
		if !ok {
			return fmt.Errorf("%s names no personality in this roster", where("personality", name))
		}
		appended, err := appendActs(where("personality", name), personality.Acts, overlay.Personalities[name], "")
		if err != nil {
			return err
		}
		personality.Acts = appended
		p.Personalities[name] = personality
	}
	for _, name := range sortedKeys(overlay.Boundaries) {
		boundary, ok := p.Boundaries[name]
		if !ok {
			return fmt.Errorf("%s names no boundary in this roster", where("boundary", name))
		}
		for _, side := range sortedKeys(overlay.Boundaries[name]) {
			if !slices.Contains(boundaryActSides, side) {
				return fmt.Errorf("%s side %q is not one of %s",
					where("boundary", name), side, strings.Join(boundaryActSides, ", "))
			}
			appended, err := appendActs(where("boundary", name)+" "+side, boundary.Acts,
				overlay.Boundaries[name][side], side)
			if err != nil {
				return err
			}
			boundary.Acts = appended
		}
		p.Boundaries[name] = boundary
	}
	return nil
}

// appendActs keeps the shipped acts first, so a host holding both runs the
// portable one before the estate one.
func appendActs(owner string, base []Act, extra []overlayAct, side string) ([]Act, error) {
	merged := slices.Clone(base)
	seen := map[string]bool{}
	for _, act := range base {
		seen[act.Text] = true
	}
	for _, act := range extra {
		tool, text := strings.TrimSpace(act.Tool), strings.TrimSpace(act.Text)
		if tool == "" || text == "" {
			return nil, fmt.Errorf("%s has an act without a tool or text", owner)
		}
		if !strings.Contains(text, tool) {
			return nil, fmt.Errorf("%s act %q does not name its tool %q", owner, text, tool)
		}
		if seen[text] {
			return nil, fmt.Errorf("%s repeats act %q", owner, text)
		}
		seen[text] = true
		merged = append(merged, Act{Tool: tool, Text: text, Side: side})
	}
	return merged, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
