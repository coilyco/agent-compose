package skillaudit

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func writeSkill(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func setMtime(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

func find(report *Report, name string) *Finding {
	for i := range report.Findings {
		if report.Findings[i].Name == name {
			return &report.Findings[i]
		}
	}
	return nil
}

func TestDivergentCopiesFailAndNameBothPaths(t *testing.T) {
	older := filepath.Join(t.TempDir(), "skills")
	newer := filepath.Join(t.TempDir(), "skills")
	oldDir := writeSkill(t, older, "boundary", "short body\n")
	newDir := writeSkill(t, newer, "boundary", "short body\nplus a paragraph the older copy lacks\n")
	setMtime(t, filepath.Join(oldDir, "SKILL.md"), time.Date(2026, 9, 16, 2, 50, 0, 0, time.UTC))
	setMtime(t, filepath.Join(newDir, "SKILL.md"), time.Date(2026, 9, 19, 18, 39, 0, 0, time.UTC))

	report, err := Audit([]Root{{Label: "project", Path: older}, {Label: "home", Path: newer}})
	if err != nil {
		t.Fatal(err)
	}
	got := find(report, "boundary")
	if got == nil || got.Kind != Divergent {
		t.Fatalf("want a divergent finding, got %+v", report.Findings)
	}
	if report.Count(Divergent) != 1 {
		t.Fatalf("divergent count = %d", report.Count(Divergent))
	}
	var out bytes.Buffer
	report.Write(&out)
	for _, want := range []string{oldDir, newDir, "divergent", "older"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output lacks %q:\n%s", want, out.String())
		}
	}
	for _, copy := range got.Copies {
		if copy.Older != (copy.Path == oldDir) {
			t.Fatalf("only the earlier copy is older: %+v", copy)
		}
		if copy.Bytes == 0 {
			t.Fatalf("SKILL.md size not recorded: %+v", copy)
		}
	}
}

func TestIdenticalCopiesAreANoteNotAFailure(t *testing.T) {
	a := filepath.Join(t.TempDir(), "skills")
	b := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, a, "same", "one body\n")
	writeSkill(t, b, "same", "one body\n")

	report, err := Audit([]Root{{Label: "a", Path: a}, {Label: "b", Path: b}})
	if err != nil {
		t.Fatal(err)
	}
	got := find(report, "same")
	if got == nil || got.Kind != Identical {
		t.Fatalf("want identical, got %+v", report.Findings)
	}
	if report.Count(Divergent) != 0 {
		t.Fatal("identical copies must not count as divergent")
	}
}

func TestNameInOneRootOnlyIsNotAFinding(t *testing.T) {
	a := filepath.Join(t.TempDir(), "skills")
	b := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, a, "only-a", "x\n")
	writeSkill(t, b, "only-b", "y\n")

	report, err := Audit([]Root{{Label: "a", Path: a}, {Label: "b", Path: b}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("unexpected findings: %+v", report.Findings)
	}
}

func TestOneDirectoryReachedTwiceIsAuditedOnce(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	writeSkill(t, real, "x", "body\n")
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	report, err := Audit([]Root{
		{Label: "first", Path: real},
		{Label: "again", Path: real},
		{Label: "via-link", Path: link},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots) != 1 || report.Roots[0].Label != "first" {
		t.Fatalf("want one root labelled first, got %+v", report.Roots)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("a directory must not duplicate itself: %+v", report.Findings)
	}
}

func TestSymlinkedSkillIsComparedByTargetContent(t *testing.T) {
	src := t.TempDir()
	target := writeSkill(t, src, "x", "shared body\n")
	mounted := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(mounted, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(mounted, "x")); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, copied, "x", "shared body\n")

	report, err := Audit([]Root{{Label: "mounted", Path: mounted}, {Label: "copied", Path: copied}})
	if err != nil {
		t.Fatal(err)
	}
	got := find(report, "x")
	if got == nil || got.Kind != Identical {
		t.Fatalf("a symlinked mount and a copy of the same content are identical: %+v", report.Findings)
	}
}

func TestInternalSymlinkIsReportedUnverified(t *testing.T) {
	a := filepath.Join(t.TempDir(), "skills")
	b := filepath.Join(t.TempDir(), "skills")
	dir := writeSkill(t, a, "x", "body\n")
	if err := os.Symlink("SKILL.md", filepath.Join(dir, "alias.md")); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, b, "x", "body\n")

	report, err := Audit([]Root{{Label: "a", Path: a}, {Label: "b", Path: b}})
	if err != nil {
		t.Fatalf("an unhashable skill must not abort the audit: %v", err)
	}
	got := find(report, "x")
	if got == nil || got.Kind != Unverified {
		t.Fatalf("want unverified, got %+v", report.Findings)
	}
	var out bytes.Buffer
	report.Write(&out)
	if !strings.Contains(out.String(), "unverified") {
		t.Fatalf("an unverified copy must be printed:\n%s", out.String())
	}
	if report.Count(Divergent) != 0 {
		t.Fatal("unverified is not divergent")
	}
}

