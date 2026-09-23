package project

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/compose"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

func composeFixture(t *testing.T, name string) string {
	t.Helper()
	result, err := compose.Run(filepath.Join("..", "..", "testdata", "contracts", name), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return result.Bundle.Dir
}

func readTarget(t *testing.T, target, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("expected %s at load point: %v", rel, err)
	}
	return string(raw)
}

func TestProjectNativeLayouts(t *testing.T) {
	bundleDir := composeFixture(t, "native.kdl")
	cases := map[string]struct{ instructions, skillsDir string }{
		"claude":   {"CLAUDE.md", ".claude/skills"},
		"codex":    {"AGENTS.md", ".agents/skills"},
		"goose":    {".goosehints", ".agents/skills"},
		"opencode": {"AGENTS.md", ".agents/skills"},
	}
	for layout, want := range cases {
		t.Run(layout, func(t *testing.T) {
			target := t.TempDir()
			result, err := Project(bundleDir, layout, target)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(readTarget(t, target, want.instructions), "Fixture foundation") {
				t.Fatal("instructions load point missing foundation content")
			}
			if !strings.Contains(
				readTarget(t, target, want.instructions),
				"**Role skill // `role-platform-eng`**",
			) {
				t.Fatal("instructions load point missing the selected role identity card")
			}
			for _, identity := range selectedFixtureSkills(t, "platform-eng") {
				skill := readTarget(t, target, want.skillsDir+"/"+identity+"/SKILL.md")
				if !strings.Contains(skill, skillHeading(identity)) {
					t.Fatalf("skill load point %s has wrong content:\n%s", identity, skill)
				}
			}
			if len(result.Files) < 2 {
				t.Fatalf("expected instructions and personality files, got %v", result.Files)
			}
		})
	}
}

func TestProjectCompiledLayouts(t *testing.T) {
	bundleDir := composeFixture(t, "compiled.kdl")
	cases := map[string]string{
		"claude":   "CLAUDE.md",
		"codex":    "AGENTS.md",
		"goose":    ".goosehints",
		"opencode": "AGENTS.md",
	}
	for layout, instructions := range cases {
		t.Run(layout, func(t *testing.T) {
			target := t.TempDir()
			result, err := Project(bundleDir, layout, target)
			if err != nil {
				t.Fatal(err)
			}
			compiled := readTarget(t, target, instructions)
			for _, identity := range selectedFixtureSkills(t, "platform-eng") {
				if !strings.Contains(compiled, skillHeading(identity)) {
					t.Fatalf("%s load point missing compiled prose for %s", layout, identity)
				}
			}
			if len(result.Files) != 1 {
				t.Fatalf("compiled projection must place one file, got %v", result.Files)
			}
		})
	}

}

func TestProjectRejectsUnknownLayoutAndUnsupportedMode(t *testing.T) {
	native := composeFixture(t, "native.kdl")
	if _, err := Project(native, "emacs", t.TempDir()); err == nil || !strings.Contains(err.Error(), "v0.1 layouts") {
		t.Fatalf("expected unknown-layout diagnostic, got %v", err)
	}

	Registry["compiled-only"] = Layout{Compiled: &LoadPoints{Instructions: "CONTEXT.md"}}
	defer delete(Registry, "compiled-only")
	if _, err := Project(native, "compiled-only", t.TempDir()); err == nil || !strings.Contains(err.Error(), "does not support bundles") {
		t.Fatalf("expected unsupported-mode diagnostic, got %v", err)
	}
}

func TestProjectRefusesForeignFiles(t *testing.T) {
	bundleDir := composeFixture(t, "native.kdl")
	target := t.TempDir()
	handAuthored := filepath.Join(target, "CLAUDE.md")
	if err := os.WriteFile(handAuthored, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Project(bundleDir, "claude", target); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected ownership refusal, got %v", err)
	}
	if raw, _ := os.ReadFile(handAuthored); string(raw) != "mine\n" {
		t.Fatalf("foreign file must stay untouched, got %q", raw)
	}
}

func TestApplyOwnedAcceptsIdenticalForeignFilesWithoutClaimingThem(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "AGENTS.md")
	content := []byte("canonical\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplyOwned(target, map[string][]byte{"AGENTS.md": content}, "fixture", "bundle")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("identical foreign file was claimed: %v", result.Files)
	}
	if got := readTarget(t, target, "AGENTS.md"); got != string(content) {
		t.Fatalf("canonical file changed: %q", got)
	}
	if err := Validate(target); err == nil {
		t.Fatal("empty sidecar unexpectedly validated as a complete projection")
	}
}

