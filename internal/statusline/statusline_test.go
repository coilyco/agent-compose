package statusline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/agentid"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
)

func TestRenderShowsSelectedIdentityFootprintAndHealth(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	target := t.TempDir()
	bundleDir := t.TempDir()
	writeJSON(t, filepath.Join(target, ".agent-compose", "projection.json"), project.Projection{
		Layout: "codex", Bundle: bundleDir, Files: []string{"AGENTS.md"},
	})
	writeJSON(t, filepath.Join(bundleDir, "manifest.json"), bundle.Manifest{
		Format: "agent-compose.bundle", Role: "platform-eng", ModelTier: "frontier",
		Color: "#959e5f",
		Identity: bundle.RoleIdentity{
			Person: "core",
			Seats: []person.Seat{
				{Key: "codex", Name: "opal engineer"},
			},
			Personalities: []bundle.IdentityPersonality{
				{Name: "tenacious", Color: "#d98e48", Emblem: person.Emblem{Emoji: "🧭"}},
				{Name: "grounded", Color: "#5fa87a", Emblem: person.Emblem{Emoji: "🪨"}},
			},
		},
	})
	writeJSON(t, filepath.Join(bundleDir, "trace.json"), bundle.Trace{
		Format: "agent-compose.trace",
		Providers: []resolver.ProviderReport{
			{Source: "roster:core", Outcome: resolver.OutcomeSelected, Skills: 5, ApproximateTokens: 1962},
			{Source: "aos", Outcome: resolver.OutcomeSelected, Skills: 94, ApproximateTokens: 94000},
			{Source: "ops-only", Outcome: resolver.OutcomeExcluded},
		},
	})

	nested := filepath.Join(target, "owner", "repo")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Render(Options{Target: nested})
	if err != nil {
		t.Fatal(err)
	}
	want := "  🧭 🪨  opal engineer // platform-eng@codex // frontier // 99 skills / ~96k catalog // ✓ composed"
	if got != want {
		t.Fatalf("statusline = %q, want %q", got, want)
	}
}

func TestRenderSurfacesOnlyWarningClassifiedSkippedSources(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	target := t.TempDir()
	bundleDir := t.TempDir()
	writeJSON(t, filepath.Join(target, ".agent-compose", "projection.json"), project.Projection{
		Layout: "claude", Bundle: bundleDir,
	})
	writeJSON(t, filepath.Join(bundleDir, "manifest.json"), bundle.Manifest{
		Format: "agent-compose.bundle", Role: "sysadmin",
	})
	writeJSON(t, filepath.Join(bundleDir, "trace.json"), bundle.Trace{
		Format: "agent-compose.trace",
		Providers: []resolver.ProviderReport{
			{Outcome: resolver.OutcomeSelected, Skills: 4, ApproximateTokens: 900},
			{Outcome: resolver.OutcomeExcluded, Warning: false},
			{Outcome: resolver.OutcomeExcluded, Warning: true},
		},
	})

	got, err := Render(Options{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	want := "  🧩  sysadmin // sysadmin@claude // frontier // 4 skills / ~900 catalog // ⚠ 1 source skipped"
	if got != want {
		t.Fatalf("statusline = %q, want %q", got, want)
	}
}

func TestRenderSuppressesWhenNoProjectionExists(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	got, err := Render(Options{Target: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("statusline without projection = %q", got)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
