package person

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// A derived role declares only its deltas against one parent. Merge rules and
// what never inherits: docs/roster-composition.md.

// uninherited keys always come from the child, because inheriting them would
// make the child answer to its parent's name, slot, charter, or retirement.
var uninherited = map[string]bool{
	"role":     true,
	"order":    true,
	"skill":    true,
	"archived": true,
	// A coordinator's pin is that role's, so a role derived from it does not carry it.
	"pinned_by_default": true,
}

// deriveRoleFragment returns the role fragment with its parent merged under it,
// or the fragment unchanged when it derives from nothing.
func deriveRoleFragment(source fs.FS, slug string, raw []byte) ([]byte, error) {
	var child map[string]any
	if err := yaml.Unmarshal(raw, &child); err != nil {
		return nil, err
	}
	parentSlug, _ := child["derives"].(string)
	if parentSlug == "" {
		if _, declared := child["derives"]; declared {
			return nil, fmt.Errorf("derives needs one parent role id")
		}
		return raw, nil
	}
	if parentSlug == slug {
		return nil, fmt.Errorf("derives cannot name itself")
	}
	for _, key := range []string{"role", "order"} {
		if _, ok := child[key]; !ok {
			return nil, fmt.Errorf("a derived role declares its own %s", key)
		}
	}
	parentRaw, err := fs.ReadFile(source, dataRoot+"/role-"+parentSlug+"/role"+yamlFragmentExt)
	if err != nil {
		return nil, fmt.Errorf("derives %q, which is not defined", parentSlug)
	}
	var parent map[string]any
	if err := yaml.Unmarshal(parentRaw, &parent); err != nil {
		return nil, fmt.Errorf("derives %q: %w", parentSlug, err)
	}
	if _, chained := parent["derives"]; chained {
		return nil, fmt.Errorf("derives %q, which itself derives, no chaining", parentSlug)
	}
	for key := range uninherited {
		delete(parent, key)
	}
	merged := mergeRoleFields(parent, child)
	if _, ok := merged["skill"]; !ok {
		merged["skill"] = "role-" + slug
	}
	if twin, ok := merged["color_twin"]; !ok {
		merged["color_twin"] = parentSlug
	} else if twin != parentSlug {
		return nil, fmt.Errorf("color_twin %v differs from derives %q", twin, parentSlug)
	}
	return yaml.Marshal(merged)
}

// mergeRoleFields lays child over parent: mappings merge key by key, anything
// else replaces, and an explicit null removes the parent's key.
func mergeRoleFields(parent, child map[string]any) map[string]any {
	out := make(map[string]any, len(parent)+len(child))
	for key, value := range parent {
		out[key] = value
	}
	for key, value := range child {
		if value == nil {
			delete(out, key)
			continue
		}
		if childMap, ok := value.(map[string]any); ok {
			if parentMap, ok := out[key].(map[string]any); ok {
				out[key] = mergeRoleFields(parentMap, childMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}