func TestCRLFAgainstLFIsIdentical(t *testing.T) {
	a := filepath.Join(t.TempDir(), "skills")
	b := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, a, "x", "line one\r\nline two\r\n")
	writeSkill(t, b, "x", "line one\nline two\n")

	report, err := Audit([]Root{{Label: "a", Path: a}, {Label: "b", Path: b}})
	if err != nil {
		t.Fatal(err)
	}
	if got := find(report, "x"); got == nil || got.Kind != Identical {
		t.Fatalf("CRLF must fold, got %+v", report.Findings)
	}
}

func TestAuditChangesNothing(t *testing.T) {
	a := filepath.Join(t.TempDir(), "skills")
	b := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, a, "x", "old\n")
	writeSkill(t, b, "x", "new\n")
	before := snapshot(t, a, b)

	if _, err := Audit([]Root{{Label: "a", Path: a}, {Label: "b", Path: b}}); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, a, b); after != before {
		t.Fatalf("audit modified a root:\nbefore %s\nafter  %s", before, after)
	}
}

func snapshot(t *testing.T, roots ...string) string {
	t.Helper()
	h := sha256.New()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "%s|%d|%d|%v\n", p, info.Size(), info.ModTime().UnixNano(), info.Mode())
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestAbsentRootIsSkippedAndNamed(t *testing.T) {
	present := filepath.Join(t.TempDir(), "skills")
	writeSkill(t, present, "x", "body\n")
	missing := filepath.Join(t.TempDir(), "nowhere", "skills")

	report, err := Audit([]Root{{Label: "here", Path: present}, {Label: "gone", Path: missing}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots) != 1 {
		t.Fatalf("roots = %+v", report.Roots)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != missing {
		t.Fatalf("skipped = %v", report.Skipped)
	}
	var out bytes.Buffer
	report.Write(&out)
	if !strings.Contains(out.String(), "absent: 1 candidate roots") || strings.Contains(out.String(), missing) {
		t.Fatalf("absent roots print as a count, not a line each:\n%s", out.String())
	}
}

func TestDiscoverCoversBothHomesAndEveryAncestor(t *testing.T) {
	realHome := "/real/kai"
	sessionHome := "/shadow/home"
	cwd := "/shadow/projects/org/repo"

	roots := Discover(realHome, sessionHome, cwd)
	var paths []string
	for _, root := range roots {
		paths = append(paths, root.Path)
	}
	for _, want := range []string{
		"/real/kai/.claude/skills",
		"/real/kai/.agents/skills",
		"/shadow/home/.claude/skills",
		"/shadow/home/.agents/skills",
		"/shadow/projects/org/repo/.claude/skills",
		"/shadow/projects/org/.claude/skills",
		"/shadow/projects/.claude/skills",
		"/shadow/.claude/skills",
		"/.claude/skills",
	} {
		if !contains(paths, want) {
			t.Fatalf("discovery lacks %s in %v", want, paths)
		}
	}
	if dupes := duplicates(paths); len(dupes) != 0 {
		t.Fatalf("discovery repeats %v", dupes)
	}
	for _, root := range roots {
		if strings.HasSuffix(root.Path, "/.claude/skills") && root.Path == "/real/kai/.claude/skills" && root.Label != "home" {
			t.Fatalf("real home labelled %q", root.Label)
		}
	}
}

func TestDiscoverCollapsesASessionHomeThatIsTheRealHome(t *testing.T) {
	roots := Discover("/h", "/h", "/h/p")
	var paths []string
	for _, root := range roots {
		paths = append(paths, root.Path)
	}
	if dupes := duplicates(paths); len(dupes) != 0 {
		t.Fatalf("repeats %v", dupes)
	}
}

func TestExplicitRootsReplaceDiscovery(t *testing.T) {
	roots := Roots([]string{"/a", "/b"}, "/real", "/session", "/cwd")
	if len(roots) != 2 || roots[0].Path != "/a" || roots[1].Path != "/b" {
		t.Fatalf("explicit roots must stand alone: %+v", roots)
	}
	if got := Roots(nil, "/real", "/session", "/cwd"); len(got) < 3 {
		t.Fatalf("no explicit roots must fall back to discovery: %+v", got)
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func duplicates(list []string) []string {
	seen := map[string]bool{}
	var dupes []string
	for _, item := range list {
		if seen[item] {
			dupes = append(dupes, item)
		}
		seen[item] = true
	}
	sort.Strings(dupes)
	return dupes
}
