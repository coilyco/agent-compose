// Package skillaudit names every skill that two load roots both offer and says
// whether the copies match. It only reads: a root a session may load from is
// somewhere to look, never to repair. See docs/projection.md.
package skillaudit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/treehash"
)

// Root is one directory of skills, as its caller spelled it.
type Root struct {
	Label string
	Path  string
}

// Kind classifies one name held by more than one root.
type Kind string

const (
	Identical  Kind = "identical"
	Divergent  Kind = "divergent"
	Unverified Kind = "unverified"
)

// Copy is one root's skill directory. Bytes and Mtime describe its SKILL.md.
type Copy struct {
	Label      string
	Path       string
	Digest     string
	Bytes      int64
	Mtime      time.Time
	Older      bool
	Unverified string
}

// Finding is a name offered by two or more roots.
type Finding struct {
	Name   string
	Kind   Kind
	Copies []Copy
}

// Audited is a root that was read, with the skills it holds.
type Audited struct {
	Root
	Skills int
}

// Report is the outcome of one audit.
type Report struct {
	Roots    []Audited
	Skipped  []string
	Findings []Finding
}

// Count returns how many findings are of kind.
func (r *Report) Count(kind Kind) int {
	n := 0
	for _, finding := range r.Findings {
		if finding.Kind == kind {
			n++
		}
	}
	return n
}

// Roots is the explicit list when one is given, discovery otherwise.
func Roots(explicit []string, realHome, sessionHome, cwd string) ([]Root, error) {
	if len(explicit) == 0 {
		return Discover(realHome, sessionHome, cwd)
	}
	roots := make([]Root, 0, len(explicit))
	for _, path := range explicit {
		roots = append(roots, Root{Label: "explicit", Path: path})
	}
	return roots, nil
}

// Discover lists the home layout under both homes and the repo layout at cwd
// and each ancestor, with suffixes read from the project registries.
func Discover(realHome, sessionHome, cwd string) ([]Root, error) {
	repoRegistry, homeRegistry, err := project.Registries()
	if err != nil {
		return nil, err
	}
	var roots []Root
	seen := map[string]bool{}
	add := func(label, path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		roots = append(roots, Root{Label: label, Path: path})
	}
	homeDirs := skillDirs(homeRegistry)
	for _, dir := range homeDirs {
		add("home", filepath.Join(realHome, dir))
	}
	if sessionHome != realHome {
		for _, dir := range homeDirs {
			add("session-home", filepath.Join(sessionHome, dir))
		}
	}
	repoDirs := skillDirs(repoRegistry)
	for dir := cwd; dir != ""; {
		for _, rel := range repoDirs {
			add("project", filepath.Join(dir, rel))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return roots, nil
}

func skillDirs(registry map[string]project.Layout) []string {
	set := map[string]bool{}
	for _, layout := range registry {
		if layout.Native != nil && layout.Native.SkillsDir != "" {
			set[layout.Native.SkillsDir] = true
		}
	}
	dirs := make([]string, 0, len(set))
	for dir := range set {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

// Audit reads every root that exists, once per real directory, and reports the
// names two roots both offer.
func Audit(roots []Root) (*Report, error) {
	report := &Report{}
	byName := map[string][]Copy{}
	var names []string
	resolved := map[string]bool{}
	for _, root := range roots {
		real, err := filepath.EvalSymlinks(root.Path)
		if err != nil {
			report.Skipped = append(report.Skipped, root.Path)
			continue
		}
		if info, err := os.Stat(real); err != nil || !info.IsDir() {
			report.Skipped = append(report.Skipped, root.Path)
			continue
		}
		if resolved[real] {
			continue
		}
		resolved[real] = true
		entries, err := os.ReadDir(real)
		if err != nil {
			return nil, fmt.Errorf("read skill root %s: %w", root.Path, err)
		}
		held := 0
		for _, entry := range entries {
			skill := filepath.Join(root.Path, entry.Name())
			info, err := os.Stat(filepath.Join(skill, "SKILL.md"))
			if err != nil || info.IsDir() {
				continue
			}
			held++
			if _, seen := byName[entry.Name()]; !seen {
				names = append(names, entry.Name())
			}
			byName[entry.Name()] = append(byName[entry.Name()], Copy{
				Label: root.Label, Path: skill, Bytes: info.Size(), Mtime: info.ModTime(),
			})
		}
		report.Roots = append(report.Roots, Audited{Root: root, Skills: held})
	}
	sort.Strings(names)
	for _, name := range names {
		copies := byName[name]
		if len(copies) < 2 {
			continue
		}
		report.Findings = append(report.Findings, classify(name, copies))
	}
	return report, nil
}

func classify(name string, copies []Copy) Finding {
	digests := map[string]bool{}
	unverified := false
	for i := range copies {
		digest, err := digestSkill(copies[i].Path)
		if err != nil {
			copies[i].Unverified = err.Error()
			unverified = true
			continue
		}
		copies[i].Digest = digest
		digests[digest] = true
	}
	kind := Identical
	switch {
	case len(digests) > 1:
		kind = Divergent
		newest := copies[0].Mtime
		for _, copy := range copies {
			if copy.Mtime.After(newest) {
				newest = copy.Mtime
			}
		}
		for i := range copies {
			copies[i].Older = copies[i].Mtime.Before(newest)
		}
	case unverified:
		kind = Unverified
	}
	return Finding{Name: name, Kind: kind, Copies: copies}
}

// digestSkill hashes the directory a skill path resolves to, so a symlinked
// mount compares by what it points at.
func digestSkill(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return treehash.Digest(os.DirFS(filepath.Dir(real)), filepath.Base(real))
}

// Write prints the roots, then each divergent and unverified name in full and
// each identical name on one line.
func (r *Report) Write(w io.Writer) {
	fmt.Fprintln(w, "roots")
	for _, root := range r.Roots {
		fmt.Fprintf(w, "  %s (%s) %d skills\n", root.Path, root.Label, root.Skills)
	}
	if len(r.Skipped) > 0 {
		fmt.Fprintf(w, "  absent: %d candidate roots not on disk\n", len(r.Skipped))
	}
	fmt.Fprintf(w, "names in more than one root: %d (%d divergent, %d unverified, %d identical)\n",
		len(r.Findings), r.Count(Divergent), r.Count(Unverified), r.Count(Identical))
	for _, finding := range r.Findings {
		if finding.Kind == Identical {
			continue
		}
		fmt.Fprintf(w, "%s %s\n", finding.Kind, finding.Name)
		for _, copy := range finding.Copies {
			fmt.Fprintf(w, "  %s\n", describe(copy))
		}
	}
	for _, finding := range r.Findings {
		if finding.Kind == Identical {
			fmt.Fprintf(w, "identical %s in %d roots\n", finding.Name, len(finding.Copies))
		}
	}
}

func describe(copy Copy) string {
	parts := []string{
		copy.Path,
		fmt.Sprintf("SKILL.md %d bytes", copy.Bytes),
		copy.Mtime.UTC().Format(time.RFC3339),
	}
	if copy.Digest != "" {
		parts = append(parts, "sha "+copy.Digest[:8])
	}
	if copy.Unverified != "" {
		parts = append(parts, "unverified: "+copy.Unverified)
	}
	if copy.Older {
		parts = append(parts, "older")
	}
	return strings.Join(parts, "  ")
}
