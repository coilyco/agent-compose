package person

import (
	"fmt"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/color"
	"github.com/coilyco/agent-compose/v2/internal/schema"
)

// The value checks the KDL walker used to make inline. A typed decode already
// rejects every shape error it also carried. See docs/person-packages.md.

func validateDecodedPerson(p *Person) error {
	for _, name := range p.RoleOrder {
		role := p.Roles[name]
		if err := validateDecodedRole(name, &role); err != nil {
			return err
		}
		p.Roles[name] = role
	}
	for _, name := range p.PersonalityOrder {
		if err := validateDecodedPersonality(name, p.Personalities[name]); err != nil {
			return err
		}
	}
	for _, name := range p.BoundaryOrder {
		if err := validateDecodedBoundary(name, p.Boundaries[name]); err != nil {
			return err
		}
		// A derived role narrows its parent's charter, so it never owns one.
		if owner := p.Boundaries[name].Owner; owner != "" && p.Roles[owner].Derives != "" {
			return fmt.Errorf("boundary %q: owner %q is a derived role", name, owner)
		}
		// It may share its parent's own boundary, which narrows that charter.
		for _, co := range p.Boundaries[name].CoOwners {
			if !validSemanticToken(co) {
				return fmt.Errorf("boundary %q co-owner needs a stable role id", name)
			}
			if parent := p.Roles[co].Derives; parent != "" && parent != p.Boundaries[name].Owner {
				return fmt.Errorf(
					"boundary %q: co-owner %q derives %q, which does not own it",
					name, co, parent)
			}
		}
	}
	return nil
}

func validateDecodedRole(name string, role *Role) error {
	if role.Briefing != "" && role.Skill != "" {
		return fmt.Errorf("role %q cannot define both briefing and skill", name)
	}
	if len(role.Personalities) == 0 {
		return fmt.Errorf("role %q needs at least one personality", name)
	}
	if err := noRepeats(role.Personalities, func(value string) error {
		return fmt.Errorf("role %q repeats personality %q", name, value)
	}); err != nil {
		return err
	}
	if err := noRepeats(role.Boundaries, func(value string) error {
		return fmt.Errorf("role %q repeats boundary %q", name, value)
	}); err != nil {
		return err
	}
	for _, boundary := range role.Boundaries {
		if !validSemanticToken(boundary) {
			return fmt.Errorf("role %q: boundary needs one stable doctrine id", name)
		}
	}
	scoped := map[string]bool{}
	for _, entry := range role.ScopedBoundaries {
		if !validSemanticToken(entry.Name) {
			return fmt.Errorf("role %q: boundary-scoped needs one stable doctrine id", name)
		}
		if scoped[entry.Name] {
			return fmt.Errorf("role %q repeats scoped boundary %q", name, entry.Name)
		}
		scoped[entry.Name] = true
		for _, deferred := range role.Boundaries {
			if deferred == entry.Name {
				return fmt.Errorf("role %q both defers and scopes boundary %q", name, entry.Name)
			}
		}
	}
	if role.ColorTwin != "" {
		if !validSemanticToken(role.ColorTwin) {
			return fmt.Errorf("role %q: color_twin needs one stable role id", name)
		}
		if role.ColorTwin == name {
			return fmt.Errorf("role %q: color_twin cannot name itself", name)
		}
	}
	adjacent := map[string]bool{}
	for _, entry := range role.Adjacents {
		if !validSemanticToken(entry.Role) {
			return fmt.Errorf("role %q: adjacent needs one stable role id", name)
		}
		if entry.Role == name {
			return fmt.Errorf("role %q: adjacent cannot name itself", name)
		}
		if adjacent[entry.Role] {
			return fmt.Errorf("role %q repeats adjacent %q", name, entry.Role)
		}
		adjacent[entry.Role] = true
	}
	tiers := map[string]bool{}
	for _, tier := range role.SupportedModelTiers {
		if !schema.IsModelTier(tier) {
			return fmt.Errorf("role %q: unsupported model tier %q", name, tier)
		}
		if tiers[tier] {
			return fmt.Errorf("role %q repeats model tier %q", name, tier)
		}
		tiers[tier] = true
	}
	if err := noRepeats(role.Methods, func(value string) error {
		return fmt.Errorf("role %q repeats method skill %q", name, value)
	}); err != nil {
		return err
	}
	for _, method := range role.Methods {
		if !validSemanticToken(method) {
			return fmt.Errorf("role %q: method needs one stable skill id", name)
		}
		if method == role.Skill {
			return fmt.Errorf("role %q method skill %q duplicates its role skill", name, method)
		}
	}
	if role.Identity != nil {
		if strings.TrimSpace(role.Identity.Name) == "" {
			return fmt.Errorf("role %q: identity needs a name property", name)
		}
	}
	if err := resolveDecodedSeats(name, role); err != nil {
		return err
	}
	return validateDecodedCopyContract(name, role.CopyContract)
}