func TestApplyOwnedAcceptsIdenticalForeignSymlinkWithoutReplacingIt(t *testing.T) {
	target := t.TempDir()
	source := filepath.Join(t.TempDir(), "SKILL.md")
	content := []byte("canonical skill\n")
	if err := os.WriteFile(source, content, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "SKILL.md")
	if err := os.Symlink(source, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	result, err := ApplyOwned(target, map[string][]byte{"SKILL.md": content}, "fixture", "bundle")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("identical foreign symlink was claimed: %v", result.Files)
	}
	resolved, err := os.Readlink(path)
	if err != nil || resolved != source {
		t.Fatalf("canonical symlink was replaced: %q, %v", resolved, err)
	}
}

func TestHomeScopeProjection(t *testing.T) {
	native := composeFixture(t, "native.kdl")
	compiled := composeFixture(t, "compiled.kdl")
	nativeBefore := treeFingerprint(t, native)
	compiledBefore := treeFingerprint(t, compiled)
	expectedIdentities := selectedFixtureSkills(t, "platform-eng")
	cases := map[string]struct{ instructions, skillsDir string }{
		"claude":   {".claude/CLAUDE.md", ".claude/skills"},
		"codex":    {".codex/AGENTS.md", ".agents/skills"},
		"goose":    {".config/goose/.goosehints", ".agents/skills"},
		"opencode": {".config/opencode/AGENTS.md", ".agents/skills"},
	}
	for layout, want := range cases {
		t.Run(layout, func(t *testing.T) {
			home := t.TempDir()
			if _, err := ProjectScoped(native, layout, home, ScopeHome); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(readTarget(t, home, want.instructions), "Fixture foundation") {
				t.Fatalf("home instructions load point %s missing content", want.instructions)
			}
			for _, identity := range expectedIdentities {
				readTarget(t, home, want.skillsDir+"/"+identity+"/SKILL.md")
			}
			identities := projectedIdentities(t, home)
			slices.Sort(identities)
			if !slices.Equal(identities, expectedIdentities) {
				t.Fatalf("native home identities = %v, want %v", identities, expectedIdentities)
			}

			home = t.TempDir()
			if _, err := ProjectScoped(compiled, layout, home, ScopeHome); err != nil {
				t.Fatal(err)
			}
			compiledContext := readTarget(t, home, want.instructions)
			for _, identity := range expectedIdentities {
				if !strings.Contains(compiledContext, skillHeading(identity)) {
					t.Fatalf("home compiled load point missing %s", identity)
				}
			}
			if identities := projectedIdentities(t, home); len(identities) != 0 {
				t.Fatalf("compiled home unexpectedly mounted identity trees: %v", identities)
			}
		})
	}

	if _, err := ProjectScoped(native, "claude", t.TempDir(), "galaxy"); err == nil || !strings.Contains(err.Error(), "unknown scope") {
		t.Fatalf("expected unknown-scope diagnostic, got %v", err)
	}
	if after := treeFingerprint(t, native); after != nativeBefore {
		t.Fatal("home projection mutated the native input bundle")
	}
	if after := treeFingerprint(t, compiled); after != compiledBefore {
		t.Fatal("home projection mutated the compiled input bundle")
	}
}

