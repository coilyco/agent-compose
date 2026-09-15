package compose

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogset"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

// OrgRoots expands every org a role uses into one root per contributing
// catalogue, bounded by the org's selectors. See docs/skill-selectors.md.
func OrgRoots(
	role string,
	sources []*schema.Source,
	set *catalogset.Set,
) ([]RootSource, error) {
	if set == nil {
		return nil, nil
	}
	roots := make([]RootSource, 0)
	seen := map[string]string{}
	for _, src := range sources {
		for _, use := range src.RoleOrgs[role] {
			definition, declared := src.Orgs[use.Org]
			if !declared {
				return nil, fmt.Errorf(
					"role %q uses org %q, which %s does not declare", role, use.Org, src.ID)
			}
			admitted, err := set.SelectOrg(definition.Owner, definition.Skills)
			if err != nil {
				return nil, fmt.Errorf("role %q org %q: %w", role, use.Org, err)
			}
			byCatalogue := map[string][]string{}
			paths := map[string]string{}
			for _, skill := range admitted {
				key := skill.Source.String()
				byCatalogue[key] = append(byCatalogue[key], skill.Name)
				paths[key] = skill.Root
			}
			for _, key := range sortedKeys(byCatalogue) {
				id := use.Org + ":" + sourceID(key)
				if previous, repeat := seen[id]; repeat {
					return nil, fmt.Errorf(
						"role %q admits %s from both %s and %s", role, key, previous, src.ID)
				}
				seen[id] = src.ID
				names := byCatalogue[key]
				sort.Strings(names)
				roots = append(roots, RootSource{
					ID:        id,
					Root:      paths[key],
					Catalogue: true,
					Scope:     schema.ProviderScopeOrg,
					Reason: fmt.Sprintf(
						"org %q admits this catalogue to role %q", use.Org, role),
					Skills: names,
				})
			}
		}
	}
	return roots, nil
}

// sourceID makes a rendered source usable as a bundle source id, which admits
// namespace colons and no separators. A redacted source stays redacted.
func sourceID(rendered string) string {
	return strings.ReplaceAll(strings.ReplaceAll(rendered, "/", ":"), " ", "-")
}

func sortedKeys(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
