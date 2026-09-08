package person

import (
	"fmt"
	"io/fs"
	"path"
)

type PersonalityCatalogEntry struct {
	Slug          string            `json:"slug"`
	Skill         string            `json:"skill"`
	Description   string            `json:"description"`
	Aliases       []string          `json:"aliases"`
	Color         string            `json:"color"`
	Motif         string            `json:"motif"`
	Geometry      string            `json:"geometry"`
	Emblem        Emblem            `json:"emblem"`
	Body          Body              `json:"body"`
	SoundMark     SoundMark         `json:"sound_mark"`
	SourceLibrary string            `json:"source_library"`
	Digest        string            `json:"digest"`
	Affinities    []PersonalityMeld `json:"affinities"`
}

type RoleCatalogEntry struct {
	Slug          string         `json:"slug"`
	DisplayName   string         `json:"display_name"`
	Purpose       string         `json:"purpose"`
	Skill         string         `json:"skill"`
	SkillSource   string         `json:"skill_source"`
	SkillDigest   string         `json:"skill_digest"`
	Methods       []string       `json:"methods,omitempty"`
	Identity      *AgentIdentity `json:"identity,omitempty"`
	Seats         []Seat         `json:"seats"`
	Personalities []string       `json:"personalities"`
	FavoriteColor string         `json:"favorite_color"`
	Background    string         `json:"background"`
	Archived      bool           `json:"archived,omitempty"`
}

type SeatCatalogEntry struct {
	Role string `json:"role"`
	Seat Seat   `json:"seat"`
}

func (p *Person) PersonalityCatalog(names []string) ([]PersonalityCatalogEntry, error) {
	snapshot, err := BuildSnapshotV4(p)
	if err != nil {
		return nil, err
	}
	if err := ValidateSnapshotV4(snapshot); err != nil {
		return nil, err
	}
	source, err := Source(p)
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = p.personalityOrder()
	}
	out := make([]PersonalityCatalogEntry, 0, len(names))
	for _, name := range names {
		binding, ok := p.Personalities[name]
		if !ok {
			return nil, fmt.Errorf("catalogue personality %q is not defined", name)
		}
		raw, err := fs.ReadFile(source.FileSystem(), path.Join(
			"skills", binding.Skill, "SKILL.md",
		))
		if err != nil {
			return nil, fmt.Errorf("read personality %q definition: %w", name, err)
		}
		description, err := skillDescription(raw)
		if err != nil {
			return nil, fmt.Errorf("personality %q: %w", name, err)
		}
		projected := snapshot.Personalities[name]
		out = append(out, PersonalityCatalogEntry{
			Slug:          name,
			Skill:         binding.Skill,
			Description:   description,
			Aliases:       append([]string(nil), binding.Aliases...),
			Color:         binding.Color,
			Motif:         binding.Motif,
			Geometry:      binding.Geometry,
			Emblem:        binding.Emblem,
			Body:          binding.Body,
			SoundMark:     binding.SoundMark,
			SourceLibrary: projected.SourceLibrary,
			Digest:        projected.Digest,
			Affinities:    append([]PersonalityMeld(nil), projected.Affinities...),
		})
	}
	return out, nil
}

func (p *Person) RoleCatalog() ([]RoleCatalogEntry, error) {
	snapshot, err := BuildSnapshot(p)
	if err != nil {
		return nil, err
	}
	out := make([]RoleCatalogEntry, 0, len(p.RoleOrder))
	for _, name := range p.RoleOrder {
		role := p.Roles[name]
		out = append(out, RoleCatalogEntry{
			Slug:          name,
			DisplayName:   p.RoleDisplayName(name),
			Purpose:       role.Purpose,
			Skill:         p.RoleSkillID(name),
			SkillSource:   role.SkillSource,
			SkillDigest:   role.SkillDigest,
			Methods:       append([]string(nil), role.Methods...),
			Identity:      role.Identity,
			Seats:         append([]Seat(nil), role.Seats...),
			Personalities: append([]string(nil), role.Personalities...),
			FavoriteColor: snapshot.Roles[name].FavoriteColor,
			Background:    role.Background,
			Archived:      role.Archived,
		})
	}
	return out, nil
}

func (p *Person) SeatCatalog(roleFilter string) ([]SeatCatalogEntry, error) {
	roleNames := p.RoleOrder
	if roleFilter != "" {
		if _, ok := p.Roles[roleFilter]; !ok {
			return nil, fmt.Errorf("role %q is not defined", roleFilter)
		}
		roleNames = []string{roleFilter}
	}
	var out []SeatCatalogEntry
	for _, roleName := range roleNames {
		for _, seat := range p.Roles[roleName].Seats {
			out = append(out, SeatCatalogEntry{Role: roleName, Seat: seat})
		}
	}
	return out, nil
}

// BoundaryMatrixEntry is one boundary row: its verb per role, in role order.
// See docs/role-boundaries.md. agent-compose#325
type BoundaryMatrixEntry struct {
	Boundary string            `json:"boundary"`
	Owner    string            `json:"owner"`
	Verbs    map[string]string `json:"verbs"`
}

// BoundaryMatrix returns one row per boundary in stable boundary order, with a
// single verb per role rather than a justification sentence.
func (p *Person) BoundaryMatrix() ([]string, []BoundaryMatrixEntry) {
	roles := p.roleOrder()
	out := make([]BoundaryMatrixEntry, 0, len(p.Boundaries))
	for _, name := range p.boundaryOrder() {
		entry := BoundaryMatrixEntry{
			Boundary: name,
			Owner:    p.Boundaries[name].Owner,
			Verbs:    make(map[string]string, len(roles)),
		}
		for _, roleName := range roles {
			entry.Verbs[roleName] = boundaryVerb(p.Roles[roleName], name, entry.Owner == roleName)
		}
		out = append(out, entry)
	}
	return roles, out
}

func boundaryVerb(role Role, boundary string, owns bool) string {
	if owns {
		return "OWNS"
	}
	for _, scoped := range role.ScopedBoundaries {
		if scoped.Name == boundary {
			return "scope"
		}
	}
	for _, declared := range role.Boundaries {
		if declared == boundary {
			return "defers"
		}
	}
	return "-"
}