func TestReprojectionChangesDeliveryAndPreservesForeignFiles(t *testing.T) {
	target := t.TempDir()
	native := composeFixture(t, "native.kdl")
	if _, err := Project(native, "claude", target); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(target, "notes.md")
	if err := os.WriteFile(foreign, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	compiled := composeFixture(t, "compiled.kdl")
	result, err := Project(compiled, "claude", target)
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range selectedFixtureSkills(t, "platform-eng") {
		if _, err := os.Stat(filepath.Join(target, ".claude", "skills", identity)); !os.IsNotExist(err) {
			t.Fatalf("stale native identity %s survived compiled re-projection: %v", identity, err)
		}
	}
	if !strings.Contains(readTarget(t, target, "CLAUDE.md"), "# Tenacious") {
		t.Fatal("compiled re-projection omitted canonical personality prose")
	}
	if raw, _ := os.ReadFile(foreign); string(raw) != "keep me\n" {
		t.Fatal("foreign file must survive re-projection")
	}
	if !slices.ContainsFunc(result.Files, func(rel string) bool {
		return rel == "CLAUDE.md"
	}) {
		t.Fatalf("sidecar omitted the compiled load point: %v", result.Files)
	}
}

func TestProjectionRollsBackAfterWriteFailure(t *testing.T) {
	target := t.TempDir()
	initial := map[string][]byte{"one.txt": []byte("old\n")}
	if _, err := ApplyOwned(target, initial, "fixture", "old-bundle"); err != nil {
		t.Fatal(err)
	}
	sidecarPath := filepath.Join(target, filepath.FromSlash(sidecarRel))
	sidecarBefore, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatal(err)
	}

	writeFile := func(path string, content []byte, mode fs.FileMode) error {
		if filepath.Base(path) == "two.txt" {
			return errors.New("injected write failure")
		}
		return os.WriteFile(path, content, mode)
	}
	next := map[string][]byte{
		"one.txt": []byte("new\n"),
		"two.txt": []byte("new\n"),
	}
	if _, err := applyOwnedWithWriter(target, next, "fixture", "new-bundle", writeFile); err == nil ||
		!strings.Contains(err.Error(), "previous projection restored") {
		t.Fatalf("expected restored projection failure, got %v", err)
	}
	if got := readTarget(t, target, "one.txt"); got != "old\n" {
		t.Fatalf("owned file was not restored, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(target, "two.txt")); !os.IsNotExist(err) {
		t.Fatalf("partially written file survived rollback: %v", err)
	}
	sidecarAfter, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(sidecarAfter) != string(sidecarBefore) {
		t.Fatal("projection sidecar changed despite rollback")
	}
}

func TestInvalidBundleDoesNotTouchProjectionTarget(t *testing.T) {
	bundleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(bundleDir, "manifest.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	marker := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(marker, []byte("unchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeFingerprint(t, target)
	if _, err := ProjectScoped(bundleDir, "claude", target, ScopeHome); err == nil {
		t.Fatal("invalid bundle unexpectedly projected")
	}
	if after := treeFingerprint(t, target); after != before {
		t.Fatal("bundle verification failure changed the projection target")
	}
}

func TestProjectionCannotTargetItsInputBundle(t *testing.T) {
	bundleDir := composeFixture(t, "native.kdl")
	before := treeFingerprint(t, bundleDir)
	if _, err := ProjectScoped(bundleDir, "claude", bundleDir, ScopeHome); err == nil ||
		!strings.Contains(err.Error(), "must not be the bundle") {
		t.Fatalf("expected input-bundle target failure, got %v", err)
	}
	if after := treeFingerprint(t, bundleDir); after != before {
		t.Fatal("rejected projection changed its input bundle")
	}
}

func TestApplyOwnedRejectsUnsafeTargetPath(t *testing.T) {
	target := t.TempDir()
	if _, err := ApplyOwned(target, map[string][]byte{"../outside": []byte("bad")}, "fixture", "bundle"); err == nil ||
		!strings.Contains(err.Error(), "safe relative path") {
		t.Fatalf("expected unsafe target-path failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target), "outside")); !os.IsNotExist(err) {
		t.Fatalf("unsafe target path escaped projection root: %v", err)
	}
}

func TestApplyOwnedRejectsSymlinkParent(t *testing.T) {
	target := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(target, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ApplyOwned(target, map[string][]byte{"linked/context.md": []byte("bad")}, "fixture", "bundle"); err == nil ||
		!strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("expected symlink-parent failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(external, "context.md")); !os.IsNotExist(err) {
		t.Fatalf("projection escaped through symlink parent: %v", err)
	}
}

func projectedIdentities(t *testing.T, root string) []string {
	t.Helper()
	var identities []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == "SKILL.md" {
			identities = append(identities, filepath.Base(filepath.Dir(path)))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return identities
}

func selectedFixtureSkills(t *testing.T, roleName string) []string {
	t.Helper()
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	role, ok := p.Roles[roleName]
	if !ok {
		t.Fatalf("fixture role %q is absent", roleName)
	}
	skills := make([]string, 0, len(role.Personalities)+len(role.Boundaries)+2)
	skills = append(skills, p.RoleSkillID(roleName))
	for _, boundaryName := range role.Boundaries {
		skills = append(skills, p.Boundaries[boundaryName].Skill)
	}
	// A role declares only the boundaries it defers, so a scoped grant and an
	// owned boundary reach it through the catalog instead.
	for _, scoped := range role.ScopedBoundaries {
		skills = append(skills, p.Boundaries[scoped.Name].Skill)
	}
	for _, boundary := range p.Boundaries {
		if boundary.Owner == roleName {
			skills = append(skills, boundary.Skill)
		}
	}
	for _, personalityName := range role.Personalities {
		skills = append(skills, p.Personalities[personalityName].Skill)
	}
	skills = append(skills, "fixture-review")
	slices.Sort(skills)
	return skills
}

func skillHeading(skillID string) string {
	if skillID == "fixture-review" {
		return "# Fixture review"
	}
	// A boundary body carries an authored doctrine title rather than a title derived
	// from its id, so its frontmatter name is the identity worth asserting.
	if strings.HasPrefix(skillID, "boundary-") {
		return "name: " + skillID
	}
	// A role body is titled with its display name rather than its slug, so
	// the frontmatter name is again the identity worth asserting.
	if strings.HasPrefix(skillID, "role-") {
		return "name: " + skillID
	}
	name := strings.TrimPrefix(skillID, "personality-")
	return "# " + strings.ToUpper(name[:1]) + name[1:]
}

func treeFingerprint(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%s\x00%s\x00", filepath.ToSlash(rel), info.Mode())
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hash.Write(raw)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
