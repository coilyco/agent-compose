// Package describe renders stored decision evidence for humans. It only
// reads trace.json and manifest.json; reasons are never reconstructed.
package describe

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/bundle"
	"github.com/coilyco/agent-compose/v2/internal/color"
	"github.com/coilyco/agent-compose/v2/internal/person"
	"github.com/coilyco/agent-compose/v2/internal/resolver"
)

type Options struct {
	All       bool
	Color     bool
	TrueColor bool
}

// collapseAt is the excluded-skill count where describe folds repeats into
// one line; --all expands them.
const collapseAt = 4

func Bundle(dir string, opts Options) (string, error) {
	manifest, trace, err := load(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "bundle %s // %s/%s // %s // %d body bytes%s\n",
		filepath.Base(dir), manifest.Role, strings.Join(manifest.Personalities, "+"),
		manifest.Delivery.Mode, manifest.Delivery.BodyBytes, favoriteSuffix(manifest, opts))

	sections := []struct {
		title string
		kinds []string
	}{
		{"profile", []string{"profile"}},
		{"repositories", []string{"repository"}},
		{"providers", []string{"source"}},
		{"context budget", nil},
		{"selection", []string{"instruction", "skill", "selector"}},
		{"delivery", []string{"delivery"}},
	}
	for _, section := range sections {
		var lines []string
		var excludedSkills []resolver.Decision
		if section.title == "providers" && len(trace.Providers) > 0 {
			for _, provider := range trace.Providers {
				lines = append(lines, renderProviderLine(provider, opts))
			}
		} else if section.title == "context budget" {
			for _, provider := range trace.Providers {
				lines = append(lines, renderBudgetLine(provider))
			}
		} else if section.title == "providers" {
			traced := map[string]bool{}
			for _, decision := range trace.Decisions {
				if decision.Kind == "source" {
					traced[strings.TrimPrefix(decision.Subject, "source:")] = true
				}
			}
			for _, id := range manifest.Sources {
				if traced[id] {
					continue
				}
				lines = append(lines, fmt.Sprintf("  %s %-36s %-14s %s",
					symbol(resolver.OutcomeSelected, opts), id, "", "composed"))
			}
		}
		if section.title != "context budget" && !(section.title == "providers" && len(trace.Providers) > 0) {
			for _, d := range trace.Decisions {
				if !contains(section.kinds, d.Kind) {
					continue
				}
				if section.title == "selection" && d.Kind == "skill" && d.Outcome == resolver.OutcomeExcluded && !opts.All {
					excludedSkills = append(excludedSkills, d)
					continue
				}
				lines = append(lines, renderLine(d, opts))
			}
		}
		lines = append(lines, collapse(excludedSkills, opts)...)
		if len(lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n%s\n", section.title)
		for _, line := range lines {
			fmt.Fprintln(&b, line)
		}
	}

	fmt.Fprintf(&b, "\nmachine-readable trace: trace.json\n")
	return b.String(), nil
}

// Why follows one item through consideration to its outcome, adding what
// would have selected an excluded personality skill.
func Why(dir, subject string, opts Options) (string, error) {
	manifest, trace, err := load(dir)
	if err != nil {
		return "", err
	}
	var matched []resolver.Decision
	for _, d := range trace.Decisions {
		if d.Subject == subject || strings.TrimPrefix(d.Subject, d.Kind+":") == subject {
			matched = append(matched, d)
		}
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no decision mentions %q; try `agent-compose describe %s` for the full list", subject, dir)
	}
	var b strings.Builder
	for _, d := range matched {
		fmt.Fprintf(&b, "%s", subjectLabel(d))
		if d.Source != "" {
			fmt.Fprintf(&b, " (%s)", d.Source)
		}
		fmt.Fprintln(&b)
		if d.Source != "" {
			fmt.Fprintf(&b, "  considered: declared by %s\n", d.Source)
		}
		fmt.Fprintf(&b, "  outcome: %s\n", d.Outcome)
		fmt.Fprintf(&b, "  reason: %s\n", d.Reason)
		if provider, ok := providerBySource(trace, d.Source); ok {
			fmt.Fprintf(&b, "  provider: %s/%s\n", provider.Category, provider.Scope)
			if provider.SelectorReason != "" {
				fmt.Fprintf(&b, "  selector: %s\n", provider.SelectorReason)
			}
			fmt.Fprintf(
				&b,
				"  context: %d skills, %d bytes, approximately %d tokens\n",
				provider.Skills,
				provider.ContextBytes,
				provider.ApproximateTokens,
			)
		}
		if hint := selectionHint(d, manifest); hint != "" {
			fmt.Fprintf(&b, "  %s\n", hint)
		}
	}
	return b.String(), nil
}

// Diff reports semantic decision changes between two bundles, keyed by
// subject rather than by file bytes.
func Diff(leftDir, rightDir string) (string, error) {
	leftManifest, leftTrace, err := load(leftDir)
	if err != nil {
		return "", err
	}
	rightManifest, rightTrace, err := load(rightDir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s/%s (%s) → %s/%s (%s)\n",
		leftManifest.Role, strings.Join(leftManifest.Personalities, "+"), leftManifest.Delivery.Mode,
		rightManifest.Role, strings.Join(rightManifest.Personalities, "+"), rightManifest.Delivery.Mode)

	left, right := bySubject(leftTrace), bySubject(rightTrace)
	var subjects []string
	seen := map[string]bool{}
	for _, key := range append(keys(left), keys(right)...) {
		if !seen[key] {
			seen[key] = true
			subjects = append(subjects, key)
		}
	}
	sort.Strings(subjects)

	unchanged, changed := 0, 0
	for _, subject := range subjects {
		l, inLeft := left[subject]
		r, inRight := right[subject]
		switch {
		case inLeft && inRight && l.Outcome == r.Outcome && l.Reason == r.Reason:
			unchanged++
		case inLeft && inRight && l.Outcome == r.Outcome:
			changed++
			fmt.Fprintf(&b, "  ~ %-36s still %s, reason changed\n", subjectLabel(l), l.Outcome)
		case inLeft && inRight:
			changed++
			fmt.Fprintf(&b, "  ~ %-36s %s → %s\n", subjectLabel(l), l.Outcome, r.Outcome)
		case inLeft:
			changed++
			fmt.Fprintf(&b, "  - %-36s (%s)\n", subjectLabel(l), l.Outcome)
		default:
			changed++
			fmt.Fprintf(&b, "  + %-36s (%s)\n", subjectLabel(r), r.Outcome)
		}
	}
	if changed == 0 {
		fmt.Fprintln(&b, "  no semantic differences")
	}
	fmt.Fprintf(&b, "  %d decisions unchanged\n", unchanged)
	leftContent, err := contentDigests(leftDir, leftManifest)
	if err != nil {
		return "", err
	}
	rightContent, err := contentDigests(rightDir, rightManifest)
	if err != nil {
		return "", err
	}
	contentPaths := append(keysString(leftContent), keysString(rightContent)...)
	seenContent := map[string]bool{}
	var contentChanged []string
	for _, path := range contentPaths {
		if seenContent[path] {
			continue
		}
		seenContent[path] = true
		if leftContent[path] != rightContent[path] {
			contentChanged = append(contentChanged, path)
		}
	}
	sort.Strings(contentChanged)
	if len(contentChanged) > 0 {
		fmt.Fprintln(&b, "  content changes:")
		for _, id := range contentChanged {
			switch {
			case leftContent[id] == "":
				fmt.Fprintf(&b, "  + %s %s\n", id, shortDigest(rightContent[id]))
			case rightContent[id] == "":
				fmt.Fprintf(&b, "  - %s %s\n", id, shortDigest(leftContent[id]))
			default:
				fmt.Fprintf(
					&b,
					"  ~ %s %s → %s\n",
					id,
					shortDigest(leftContent[id]),
					shortDigest(rightContent[id]),
				)
			}
		}
	}
	return b.String(), nil
}

func contentDigests(root string, manifest *bundle.Manifest) (map[string]string, error) {
	digests := map[string]string{}
	for _, entry := range manifest.Content {
		digests[entry.ID] = entry.Digest
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "manifest.json" || rel == "trace.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		digests[filepath.ToSlash(rel)] = fmt.Sprintf("sha256:%x", digest)
		return nil
	})
	return digests, err
}

