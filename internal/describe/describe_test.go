package describe

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
)

func composeFixture(t *testing.T, name string) string {
	t.Helper()
	result, err := compose.Run(filepath.Join("..", "..", "testdata", "contracts", name), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return result.Bundle.Dir
}

func TestBundleRendersSections(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	personalitySet := strings.Join(p.Roles["platform-eng"].Personalities, "+")
	dir := composeFixture(t, "native.kdl")
	out, err := Bundle(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{
		"platform-eng/" + personalitySet,
		"melded #",
		"\nprofile\n", "\nproviders\n", "\ncontext budget\n", "\nselection\n", "\ndelivery\n",
		"role platform-eng", "roster:core defines this role",
		"personality tenacious", "activates its full personality set",
		"✓ roster:core", "(person-package/person)",
		"✓ aos-public", "(catalogue/request)",
		"skills //", "bytes // ~", "tokens",
		"✓ skill fixture-review",
		"→ content/skills",
		"machine-readable trace: trace.json",
	}
	for _, personalityName := range p.Roles["platform-eng"].Personalities {
		wants = append(wants, "✓ skill "+p.Personalities[personalityName].Skill)
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("describe output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b") {
		t.Fatal("plain output must carry no ANSI codes")
	}
}

func TestCollapseFoldsLargeExclusionGroups(t *testing.T) {
	var excluded []resolver.Decision
	for range 5 {
		excluded = append(excluded, resolver.Decision{
			Subject: "skill:x", Kind: "skill",
			Outcome: resolver.OutcomeExcluded, Reason: "not bound",
		})
	}
	lines := collapse(excluded, Options{})
	if len(lines) != 1 || !strings.Contains(lines[0], "5 skills excluded") || !strings.Contains(lines[0], "--all") {
		t.Fatalf("expected one collapsed line, got %v", lines)
	}
	if got := collapse(excluded[:2], Options{}); len(got) != 2 {
		t.Fatalf("small groups must stay expanded, got %v", got)
	}
}

func TestWhyFollowsOneItem(t *testing.T) {
	dir := composeFixture(t, "native.kdl")

	out, err := Why(dir, "skill:fixture-review", Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"outcome: selected",
		"considered: declared by aos-public",
		"provider: catalogue/request",
		"ordinary provider skills are discoverable for every role",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("why output missing %q:\n%s", want, out)
		}
	}

	out, err = Why(dir, "personality-tenacious", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "outcome: selected") {
		t.Fatalf("bare-name lookup failed:\n%s", out)
	}

	if _, err := Why(dir, "skill:nonexistent", Options{}); err == nil {
		t.Fatal("expected unknown subject to error with guidance")
	}
}

func TestDiffReportsSemanticChanges(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	identity := "platform-eng/" + strings.Join(p.Roles["platform-eng"].Personalities, "+")
	native := composeFixture(t, "native.kdl")
	compiled := composeFixture(t, "compiled.kdl")

	out, err := Diff(native, compiled)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		identity,
		"native-skills",
		"compiled",
		"+ delivery/compiled.md",
		"decisions unchanged",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("diff output missing %q:\n%s", want, out)
		}
	}

	same, err := Diff(native, native)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(same, "no semantic differences") {
		t.Fatalf("identical bundles must diff clean:\n%s", same)
	}
}

func TestDiffReportsLogicalProseAndIdentityChanges(t *testing.T) {
	left := composeFixture(t, "native.kdl")
	right := composeFixture(t, "native.kdl")
	manifest, err := bundle.ReadManifest(right)
	if err != nil {
		t.Fatal(err)
	}
	changed := map[string]bool{
		"roster:core:role:platform-eng":          false,
		"roster:core:role:platform-eng:identity": false,
	}
	for index := range manifest.Content {
		if _, ok := changed[manifest.Content[index].ID]; ok {
			manifest.Content[index].Digest = "sha256:" + strings.Repeat(
				map[bool]string{true: "a", false: "b"}[manifest.Content[index].ID == "roster:core:role:platform-eng"],
				64,
			)
			changed[manifest.Content[index].ID] = true
		}
	}
	for id, found := range changed {
		if !found {
			t.Fatalf("manifest omitted logical content %q: %+v", id, manifest.Content)
		}
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(right, "manifest.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Diff(left, right)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"roster:core:role:platform-eng ",
		"roster:core:role:platform-eng:identity ",
		"sha256:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logical content diff omitted %q:\n%s", want, out)
		}
	}
}
