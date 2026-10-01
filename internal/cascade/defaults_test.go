package cascade

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// defaultsEnv is a state dir plus a projects root holding two operating-context
// repositories, one of which has no AGENTS.md.
func defaultsEnv(t *testing.T) (state, projects string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	state = filepath.Join(home, ".agent-compose")
	projects = filepath.Join(home, "projects")
	for path, body := range map[string]string{
		filepath.Join(projects, "org-a", "alpha", "AGENTS.md"): "# Alpha\n",
		filepath.Join(projects, "org-b", "beta", ".keep"):      "",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(state, "sources"), 0o755); err != nil {
		t.Fatal(err)
	}
	return state, projects
}

func loadDefaults(t *testing.T, state, projects, body string) *Config {
	t.Helper()
	path := filepath.Join(state, "agent-compose.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path, projects)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

const twoRepositories = "operating_context:\n  - org-a/alpha\n  - org-b/beta\n"

func TestShortConfigDerivesSourcesRootsAndDelivery(t *testing.T) {
	state, projects := defaultsEnv(t)
	cfg := loadDefaults(t, state, projects, twoRepositories)

	// beta has no AGENTS.md, so it has no doctrine to compose and is skipped.
	want := []string{filepath.Join(projects, "org-a", "alpha", "AGENTS.md")}
	if !slices.Equal(cfg.Sources, want) {
		t.Fatalf("derived sources = %v, want %v", cfg.Sources, want)
	}
	if cfg.SourceDelivery != DeliveryImport {
		t.Fatalf("derived sources deliver as %q, want %q", cfg.SourceDelivery, DeliveryImport)
	}
	if want := []string{filepath.Join(state, "sources")}; !slices.Equal(cfg.Roots, want) {
		t.Fatalf("default roots = %v, want %v", cfg.Roots, want)
	}
	if err := ValidateSources(cfg); err != nil {
		t.Fatalf("derived import sources must validate: %v", err)
	}
}

func TestExplicitKeysBeatTheirDefaults(t *testing.T) {
	state, projects := defaultsEnv(t)
	named := filepath.Join(projects, "org-a", "alpha", "AGENTS.md")
	cfg := loadDefaults(t, state, projects,
		"sources:\n  - "+named+"\nroots: []\n"+twoRepositories)

	if cfg.SourceDelivery != "" {
		t.Fatalf("named sources keep the inline default, got delivery %q", cfg.SourceDelivery)
	}
	if len(cfg.Roots) != 0 {
		t.Fatalf("`roots: []` must opt out of the default root, got %v", cfg.Roots)
	}

	cfg = loadDefaults(t, state, projects, "source_delivery: inline\n"+twoRepositories)
	if cfg.SourceDelivery != DeliveryInline || len(cfg.Sources) != 1 {
		t.Fatalf("explicit inline delivery must survive derivation: %q %v", cfg.SourceDelivery, cfg.Sources)
	}
}

func TestNoOperatingContextDerivesNothing(t *testing.T) {
	state, projects := defaultsEnv(t)
	cfg := loadDefaults(t, state, projects, "load_points:\n  claude: "+filepath.Join(state, "c.md")+"\n")
	if len(cfg.Sources) != 0 || cfg.SourceDelivery != "" {
		t.Fatalf("a config with no operating_context derived sources %v delivery %q", cfg.Sources, cfg.SourceDelivery)
	}
}

func TestMissingDefaultInputsAreSkippedNotFatal(t *testing.T) {
	state, projects := defaultsEnv(t)
	if err := os.RemoveAll(filepath.Join(state, "sources")); err != nil {
		t.Fatal(err)
	}
	cfg := loadDefaults(t, state, projects, "operating_context:\n  - org-b/beta\n")
	if len(cfg.Roots) != 0 || len(cfg.Sources) != 0 || cfg.SkillCatalogManifest != "" {
		t.Fatalf("absent inputs must leave defaults empty: %+v", cfg)
	}
}

func TestSkillCatalogManifestDefaultsToTheHostManifestWhenPresent(t *testing.T) {
	state, projects := defaultsEnv(t)
	home := filepath.Dir(state)
	manifest := filepath.Join(home, ".config", "aos", "catalogues.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg := loadDefaults(t, state, projects, twoRepositories); cfg.SkillCatalogManifest != manifest {
		t.Fatalf("default manifest = %q, want %q", cfg.SkillCatalogManifest, manifest)
	}
	elsewhere := filepath.Join(home, "elsewhere.json")
	if cfg := loadDefaults(t, state, projects, "skill_catalog_manifest: "+elsewhere+"\n"); cfg.SkillCatalogManifest != elsewhere {
		t.Fatalf("explicit manifest overridden: %q", cfg.SkillCatalogManifest)
	}
}

func TestBareTrueOptsAHarnessInAtItsTableDefault(t *testing.T) {
	state, projects := defaultsEnv(t)
	home := filepath.Dir(state)
	cfg := loadDefaults(t, state, projects, "load_points:\n  opencode: true\nskill_load_points:\n  opencode: true\n")

	points, err := ResolveLoadPoints(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "opencode", "AGENTS.md"); points["opencode"] != want {
		t.Fatalf("opencode load point = %q, want %q", points["opencode"], want)
	}
	if _, cascading := points["claude"]; !cascading {
		t.Fatalf("an opt-in must not displace the cascade defaults: %v", points)
	}
	skills, err := ResolveSkillLoadPoints(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".agents", "skills"); skills["opencode"] != want {
		t.Fatalf("opencode skill load point = %q, want %q", skills["opencode"], want)
	}
}

func TestBareTrueRefusesAnUnknownHarness(t *testing.T) {
	state, projects := defaultsEnv(t)
	cfg := loadDefaults(t, state, projects, "load_points:\n  nosuch: true\n")
	if _, err := ResolveLoadPoints(cfg); err == nil {
		t.Fatal("`nosuch: true` named no harness and must fail rather than link a path called true")
	}
}
