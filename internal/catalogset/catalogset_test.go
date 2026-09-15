package catalogset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogmanifest"
)

func makeCatalog(t *testing.T, root string, skills ...string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range skills {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func source(t *testing.T, raw string) catalogmanifest.Source {
	t.Helper()
	parsed, err := catalogmanifest.ParseSource(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func build(t *testing.T, entries ...catalogmanifest.Catalog) *Set {
	t.Helper()
	set, err := Build(entries)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// A skill is a directory carrying SKILL.md. Five repositories keep a
// categories.yaml beside their skills, and a bare "*" must not admit it.
func TestBuildAdmitsOnlyDirectoriesCarryingASkillFile(t *testing.T) {
	root := makeCatalog(t, filepath.Join(t.TempDir(), "catalogue"), "coding-go", "repo-lore")
	if err := os.WriteFile(filepath.Join(root, "categories.yaml"), []byte("a: b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	set := build(t, catalogmanifest.Catalog{
		Path:   root,
		Source: source(t, "https://forgejo.example.test/org/one/.agents/skills@main"),
	})
	got := set.Catalogs()[0].Skills
	if strings.Join(got, ",") != "coding-go,repo-lore" {
		t.Fatalf("skills = %v, want the two carrying SKILL.md", got)
	}
}

func TestOwnerExpandsEveryCatalogueUnderIt(t *testing.T) {
	dir := t.TempDir()
	set := build(t,
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "a"), "sirens-game-eco"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/eco-ops/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "b"), "repo-lore"),
			Source: source(t, "https://forgejo.example.test/coilyco-bridge/lore/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "c"), "sirens-game-enshrouded"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main"),
		},
	)
	found := set.Owner("coilyco-gaming")
	if len(found) != 2 ||
		found[0].Source.Repo != "eco-ops" ||
		found[1].Source.Repo != "enshrouded" {
		t.Fatalf("owner expansion = %+v", found)
	}
}

func TestSelectOrgBoundsTheGrantAndFailsClosed(t *testing.T) {
	dir := t.TempDir()
	set := build(t,
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "a"), "sirens-game-eco", "coding-csharp"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/eco-ops/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "b"), "sirens-game-enshrouded"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main"),
		},
	)

	admitted, err := set.SelectOrg("coilyco-gaming", []string{"sirens-game-*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(admitted) != 2 {
		t.Fatalf("admitted = %+v, want both game focuses across two repos", admitted)
	}
	// Ordered by skill name rather than catalogue, and the address crosses the
	// repository boundary the org grant exists to erase.
	if admitted[0].Address != "forgejo.example.test/coilyco-gaming/eco-ops/sirens-game-eco" {
		t.Fatalf("address = %q", admitted[0].Address)
	}
	if admitted[1].Address != "forgejo.example.test/coilyco-gaming/enshrouded/sirens-game-enshrouded" {
		t.Fatalf("address = %q", admitted[1].Address)
	}

	if _, err := set.SelectOrg("coilyco-gaming", []string{"sirens-game-*", "nothing-*"}); err == nil {
		t.Fatal("a pattern matching nothing was admitted")
	}
	if _, err := set.SelectOrg("coilyco-absent", []string{"*"}); err == nil {
		t.Fatal("an org with no catalogue was admitted")
	}
}

// A name owned by two catalogues in one org is fatal and names both, rather
// than the last entry silently winning as a single manifest read would.
func TestSelectOrgRefusesASkillOwnedTwice(t *testing.T) {
	dir := t.TempDir()
	set := build(t,
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "a"), "shared-skill"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/one/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "b"), "shared-skill"),
			Source: source(t, "https://forgejo.example.test/coilyco-gaming/two/.agents/skills@main"),
		},
	)
	_, err := set.SelectOrg("coilyco-gaming", []string{"shared-*"})
	if err == nil {
		t.Fatal("a duplicate skill name was admitted")
	}
	for _, want := range []string{"coilyco-gaming/one", "coilyco-gaming/two"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %s", err, want)
		}
	}
}

func TestResolveReachesASkillByAddress(t *testing.T) {
	dir := t.TempDir()
	root := makeCatalog(t, filepath.Join(dir, "a"), "sirens-game-enshrouded")
	set := build(t, catalogmanifest.Catalog{
		Path:   root,
		Source: source(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main"),
	})
	for _, address := range []string{
		"coilyco-gaming/enshrouded/sirens-game-enshrouded",
		"forgejo.example.test/coilyco-gaming/enshrouded/sirens-game-enshrouded",
	} {
		got, err := set.Resolve(address)
		if err != nil {
			t.Fatalf("%s: %v", address, err)
		}
		if got.Path != filepath.Join(root, "sirens-game-enshrouded") {
			t.Fatalf("%s resolved to %q", address, got.Path)
		}
	}
}

func TestResolveFailsAndNamesTheAddress(t *testing.T) {
	dir := t.TempDir()
	set := build(t, catalogmanifest.Catalog{
		Path:   makeCatalog(t, filepath.Join(dir, "a"), "present"),
		Source: source(t, "https://forgejo.example.test/coilyco-gaming/enshrouded/.agents/skills@main"),
	})
	for name, address := range map[string]string{
		"absent skill": "coilyco-gaming/enshrouded/missing",
		"absent repo":  "coilyco-gaming/other/present",
		"wrong forge":  "github.com/coilyco-gaming/enshrouded/present",
		"too short":    "enshrouded/present",
		"too long":     "a/b/c/d/e",
		"empty":        "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := set.Resolve(address)
			if err == nil {
				t.Fatalf("address %q resolved", address)
			}
			if address != "" && !strings.Contains(err.Error(), address) {
				t.Fatalf("error %q does not name the address", err)
			}
		})
	}
}

