package overlay

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

func TestBuildProjectsOneCanonicalMember(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "platform-eng", "codex", "acting")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Format != Format || doc.SchemaVersion != SchemaVersion ||
		doc.Person != p.Name || doc.Role != "platform-eng" ||
		doc.Seat.Harness != "codex" || doc.Seat.Name == "" ||
		doc.Expression != "acting" || doc.FavoriteColor == "" {
		t.Fatalf("overlay identity is incomplete: %+v", doc)
	}
	if len(doc.Personalities) != len(p.Roles["platform-eng"].Personalities) {
		t.Fatalf("overlay personalities = %d", len(doc.Personalities))
	}
	if doc.Stance == "" {
		t.Fatalf("overlay carries no role stance: %+v", doc)
	}
	for _, personality := range doc.Personalities {
		if personality.Emblem.Name() == "" || personality.Motif == "" ||
			personality.Geometry == "" || personality.Body.Archetype == "" ||
			personality.Body.Attachment == "" {
			t.Fatalf("overlay personality is incomplete: %+v", personality)
		}
	}
}

// TestBuildComposesTheSeatAnnotation pins the identity string every terminal
// renderer shows verbatim, so a consumer never reassembles it from parts.
func TestBuildComposesTheSeatAnnotation(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "platform-eng", "claude", "acting")
	if err != nil {
		t.Fatal(err)
	}
	if doc.RoleDisplayName != "Platform Engineer" {
		t.Errorf("role display name = %q, want Platform Engineer", doc.RoleDisplayName)
	}
	want := person.SeatAnnotation(doc.Seat.Name, "Platform Engineer")
	if doc.Annotation != want || !strings.HasSuffix(doc.Annotation, "(Platform Engineer)") {
		t.Errorf("annotation = %q, want %q", doc.Annotation, want)
	}
}

// TestRenderTextCarriesTheSeatNotTheRoleLabel guards the card's split: it
// already prints the role on its own, so the seat stops at its own name.
func TestRenderTextCarriesTheSeatNotTheRoleLabel(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "prod-director", "claude", "acting")
	if err != nil {
		t.Fatal(err)
	}
	wide, err := RenderText(doc, 200)
	if err != nil {
		t.Fatal(err)
	}
	label := person.SeatLabel(doc.Seat.Name)
	if !strings.Contains(wide, label) {
		t.Errorf("card %q omits seat label %q", wide, label)
	}
	if strings.Contains(wide, "(Portfolio Director)") {
		t.Errorf("card %q repeats the role label", wide)
	}
}

func TestBuildRejectsUnknownSelectionFacts(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	for name, selection := range map[string][3]string{
		"role":       {"missing", "codex", "acting"},
		"seat":       {"platform-eng", "missing", "acting"},
		"expression": {"platform-eng", "codex", "invented"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Build(p, selection[0], selection[1], selection[2]); err == nil {
				t.Fatal("unknown overlay selection must fail")
			}
		})
	}
}

func TestRenderTextIsWidthResponsiveAndPlain(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "dev-advocate", "codex", "waiting-for-human")
	if err != nil {
		t.Fatal(err)
	}
	narrow, err := RenderText(doc, 40)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(narrow, "\n"), "\n") {
		if utf8.RuneCountInString(line) > 40 {
			t.Fatalf("narrow line exceeds width: %q", line)
		}
	}
	wide, err := RenderText(doc, 200)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(wide, "\n") != 1 || strings.Contains(wide, "\x1b") {
		t.Fatalf("wide overlay is not one plain line: %q", wide)
	}
}

func TestMarshalProducesVersionedJSON(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "sysadmin-senior", "claude", "blocked")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Document
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Format != Format || decoded.Expression != "blocked" {
		t.Fatalf("unexpected overlay JSON: %+v", decoded)
	}
}

// Pins #7362 and #7485: the lineage was only prose, and is now a field derived
// from the meld rather than read off the role.
func TestBuildDerivesTheCreaturePairFromTheMeld(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"platform-eng", "sysadmin-senior", "scientist"} {
		doc, err := Build(p, role, "claude", "acting")
		if err != nil {
			t.Fatal(err)
		}
		names := p.Roles[role].Personalities
		if len(names) != 2 {
			t.Fatalf("role %q carries %d personalities, want 2", role, len(names))
		}
		want := p.Personalities[names[0]].Species + "-" + p.Personalities[names[1]].Species
		if doc.Creature != want {
			t.Fatalf("role %q creature = %q, meld derives %q", role, doc.Creature, want)
		}
		// The regression this replaces: lineage read back out of prose.
		if !strings.Contains(doc.Identity, want) {
			t.Fatalf("role %q identity dropped the creature: %q", role, doc.Identity)
		}
		for index, name := range names {
			if got := doc.Personalities[index].Species; got != p.Personalities[name].Species {
				t.Fatalf("role %q personality %q species = %q, record carries %q",
					role, name, got, p.Personalities[name].Species)
			}
		}
	}
}

// The surface the consumer reads. Element's absence is asserted, not assumed:
// a stale key would keep aosx parsing it.
func TestMarshalCarriesTheDerivedCreatureAndNoElement(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(p, "sysadmin-senior", "claude", "acting")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	creature, present := decoded["creature"]
	if !present {
		t.Fatalf("overlay JSON has no \"creature\" key: %v", keysOf(decoded))
	}
	if text, _ := creature.(string); !strings.Contains(text, "-") {
		t.Fatalf("overlay JSON creature %q is not a pair", text)
	}
	if _, present := decoded["element"]; present {
		t.Fatalf("overlay JSON still carries an element key: %v", keysOf(decoded))
	}
	melds, _ := decoded["personalities"].([]any)
	if len(melds) == 0 {
		t.Fatalf("overlay JSON carries no personalities: %v", keysOf(decoded))
	}
	for index, entry := range melds {
		fields, _ := entry.(map[string]any)
		if text, _ := fields["species"].(string); text == "" {
			t.Fatalf("personality %d carries no species: %v", index, fields)
		}
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
