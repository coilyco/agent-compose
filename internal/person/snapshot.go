package person

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
)

const (
	SnapshotFormat        = "agent-compose.person-snapshot.v3"
	SnapshotSchemaVersion = 3
)

// Snapshot is the complete public person boundary emitted during convergence.
// Model selection, authority, and runtime routing stay outside this contract.
type Snapshot struct {
	Format         string                  `json:"format"`
	SchemaVersion  int                     `json:"schema_version"`
	Source         string                  `json:"source"`
	Person         string                  `json:"person"`
	RoleOrder      []string                `json:"role_order"`
	Roles          map[string]SnapshotRole `json:"roles"`
	BoundaryOrder  []string                `json:"boundary_order,omitempty"`
	Boundaries     map[string]Boundary     `json:"boundaries,omitempty"`
	Personalities  map[string]Personality  `json:"personalities"`
	GuardrailOrder []string                `json:"guardrail_order,omitempty"`
	Guardrails     map[string]Guardrail    `json:"guardrails,omitempty"`
	Expressions    []string                `json:"expressions"`
}

// SnapshotRole embeds the canonical role so future role fields enter the
// export automatically, FavoriteColor among them.
type SnapshotRole struct {
	Role
}

// BuildSnapshot converts the loaded person model without maintaining a second
// policy copy.
func BuildSnapshot(p *Person) (*Snapshot, error) {
	if p == nil {
		return nil, fmt.Errorf("build person snapshot: person is nil")
	}
	roles := make(map[string]SnapshotRole, len(p.Roles))
	for _, name := range p.RoleOrder {
		role, ok := p.Roles[name]
		if !ok {
			return nil, fmt.Errorf("build person snapshot: role order names missing role %q", name)
		}
		if role.Guardrail != "" {
			rail, ok := p.Guardrails[role.Guardrail]
			if !ok {
				return nil, fmt.Errorf(
					"build person snapshot: role %q names missing guardrail %q",
					name, role.Guardrail,
				)
			}
			if rail.Role != name {
				return nil, fmt.Errorf(
					"build person snapshot: role %q names guardrail %q, which claims role %q",
					name, role.Guardrail, rail.Role,
				)
			}
		}
		for _, personalityName := range role.Personalities {
			if _, ok := p.Personalities[personalityName]; !ok {
				return nil, fmt.Errorf(
					"build person snapshot: role %q names missing personality %q",
					name, personalityName,
				)
			}
		}
		roles[name] = SnapshotRole{Role: role}
	}
	if len(roles) != len(p.Roles) {
		return nil, fmt.Errorf(
			"build person snapshot: role order covers %d of %d roles",
			len(roles), len(p.Roles),
		)
	}
	return &Snapshot{
		Format:         SnapshotFormat,
		SchemaVersion:  SnapshotSchemaVersion,
		Source:         p.ProviderID(),
		Person:         p.Name,
		RoleOrder:      append([]string(nil), p.RoleOrder...),
		Roles:          roles,
		BoundaryOrder:  append([]string(nil), p.BoundaryOrder...),
		Boundaries:     p.Boundaries,
		Personalities:  p.Personalities,
		GuardrailOrder: append([]string(nil), p.GuardrailOrder...),
		Guardrails:     p.Guardrails,
		Expressions:    ExpressionVocabulary(),
	}, nil
}

func MarshalSnapshot(p *Person) ([]byte, error) {
	snapshot, err := BuildSnapshot(p)
	if err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal person snapshot: %w", err)
	}
	return append(raw, '\n'), nil
}

// SnapshotV4 is an additive inspection projection. The v3 snapshot remains
// available for consumers that have not adopted aliases and affinities.
type SnapshotV4 struct {
	Format        string                         `json:"format"`
	SchemaVersion int                            `json:"schema_version"`
	Person        string                         `json:"person"`
	Roles         map[string]SnapshotRole        `json:"roles"`
	Personalities map[string]SnapshotPersonality `json:"personalities"`
	Expressions   []string                       `json:"expressions"`
}

type SnapshotPersonality struct {
	Personality
	SourceLibrary string            `json:"source_library"`
	Digest        string            `json:"digest"`
	Affinities    []PersonalityMeld `json:"affinities"`
}

