package cascade

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/repositoryplan"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

var headingRe = regexp.MustCompile(`^(#{1,6}) +\S`)

// linkRe matches a markdown inline-link tail after "]", capturing the target
// apart from any title; v1 used a lookbehind, Go keeps the bracket literal.
var linkRe = regexp.MustCompile(`\]\(([^)\s]+)(\s+[^)]*)?\)`)

var schemeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

var navigationHeadings = map[string]bool{"see also": true}

func headerLevel(line string) int {
	match := headingRe.FindStringSubmatch(line)
	if match == nil {
		return 0
	}
	return len(match[1])
}

// sectionEnd returns the index just past the section opened at start: a
// section runs to the next heading of the same or shallower level.
func sectionEnd(lines []string, start int) int {
	level := headerLevel(lines[start])
	end := start + 1
	for end < len(lines) {
		if here := headerLevel(lines[end]); here != 0 && here <= level {
			break
		}
		end++
	}
	return end
}

type overrideSection struct {
	heading string
	lines   []string
}

func overrideSections(body string) []overrideSection {
	lines := strings.Split(strings.Trim(body, "\n"), "\n")
	start := 0
	for start < len(lines) && headerLevel(lines[start]) == 0 {
		start++
	}
	var sections []overrideSection
	for start < len(lines) {
		end := sectionEnd(lines, start)
		sections = append(sections, overrideSection{
			heading: strings.TrimRight(lines[start], " \t"),
			lines:   lines[start:end],
		})
		start = end
	}
	return sections
}

