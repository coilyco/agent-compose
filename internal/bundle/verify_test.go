package bundle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/compose"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func composeBundle(t *testing.T, name string) string {
	t.Helper()
	result, err := compose.Run(filepath.Join("..", "..", "testdata", "contracts", name), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return result.Bundle.Dir
}

func TestVerifyNativeAndCompiledBundles(t *testing.T) {
	cases := []struct {
		request string
		mode    string
	}{
		{"native.kdl", schema.DeliveryNativeSkills},
		{"compiled.kdl", schema.DeliveryCompiled},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			verified, err := bundle.Verify(composeBundle(t, tc.request))
			if err != nil {
				t.Fatal(err)
			}
			if verified.Manifest.Delivery.Mode != tc.mode {
				t.Fatalf("delivery = %q, want %q", verified.Manifest.Delivery.Mode, tc.mode)
			}
			if verified.Manifest.ModelTier != schema.ModelTierFrontier {
				t.Fatalf("model tier = %q", verified.Manifest.ModelTier)
			}
			// Identities cover the personalities, the melded shared doctrine,
			// the role charter, and the one ordinary fixture skill.
			wantIdentities := len(verified.Manifest.Personalities) + len(verified.Manifest.Boundaries) + 2
			if len(verified.Identities) != wantIdentities {
				t.Fatalf("identities = %+v, want %d covering personalities %v and boundaries %v",
					verified.Identities, wantIdentities,
					verified.Manifest.Personalities, verified.Manifest.Boundaries)
			}
			var foundOrdinary bool
			for _, identity := range verified.Identities {
				if identity.Skill == "fixture-review" {
					if identity.Source != "aos-public" {
						t.Fatalf("ordinary identity source = %q, want aos-public", identity.Source)
					}
					foundOrdinary = true
				} else if identity.Source != "roster:core" {
					t.Fatalf("personality identity source = %q, want roster:core", identity.Source)
				}
			}
			if !foundOrdinary {
				t.Fatalf("ordinary skill missing from verified identities: %+v", verified.Identities)
			}
			if verified.Files == 0 {
				t.Fatal("verified bundle reported no files")
			}
		})
	}
}

func TestVerifyRejectsUnsafeIncompleteAndAmbiguousBundles(t *testing.T) {
	t.Run("unknown model tier", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		manifest, err := bundle.ReadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		manifest.ModelTier = "premium"
		writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "model tier") {
			t.Fatalf("expected model-tier failure, got %v", err)
		}
	})

	t.Run("legacy model class is ignored", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		manifest := map[string]any{}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatal(err)
		}
		manifest["model_class"] = "low-context"
		writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
		if _, err := bundle.Verify(dir); err != nil {
			t.Fatalf("legacy model_class field changed verification: %v", err)
		}
	})

	t.Run("unsafe manifest path", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		manifest, err := bundle.ReadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Delivery.Instructions = "../outside"
		writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "safe relative path") {
			t.Fatalf("expected unsafe-path failure, got %v", err)
		}
	})

	t.Run("missing entry point", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "compiled.kdl"))
		if err := os.Remove(filepath.Join(dir, "delivery", "compiled.md")); err != nil {
			t.Fatal(err)
		}
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "compiled_context") {
			t.Fatalf("expected missing-entry failure, got %v", err)
		}
	})

	t.Run("extra identity", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		extra := filepath.Join(dir, "content", "skills", "aos-public", "personality-extra")
		if err := os.MkdirAll(extra, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(extra, "SKILL.md"), []byte("# Grounded\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "does not match selected identities") {
			t.Fatalf("expected identity-cardinality failure, got %v", err)
		}
	})

	t.Run("manifest trace mismatch", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		manifest, err := bundle.ReadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Personalities[0] = "other"
		writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "manifest profile") {
			t.Fatalf("expected profile-trace failure, got %v", err)
		}
	})

	t.Run("provider context budget mismatch", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		trace, err := bundle.ReadTrace(dir)
		if err != nil {
			t.Fatal(err)
		}
		trace.Providers[0].ContextBytes++
		trace.Providers[0].ApproximateTokens = (trace.Providers[0].ContextBytes + 3) / 4
		writeJSON(t, filepath.Join(dir, "trace.json"), trace)
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "budget is") {
			t.Fatalf("expected provider-budget failure, got %v", err)
		}
	})

	t.Run("missing identity document", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		skillDoc := filepath.Join(dir, "content", "skills", "roster%3Acore", "personality-tenacious", "SKILL.md")
		if err := os.Remove(skillDoc); err != nil {
			t.Fatal(err)
		}
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "has no SKILL.md") {
			t.Fatalf("expected missing-identity-document failure, got %v", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dir := copyBundle(t, composeBundle(t, "native.kdl"))
		link := filepath.Join(dir, "content", "linked")
		if err := os.Symlink("instructions.md", link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := bundle.Verify(dir); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("expected symlink failure, got %v", err)
		}
	})
}

func TestVerifyDefaultsLegacyBundleToFrontierModelTier(t *testing.T) {
	dir := copyBundle(t, composeBundle(t, "native.kdl"))
	manifest, err := bundle.ReadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ModelTier = ""
	writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
	verified, err := bundle.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Manifest.ModelTier != schema.ModelTierFrontier {
		t.Fatalf("legacy model tier = %q, want frontier", verified.Manifest.ModelTier)
	}
}

func TestMaterializeVerifiesCacheReuse(t *testing.T) {
	out := t.TempDir()
	request := filepath.Join("..", "..", "testdata", "contracts", "native.kdl")
	first, err := compose.Run(request, out)
	if err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(first.Bundle.Dir, "content", "skills", "aos-public", "personality-extra")
	if err := os.MkdirAll(extra, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := compose.Run(request, out); err == nil || !strings.Contains(err.Error(), "cached bundle") {
		t.Fatalf("expected invalid-cache failure, got %v", err)
	}
}

func TestMaterializeKeysRenderedInstructions(t *testing.T) {
	out := t.TempDir()
	request := filepath.Join("..", "..", "testdata", "contracts", "native.kdl")
	first, err := compose.Run(request, out)
	if err != nil {
		t.Fatal(err)
	}

	first.Resolution.RoleBriefing += "\n\nThe agent follows the revised role contract."
	second, err := bundle.Materialize(first.Resolution, out)
	if err != nil {
		t.Fatal(err)
	}
	if second.Key == first.Bundle.Key || second.Reused {
		t.Fatalf("rendered instruction change reused stale bundle: %+v then %+v", first.Bundle, second)
	}
}

func TestMaterializeRejectsUnsafeIdentitySegments(t *testing.T) {
	resolution := &resolver.Resolution{
		Request: &schema.Request{
			Role:     "platform-eng",
			Delivery: schema.DeliveryNativeSkills,
		},
		Person: &person.Person{Raw: []byte("fixture")},
		Skills: []resolver.Selected{{
			Source: "aos-public", ID: "../outside", Path: t.TempDir(),
		}},
	}
	if _, err := bundle.Materialize(resolution, t.TempDir()); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected unsafe identity failure, got %v", err)
	}
}

func copyBundle(t *testing.T, source string) string {
	t.Helper()
	target := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		out := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
