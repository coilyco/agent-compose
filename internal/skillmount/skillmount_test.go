package skillmount

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/repositoryplan"
)

// notExistCause is this host own spelling of a missing-file errno, probed
// rather than pinned because Windows and unix word it differently.
func notExistCause(t *testing.T, dir string) string {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, "definitely-absent"))
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("probe for a missing path returned %v, want a *fs.PathError", err)
	}
	return pathErr.Err.Error()
}

func makeSkill(t *testing.T, repo, name string) string {
	t.Helper()
	path := filepath.Join(repo, ".agents", "skills", name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeEligibility(t *testing.T, path string, defaults []string, harnesses map[string][]string) {
	t.Helper()
	all := append([]string(nil), defaults...)
	for _, repositories := range harnesses {
		all = append(all, repositories...)
	}
	selections := make([]repositoryplan.Selection, 0, len(all))
	for index, repository := range all {
		selections = append(selections, repositoryplan.Selection{
			Identity: fmt.Sprintf("test/repository-%03d", index), Path: repository,
			Source: "test/policy", Scope: "role-union", Reason: "test repository",
		})
	}
	raw := marshalPlan(t, repositoryplan.Plan{
		Format: repositoryplan.Format, ProjectsRoot: filepath.Dir(path),
		Inputs: testInputs("test/policy"),
		Roles:  map[string][]repositoryplan.Selection{"sysadmin": selections}, Residency: selections,
	})
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryPlanRoleProviderOrderingAndStrictLoading(t *testing.T) {
	dir := t.TempDir()
	defaultProvider := filepath.Join(dir, "default")
	harnessProvider := filepath.Join(dir, "harness")
	roleOne := filepath.Join(dir, "role-one")
	roleTwo := filepath.Join(dir, "role-two")
	manifestPath := filepath.Join(dir, "repository-plan.yaml")
	raw := marshalPlan(t, repositoryplan.Plan{
		Format: repositoryplan.Format, ProjectsRoot: dir,
		Inputs: testInputs("example/aosk"),
		Roles: map[string][]repositoryplan.Selection{
			"sysadmin": {
				{Identity: "example/default", Path: defaultProvider, Source: "example/aosk", Scope: "operating-context", Reason: "test"},
				{Identity: "example/harness", Path: harnessProvider, Source: "example/aosk", Scope: "global", Reason: "test"},
				{Identity: "example/role-one", Path: roleOne, Source: "example/aosk", Scope: "provider", Reason: "test", Required: true, Skills: []string{"compute-stack", "machine-*"}, Name: "hardware", DeclaredBy: "example/aosk"},
				{Identity: "example/role-two", Path: roleTwo, Source: "example/aosk", Scope: "provider", Reason: "test", Name: "software", DeclaredBy: "example/aosk"},
			},
		},
		Residency: []repositoryplan.Selection{
			{Identity: "example/default", Path: defaultProvider, Source: "example/aosk", Scope: "role-union", Reason: "test"},
			{Identity: "example/harness", Path: harnessProvider, Source: "example/aosk", Scope: "role-union", Reason: "test"},
		},
	})
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := repositoryplan.Load(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	providers := manifest.Roles["sysadmin"]
	want := []string{defaultProvider, harnessProvider, roleOne, roleTwo}
	if len(providers) != len(want) {
		t.Fatalf("providers = %+v, want %v", providers, want)
	}
	for index, path := range want {
		if providers[index].Path != path {
			t.Fatalf("provider order = %+v, want %v", providers, want)
		}
	}
	if got := providers[2].Skills; len(got) != 2 || got[0] != "compute-stack" || got[1] != "machine-*" {
		t.Fatalf("role provider selector = %v", got)
	}
	if providers[2].Name != "hardware" || providers[2].DeclaredBy != "example/aosk" {
		t.Fatalf("role provider provenance = %+v", providers[2])
	}
	if got := manifest.Residency; len(got) != 2 || got[0].Path != defaultProvider || got[1].Path != harnessProvider {
		t.Fatalf("residency leaked role providers: %v", got)
	}
	makeSkill(t, defaultProvider, "global")
	makeSkill(t, roleOne, "role-only")
	destination := filepath.Join(dir, "native-skills")
	if _, err := Apply(
		manifestPath,
		map[string]string{"codex": destination},
		filepath.Join(dir, "state"),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "global")); err != nil {
		t.Fatalf("bare convergence omitted global skill: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(destination, "role-only")); !os.IsNotExist(err) {
		t.Fatalf("bare convergence leaked role-only skill: %v", err)
	}

	for name, body := range map[string]string{
		"top-level":  "format: agent-compose.repositories.v2\nprojects_root: /tmp\ninputs: []\nroles: {}\nresidency: []\nextra: true\n",
		"nested":     "format: agent-compose.repositories.v2\nprojects_root: /tmp\ninputs:\n  - identity: example/aosk\n    revision: 0123456789012345678901234567890123456789\n    policy:\n      path: .agents/roles.kdl\n      sha256: sha256:0123456789012345678901234567890123456789012345678901234567890123\nroles:\n  ops:\n    - identity: example/provider\n      path: /tmp/provider\n      source: example/aosk\n      scope: provider\n      reason: test\n      name: hardware\n      declared_by: example/aosk\n      extra: true\nresidency: []\n",
		"trailing":   "format: agent-compose.repositories.v2\nprojects_root: /tmp\ninputs: []\nroles: {}\nresidency: []\n---\nextra: true\n",
		"provenance": "format: agent-compose.repositories.v2\nprojects_root: /tmp\ninputs:\n  - identity: example/aosk\n    revision: 0123456789012345678901234567890123456789\n    policy:\n      path: .agents/roles.kdl\n      sha256: sha256:0123456789012345678901234567890123456789012345678901234567890123\nroles:\n  ops:\n    - identity: example/provider\n      path: /tmp/provider\n      source: example/aosk\n      scope: provider\n      reason: test\n      name: hardware\nresidency: []\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.yaml")
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := repositoryplan.Load(path); err == nil {
				t.Fatal("invalid repository plan passed strict loading")
			}
		})
	}
}

func marshalPlan(t *testing.T, plan repositoryplan.Plan) []byte {
	t.Helper()
	raw, err := repositoryplan.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testInputs(identities ...string) []repositoryplan.Input {
	inputs := make([]repositoryplan.Input, 0, len(identities))
	for _, identity := range identities {
		inputs = append(inputs, repositoryplan.Input{
			Identity: identity,
			Revision: "0123456789012345678901234567890123456789",
			Policy: repositoryplan.PolicyInput{
				Path:   repositoryplan.PolicyPath,
				SHA256: "sha256:0123456789012345678901234567890123456789012345678901234567890123",
			},
		})
	}
	return inputs
}

func TestApplyOverlaysAndMultipleLoadPoints(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	private := filepath.Join(dir, "private")
	makeSkill(t, public, "coding-go")
	makeSkill(t, public, "shared")
	privateShared := makeSkill(t, private, "shared")
	makeSkill(t, private, "kai-local")
	loadPoints := map[string]string{
		"claude": filepath.Join(dir, "claude"),
		"codex":  filepath.Join(dir, "codex"),
	}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{public, private}, map[string][]string{})

	result, err := Apply(manifest, loadPoints, filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Linked != 6 {
		t.Fatalf("linked %d, want 6", result.Linked)
	}
	for _, destination := range loadPoints {
		target, err := os.Readlink(filepath.Join(destination, "shared"))
		if err != nil {
			t.Fatal(err)
		}
		if target != privateShared {
			t.Fatalf("later root must win: %s", target)
		}
	}
}

func TestApplyRemovesStaleOwnedLinksAndPreservesForeignEntries(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	stale := makeSkill(t, root, "stale")
	makeSkill(t, root, "kept")
	destination := filepath.Join(dir, "skills")
	state := filepath.Join(dir, "state")
	points := map[string]string{"codex": destination}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{root}, map[string][]string{})
	if _, err := Apply(manifest, points, state); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(stale); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destination, "foreign")
	if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(manifest, points, state)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Fatalf("removed %d, want 1", result.Removed)
	}
	if _, err := os.Lstat(filepath.Join(destination, "stale")); !os.IsNotExist(err) {
		t.Fatal("stale owned link must be removed")
	}
	if raw, err := os.ReadFile(foreign); err != nil || string(raw) != "mine" {
		t.Fatal("foreign entry must survive")
	}
}

