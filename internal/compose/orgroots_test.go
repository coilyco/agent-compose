package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogmanifest"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogset"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func writeCatalogueSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func buildSet(t *testing.T, entries ...catalogmanifest.Catalog) *catalogset.Set {
	t.Helper()
	set, err := catalogset.Build(entries)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func catalogueSource(t *testing.T, raw, path string) catalogmanifest.Catalog {
	t.Helper()
	parsed, err := catalogmanifest.ParseSource(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	return catalogmanifest.Catalog{Path: path, Source: parsed}
}

func orgSource(owner string, patterns ...string) *schema.Source {
	return &schema.Source{
		ID: "aosk",
		Orgs: map[string]schema.OrgDefinition{
			"gaming": {ID: "gaming", Owner: owner, Skills: patterns},
		},
		RoleOrgs: map[string][]schema.OrgUse{"game-dev": {{Org: "gaming"}}},
	}
}

func TestOrgRootsExpandsOneRootPerContributingCatalogue(t *testing.T) {
	dir := t.TempDir()
	eco := filepath.Join(dir, "eco")
	ens := filepath.Join(dir, "ens")
	writeCatalogueSkill(t, eco, "sirens-game-eco")
	writeCatalogueSkill(t, eco, "coding-csharp")
	writeCatalogueSkill(t, ens, "sirens-game-enshrouded")
	set := buildSet(t,
		catalogueSource(t, "https://forgejo.example.test/coilyco-gaming/eco-ops/.agents/skills@main", eco),
		catalogueSource(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main", ens),
	)

	roots, err := OrgRoots("game-dev", []*schema.Source{orgSource("coilyco-gaming", "sirens-game-*")}, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 {
		t.Fatalf("roots = %+v, want one per contributing catalogue", roots)
	}
	for _, root := range roots {
		if !root.Catalogue || root.Scope != "org" {
			t.Fatalf("root = %+v", root)
		}
		if len(root.Skills) != 1 || !strings.HasPrefix(root.Skills[0], "sirens-game-") {
			t.Fatalf("root %q admitted %v, want only the selected slice", root.ID, root.Skills)
		}
	}
	// The unselected skill in the same catalogue must not ride along.
	for _, root := range roots {
		for _, skill := range root.Skills {
			if skill == "coding-csharp" {
				t.Fatal("a skill outside the org selector was admitted")
			}
		}
	}
}

// A catalogue root loads as a source, so an org grant reaches the bundle
// through the same path a provider does.
func TestLoadCatalogueReadsABareSkillsDirectory(t *testing.T) {
	dir := t.TempDir()
	writeCatalogueSkill(t, dir, "sirens-game-enshrouded")
	if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte("a: b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := schema.LoadCatalogue("gaming:enshrouded", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.Skills) != 1 || source.Skills[0].ID != "sirens-game-enshrouded" {
		t.Fatalf("skills = %+v, want only the directory carrying SKILL.md", source.Skills)
	}
	// The path is the skill name, because a catalogue has no .agents/skills.
	if source.Skills[0].Path != "sirens-game-enshrouded" {
		t.Fatalf("path = %q", source.Skills[0].Path)
	}
	if _, err := schema.LoadCatalogue("empty", filepath.Join(dir, "references")); err == nil {
		t.Fatal("a directory holding no skill loaded as a catalogue")
	}
}

func TestOrgRootsFailsClosed(t *testing.T) {
	dir := t.TempDir()
	eco := filepath.Join(dir, "eco")
	writeCatalogueSkill(t, eco, "sirens-game-eco")
	set := buildSet(t,
		catalogueSource(t, "https://forgejo.example.test/coilyco-gaming/eco-ops/.agents/skills@main", eco))

	if _, err := OrgRoots("game-dev", []*schema.Source{orgSource("coilyco-absent", "*")}, set); err == nil {
		t.Fatal("an org with no catalogue in the compiled set expanded")
	}
	if _, err := OrgRoots("game-dev", []*schema.Source{orgSource("coilyco-gaming", "nothing-*")}, set); err == nil {
		t.Fatal("a selector matching nothing expanded")
	}
	// No compiled set means no org roots, rather than a guess.
	roots, err := OrgRoots("game-dev", []*schema.Source{orgSource("coilyco-gaming", "*")}, nil)
	if err != nil || len(roots) != 0 {
		t.Fatalf("roots = %+v err = %v", roots, err)
	}
	// A role that uses no org gets none.
	roots, err = OrgRoots("platform-eng", []*schema.Source{orgSource("coilyco-gaming", "*")}, set)
	if err != nil || len(roots) != 0 {
		t.Fatalf("unrelated role got org roots: %+v %v", roots, err)
	}
}

// The end of #7732: an org grant reaches a rendered bundle. Before this the
// declaration parsed and expanded and changed no bundle at all.
func TestOrgRootComposesIntoARenderedBundle(t *testing.T) {
	dir := t.TempDir()
	ens := filepath.Join(dir, "ens")
	writeCatalogueSkill(t, ens, "sirens-game-enshrouded")
	writeCatalogueSkill(t, ens, "coding-csharp")
	set := buildSet(t,
		catalogueSource(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main", ens))

	roots, err := OrgRoots("game-dev", []*schema.Source{orgSource("coilyco-gaming", "sirens-game-*")}, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("roots = %+v", roots)
	}

	result, err := RunRoots(
		&schema.Request{Role: "game-dev", Delivery: schema.DeliveryNativeSkills},
		roots,
		t.TempDir(),
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, selected := range result.Resolution.Skills {
		found[selected.ID] = true
	}
	if !found["sirens-game-enshrouded"] {
		t.Fatalf("the org-admitted skill is not in the bundle: %+v", found)
	}
	if found["coding-csharp"] {
		t.Fatal("a skill outside the org selector reached the bundle")
	}
}
