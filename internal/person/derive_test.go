package person

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"
)

func roleFile(slug, body string) fstest.MapFS {
	return fstest.MapFS{
		dataRoot + "/role-" + slug + "/role" + yamlFragmentExt: {Data: []byte(body), Mode: 0o644},
	}
}

func derivedFields(t *testing.T, source fs.FS, slug string) map[string]any {
	t.Helper()
	raw, err := fs.ReadFile(source, dataRoot+"/role-"+slug+"/role"+yamlFragmentExt)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := deriveRoleFragment(source, slug, raw)
	if err != nil {
		t.Fatalf("derive %s: %v", slug, err)
	}
	var fields map[string]any
	if err := yaml.Unmarshal(merged, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func TestDeriveMergesMappingsReplacesListsAndDropsNull(t *testing.T) {
	source := mergeFS(
		roleFile("senior", "role: senior\norder: 2\nskill: role-senior\nguardrail: reversible-steps\n"+
			"boundaries: [a, b]\nvoice:\n  summary: parent summary\n  tell: parent tell\narchived: true\npinned_by_default: true\n"),
		roleFile("junior", "role: junior\norder: 11\nderives: senior\nguardrail: null\n"+
			"boundaries: [a]\nvoice:\n  tell: child tell\n"),
	)
	fields := derivedFields(t, source, "junior")
	if _, ok := fields["guardrail"]; ok {
		t.Error("an explicit null must remove the parent's guardrail")
	}
	if got := fields["boundaries"].([]any); len(got) != 1 || got[0] != "a" {
		t.Errorf("a child list replaces the parent's, got %v", got)
	}
	voice := fields["voice"].(map[string]any)
	if voice["summary"] != "parent summary" || voice["tell"] != "child tell" {
		t.Errorf("a mapping merges key by key, got %v", voice)
	}
	if fields["role"] != "junior" || fields["order"] != 11 || fields["skill"] != "role-junior" {
		t.Errorf("identity, slot, and charter stay the child's, got %v", fields)
	}
	if _, ok := fields["archived"]; ok {
		t.Error("retirement must not inherit")
	}
	if _, ok := fields["pinned_by_default"]; ok {
		t.Error("a pin default must not inherit")
	}
	if fields["color_twin"] != "senior" || fields["derives"] != "senior" {
		t.Errorf("derives implies color_twin and survives into the fragment, got %v", fields)
	}
}

func TestDeriveRefusesWhatWouldHideTheParent(t *testing.T) {
	cases := map[string]struct {
		source fstest.MapFS
		want   string
	}{
		"chained": {mergeFS(
			roleFile("a", "role: a\norder: 1\n"),
			roleFile("b", "role: b\norder: 2\nderives: a\n"),
			roleFile("c", "role: c\norder: 3\nderives: b\n"),
		), "no chaining"},
		"self":    {roleFile("c", "role: c\norder: 3\nderives: c\n"), "cannot name itself"},
		"missing": {roleFile("c", "role: c\norder: 3\nderives: ghost\n"), "not defined"},
		"no order": {mergeFS(
			roleFile("a", "role: a\norder: 1\n"),
			roleFile("c", "role: c\nderives: a\n"),
		), "its own order"},
		"other twin": {mergeFS(
			roleFile("a", "role: a\norder: 1\n"),
			roleFile("b", "role: b\norder: 2\n"),
			roleFile("c", "role: c\norder: 3\nderives: a\ncolor_twin: b\n"),
		), "differs from derives"},
	}
	for name, tc := range cases {
		raw, _ := fs.ReadFile(tc.source, dataRoot+"/role-c/role"+yamlFragmentExt)
		_, err := deriveRoleFragment(tc.source, "c", raw)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want %q, got %v", name, tc.want, err)
		}
	}
}

// shippedWith copies the mounted roster and lays extra files over it.
func shippedWith(t *testing.T, extra fstest.MapFS) fstest.MapFS {
	t.Helper()
	seed := seedForTest(t)
	out := fstest.MapFS{}
	err := fs.WalkDir(seed, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := fs.ReadFile(seed, path)
		out[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return mergeFS(out, extra)
}

const derivedEngineer = "role: junior-eng\norder: 90\nderives: eng-platform\n" +
	"display_name: Junior Engineer\nagents:\n  - harness: opencode\n    legal_name: OpenCode\n"

func TestDerivedRoleLoadsFromTheShippedRoster(t *testing.T) {
	source := shippedWith(t, roleFile("junior-eng", derivedEngineer))
	charter := strings.Replace(string(source[dataRoot+"/role-eng-platform/SKILL.md"].Data),
		"name: role-eng-platform", "name: role-junior-eng", 1)
	source[dataRoot+"/role-junior-eng/SKILL.md"] = &fstest.MapFile{Data: []byte(charter), Mode: 0o644}
	p, err := loadSource(source, "fixture")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := resolveAndValidatePerson(p); err != nil {
		t.Fatalf("validate: %v", err)
	}
	child, parent := p.Roles["junior-eng"], p.Roles["eng-platform"]
	if child.Derives != "eng-platform" || child.ColorTwin != "eng-platform" {
		t.Errorf("derives and its implied twin must reach the snapshot, got %q %q", child.Derives, child.ColorTwin)
	}
	if strings.Join(child.Personalities, ",") != strings.Join(parent.Personalities, ",") {
		t.Errorf("personalities inherit, got %v want %v", child.Personalities, parent.Personalities)
	}
	if child.Purpose != parent.Purpose || child.DisplayName != "Junior Engineer" {
		t.Errorf("undeclared fields inherit and declared ones win, got %q %q", child.Purpose, child.DisplayName)
	}
}

func TestDerivedRoleCannotOwnABoundary(t *testing.T) {
	path := dataRoot + "/boundary-build-foundational-software/boundary" + yamlFragmentExt
	source := shippedWith(t, roleFile("junior-eng", derivedEngineer))
	raw := strings.Replace(string(source[path].Data), "owner: eng-platform", "owner: junior-eng", 1)
	source[path] = &fstest.MapFile{Data: []byte(raw), Mode: 0o644}
	_, err := loadSource(source, "fixture")
	if err == nil || !strings.Contains(err.Error(), "is a derived role") {
		t.Fatalf("a derived boundary owner must be refused, got %v", err)
	}
}

func TestPinnedByDefaultLoadsFromARoleAndReachesTheCatalog(t *testing.T) {
	source := shippedWith(t, roleFile("junior-eng", derivedEngineer+"pinned_by_default: true\n"))
	charter := strings.Replace(string(source[dataRoot+"/role-eng-platform/SKILL.md"].Data),
		"name: role-eng-platform", "name: role-junior-eng", 1)
	source[dataRoot+"/role-junior-eng/SKILL.md"] = &fstest.MapFile{Data: []byte(charter), Mode: 0o644}
	p, err := loadSource(source, "fixture")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := resolveAndValidatePerson(p); err != nil {
		t.Fatalf("validate: %v", err)
	}
	entries, err := p.RoleCatalog()
	if err != nil {
		t.Fatal(err)
	}
	declared := false
	for _, entry := range entries {
		if entry.PinnedByDefault != p.Roles[entry.Slug].PinnedByDefault {
			t.Errorf("%s: the catalog says %v, the role says %v", entry.Slug, entry.PinnedByDefault, p.Roles[entry.Slug].PinnedByDefault)
		}
		declared = declared || (entry.Slug == "junior-eng" && entry.PinnedByDefault)
	}
	if !declared {
		t.Error("a role that declares pinned_by_default must read true in the catalog")
	}
	if p.Roles["eng-platform"].PinnedByDefault {
		t.Error("declaring it on a derived role must not pin its parent")
	}
}