// resolveDecodedSeats applies role identity across the seats that inherit it,
// which is where a seat gets its name when the role names one.
func resolveDecodedSeats(name string, role *Role) error {
	seen := map[string]bool{}
	for index := range role.Seats {
		seat := &role.Seats[index]
		if seen[seat.Key] {
			return fmt.Errorf("role %q: duplicate seat %q", name, seat.Key)
		}
		seen[seat.Key] = true
		if seat.Tier != "" {
			if !schema.IsModelTier(seat.Tier) {
				return fmt.Errorf(
					"role %q: seat %q has unsupported model tier %q", name, seat.Key, seat.Tier)
			}
			if !role.SupportsModelTier(seat.Tier) {
				return fmt.Errorf(
					"role %q: seat %q uses model tier %q outside the role compatibility set",
					name, seat.Key, seat.Tier)
			}
		}
		if role.Identity != nil {
			if seat.Name != "" {
				return fmt.Errorf("role %q: seat %q cannot redefine role identity", name, seat.Key)
			}
			seat.Name = role.Identity.Name
			continue
		}
		// An unauthored, unnamed seat is not an error: resolveIdentities fills
		// it in once the full roster's species are in scope.
	}
	return nil
}

func validateDecodedCopyContract(name string, contract *CopyContract) error {
	if contract == nil {
		return nil
	}
	if contract.Scope != "tool-response" {
		return fmt.Errorf("role %q: copy-contract needs supported scope tool-response", name)
	}
	if len(contract.Rules) == 0 {
		return fmt.Errorf("role %q: copy-contract needs forbid rules", name)
	}
	seen := map[string]bool{}
	for _, rule := range contract.Rules {
		if strings.TrimSpace(rule.Forbid) == "" {
			return fmt.Errorf("role %q: copy-contract needs a rule", name)
		}
		if strings.TrimSpace(rule.Prefer) == "" {
			return fmt.Errorf("role %q: copy-contract forbid %q needs prefer", name, rule.Forbid)
		}
		if seen[rule.Forbid] {
			return fmt.Errorf("role %q: copy-contract repeats forbid %q", name, rule.Forbid)
		}
		seen[rule.Forbid] = true
	}
	return nil
}

func validateDecodedPersonality(name string, personality Personality) error {
	if strings.TrimSpace(personality.Skill) == "" {
		return fmt.Errorf("personality %q needs a skill property", name)
	}
	if strings.TrimSpace(personality.Color) == "" {
		return fmt.Errorf("personality %q needs a color property", name)
	}
	if err := color.Legible(personality.Color); err != nil {
		return fmt.Errorf("personality %q: %w", name, err)
	}
	if !validSemanticToken(personality.Motif) {
		return fmt.Errorf("personality %q needs a semantic motif property", name)
	}
	if !validSemanticToken(personality.Geometry) {
		return fmt.Errorf("personality %q needs a semantic geometry property", name)
	}
	if len(personality.Emblem.Names) == 0 || personality.Emblem.Emoji == "" {
		return fmt.Errorf("personality %q needs an emblem", name)
	}
	if personality.Body.Archetype == "" || personality.Body.Attachment == "" {
		return fmt.Errorf("personality %q needs a body", name)
	}
	seenVerb := map[string]bool{}
	for _, verb := range personality.Verbs {
		if strings.TrimSpace(verb) == "" {
			return fmt.Errorf("personality %q has an empty verb", name)
		}
		if seenVerb[verb] {
			return fmt.Errorf("personality %q repeats verb %q", name, verb)
		}
		seenVerb[verb] = true
	}
	seenAlias := map[string]bool{}
	for _, alias := range personality.Aliases {
		normalized, err := NormalizeCue(alias)
		if err != nil {
			return fmt.Errorf("personality %q alias %q: %w", name, alias, err)
		}
		if seenAlias[normalized] {
			return fmt.Errorf("personality %q repeats alias %q", name, alias)
		}
		seenAlias[normalized] = true
	}
	return nil
}

func validateDecodedBoundary(name string, boundary Boundary) error {
	if !validSemanticToken(name) {
		return fmt.Errorf("person boundary %q needs a stable doctrine id", name)
	}
	if strings.TrimSpace(boundary.Skill) == "" {
		return fmt.Errorf("boundary %q needs a skill property", name)
	}
	if strings.TrimSpace(boundary.Summary) == "" {
		return fmt.Errorf("boundary %q needs a summary property", name)
	}
	if strings.TrimSpace(boundary.Owner) == "" {
		return fmt.Errorf("boundary %q needs an owner property", name)
	}
	if !validSemanticToken(boundary.Owner) {
		return fmt.Errorf("boundary %q owner needs a stable role id", name)
	}
	return nil
}

func noRepeats(values []string, onRepeat func(string) error) error {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return onRepeat(value)
		}
		seen[value] = true
	}
	return nil
}
