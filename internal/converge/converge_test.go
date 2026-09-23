package converge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/cascade"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

func run(t *testing.T, paths cascade.Paths) (int, string, string) {
	t.Helper()
	// Unset load points fall back to home-resolved defaults, so anchor HOME and
	// USERPROFILE beside the config rather than the developer real home.
	isolated := filepath.Join(filepath.Dir(paths.Config), "home")
	if err := os.MkdirAll(isolated, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", isolated)
	t.Setenv("USERPROFILE", isolated)
	if raw, err := os.ReadFile(paths.Config); err == nil && !strings.Contains(string(raw), "operating_context:") {
		root := filepath.Join(paths.ProjectsRoot, "test", "context")
		if err := os.MkdirAll(filepath.Join(root, ".agents", "skills", "fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "skills", "fixture", "SKILL.md"), []byte("# Fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		roles := "roles { role ai {}; role creator {}; role design {}; role prod-director {}; role engineer {}; role ops {}; role qa {}; role exec {} }\n"
		if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(roles), 0o644); err != nil {
			t.Fatal(err)
		}
		ensureGitRepository(t, root)
		raw = append(raw, []byte("operating_context:\n  - test/context\n")...)
		if err := os.WriteFile(paths.Config, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	code := Run(paths, Options{Verbose: true}, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestConvergeComposesRosterIntoCascade(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "doctrine", "AGENTS.COMPOSE.md")
	if err := os.MkdirAll(filepath.Dir(doctrine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	providerRoot := filepath.Join(paths.ProjectsRoot, "coilyco-flight-deck", "agentic-os")
	skillRoot := filepath.Join(providerRoot, ".agents", "skills")
	if err := os.MkdirAll(filepath.Join(skillRoot, "coding-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "coding-go", "SKILL.md"), []byte("# Go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingSkill := filepath.Join(skillRoot, "missing-skill")
	if err := os.Symlink(filepath.Join(dir, "vanished-skill"), missingSkill); err != nil {
		t.Fatal(err)
	}
	roles := "roles { role ai {}; role creator {}; role design {}; role prod-director {}; role engineer {}; role ops {}; role qa {}; role exec {} }\n"
	if err := os.WriteFile(filepath.Join(providerRoot, ".agents", "roles.kdl"), []byte(roles), 0o644); err != nil {
		t.Fatal(err)
	}
	ensureGitRepository(t, providerRoot)
	config := "sources:\n  - " + doctrine + "\nroots:\n  - " + filepath.Join(dir, "sources") + "\n" +
		"operating_context:\n  - coilyco-flight-deck/agentic-os\n" +
		"roster_sources:\n  - " + providerRoot + "\n" +
		"skill_load_points:\n  codex: " + filepath.Join(dir, "links", "skills") + "\n" +
		"load_points:\n  claude: " + filepath.Join(dir, "links", "CLAUDE.md") + "\n  codex: null\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := run(t, paths)
	if code != 0 {
		t.Fatalf("converge failed: %s %s", out, errOut)
	}
	if !strings.Contains(errOut, "warning: inspect skill ") ||
		!strings.Contains(errOut, filepath.Base(missingSkill)) ||
		!strings.Contains(errOut, "(skipped)") {
		t.Fatalf("vanished skill warning missing: %s", errOut)
	}
	if !strings.Contains(out, "roster  ") || !strings.Contains(out, "wrote") {
		t.Fatalf("converge must refresh roster then cascade: %s", out)
	}
	composed := readFile(t, paths.Composed)
	if !strings.Contains(composed, "# Doctrine") ||
		!strings.Contains(composed, "# Personality invariant") ||
		!strings.Contains(composed, "# Agent seats") {
		t.Fatalf("composed output must carry doctrine and the dispatch table:\n%s", composed)
	}
	if !strings.Contains(composed, "Agent //") {
		t.Fatalf("dispatch table incomplete:\n%s", composed)
	}
	identityRoot := filepath.Join(dir, "sources", "personality", ".agents", "skills")
	if _, err := os.Stat(filepath.Join(identityRoot, "personality-tenacious", "SKILL.md")); err != nil {
		t.Fatal("personality skills must land in the roster identity catalog")
	}
	if _, err := os.Stat(filepath.Join(identityRoot, "role-platform-eng", "SKILL.md")); err != nil {
		t.Fatal("role skills must land in the roster identity catalog")
	}
	personSnapshot := readFile(t, filepath.Join(dir, "sources", "personality", "person.json"))
	if !strings.Contains(personSnapshot, fmt.Sprintf(`"format": %q`, person.SnapshotFormat)) ||
		!strings.Contains(personSnapshot, `"briefing":`) {
		t.Fatal("normal convergence must emit the complete versioned person snapshot")
	}
	expectedSkillTarget, err := filepath.EvalSymlinks(filepath.Join(skillRoot, "coding-go"))
	if err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(dir, "links", "skills", "coding-go")); err != nil || target != expectedSkillTarget {
		t.Fatalf("skill root must mount before cascade: target=%q err=%v", target, err)
	}
	expectedRoleTarget := filepath.Join(identityRoot, "role-platform-eng")
	if target, err := os.Readlink(filepath.Join(dir, "links", "skills", "role-platform-eng")); err != nil || target != expectedRoleTarget {
		t.Fatalf("role skill must mount through the native skill catalog: target=%q err=%v", target, err)
	}
	code, out, _ = run(t, paths)
	if code != 0 || strings.Contains(out, "wrote") {
		t.Fatalf("second converge must not rewrite current cascade state: %s", out)
	}
	if !strings.Contains(out, "cascade outputs=1 load-points=1 repository-plan=1 changed=0") {
		t.Fatalf("second converge must summarize the full cascade check: %s", out)
	}
	skillEntries, err := os.ReadDir(skillRoot)
	if err != nil {
		t.Fatal(err)
	}
	identityEntries, err := os.ReadDir(identityRoot)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture configures codex only, so claude arrives from the default
	// skill load point and every managed skill is verified at both.
	managed := 2 * (len(skillEntries) - 1 + len(identityEntries))
	wantSkillSummary := fmt.Sprintf(
		"skills  managed=%d load-points=2 verified=%d linked=0 removed=0 preserved=0",
		managed,
		managed,
	)
	if !strings.Contains(out, wantSkillSummary) {
		t.Fatalf("second converge must summarize the full skill check: %s", out)
	}
}

func TestConvergeWithoutConfigIsNoOp(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: dir,
	}
	code, out, _ := run(t, paths)
	if code != 0 || !strings.Contains(out, "nothing to do (opt-in)") {
		t.Fatalf("absent config must stay the documented no-op: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "sources")); !os.IsNotExist(err) {
		t.Fatal("no-op must write nothing")
	}
}

func TestConvergeProjectsAOSLocalCatalogueManifest(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "doctrine", "AGENTS.COMPOSE.md")
	if err := os.MkdirAll(filepath.Dir(doctrine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "catalogues", "first")
	second := filepath.Join(dir, "catalogues", "second")
	// Equal content, because divergent content under one name is now fatal.
	// The refusal has its own test below.
	writeTestSkill(t, first, "shared", "shared")
	writeTestSkill(t, second, "shared", "shared")
	writeTestSkill(t, second, "aos-only", "aos")
	manifest := filepath.Join(dir, "catalogues.json")
	body := `{
  "format": "aos.catalogues.v1",
  "forge": "https://forgejo.example.test",
  "catalogues": [
    {"source": "one/catalogue/.agents/skills@main", "path": "` + filepath.ToSlash(first) + `", "commit": "1111111111111111111111111111111111111111"},
    {"source": "two/catalogue/.agents/skills@main", "path": "` + filepath.ToSlash(second) + `", "commit": "2222222222222222222222222222222222222222"}
  ]
}
`
	if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	skillLoadPoint := filepath.Join(dir, "links", "skills")
	config := "sources:\n  - " + doctrine + "\n" +
		"skill_catalog_manifest: " + manifest + "\n" +
		"skill_load_points:\n  codex: " + skillLoadPoint + "\n" +
		"load_points:\n  claude: null\n  codex: " +
		filepath.Join(dir, "links", "AGENTS.md") + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := run(t, paths)
	if code != 0 {
		t.Fatalf("local catalogue converge failed: %s %s", out, errOut)
	}
	if !strings.Contains(out, "catalog local=2") {
		t.Fatalf("local catalogue summary missing: %s", out)
	}
	// Declaration order still decides which copy is kept where content raises
	// no objection, and the dedupe is reported rather than silent.
	if !strings.Contains(out, "catalog duplicate shared kept=") {
		t.Fatalf("dedupe was not reported: %s", out)
	}
	shared, err := os.Readlink(filepath.Join(skillLoadPoint, "shared"))
	if err != nil {
		t.Fatal(err)
	}
	if body := readFile(t, filepath.Join(shared, "SKILL.md")); body != "shared" {
		t.Fatalf("shared skill body = %q", body)
	}
	if _, err := os.Readlink(filepath.Join(skillLoadPoint, "aos-only")); err != nil {
		t.Fatal("AOS-only skill was not projected")
	}
}

func TestBadLocalCatalogueManifestPrecedesCompositionWrites(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "AGENTS.COMPOSE.md")
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "catalogues.json")
	if err := os.WriteFile(
		manifest,
		[]byte(`{"format":"aos.catalogues.v2","catalogues":[]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	config := "sources:\n  - " + doctrine + "\n" +
		"skill_catalog_manifest: " + manifest + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run(t, paths)
	if code != 1 || !strings.Contains(errOut, "aos.catalogues.v2") {
		t.Fatalf("bad manifest failure = %d %s", code, errOut)
	}
	for _, path := range []string{
		paths.Composed,
		filepath.Join(dir, "sources", "personality"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("pre-compose manifest failure wrote %s: %v", path, err)
		}
	}
}

func TestConvergeUsesConfiguredExternalPersonExclusively(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: dir,
		Home:         filepath.Join(dir, "home"),
	}
	personRoot, err := filepath.Abs(filepath.Join(
		"..", "..", "testdata", "contracts", "person-independent",
	))
	if err != nil {
		t.Fatal(err)
	}
	config := "person_policy: external-only\n" +
		"person_source: " + personRoot + "\n" +
		"roots:\n  - " + filepath.Join(dir, "sources") + "\n" +
		"load_points:\n  claude: " + filepath.Join(dir, "links", "CLAUDE.md") + "\n" +
		"  codex: null\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := run(t, paths)
	if code != 0 {
		t.Fatalf("external converge failed: %s %s", out, errOut)
	}
	composed := readFile(t, paths.Composed)
	if !strings.Contains(composed, "workbench builder") ||
		!strings.Contains(composed, "# Workbench invariant") {
		t.Fatalf("external person did not reach host context:\n%s", composed)
	}
	if strings.Contains(composed, "Agent //") ||
		strings.Contains(composed, "# Personality invariant") {
		t.Fatalf("external person inherited embedded context:\n%s", composed)
	}
	snapshot := readFile(t, filepath.Join(dir, "sources", "personality", "person.json"))
	if !strings.Contains(snapshot, `"source": "person:workbench"`) ||
		strings.Contains(snapshot, `"person:kai"`) {
		t.Fatalf("external person snapshot crossed source boundaries:\n%s", snapshot)
	}
}

func TestConvergeExternalOnlyFailsWithoutSource(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: dir,
		Home:         filepath.Join(dir, "home"),
	}
	config := "person_policy: external-only\n" +
		"roots:\n  - " + filepath.Join(dir, "sources") + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run(t, paths)
	if code != 1 || !strings.Contains(errOut, "requires person_source") {
		t.Fatalf("unguarded convergence did not fail closed: %d %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "sources", "personality")); !os.IsNotExist(err) {
		t.Fatalf("failed convergence wrote a roster: %v", err)
	}
}

func TestConvergeRetainsEmbeddedPersonalitiesOnBadRosterSource(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: dir,
	}
	config := "roots:\n  - " + filepath.Join(dir, "sources") + "\n" +
		"roster_sources:\n  - " + filepath.Join(dir, "missing.kdl") + "\n" +
		"load_points:\n  claude: " + filepath.Join(dir, "links", "CLAUDE.md") + "\n  codex: null\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, paths)
	if code != 0 || !strings.Contains(errOut, "warning: roster source") {
		t.Fatalf("bad roster source must warn and continue: %d %s", code, errOut)
	}
	composed := readFile(t, paths.Composed)
	if strings.Contains(composed, "definition pending") ||
		!strings.Contains(composed, "# Personality invariant") {
		t.Fatalf("embedded personality definitions must survive a bad overlay: %s", out)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeTestSkill(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ensureGitRepository(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		return
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "."},
		{"-c", "user.name=Agent Compose Tests", "-c", "user.email=agent-compose-tests@example.invalid", "commit", "-q", "-m", "fixture"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, root, err, output)
		}
	}
}

// convergeWithRequests builds one catalogue set and converges the given
// skill_requests against it, returning the exit code and streams.
func convergeWithRequests(t *testing.T, requests string) (int, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "doctrine", "AGENTS.COMPOSE.md")
	if err := os.MkdirAll(filepath.Dir(doctrine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogue := filepath.Join(dir, "catalogues", "enshrouded")
	writeTestSkill(t, catalogue, "sirens-game-enshrouded", "focus")
	// Not a skill: five repositories keep one of these beside their skills.
	if err := os.WriteFile(filepath.Join(catalogue, "categories.yaml"), []byte("a: b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "catalogues.json")
	body := `{
  "format": "aos.catalogues.v1",
  "forge": "https://forgejo.example.test",
  "catalogues": [
    {"source": "coilyco-gaming/enshrouded/.agents/skills@main", "path": "` +
		filepath.ToSlash(catalogue) + `", "commit": "1111111111111111111111111111111111111111"}
  ]
}
`
	if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	skillLoadPoint := filepath.Join(dir, "links", "skills")
	config := "sources:\n  - " + doctrine + "\n" +
		"skill_catalog_manifest: " + manifest + "\n" + requests +
		"skill_load_points:\n  codex: " + skillLoadPoint + "\n" +
		"load_points:\n  claude: null\n  codex: " +
		filepath.Join(dir, "links", "AGENTS.md") + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, paths)
	return code, out, errOut, skillLoadPoint
}

func TestConvergeResolvesASkillRequestByAddress(t *testing.T) {
	code, out, errOut, loadPoint := convergeWithRequests(t,
		"skill_requests:\n  - coilyco-gaming/enshrouded/sirens-game-enshrouded\n")
	if code != 0 {
		t.Fatalf("addressed converge failed: %s %s", out, errOut)
	}
	// The count is the attributable evidence: catalogues still mount whole, so
	// presence at the load point happens with no request at all.
	if !strings.Contains(out, "requests resolved=1") {
		t.Fatalf("resolution summary missing: %s", out)
	}
	if _, err := os.Stat(filepath.Join(loadPoint, "sirens-game-enshrouded")); err != nil {
		t.Fatalf("addressed skill was not projected: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(loadPoint, "categories.yaml")); !os.IsNotExist(err) {
		t.Fatal("a non-skill file beside the skills was projected")
	}
}

func TestConvergeFailsAndNamesAnUnresolvableAddress(t *testing.T) {
	code, _, errOut, _ := convergeWithRequests(t,
		"skill_requests:\n  - coilyco-gaming/enshrouded/absent-focus\n")
	if code == 0 {
		t.Fatal("an unresolvable address converged")
	}
	if !strings.Contains(errOut, "coilyco-gaming/enshrouded/absent-focus") {
		t.Fatalf("failure does not name the address: %s", errOut)
	}
}

func TestConvergeRefusesRequestsWithNoCompiledSet(t *testing.T) {
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "doctrine", "AGENTS.COMPOSE.md")
	if err := os.MkdirAll(filepath.Dir(doctrine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := "sources:\n  - " + doctrine + "\n" +
		"skill_requests:\n  - coilyco-gaming/enshrouded/sirens-game-enshrouded\n" +
		"load_points:\n  claude: null\n  codex: " +
		filepath.Join(dir, "links", "AGENTS.md") + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run(t, paths)
	if code == 0 {
		t.Fatal("requests without a compiled set converged")
	}
	if !strings.Contains(errOut, "skill_catalog_manifest") {
		t.Fatalf("failure does not name what is missing: %s", errOut)
	}
}

// convergeCatalogues writes one manifest over the given catalogue roots and
// converges it, so a test can vary only the catalogue content.
func convergeCatalogues(t *testing.T, build func(dir string) []string) (int, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	paths := cascade.Paths{
		Config:       filepath.Join(dir, "agent-compose.yaml"),
		Composed:     filepath.Join(dir, "COMPOSED.md"),
		ProjectsRoot: filepath.Join(dir, "projects"),
		Home:         filepath.Join(dir, "home"),
	}
	doctrine := filepath.Join(dir, "doctrine", "AGENTS.COMPOSE.md")
	if err := os.MkdirAll(filepath.Dir(doctrine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doctrine, []byte("# Doctrine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := build(dir)
	entries := make([]string, 0, len(roots))
	for index, root := range roots {
		private := ""
		if strings.HasSuffix(root, "-private") {
			private = `"private": true, `
		}
		entries = append(entries, fmt.Sprintf(
			`    {"source": "org/repo-%02d/.agents/skills@main", %s"path": %q, "commit": %q}`,
			index, private, filepath.ToSlash(root), strings.Repeat(fmt.Sprintf("%x", index%16), 40)))
	}
	manifest := filepath.Join(dir, "catalogues.json")
	body := "{\n  \"format\": \"aos.catalogues.v1\",\n" +
		"  \"forge\": \"https://forgejo.example.test\",\n  \"catalogues\": [\n" +
		strings.Join(entries, ",\n") + "\n  ]\n}\n"
	if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	skillLoadPoint := filepath.Join(dir, "links", "skills")
	config := "sources:\n  - " + doctrine + "\n" +
		"skill_catalog_manifest: " + manifest + "\n" +
		"skill_load_points:\n  codex: " + skillLoadPoint + "\n" +
		"load_points:\n  claude: null\n  codex: " +
		filepath.Join(dir, "links", "AGENTS.md") + "\n"
	if err := os.WriteFile(paths.Config, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, paths)
	return code, out, errOut, skillLoadPoint
}

// R1: a manifest of twenty-two entries loads, every tree validates, and each
// catalogue reports its own provenance rather than one merged count.
func TestConvergeLoadsTwentyTwoCataloguesWithPerSourceProvenance(t *testing.T) {
	const count = 22
	code, out, errOut, loadPoint := convergeCatalogues(t, func(dir string) []string {
		roots := make([]string, 0, count)
		for index := 0; index < count; index++ {
			root := filepath.Join(dir, "catalogues", fmt.Sprintf("c%02d", index))
			writeTestSkill(t, root, fmt.Sprintf("skill-%02d", index), "body")
			roots = append(roots, root)
		}
		return roots
	})
	if code != 0 {
		t.Fatalf("twenty-two catalogue converge failed: %s %s", out, errOut)
	}
	if !strings.Contains(out, fmt.Sprintf("catalog local=%d", count)) {
		t.Fatalf("catalogue count missing: %s", out)
	}
	for index := 0; index < count; index++ {
		line := fmt.Sprintf("catalog forgejo.example.test/org/repo-%02d/.agents/skills@main skills=1", index)
		if !strings.Contains(out, line) {
			t.Fatalf("per-source provenance missing for entry %d: %s", index, out)
		}
		if _, err := os.Stat(filepath.Join(loadPoint, fmt.Sprintf("skill-%02d", index))); err != nil {
			t.Fatalf("catalogue %d skill was not projected: %v", index, err)
		}
	}
}

// R4: one name meaning two different skills stops the converge and names both
// sources, rather than the last manifest entry quietly winning.
func TestConvergeRefusesOneNameMeaningTwoSkills(t *testing.T) {
	code, _, errOut, _ := convergeCatalogues(t, func(dir string) []string {
		first := filepath.Join(dir, "catalogues", "a")
		second := filepath.Join(dir, "catalogues", "b")
		writeTestSkill(t, first, "shared", "one thing")
		writeTestSkill(t, second, "shared", "a different thing")
		return []string{first, second}
	})
	if code == 0 {
		t.Fatal("divergent content under one name converged")
	}
	for _, want := range []string{"org/repo-00", "org/repo-01", "shared"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("refusal does not name %s: %s", want, errOut)
		}
	}
}

// R5: a private catalogue's content reaches the load point and its identity
// reaches nothing, which is the spec's own grep check over what ships.
func TestConvergeKeepsAPrivateCatalogueOutOfEverythingItWrites(t *testing.T) {
	var loadPoint string
	code, out, errOut, lp := convergeCatalogues(t, func(dir string) []string {
		open := filepath.Join(dir, "catalogues", "a")
		shut := filepath.Join(dir, "catalogues", "b-private")
		writeTestSkill(t, open, "public-skill", "open")
		writeTestSkill(t, shut, "sirens-game-enshrouded", "secret body")
		return []string{open, shut}
	})
	loadPoint = lp
	if code != 0 {
		t.Fatalf("private catalogue converge failed: %s %s", out, errOut)
	}

	// The content ships, which is what lets a private focus reach a public lane
	// without being copied into it.
	target, err := os.Readlink(filepath.Join(loadPoint, "sirens-game-enshrouded"))
	if err != nil {
		t.Fatalf("private catalogue skill was not projected: %v", err)
	}
	if body := readFile(t, filepath.Join(target, "SKILL.md")); body != "secret body" {
		t.Fatalf("projected body = %q", body)
	}

	// The identity does not. repo-01 is the private entry's own source.
	for name, stream := range map[string]string{"stdout": out, "stderr": errOut} {
		if strings.Contains(stream, "org/repo-01") {
			t.Fatalf("%s names the private source: %s", name, stream)
		}
		if strings.Contains(stream, "repo-00") && !strings.Contains(stream, "private catalogue 1") {
			t.Fatalf("%s reports the public source but not the redacted private one: %s", name, stream)
		}
	}
	if !strings.Contains(out, "private catalogue 1") {
		t.Fatalf("the private catalogue was not reported at all: %s", out)
	}
}