func shortDigest(digest string) string {
	if len(digest) <= 19 {
		return digest
	}
	return digest[:19]
}

func keysString(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func load(dir string) (*bundle.Manifest, *bundle.Trace, error) {
	manifest, err := bundle.ReadManifest(dir)
	if err != nil {
		return nil, nil, err
	}
	trace, err := bundle.ReadTrace(dir)
	if err != nil {
		return nil, nil, err
	}
	return manifest, trace, nil
}

func renderLine(d resolver.Decision, opts Options) string {
	label := subjectLabel(d)
	source := ""
	if d.Source != "" {
		source = "(" + d.Source + ")"
	}
	return fmt.Sprintf("  %s %-36s %-14s %s", symbol(d.Outcome, opts), label, source, d.Reason)
}

func renderProviderLine(provider resolver.ProviderReport, opts Options) string {
	reason := provider.Reason
	if provider.SelectorReason != "" {
		reason += ". " + provider.SelectorReason
	}
	return fmt.Sprintf(
		"  %s %-36s %-22s %s",
		symbol(provider.Outcome, opts),
		provider.Source,
		"("+provider.Category+"/"+provider.Scope+")",
		reason,
	)
}

func renderBudgetLine(provider resolver.ProviderReport) string {
	selector := ""
	if provider.SelectorReason != "" {
		selector = " // selected slice"
	}
	return fmt.Sprintf(
		"  %-36s %-22s %d skills // %d bytes // ~%d tokens%s",
		provider.Source,
		"("+provider.Category+"/"+provider.Scope+")",
		provider.Skills,
		provider.ContextBytes,
		provider.ApproximateTokens,
		selector,
	)
}

func providerBySource(trace *bundle.Trace, source string) (resolver.ProviderReport, bool) {
	for _, provider := range trace.Providers {
		if provider.Source == source {
			return provider, true
		}
	}
	return resolver.ProviderReport{}, false
}

func collapse(excluded []resolver.Decision, opts Options) []string {
	if len(excluded) == 0 {
		return nil
	}
	if len(excluded) < collapseAt {
		lines := make([]string, 0, len(excluded))
		for _, d := range excluded {
			lines = append(lines, renderLine(d, opts))
		}
		return lines
	}
	return []string{fmt.Sprintf("  %s %d skills excluded — %s (--all to expand)",
		symbol(resolver.OutcomeExcluded, opts), len(excluded), excluded[0].Reason)}
}

func selectionHint(d resolver.Decision, manifest *bundle.Manifest) string {
	if d.Kind != "skill" || d.Outcome != resolver.OutcomeExcluded {
		return ""
	}
	p, err := person.Load()
	if err != nil {
		return ""
	}
	skillID := strings.TrimPrefix(d.Subject, "skill:")
	for personality, bound := range p.Personalities {
		if bound.Skill != skillID {
			continue
		}
		return fmt.Sprintf("personality %q binds it, but role %q does not activate that personality", personality, manifest.Role)
	}
	return "no personality binds this skill"
}

// favoriteSuffix appends the role's melded favorite color: hex always, a tinted
// swatch only where a terminal will render it.
func favoriteSuffix(manifest *bundle.Manifest, opts Options) string {
	if manifest.Color == "" {
		return ""
	}
	if !opts.Color {
		return " // melded " + manifest.Color
	}
	return " // melded " + manifest.Color + " " + color.ANSI(manifest.Color, "■", opts.TrueColor)
}

func symbol(outcome string, opts Options) string {
	symbols := map[string][2]string{
		resolver.OutcomeSelected:  {"✓", "\x1b[32m"},
		resolver.OutcomeExcluded:  {"✗", "\x1b[31m"},
		resolver.OutcomeShadowed:  {"~", "\x1b[33m"},
		resolver.OutcomeDelivered: {"→", "\x1b[36m"},
	}
	entry, ok := symbols[outcome]
	if !ok {
		entry = [2]string{"?", ""}
	}
	if !opts.Color {
		return entry[0]
	}
	return entry[1] + entry[0] + "\x1b[0m"
}

func subjectLabel(d resolver.Decision) string {
	return strings.Replace(d.Subject, ":", " ", 1)
}

func bySubject(tr *bundle.Trace) map[string]resolver.Decision {
	out := map[string]resolver.Decision{}
	for _, d := range tr.Decisions {
		out[d.Subject] = d
	}
	return out
}

func keys(m map[string]resolver.Decision) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(list []string, item string) bool {
	for _, entry := range list {
		if entry == item {
			return true
		}
	}
	return false
}
