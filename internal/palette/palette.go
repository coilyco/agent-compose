// Package palette projects a selected person source into the small JSON
// contract consumed by a personality palette explorer.
package palette

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/coilyco/agent-compose/v2/internal/person"
)

const schemaVersion = 2

type Personality struct {
	Name     string        `json:"name"`
	Color    string        `json:"color"`
	Motif    string        `json:"motif"`
	Geometry string        `json:"geometry"`
	Emblem   person.Emblem `json:"emblem"`
	Body     person.Body   `json:"body"`
}

type Role struct {
	Name          string   `json:"name"`
	Personalities []string `json:"personalities"`
	Color         string   `json:"color"`
}

type Document struct {
	Version       int           `json:"version"`
	Expressions   []string      `json:"expressions"`
	Personalities []Personality `json:"personalities"`
	Roles         []Role        `json:"roles"`
}

// Build derives browser data from the same person source used by composition.
func Build(p *person.Person) (Document, error) {
	names := make([]string, 0, len(p.Personalities))
	for name := range p.Personalities {
		names = append(names, name)
	}
	sort.Strings(names)

	doc := Document{
		Version:     schemaVersion,
		Expressions: person.ExpressionVocabulary(),
	}
	for _, name := range names {
		binding := p.Personalities[name]
		doc.Personalities = append(doc.Personalities, Personality{
			Name:     name,
			Color:    binding.Color,
			Motif:    binding.Motif,
			Geometry: binding.Geometry,
			Emblem:   binding.Emblem,
			Body:     binding.Body,
		})
	}

	for _, roleName := range p.RoleOrder {
		role, ok := p.Roles[roleName]
		if !ok {
			return Document{}, fmt.Errorf("role order names missing role %q", roleName)
		}
		colors := make([]string, 0, len(role.Personalities))
		for _, name := range role.Personalities {
			personality, ok := p.Personalities[name]
			if !ok {
				return Document{}, fmt.Errorf("role %q names missing personality %q", roleName, name)
			}
			colors = append(colors, personality.Color)
		}
		boundary := role.FavoriteColor
		doc.Roles = append(doc.Roles, Role{
			Name:          roleName,
			Personalities: append([]string(nil), role.Personalities...),
			Color:         boundary,
		})
	}
	return doc, nil
}

func Marshal(p *person.Person) ([]byte, error) {
	doc, err := Build(p)
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal palette data: %w", err)
	}
	return append(raw, '\n'), nil
}

// Write replaces generated palette data atomically so a failed refresh cannot
// leave the local explorer with a partial JSON document.
func Write(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create palette data directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".palette-*.tmp")
	if err != nil {
		return fmt.Errorf("create palette data temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(raw); err != nil {
		temp.Close()
		return fmt.Errorf("write palette data: %w", err)
	}
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return fmt.Errorf("set palette data mode: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close palette data: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace palette data: %w", err)
	}
	return nil
}
