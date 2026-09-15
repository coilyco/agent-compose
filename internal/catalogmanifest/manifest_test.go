package catalogmanifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifestTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPreservesVerifiedLocalCatalogueOrder(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, path := range []string{first, second} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := filepath.Join(root, "catalogues.json")
	writeManifestTestFile(t, manifest, `{
  "format": "aos.catalogues.v1",
  "forge": "https://forgejo.example.test",
  "catalogues": [
    {"source": "one/catalogue/.agents/skills@main", "path": "`+filepath.ToSlash(first)+`", "commit": "1111111111111111111111111111111111111111"},
    {"source": "https://github.com/two/catalogue/.agents/skills@main", "path": "`+filepath.ToSlash(second)+`", "commit": "2222222222222222222222222222222222222222"}
  ]
}
`)
	catalogs, err := Load(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogs) != 2 ||
		catalogs[0].Path != first ||
		catalogs[1].Path != second {
		t.Fatalf("catalogue order = %+v", catalogs)
	}
	// The default fills the bare entry; a scheme keeps its own forge.
	if got := catalogs[0].Source.String(); got != "forgejo.example.test/one/catalogue/.agents/skills@main" {
		t.Fatalf("defaulted source = %q", got)
	}
	if got := catalogs[1].Source.String(); got != "github.com/two/catalogue/.agents/skills@main" {
		t.Fatalf("qualified source = %q", got)
	}
	if got := catalogs[1].Source.SkillAddress("coding-go"); got != "github.com/two/catalogue/coding-go" {
		t.Fatalf("skill address = %q", got)
	}
}

func TestLoadRejectsUntrustedManifestInputs(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "catalogue")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(root, "regular")
	writeManifestTestFile(t, regular, "not a directory\n")
	commit := "1111111111111111111111111111111111111111"
	// A resolvable source, so each fixture still fails for its named reason
	// rather than stopping early at source parsing.
	source := "https://forgejo.example.test/one/catalogue/.agents/skills@main"
	for name, body := range map[string]string{
		"unsupported format": `{"format":"aos.catalogues.v2","catalogues":[]}`,
		"unknown field":      `{"format":"aos.catalogues.v1","catalogues":[],"remote":"https://example.test"}`,
		"relative path": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"relative","commit":"` + commit + `"}]}`,
		"missing path": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"` + filepath.ToSlash(filepath.Join(root, "missing")) +
			`","commit":"` + commit + `"}]}`,
		"regular file": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"` + filepath.ToSlash(regular) +
			`","commit":"` + commit + `"}]}`,
		"short commit": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"` + filepath.ToSlash(directory) +
			`","commit":"1234"}]}`,
		"trailing JSON": `{"format":"aos.catalogues.v1","catalogues":[]}` +
			`{"format":"aos.catalogues.v1","catalogues":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			manifest := filepath.Join(root, strings.ReplaceAll(name, " ", "-")+".json")
			writeManifestTestFile(t, manifest, body)
			if _, err := Load(manifest); err == nil {
				t.Fatal("invalid manifest passed")
			}
		})
	}
}

// Negative control: each fixture must fail on its named defect, since a table
// that stops early at parsing passes while testing nothing.
func TestUntrustedManifestInputsFailForTheirOwnReason(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "catalogue")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	source := "https://forgejo.example.test/one/catalogue/.agents/skills@main"
	for name, body := range map[string]string{
		"relative path": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"relative",` +
			`"commit":"1111111111111111111111111111111111111111"}]}`,
		"short commit": `{"format":"aos.catalogues.v1","catalogues":[` +
			`{"source":"` + source + `","path":"` + filepath.ToSlash(directory) +
			`","commit":"1234"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			manifest := filepath.Join(root, strings.ReplaceAll(name, " ", "-")+".json")
			writeManifestTestFile(t, manifest, body)
			_, err := Load(manifest)
			if err == nil {
				t.Fatal("invalid manifest passed")
			}
			if strings.Contains(err.Error(), "no forge") ||
				strings.Contains(err.Error(), "must name owner") {
				t.Fatalf("failed at source parsing rather than %s: %v", name, err)
			}
		})
	}
}
