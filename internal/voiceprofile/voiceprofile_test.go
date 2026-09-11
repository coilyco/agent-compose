package voiceprofile

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

func advocateBanks() []person.VoiceBank {
	return []person.VoiceBank{
		{Label: "Developer Advocate", Voice: &person.Voice{
			Avoid: []string{"simply", "just", "easy", "as you know", "reach out", "circle back", "obviously", "leverage"},
		}},
		{Label: "Warm", Voice: &person.Voice{Avoid: []string{"circle back", "delve into"}}},
	}
}

func ruleByID(t *testing.T, rules []Rule, id string) Rule {
	t.Helper()
	for _, rule := range rules {
		if rule.ID == id {
			return rule
		}
	}
	t.Fatalf("no rule %q in %d rules", id, len(rules))
	return Rule{}
}

func TestGeneratedCoversTheBankInMeldOrder(t *testing.T) {
	rules := Generated(advocateBanks())
	if len(rules) != 9 {
		t.Fatalf("8 role terms plus 1 unique personality term, got %d", len(rules))
	}
	first := ruleByID(t, rules, "avoid-circle-back")
	if first.Hint != "on the Developer Advocate avoid bank" {
		t.Fatalf("a term on both banks keeps the role bank, got %q", first.Hint)
	}
	ruleByID(t, rules, "avoid-delve-into")
}

// The over-flag that makes a generated rule worth skipping rather than reading.
func TestGeneratedAnchorsOnWordBoundaries(t *testing.T) {
	rules := Generated(advocateBanks())
	just := regexp.MustCompile(`(?i)` + ruleByID(t, rules, "avoid-just").Pattern)
	for _, clean := range []string{"adjust the value", "unjust", "justify it"} {
		if just.MatchString(clean) {
			t.Fatalf("avoid-just fired inside a longer word: %q", clean)
		}
	}
	if !just.MatchString("just run it") {
		t.Fatal("avoid-just missed the word itself")
	}
	easy := regexp.MustCompile(`(?i)` + ruleByID(t, rules, "avoid-easy").Pattern)
	if easy.MatchString("greasy") || easy.MatchString("measly") {
		t.Fatal("avoid-easy fired inside a longer word")
	}
}

// A phrase that wraps a line is the same phrase, and a linter that reads
// Markdown reads wrapped prose most of the time.
func TestGeneratedPhraseMatchesAcrossALineBreak(t *testing.T) {
	rules := Generated(advocateBanks())
	pattern := regexp.MustCompile(`(?i)` + ruleByID(t, rules, "avoid-circle-back").Pattern)
	if !pattern.MatchString("we should circle\nback on this") {
		t.Fatal("phrase did not match across a line break")
	}
}

func TestGeneratedSkipsEmptyAndDuplicateTerms(t *testing.T) {
	rules := Generated([]person.VoiceBank{
		{Label: "Role", Voice: &person.Voice{Avoid: []string{"  ", "", "leverage", "Leverage", "!!!"}}},
		{Label: "Nil", Voice: nil},
	})
	if len(rules) != 1 || rules[0].ID != "avoid-leverage" {
		t.Fatalf("expected one deduped rule, got %+v", rules)
	}
}

func TestCarriedRejectsAnIncompleteRule(t *testing.T) {
	_, err := Carried("kai-voice-guide-linter", []byte(`{"rules":[{"id":"em-dash","pattern":"—"}]}`))
	if err == nil || !strings.Contains(err.Error(), "id, pattern and hint") {
		t.Fatalf("expected a naming error, got %v", err)
	}
}

func TestLooksRefusesANonProfile(t *testing.T) {
	for _, raw := range []string{`{}`, `{"rules":[]}`, `not json`, `{"rules":[{"id":"x"}]}`} {
		if Looks([]byte(raw)) {
			t.Fatalf("accepted a non-profile: %s", raw)
		}
	}
	if !Looks([]byte(`{"rules":[{"id":"x","pattern":"y","hint":"z"}]}`)) {
		t.Fatal("refused a valid profile")
	}
}

// A hand-written pattern was written to say something a generated one cannot,
// so it wins the collision rather than being shadowed.
func TestBuildKeepsTheHandWrittenRuleOnACollision(t *testing.T) {
	carried, err := Carried("kai-voice-guide-linter", []byte(
		`{"rules":[{"id":"avoid-leverage","pattern":"\\bleverage\\b(?! ratio)","hint":"use a concrete verb"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Build("coilyco:advocate", carried, Generated(advocateBanks()))
	if err != nil {
		t.Fatal(err)
	}
	var profile Profile
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Format != Format {
		t.Fatalf("format is %q", profile.Format)
	}
	kept := ruleByID(t, profile.Rules, "avoid-leverage")
	if kept.Hint != "use a concrete verb" || kept.Source != "skill:kai-voice-guide-linter" {
		t.Fatalf("generated rule shadowed the hand-written one: %+v", kept)
	}
	if profile.Rules[0].ID != "avoid-leverage" {
		t.Fatalf("carried rules sort first, got %q", profile.Rules[0].ID)
	}
	if len(profile.Rules) != 9 {
		t.Fatalf("one carried plus eight surviving generated, got %d", len(profile.Rules))
	}
}

// Linting nothing and reporting success is what the engine refuses, so a seat
// with nothing to lint against writes no file at all.
func TestBuildReportsEmptyRatherThanWritingAnUnloadableProfile(t *testing.T) {
	_, err := Build("coilyco:none", nil, nil)
	if !Empty(err) {
		t.Fatalf("expected the empty sentinel, got %v", err)
	}
}

// blocking decides the engine's --block-only exit status, so a round trip that
// drops it turns a gating rule into a warning and the bundle reports success.
func TestCarriedAndBuildKeepBlocking(t *testing.T) {
	carried, err := Carried("kai-voice-guide-linter", []byte(
		`{"rules":[
			{"id":"em-dash","pattern":"—","hint":"replace with ' - '","blocking":true},
			{"id":"wordy","pattern":"\\bverily\\b","hint":"cut it"}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Build("coilyco:advocate", carried, nil)
	if err != nil {
		t.Fatal(err)
	}
	var profile Profile
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	if gate := ruleByID(t, profile.Rules, "em-dash"); !gate.Blocking {
		t.Fatalf("blocking was dropped on the round trip: %+v", gate)
	}
	if warn := ruleByID(t, profile.Rules, "wordy"); warn.Blocking {
		t.Fatalf("a rule with no blocking key became a gate: %+v", warn)
	}
	// omitempty keeps a warning's document identical to what it was before the
	// field existed, so adding it rewrites no shipped profile.
	if strings.Contains(string(raw), `"id": "wordy"`) && strings.Count(string(raw), `"blocking"`) != 1 {
		t.Fatalf("blocking is written for a non-gating rule:\n%s", raw)
	}
}
