package person

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// coreBindingProfile copies the example profile and rebinds one role to a core
// personality it does not define, which is the shape issue #311 describes.
func coreBindingProfile(t *testing.T) string {
	t.Helper()
	profile := filepath.Join(t.TempDir(), "profile")
	copyTestTree(t, filepath.Join("..", "..", "examples", "person-profile"), profile)
	role := filepath.Join(profile, "roles", "02-caption-review.yaml")
	raw, err := os.ReadFile(role)
	if err != nil {
		t.Fatal(err)
	}
	rebound := strings.Replace(string(raw), "personalities: [local-guide]", "personalities: [grounded]", 1)
	if rebound == string(raw) {
		t.Fatal("example profile no longer binds local-guide, so the fixture needs updating")
	}
	if err := os.WriteFile(role, []byte(rebound), 0o644); err != nil {
		t.Fatal(err)
	}
	return profile
}

func TestExternalProfileBindsCorePersonalitiesWithoutVendoring(t *testing.T) {
	profile := coreBindingProfile(t)
	library := filepath.Join("..", "..", "examples", "shared-personality-library")

	if _, err := LoadDirectoryWithLibraries(profile, library); err == nil ||
		!strings.Contains(err.Error(), `personality "grounded" has no catalog binding`) {
		t.Fatalf("unreferenced core personality must fail, got %v", err)
	}

	p, err := LoadDirectoryWithLibraries(profile, library, CoreLibraryRoot)
	if err != nil {
		t.Fatalf("core personality reference rejected: %v", err)
	}
	binding, bound := p.Personalities["grounded"]
	if !bound || binding.Skill != "personality-grounded" {
		t.Fatalf("grounded did not merge: %+v", p.Personalities)
	}
	if p.PersonalityLibraries["grounded"] != CoreLibraryRoot {
		t.Fatalf("provenance = %q, want %q", p.PersonalityLibraries["grounded"], CoreLibraryRoot)
	}

	// The body is the whole point: it must arrive from the binary, byte for byte.
	got, err := fs.ReadFile(p.source, "definitions/skills/personality-grounded/SKILL.md")
	if err != nil {
		t.Fatalf("core personality body is absent: %v", err)
	}
	want, err := fs.ReadFile(seedForTest(t), "data/personality-grounded/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("materialized core body diverged from the embedded copy")
	}
}

// Only the shared-disposition axis crosses. Roles, seats, identity, and the
// invariant stay package-exclusive.
func TestCoreLibraryCrossesPersonalitiesOnly(t *testing.T) {
	p, err := LoadDirectoryWithLibraries(
		coreBindingProfile(t),
		filepath.Join("..", "..", "examples", "shared-personality-library"),
		CoreLibraryRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "example" || p.ProviderID() != "person:example" {
		t.Fatalf("core admission changed package identity: %q %q", p.Name, p.ProviderID())
	}
	for _, role := range []string{"platform-eng", "scientist", "sysadmin", "prod-director"} {
		if _, inherited := p.Roles[role]; inherited {
			t.Fatalf("core admission leaked role %q", role)
		}
	}
	if len(p.Boundaries) != 0 {
		t.Fatalf("core admission leaked boundaries: %+v", p.Boundaries)
	}
	invariant, err := fs.ReadFile(p.source, "definitions/INVARIANT.md")
	if err != nil {
		t.Fatal(err)
	}
	core, err := fs.ReadFile(seedForTest(t), "data/invariant/INVARIANT.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(invariant) == string(core) {
		t.Fatal("core admission replaced the profile invariant")
	}
}

func TestCoreLibraryAdmissionDiagnostics(t *testing.T) {
	profile := coreBindingProfile(t)
	if _, err := LoadDirectoryWithLibraries(profile, CoreLibraryRoot, CoreLibraryRoot); err == nil ||
		!strings.Contains(err.Error(), "admitted more than once") {
		t.Fatalf("duplicate core admission diagnostic = %v", err)
	}
	if IsCoreLibrary("./roster:core") || IsCoreLibrary("roster") || IsCoreLibrary("") {
		t.Fatal("a path that merely resembles the reserved root must stay a path")
	}
	if !IsCoreLibrary("  " + CoreLibraryRoot + "  ") {
		t.Fatal("the reserved root must survive surrounding whitespace")
	}
}

func TestCoreLibrarySourceCarriesEveryEmbeddedPersonality(t *testing.T) {
	source, err := coreLibrarySource()
	if err != nil {
		t.Fatal(err)
	}
	library, id, err := loadLibrarySource(source, coreLibraryLabel)
	if err != nil {
		t.Fatal(err)
	}
	if id != CoreLibraryRoot {
		t.Fatalf("library id = %q, want %q", id, CoreLibraryRoot)
	}
	embeddedPerson, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(library.Personalities) != len(embeddedPerson.Personalities) {
		t.Fatalf("library carries %d personalities, roster has %d",
			len(library.Personalities), len(embeddedPerson.Personalities))
	}
	for name, binding := range embeddedPerson.Personalities {
		if !equalPersonality(library.Personalities[name], binding) {
			t.Fatalf("personality %q diverged from the roster copy", name)
		}
	}
	if len(library.Roles) != 0 || len(library.Boundaries) != 0 {
		t.Fatalf("core library is not personality-only: roles=%d boundaries=%d",
			len(library.Roles), len(library.Boundaries))
	}
}
