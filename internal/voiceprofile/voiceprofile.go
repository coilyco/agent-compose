// Package voiceprofile builds the linter profile a composed seat lints itself
// against. Merge rules and anchoring: docs/manifest-schema.md.
package voiceprofile

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

// Format names the artifact so a consumer can refuse an unrecognized one.
const Format = "agent-compose.voice-profile"

// Rule is one linter rule. The engine reads id, pattern, hint, scope, flags and
// blocking, and ignores anything else, so source rides along as provenance.
type Rule struct {
	ID       string   `json:"id"`
	Pattern  string   `json:"pattern"`
	Hint     string   `json:"hint"`
	Scope    string   `json:"scope,omitempty"`
	Flags    []string `json:"flags,omitempty"`
	Blocking bool     `json:"blocking,omitempty"`
	Source   string   `json:"source,omitempty"`
}

// Profile is the document the linter engine loads.
type Profile struct {
	Format string `json:"format"`
	Name   string `json:"name"`
	Rules  []Rule `json:"rules"`
}

// The engine refuses an empty rule list, so an empty build writes no file
// rather than one that fails to load at the point of use.
var errNoRules = fmt.Errorf("voice profile: no rules to write")

var whitespace = regexp.MustCompile(`\s+`)

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug turns an avoid term into a stable rule id fragment.
func slug(term string) string {
	return strings.Trim(notSlug.ReplaceAllString(strings.ToLower(term), "-"), "-")
}

func isWordChar(r rune) bool {
	return r == '_' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9')
}

// pattern anchors on word-character ends and lets internal spacing match a
// line break. Unanchored, "just" matches inside "adjust".
func pattern(term string) string {
	fields := strings.Fields(term)
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		quoted = append(quoted, regexp.QuoteMeta(field))
	}
	body := strings.Join(quoted, `\s+`)
	runes := []rune(whitespace.ReplaceAllString(term, " "))
	if isWordChar(runes[0]) {
		body = `\b` + body
	}
	if isWordChar(runes[len(runes)-1]) {
		body += `\b`
	}
	return body
}

// Generated turns melded avoid banks into rules, role first. The hint names the
// bank rather than advising: the wording belongs to the seat that owns it.
func Generated(banks []person.VoiceBank) []Rule {
	seen := map[string]bool{}
	rules := []Rule{}
	for _, bank := range banks {
		if bank.Voice == nil {
			continue
		}
		for _, term := range bank.Voice.Avoid {
			term = strings.TrimSpace(term)
			id := slug(term)
			if id == "" || seen[id] {
				continue
			}
			body := pattern(term)
			if body == "" {
				continue
			}
			seen[id] = true
			rules = append(rules, Rule{
				ID:      "avoid-" + id,
				Pattern: body,
				Hint:    "on the " + bank.Label + " avoid bank",
				Flags:   []string{"i"},
				Source:  "voice:" + bank.Label,
			})
		}
	}
	return rules
}

// Carried adopts a selected skill's hand-written rules, tagged with their
// origin. Build lets them win a collision against a generated rule.
func Carried(skillID string, raw []byte) ([]Rule, error) {
	var doc struct {
		Rules []Rule `json:"rules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("voice profile %s: %w", skillID, err)
	}
	rules := make([]Rule, 0, len(doc.Rules))
	for _, rule := range doc.Rules {
		if rule.ID == "" || rule.Pattern == "" || rule.Hint == "" {
			return nil, fmt.Errorf("voice profile %s: rule %q needs id, pattern and hint", skillID, rule.ID)
		}
		rule.Source = "skill:" + skillID
		rules = append(rules, rule)
	}
	return rules, nil
}

// Looks reports whether a shipped file is a linter profile at all, so discovery
// reads the document rather than a filename convention.
func Looks(raw []byte) bool {
	var doc struct {
		Rules []struct {
			ID      string `json:"id"`
			Pattern string `json:"pattern"`
			Hint    string `json:"hint"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc.Rules) == 0 {
		return false
	}
	for _, rule := range doc.Rules {
		if rule.ID == "" || rule.Pattern == "" || rule.Hint == "" {
			return false
		}
	}
	return true
}

// Build merges carried and generated rules into one document. Carried rules win
// a collision on id and sort first.
func Build(name string, carried, generated []Rule) ([]byte, error) {
	claimed := map[string]bool{}
	rules := make([]Rule, 0, len(carried)+len(generated))
	for _, rule := range carried {
		if claimed[rule.ID] {
			continue
		}
		claimed[rule.ID] = true
		rules = append(rules, rule)
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	kept := make([]Rule, 0, len(generated))
	for _, rule := range generated {
		if claimed[rule.ID] {
			continue
		}
		claimed[rule.ID] = true
		kept = append(kept, rule)
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].ID < kept[j].ID })
	rules = append(rules, kept...)
	if len(rules) == 0 {
		return nil, errNoRules
	}
	out, err := json.MarshalIndent(Profile{Format: Format, Name: name, Rules: rules}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal voice profile: %w", err)
	}
	return append(out, '\n'), nil
}

// Empty reports Build's sentinel for a seat with nothing to lint against,
// which is a skip rather than a failure.
func Empty(err error) bool { return err == errNoRules }
