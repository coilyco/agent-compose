// Package catalogset indexes the compiled catalogue superset by source, so an
// org can be expanded and a skill reached by address. See docs/skill-catalogues.md.
package catalogset

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogmanifest"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/skillselector"
)

// Catalog is one compiled catalogue and the skills it actually offers.
type Catalog struct {
	Source catalogmanifest.Source
	Path   string
	Skills []string
}

// Skill is one admitted skill, carrying the address a record names it by.
type Skill struct {
	Name    string
	Path    string
	Source  catalogmanifest.Source
	Address string
}

// Set is the compiled superset in manifest declaration order.
type Set struct {
	catalogs []Catalog
}

// Build reads each catalogue's skill directories. A skill is a directory
// carrying SKILL.md. See docs/skill-selectors.md.
func Build(entries []catalogmanifest.Catalog) (*Set, error) {
	set := &Set{catalogs: make([]Catalog, 0, len(entries))}
	for _, entry := range entries {
		names, err := readSkillNames(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("catalogue %s: %w", entry.Source, err)
		}
		set.catalogs = append(set.catalogs, Catalog{
			Source: entry.Source,
			Path:   entry.Path,
			Skills: names,
		})
	}
	return set, nil
}

func readSkillNames(root string) ([]string, error) {
	items, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read catalogue root %s: %w", root, err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		marker := filepath.Join(root, item.Name(), "SKILL.md")
		if info, err := os.Stat(marker); err == nil && !info.IsDir() {
			names = append(names, item.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Catalogs returns every catalogue in declaration order.
func (s *Set) Catalogs() []Catalog {
	return append([]Catalog(nil), s.catalogs...)
}

// Owner returns every catalogue under owner, in declaration order. The grant
// is open, so a repository added later needs no config change.
func (s *Set) Owner(owner string) []Catalog {
	found := make([]Catalog, 0, len(s.catalogs))
	for _, catalog := range s.catalogs {
		if catalog.Source.Owner == owner {
			found = append(found, catalog)
		}
	}
	return found
}

// SelectOrg admits the slice patterns bound, failing closed on an org holding
// no catalogue. Patterns apply across the org's whole surface.
func (s *Set) SelectOrg(owner string, patterns []string) ([]Skill, error) {
	catalogs := s.Owner(owner)
	if len(catalogs) == 0 {
		return nil, fmt.Errorf(
			"org %q matches no catalogue in the compiled set: an org that "+
				"contributes nothing is the same failure as an empty selector",
			owner,
		)
	}
	offered := make([]string, 0)
	owning := map[string][]Catalog{}
	for _, catalog := range catalogs {
		for _, name := range catalog.Skills {
			if _, seen := owning[name]; !seen {
				offered = append(offered, name)
			}
			owning[name] = append(owning[name], catalog)
		}
	}
	sort.Strings(offered)
	selected, _, err := skillselector.Select(patterns, offered)
	if err != nil {
		return nil, fmt.Errorf("org %q: %w", owner, err)
	}
	admitted := make([]Skill, 0, len(selected))
	for _, name := range selected {
		holders := owning[name]
		if len(holders) > 1 {
			return nil, fmt.Errorf(
				"org %q skill %q is owned by %s and %s",
				owner, name, holders[0].Source, holders[1].Source,
			)
		}
		admitted = append(admitted, skillOf(holders[0], name))
	}
	return admitted, nil
}

func skillOf(catalog Catalog, name string) Skill {
	return Skill{
		Name:    name,
		Path:    filepath.Join(catalog.Path, name),
		Source:  catalog.Source,
		Address: catalog.Source.SkillAddress(name),
	}
}

// Resolve reaches one skill by address, terse or forge-qualified. A terse
// address matching two forges is refused rather than resolved.
func (s *Set) Resolve(address string) (Skill, error) {
	forge, owner, repo, name, err := splitAddress(address)
	if err != nil {
		return Skill{}, err
	}
	matches := make([]Catalog, 0, 2)
	for _, catalog := range s.catalogs {
		if catalog.Source.Owner != owner || catalog.Source.Repo != repo {
			continue
		}
		if forge != "" && catalog.Source.Forge != forge {
			continue
		}
		if !contains(catalog.Skills, name) {
			continue
		}
		matches = append(matches, catalog)
	}
	switch len(matches) {
	case 0:
		return Skill{}, fmt.Errorf(
			"skill address %q resolves to nothing in the compiled set",
			address,
		)
	case 1:
		return skillOf(matches[0], name), nil
	default:
		return Skill{}, fmt.Errorf(
			"skill address %q is served by %s and %s: name the forge",
			address, matches[0].Source, matches[1].Source,
		)
	}
}

func splitAddress(address string) (forge, owner, repo, name string, err error) {
	trimmed := strings.Trim(strings.TrimSpace(address), "/")
	if trimmed == "" {
		return "", "", "", "", fmt.Errorf("skill address is empty")
	}
	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if segment == "" {
			return "", "", "", "", fmt.Errorf(
				"skill address %q has an empty segment", address)
		}
	}
	switch len(segments) {
	case 3:
		return "", segments[0], segments[1], segments[2], nil
	case 4:
		return segments[0], segments[1], segments[2], segments[3], nil
	default:
		return "", "", "", "", fmt.Errorf(
			"skill address %q must be owner/repo/skill, or forge/owner/repo/skill",
			address,
		)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