// applyOverrides merges override sections into a base body by verbatim
// heading: a match replaces the section, a new heading appends, ambiguity fails.
func applyOverrides(baseBody, overrideBody string) (string, error) {
	lines := strings.Split(baseBody, "\n")
	for _, section := range overrideSections(overrideBody) {
		var matches []int
		for i, line := range lines {
			if strings.TrimRight(line, " \t") == section.heading {
				matches = append(matches, i)
			}
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("override heading %q matches %d sections", section.heading, len(matches))
		}
		if len(matches) == 1 {
			start := matches[0]
			end := sectionEnd(lines, start)
			lines = append(lines[:start], append(append([]string{}, section.lines...), lines[end:]...)...)
		} else {
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
				lines = append(lines, "")
			}
			lines = append(lines, section.lines...)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// stripNavigationSections drops repo-local navigation (e.g. "## See also"),
// which addresses sibling files meaningless outside the source repo.
func stripNavigationSections(body string) string {
	lines := strings.Split(body, "\n")
	var kept []string
	for i := 0; i < len(lines); {
		if headerLevel(lines[i]) != 0 {
			text := strings.TrimSpace(strings.TrimLeft(lines[i], "#"))
			if navigationHeadings[strings.ToLower(text)] {
				i = sectionEnd(lines, i)
				continue
			}
		}
		kept = append(kept, lines[i])
		i++
	}
	return strings.Trim(strings.Join(kept, "\n"), "\n")
}

func isGlobalTarget(target string) bool {
	return strings.HasPrefix(target, "/") || strings.HasPrefix(target, "#") ||
		strings.HasPrefix(target, "mailto:") || schemeRe.MatchString(target)
}

// absolutizeLinks rewrites repo-relative markdown link targets against the
// source's own directory so they stay followable from the composed location.
func absolutizeLinks(body, baseDir string) string {
	return linkRe.ReplaceAllStringFunc(body, func(match string) string {
		groups := linkRe.FindStringSubmatch(match)
		target, title := groups[1], groups[2]
		if isGlobalTarget(target) {
			return match
		}
		path, frag, hasFrag := strings.Cut(target, "#")
		resolvedBase := baseDir
		if r, err := filepath.EvalSymlinks(baseDir); err == nil {
			resolvedBase = r
		}
		absolute, err := filepath.Abs(filepath.Join(resolvedBase, path))
		if err != nil {
			return match
		}
		sep := ""
		if hasFrag {
			sep = "#"
		}
		return "](" + absolute + sep + frag + title + ")"
	})
}

// Compose concatenates source bodies under the banner, each fenced by its
// source path, then the appendix blocks bound to role. See docs/cascade.md.
func Compose(
	sources []string,
	overrides map[string]string,
	appendix []AppendixBlock,
	role string,
) (string, error) {
	body, tail, err := ComposeParts(sources, overrides, appendix, role)
	if err != nil {
		return "", err
	}
	if tail == "" {
		return body, nil
	}
	return body + "\n" + tail, nil
}

// ComposeParts returns the source body and the role's appendix separately,
// because a bundle rewrites the body. See docs/cascade.md. agent-compose#6987.
func ComposeParts(
	sources []string,
	overrides map[string]string,
	appendix []AppendixBlock,
	role string,
) (string, string, error) {
	parts := []string{Banner}
	for _, src := range sources {
		_, body, err := parseSource(src)
		if err != nil {
			return "", "", err
		}
		body = strings.Trim(body, "\n")
		fence := fmt.Sprintf("<!-- source: %s -->", src)
		if override := overrides[src]; override != "" {
			_, overrideBody, err := parseSource(override)
			if err != nil {
				return "", "", err
			}
			body, err = applyOverrides(body, overrideBody)
			if err != nil {
				return "", "", err
			}
			fence = fmt.Sprintf("<!-- source: %s (override: %s) -->", src, filepath.Base(override))
		}
		body = stripNavigationSections(body)
		body = absolutizeLinks(body, filepath.Dir(src))
		parts = append(parts, fence+"\n"+body)
	}
	var blocks []string
	for _, block := range appendix {
		if !appendixBinds(block, role) {
			continue
		}
		blocks = append(blocks, block.Fence+"\n"+block.Body)
	}
	composed := strings.Join(parts, "\n\n") + "\n"
	if len(blocks) == 0 {
		return composed, "", nil
	}
	return composed, strings.Join(blocks, "\n\n") + "\n", nil
}

// appendixBinds reports whether a block composes for role; an empty role
// drops every scoped block, which is what the host load point wants.
func appendixBinds(block AppendixBlock, role string) bool {
	if len(block.Roles) == 0 {
		return true
	}
	if role == "" {
		return false
	}
	for _, candidate := range block.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

type plan struct {
	slices    map[string][]string
	overrides map[string]map[string]string
	outputs   map[string]string
	appendix  []AppendixBlock
	errors    []string
}

// planOutputs selects each harness slice and decides shared versus divergent
// outputs: differing sources or overrides split COMPOSED per harness.
func planOutputs(sources []string, loadPoints map[string]string, composedPath string) plan {
	p := plan{
		slices:    map[string][]string{},
		overrides: map[string]map[string]string{},
		outputs:   map[string]string{},
	}
	signatures := map[string]bool{}
	for harness := range loadPoints {
		selected := selectByHarness(sources, harness)
		p.slices[harness] = selected
		overrideMap := map[string]string{}
		var overridePairs []string
		for _, src := range selected {
			if override := discoverOverride(src, harness); override != "" {
				overrideMap[src] = override
				overridePairs = append(overridePairs, src+"|"+override)
			}
		}
		p.overrides[harness] = overrideMap
		if len(selected) == 0 {
			p.errors = append(p.errors, fmt.Sprintf("no sources matched harness %q", harness))
		}
		sort.Strings(overridePairs)
		signatures[strings.Join(selected, "\x00")+"\x01"+strings.Join(overridePairs, "\x00")] = true
	}
	for harness := range loadPoints {
		if len(signatures) <= 1 {
			p.outputs[harness] = composedPath
		} else {
			p.outputs[harness] = harnessOutputPath(composedPath, harness)
		}
	}
	return p
}

func harnessOutputPath(composedPath, harness string) string {
	ext := filepath.Ext(composedPath)
	stem := strings.TrimSuffix(filepath.Base(composedPath), ext)
	return filepath.Join(filepath.Dir(composedPath), stem+"."+harness+ext)
}

// staleGeneratedOutputs finds obsolete composer-owned files beside
// composedPath; only banner-carrying files qualify, never user files.
func staleGeneratedOutputs(composedPath string, active map[string]bool) []string {
	ext := filepath.Ext(composedPath)
	stem := strings.TrimSuffix(filepath.Base(composedPath), ext)
	candidates := []string{composedPath}
	pattern := filepath.Join(filepath.Dir(composedPath), stem+".*"+ext)
	if globbed, err := filepath.Glob(pattern); err == nil {
		candidates = append(candidates, globbed...)
	}
	stale := map[string]bool{}
	for _, path := range candidates {
		if active[path] {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		first, _, _ := strings.Cut(string(raw), "\n")
		if first == Banner {
			stale[path] = true
		}
	}
	out := make([]string, 0, len(stale))
	for path := range stale {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

// repoForSource maps a source to the projects-root repo backing it; sources
// outside the projects tree back no mountable repo.
func repoForSource(source, projects string) string {
	absSource, err := canonicalPath(source)
	if err != nil {
		return ""
	}
	absProjects, err := canonicalPath(projects)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absProjects, absSource)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 3 {
		return ""
	}
	return filepath.Join(absProjects, parts[0], parts[1])
}

func canonicalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved, nil
	}
	return absolute, nil
}

type trustedRoleGraph struct {
	root     string
	relative string
	source   *schema.Source
	input    repositoryplan.Input
}

func resolveProviderPath(projects, logical string) (string, error) {
	resolved, err := canonicalPath(filepath.Join(projects, filepath.FromSlash(logical)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(projects, resolved)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolved path %s is outside projects_root %s", resolved, projects)
	}
	return resolved, nil
}

func rejectProviderCycles(graphs []trustedRoleGraph, definitions map[string]map[string]string) error {
	graphRoots := map[string]trustedRoleGraph{}
	for _, graph := range graphs {
		graphRoots[graph.root] = graph
	}
	adjacency := map[string][]string{}
	for _, graph := range graphs {
		seen := map[string]bool{}
		for _, uses := range graph.source.RoleProviders {
			for _, use := range uses {
				target := definitions[graph.root][use.Provider]
				if _, isGraph := graphRoots[target]; isGraph && !seen[target] {
					adjacency[graph.root] = append(adjacency[graph.root], target)
					seen[target] = true
				}
			}
		}
		sort.Strings(adjacency[graph.root])
	}
	state := map[string]int{}
	var visit func(string, []string) error
	visit = func(root string, stack []string) error {
		if state[root] == 1 {
			cycle := append(stack, graphRoots[root].relative)
			return fmt.Errorf("role provider cycle: %s", strings.Join(cycle, " -> "))
		}
		if state[root] == 2 {
			return nil
		}
		state[root] = 1
		for _, target := range adjacency[root] {
			if err := visit(target, append(stack, graphRoots[root].relative)); err != nil {
				return err
			}
		}
		state[root] = 2
		return nil
	}
	for _, graph := range graphs {
		if err := visit(graph.root, nil); err != nil {
			return err
		}
	}
	return nil
}

func symlinkUpToDate(dst, target string) bool {
	link, err := os.Readlink(dst)
	return err == nil && link == target
}

// installSymlink points dst at target, backing up a pre-existing real file
// to dst.bak. Reapply recreates an already-correct link.
func installSymlink(dst, target string, reapply bool) (string, error) {
	if !reapply && symlinkUpToDate(dst, target) {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	note := ""
	if info, err := os.Lstat(dst); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(dst); err != nil {
				return "", err
			}
		} else {
			backup := dst + ".bak"
			if err := os.Rename(dst, backup); err != nil {
				return "", err
			}
			note = fmt.Sprintf(" (backed up prior file to %s)", backup)
		}
	}
	if err := os.Symlink(target, dst); err != nil {
		return "", err
	}
	return fmt.Sprintf("linked  %s -> %s%s", dst, target, note), nil
}
