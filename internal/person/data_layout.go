package person

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing/fstest"

	"gopkg.in/yaml.v3"
)

// Entity data lives in one flat data/<kind>-<slug>/ directory per first-class
// entity. See docs/person-packages.md.
const dataRoot = "data"

var entityOrder = regexp.MustCompile(`(?m)^\s*order[ =](\d+)\s*$|(?m)\s+order=(\d+)`)

// maxEntityOrder is what the two-digit projected prefix can carry. Declaring
// past it is refused here. See docs/person-packages.md.
const maxEntityOrder = 99

var entityKinds = map[string]string{
	"role":        "roles",
	"personality": "personalities",
	"boundary":    "boundaries",
	"guardrail":   "guardrails",
}

// dataLayout projects the flat entity tree onto the section layout the loader
// consumes, so the on-disk shape and the parser stay independent.
func dataLayout(source fs.FS, label string) (fs.FS, bool, error) {
	entries, err := fs.ReadDir(source, dataRoot)
	if os.IsNotExist(err) {
		return source, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: read %s: %w", label, dataRoot, err)
	}
	projected := fstest.MapFS{}
	// A duplicate order projects to distinct filenames, so the tie falls to the
	// slug and nothing downstream can see it. See docs/person-packages.md.
	claimed := map[string]map[int]string{}
	if manifest, err := fs.ReadFile(source, "person"+yamlFragmentExt); err == nil {
		projected["person"+yamlFragmentExt] = &fstest.MapFile{Data: manifest, Mode: 0o644}
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, false, fmt.Errorf("%s: %s has unexpected file %q", label, dataRoot, entry.Name())
		}
		if entry.Name() == "invariant" {
			raw, err := fs.ReadFile(source, dataRoot+"/invariant/INVARIANT.md")
			if err != nil {
				return nil, false, fmt.Errorf("%s: read invariant: %w", label, err)
			}
			projected["definitions/INVARIANT.md"] = &fstest.MapFile{Data: raw, Mode: 0o644}
			continue
		}
		kind, slug, ok := splitEntityDirectory(entry.Name())
		if !ok {
			return nil, false, fmt.Errorf("%s: %s has unexpected entry %q", label, dataRoot, entry.Name())
		}
		order, err := projectEntity(source, projected, kind, slug, label)
		if err != nil {
			return nil, false, err
		}
		if taken, exists := claimed[kind][order]; exists {
			return nil, false, fmt.Errorf(
				"%s: %s %q and %q both declare order %d", label, kind, taken, slug, order)
		}
		if claimed[kind] == nil {
			claimed[kind] = map[int]string{}
		}
		claimed[kind][order] = slug
	}
	return projected, true, nil
}

func splitEntityDirectory(name string) (string, string, bool) {
	for kind := range entityKinds {
		if strings.HasPrefix(name, kind+"-") {
			slug := strings.TrimPrefix(name, kind+"-")
			if slug == "" {
				return "", "", false
			}
			return kind, slug, true
		}
	}
	return "", "", false
}

// projectEntity writes the entity into the section layout and reports the order
// it declared, which dataLayout uses to refuse a duplicate claim.
func projectEntity(source fs.FS, projected fstest.MapFS, kind, slug, label string) (int, error) {
	dir := dataRoot + "/" + kind + "-" + slug
	extension := yamlFragmentExt
	raw, err := fs.ReadFile(source, dir+"/"+kind+extension)
	if err != nil {
		return 0, fmt.Errorf("%s: read %s %q: %w", label, kind, slug, err)
	}
	order, fragment, err := entityOrderOf(string(raw), extension)
	if err != nil {
		return 0, fmt.Errorf("%s: %s %q: %w", label, kind, slug, err)
	}
	name := fmt.Sprintf("%02d-%s%s", order, slug, extension)
	projected[entityKinds[kind]+"/"+name] = &fstest.MapFile{Data: []byte(fragment), Mode: 0o644}
	if body, err := fs.ReadFile(source, dir+"/SKILL.md"); err == nil {
		path := "definitions/skills/" + kind + "-" + slug + "/SKILL.md"
		if kind == "role" {
			path = "roles/" + slug + "/SKILL.md"
		}
		projected[path] = &fstest.MapFile{Data: body, Mode: 0o644}
	}
	return order, nil
}

// entityOrderOf reads the order the entity declares. A YAML entity keeps the
// field, because its decoder already accepts it.
func entityOrderOf(fragment, extension string) (int, string, error) {
	if extension != yamlFragmentExt {
		return takeEntityOrder(fragment)
	}
	var declared struct {
		Order int `yaml:"order"`
	}
	if err := yaml.Unmarshal([]byte(fragment), &declared); err != nil {
		return 0, "", err
	}
	if declared.Order < 1 {
		return 0, "", fmt.Errorf("needs an order")
	}
	if declared.Order > maxEntityOrder {
		return 0, "", fmt.Errorf("order %d is above the maximum of %d", declared.Order, maxEntityOrder)
	}
	return declared.Order, fragment, nil
}

// takeEntityOrder reads the order the entity declares and removes it, so the
// parser never sees a field that exists only to sequence the roster.
func takeEntityOrder(fragment string) (int, string, error) {
	match := entityOrder.FindStringSubmatch(fragment)
	if match == nil {
		return 0, "", fmt.Errorf("needs an order")
	}
	digits := match[1]
	if digits == "" {
		digits = match[2]
	}
	order, err := strconv.Atoi(digits)
	if err != nil || order < 1 {
		return 0, "", fmt.Errorf("order %q is not a positive integer", digits)
	}
	if order > maxEntityOrder {
		return 0, "", fmt.Errorf("order %d is above the maximum of %d", order, maxEntityOrder)
	}
	stripped := entityOrder.ReplaceAllString(fragment, "")
	stripped = strings.ReplaceAll(stripped, "{\n\n", "{\n")
	return order, strings.TrimSpace(stripped) + "\n", nil
}