// The R0 ambiguity, one level down: a terse address matching two forges is
// refused rather than picking one, and the error names both.
func TestResolveRefusesATerseAddressServedByTwoForges(t *testing.T) {
	dir := t.TempDir()
	set := build(t,
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "a"), "shared"),
			Source: source(t, "https://forgejo.example.test/owner/repo/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "b"), "shared"),
			Source: source(t, "https://github.com/owner/repo/.agents/skills@main"),
		},
	)
	_, err := set.Resolve("owner/repo/shared")
	if err == nil {
		t.Fatal("an ambiguous terse address resolved")
	}
	for _, want := range []string{"forgejo.example.test", "github.com"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %s", err, want)
		}
	}
	if _, err := set.Resolve("github.com/owner/repo/shared"); err != nil {
		t.Fatalf("naming the forge must settle it: %v", err)
	}
}

func TestDuplicatesDedupesEqualContentAndNamesIt(t *testing.T) {
	dir := t.TempDir()
	first := makeCatalog(t, filepath.Join(dir, "a"), "shared", "only-here")
	second := makeCatalog(t, filepath.Join(dir, "b"), "shared")
	set := build(t,
		catalogmanifest.Catalog{
			Path:   first,
			Source: source(t, "https://forgejo.example.test/org/one/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   second,
			Source: source(t, "https://forgejo.example.test/org/two/.agents/skills@main"),
		},
	)
	notes, err := set.Duplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %+v, want one", notes)
	}
	// Declaration order keeps the first, which is the manifest contract's own
	// rule where content raises no objection.
	if notes[0].Name != "shared" ||
		notes[0].Kept.Repo != "one" ||
		notes[0].Shadow.Repo != "two" {
		t.Fatalf("note = %+v", notes[0])
	}
}

func TestDuplicatesRefusesDivergentContentAndNamesBoth(t *testing.T) {
	dir := t.TempDir()
	first := makeCatalog(t, filepath.Join(dir, "a"), "shared")
	second := makeCatalog(t, filepath.Join(dir, "b"), "shared")
	body := filepath.Join(second, "shared", "SKILL.md")
	if err := os.WriteFile(body, []byte("a different skill wearing the same name\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	set := build(t,
		catalogmanifest.Catalog{
			Path:   first,
			Source: source(t, "https://forgejo.example.test/org/one/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   second,
			Source: source(t, "https://github.com/org/two/.agents/skills@main"),
		},
	)
	_, err := set.Duplicates()
	if err == nil {
		t.Fatal("divergent content under one name was admitted")
	}
	for _, want := range []string{"forgejo.example.test/org/one", "github.com/org/two", "shared"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %s", err, want)
		}
	}
}

// Line endings must not decide the ruling: the same skill checked out on two
// hosts would otherwise read as two different skills.
func TestDuplicatesFoldsLineEndings(t *testing.T) {
	dir := t.TempDir()
	first := makeCatalog(t, filepath.Join(dir, "a"), "shared")
	second := makeCatalog(t, filepath.Join(dir, "b"), "shared")
	if err := os.WriteFile(filepath.Join(first, "shared", "SKILL.md"),
		[]byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(second, "shared", "SKILL.md"),
		[]byte("one\r\ntwo\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	set := build(t,
		catalogmanifest.Catalog{
			Path:   first,
			Source: source(t, "https://forgejo.example.test/org/one/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   second,
			Source: source(t, "https://forgejo.example.test/org/two/.agents/skills@main"),
		},
	)
	notes, err := set.Duplicates()
	if err != nil {
		t.Fatalf("CRLF alone must not make two skills: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %+v, want one dedupe note", notes)
	}
}

func TestDuplicatesIsSilentWhenNoNameIsSharedTwice(t *testing.T) {
	dir := t.TempDir()
	set := build(t,
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "a"), "one"),
			Source: source(t, "https://forgejo.example.test/org/one/.agents/skills@main"),
		},
		catalogmanifest.Catalog{
			Path:   makeCatalog(t, filepath.Join(dir, "b"), "two"),
			Source: source(t, "https://forgejo.example.test/org/two/.agents/skills@main"),
		},
	)
	notes, err := set.Duplicates()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %+v, want none", notes)
	}
}