func TestApplyWarnsAndRemovesOwnedLinkWhenSkillTargetDisappears(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	skillRoot := filepath.Join(root, ".agents", "skills")
	makeSkill(t, root, "kept")
	target := filepath.Join(dir, "fragile-target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	fragile := filepath.Join(skillRoot, "fragile")
	if err := os.Symlink(target, fragile); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "skills")
	state := filepath.Join(dir, "state")
	points := map[string]string{"codex": destination}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{root}, map[string][]string{})

	if _, err := Apply(manifest, points, state); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(manifest, points, state)
	if err != nil {
		t.Fatalf("vanished skill target must not fail convergence: %v", err)
	}
	if result.Removed != 1 || result.Verified != 1 {
		t.Fatalf("result = %+v, want one stale removal and one verified skill", result)
	}
	if len(result.Warnings) != 1 ||
		!strings.Contains(result.Warnings[0], fragile) ||
		!strings.Contains(result.Warnings[0], notExistCause(t, dir)) {
		t.Fatalf("warnings = %q, want vanished skill path and cause", result.Warnings)
	}
	if _, err := os.Lstat(filepath.Join(destination, "fragile")); !os.IsNotExist(err) {
		t.Fatal("owned projection for vanished skill must be removed")
	}
	if _, err := os.Stat(filepath.Join(destination, "kept")); err != nil {
		t.Fatalf("available skill must remain projected: %v", err)
	}
}

