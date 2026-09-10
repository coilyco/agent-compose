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
		if err := projectEntity(source, projected, kind, slug, label); err != nil {
			return nil, false, err
		}
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

func projectEntity(source fs.FS, projected fstest.MapFS, kind, slug, label string) error {
	dir := dataRoot + "/" + kind + "-" + slug
	extension := yamlFragmentExt
	raw, err := fs.ReadFile(source, dir+"/"+kind+extension)
	if err != nil {
		return fmt.Errorf("%s: read %s %q: %w", label, kind, slug, err)
	}
	order, fragment, err := entityOrderOf(string(raw), extension)
	if err != nil {
		return fmt.Errorf("%s: %s %q: %w", label, kind, slug, err)
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
	return nil
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
	stripped := entityOrder.ReplaceAllString(fragment, "")
	stripped = strings.ReplaceAll(stripped, "{\n\n", "{\n")
	return order, strings.TrimSpace(stripped) + "\n", nil
}