type PersonalityMeld struct {
	Role          string   `json:"role"`
	Personalities []string `json:"personalities"`
}

// BuildSnapshotV4 derives aliases and role affinity from the effective
// profile graph. Legacy complete packages are represented as one local source.
func BuildSnapshotV4(p *Person) (*SnapshotV4, error) {
	v3, err := BuildSnapshot(p)
	if err != nil {
		return nil, err
	}
	personalities := make(map[string]SnapshotPersonality, len(p.Personalities))
	for _, name := range p.personalityOrder() {
		binding := p.Personalities[name]
		raw, err := json.Marshal(binding)
		if err != nil {
			return nil, fmt.Errorf("marshal personality %q for digest: %w", name, err)
		}
		digest := sha256.Sum256(raw)
		sourceLibrary := p.PersonalityLibraries[name]
		if sourceLibrary == "" {
			sourceLibrary = p.localSourceID()
		}
		entry := SnapshotPersonality{
			Personality:   binding,
			SourceLibrary: sourceLibrary,
			Digest:        fmt.Sprintf("sha256:%x", digest),
		}
		for _, roleName := range p.RoleOrder {
			role := p.Roles[roleName]
			for _, member := range role.Personalities {
				if member == name {
					entry.Affinities = append(entry.Affinities, PersonalityMeld{
						Role:          roleName,
						Personalities: append([]string(nil), role.Personalities...),
					})
					break
				}
			}
		}
		personalities[name] = entry
	}
	return &SnapshotV4{
		Format:        "agent-compose.person-snapshot.v4",
		SchemaVersion: 4,
		Person:        p.Name,
		Roles:         v3.Roles,
		Personalities: personalities,
		Expressions:   v3.Expressions,
	}, nil
}

func MarshalSnapshotV4(p *Person) ([]byte, error) {
	snapshot, err := BuildSnapshotV4(p)
	if err != nil {
		return nil, err
	}
	if err := ValidateSnapshotV4(snapshot); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal person snapshot v4: %w", err)
	}
	return append(raw, '\n'), nil
}

// ValidateSnapshotV4 proves that aliases, provenance, and affinities form one
// self-consistent effective profile projection.
func ValidateSnapshotV4(snapshot *SnapshotV4) error {
	if snapshot == nil || snapshot.Format != "agent-compose.person-snapshot.v4" ||
		snapshot.SchemaVersion != 4 || snapshot.Person == "" {
		return fmt.Errorf("person snapshot v4 identity is incomplete")
	}
	for name, entry := range snapshot.Personalities {
		if entry.SourceLibrary == "" || len(entry.Digest) != 71 ||
			entry.Digest[:7] != "sha256:" {
			return fmt.Errorf("personality %q has invalid provenance", name)
		}
		seenAliases := map[string]bool{}
		for _, alias := range entry.Aliases {
			normalized, err := NormalizeCue(alias)
			if err != nil {
				return fmt.Errorf("personality %q alias: %w", name, err)
			}
			if seenAliases[normalized] {
				return fmt.Errorf("personality %q repeats normalized alias %q", name, normalized)
			}
			seenAliases[normalized] = true
		}
		seenRoles := map[string]bool{}
		for _, affinity := range entry.Affinities {
			role, ok := snapshot.Roles[affinity.Role]
			if !ok {
				return fmt.Errorf("personality %q affinity names unknown role %q", name, affinity.Role)
			}
			if seenRoles[affinity.Role] {
				return fmt.Errorf("personality %q repeats affinity role %q", name, affinity.Role)
			}
			seenRoles[affinity.Role] = true
			if !slices.Equal(affinity.Personalities, role.Personalities) ||
				!slices.Contains(affinity.Personalities, name) {
				return fmt.Errorf("personality %q affinity for role %q has inconsistent boundary", name, affinity.Role)
			}
		}
		for roleName, role := range snapshot.Roles {
			if slices.Contains(role.Personalities, name) != seenRoles[roleName] {
				return fmt.Errorf("personality %q affinity coverage disagrees with role %q", name, roleName)
			}
		}
	}
	return nil
}