func TestApplyStillFailsOnNonMissingSkillInspectionError(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	skillRoot := filepath.Join(root, ".agents", "skills")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(skillRoot, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{root}, map[string][]string{})

	_, err := Apply(
		manifest,
		map[string]string{"codex": filepath.Join(dir, "skills")},
		filepath.Join(dir, "state"),
	)
	if err == nil || !strings.Contains(err.Error(), "inspect skill") {
		t.Fatalf("symlink loop must remain fatal, got %v", err)
	}
}

func TestApplyDoesNotMutateWhenManifestIsMissing(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "skills")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destination, "foreign")
	if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Apply(filepath.Join(dir, "missing.yaml"), map[string]string{"codex": destination}, filepath.Join(dir, "state"))
	if err == nil {
		t.Fatal("missing root must fail")
	}
	if raw, readErr := os.ReadFile(foreign); readErr != nil || string(raw) != "mine" {
		t.Fatal("validation failure must not mutate the destination")
	}
}

func TestApplyWithCatalogsOverlaysLocalSkillsForEveryLoadPoint(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "local")
	additional := filepath.Join(dir, "additional-skills")
	makeSkill(t, local, "local-only")
	makeSkill(t, local, "shared")
	additionalShared := filepath.Join(additional, "shared")
	if err := os.MkdirAll(additionalShared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(additionalShared, "SKILL.md"), []byte("additional"), 0o644); err != nil {
		t.Fatal(err)
	}
	additionalOnly := filepath.Join(additional, "additional-only")
	if err := os.MkdirAll(additionalOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(additionalOnly, "SKILL.md"), []byte("additional"), 0o644); err != nil {
		t.Fatal(err)
	}

	loadPoints := map[string]string{
		"claude": filepath.Join(dir, "claude"),
		"codex":  filepath.Join(dir, "codex"),
	}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{local}, map[string][]string{})
	result, err := ApplyWithCatalogs(
		manifest,
		loadPoints,
		filepath.Join(dir, "state"),
		[]Catalog{{Path: additional}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Linked != 6 {
		t.Fatalf("linked %d, want 6", result.Linked)
	}
	for harness, destination := range loadPoints {
		target, err := os.Readlink(filepath.Join(destination, "shared"))
		if err != nil || target != additionalShared {
			t.Fatalf("additional catalog must overlay local for %s: target=%q err=%v", harness, target, err)
		}
		if _, err := os.Readlink(filepath.Join(destination, "additional-only")); err != nil {
			t.Fatalf("additional-only skill missing from %s: %v", harness, err)
		}
	}
}

// A vanished catalogue skill is reported by source: the cache path names
// neither the forge nor the repository to go fix.
func TestApplyWarnsWithTheCatalogueSourceRatherThanItsCachePath(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	makeSkill(t, root, "kept")

	catalogRoot := filepath.Join(dir, "cache", "0a1b2c3d")
	if err := os.MkdirAll(catalogRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "fragile-target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(catalogRoot, "sirens-game-enshrouded")); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(dir, "skills")
	state := filepath.Join(dir, "state")
	points := map[string]string{"codex": destination}
	manifest := filepath.Join(dir, "repository-plan.yaml")
	writeEligibility(t, manifest, []string{root}, map[string][]string{})
	source := "forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main"
	catalogs := []Catalog{{Path: catalogRoot, Source: source}}

	if _, err := ApplyWithCatalogs(manifest, points, state, catalogs); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}

	result, err := ApplyWithCatalogs(manifest, points, state, catalogs)
	if err != nil {
		t.Fatalf("vanished catalogue skill must not fail convergence: %v", err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %q, want one", result.Warnings)
	}
	want := "forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main/sirens-game-enshrouded"
	if !strings.Contains(result.Warnings[0], want) {
		t.Fatalf("warning = %q, want it to name %q", result.Warnings[0], want)
	}
	// The path stays in the wrapped cause, as Result documents. What changes
	// is that the skill is identified by source first.
	if !strings.Contains(result.Warnings[0], catalogRoot) {
		t.Fatalf("warning = %q, want the inspected path retained in the cause",
			result.Warnings[0])
	}
	if !strings.HasPrefix(result.Warnings[0], "inspect skill "+want) {
		t.Fatalf("warning = %q, want it to lead with the qualified source",
			result.Warnings[0])
	}
}

// A residency repository has no source, so its warning keeps naming the path.
func TestApplyWarnsWithThePathWhenARootHasNoSource(t *testing.T) {
	if got := describeSkill("", "fragile", "/tmp/root/.agents/skills/fragile"); got !=
		"inspect skill /tmp/root/.agents/skills/fragile" {
		t.Fatalf("pathless description = %q", got)
	}
}
