package person

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
)

func TestRenderRoleMetadataIncludesCompleteSelectedFacts(t *testing.T) {
	p := &Person{
		Name: "fixture",
		Roles: map[string]Role{
			"builder": {
				Purpose:       "Build the fixture.",
				Personalities: []string{"bright", "steady"},
				Seats: []Seat{
					{Harness: "alpha", Name: "bright builder"},
					{Harness: "beta", Name: "steady builder"},
				},
			},
		},
		Personalities: map[string]Personality{
			"bright": {
				Skill:    "personality-bright",
				Color:    "#d98e48",
				Motif:    "sunbeam",
				Geometry: "open-rays",
				Emblem:   Emblem{Names: []string{"lantern", "beacon"}, Emoji: "🏮"},
				Body:     Body{Archetype: "small upright body", Attachment: "the flame sits in its chest"},
			},
			"steady": {
				Skill:    "personality-steady",
				Color:    "#5fa87a",
				Motif:    "stone",
				Geometry: "stacked-rounds",
				Emblem:   Emblem{Names: []string{"anchor", "cairn"}, Emoji: "⚓"},
				Body:     Body{Archetype: "low and rounded, stacked stone", Attachment: "a cairn at its foot"},
			},
			"inactive": {
				Skill: "personality-inactive",
				Color: "#7d9fd3",
			},
		},
	}

	got, err := p.RenderRoleMetadata("builder", "#90a66a")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Active role metadata",
		"* Provider: `person:fixture`",
		"* Role: `builder`",
		"* Purpose: Build the fixture.",
		"`bright`: skill `personality-bright`, favorite color `#d98e48`",
		"emblem `🏮` `lantern / beacon`, motif `sunbeam`",
		"geometry `open-rays`",
		"`steady`: skill `personality-steady`, favorite color `#5fa87a`",
		"* Melded favorite color: `#90a66a`",
		"`alpha`: `bright builder`",
		"`beta`: `steady builder`",
		"* Renderer expressions: `available`, `listening`, `thinking`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("metadata missing %q:\n%s", want, got)
		}
	}
	for _, excluded := range []string{
		"personality-inactive",
		"inspiration",
		"Credit",
	} {
		if strings.Contains(got, excluded) {
			t.Errorf("metadata contains excluded field %q:\n%s", excluded, got)
		}
	}
}

func TestRenderRoleMetadataRejectsIncompleteRelationships(t *testing.T) {
	p := &Person{
		Name: "fixture",
		Roles: map[string]Role{
			"builder": {
				Personalities: []string{"missing"},
			},
		},
	}
	if _, err := p.RenderRoleMetadata("unknown", "#90a66a"); err == nil {
		t.Fatal("unknown role must fail")
	}
	if _, err := p.RenderRoleMetadata("builder", ""); err == nil {
		t.Fatal("missing melded color must fail")
	}
	if _, err := p.RenderRoleMetadata("builder", "#90a66a"); err == nil {
		t.Fatal("missing personality must fail")
	}
}

func TestRenderRoleTranscriptIncludesCompleteSelectedMetadata(t *testing.T) {
	t.Parallel()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.RenderRoleTranscript("platform-eng", "#90a66a", RoleTranscriptOptions{Expanded: true})
	if err != nil {
		t.Fatal(err)
	}
	role := p.Roles["platform-eng"]
	personalityMetadata := strings.Index(got, "personality metadata")
	rendererExpressions := strings.Index(got, "renderer expressions:")
	roleMetadata := strings.Index(got, "role metadata")
	if personalityMetadata < 0 ||
		rendererExpressions < personalityMetadata ||
		roleMetadata < rendererExpressions {
		t.Fatalf("transcript does not end with role metadata:\n%s", got)
	}
	for _, want := range []string{
		"role metadata",
		"roster: core // provided by: roster:core",
		"role: platform-eng",
		"purpose: " + role.Purpose,
		"agent identity: " + role.Identity.Name,
		"personalities: " + strings.Join(role.Personalities, " // "),
		"melded color: #90a66a",
		"briefing:",
		"seats:",
		"personality metadata",
		"renderer expressions: " + strings.Join(ExpressionVocabulary(), " // "),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing %q:\n%s", want, got)
		}
	}
	for _, seat := range role.Seats {
		want := fmt.Sprintf("seat %s%s", seat.Selector(), seatRoutingSuffix(seat))
		if !strings.Contains(got, want) {
			t.Errorf("transcript missing seat %q:\n%s", want, got)
		}
	}
	for _, name := range role.Personalities {
		binding := p.Personalities[name]
		for _, want := range []string{
			binding.Emblem.Emoji + " " + strings.Join(binding.Emblem.Names, " / ") +
				" // " + binding.Motif + " // " + binding.Color,
			"// geometry: " + binding.Geometry,
			"// body: " + binding.Body.Archetype,
			"// emblem sits: " + binding.Body.Attachment,
			"// skill: " + binding.Skill,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("transcript missing personality field %q:\n%s", want, got)
			}
		}
	}
	for _, excluded := range []string{
		"additional linked metadata",
		"inspiration",
	} {
		if strings.Contains(got, excluded) {
			t.Errorf("transcript retained excluded key %q:\n%s", excluded, got)
		}
	}
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if strings.HasPrefix(line, " ") {
			t.Fatalf("transcript line is not flush-left: %q", line)
		}
	}
}

func TestRenderRoleTranscriptUsesCanonicalColors(t *testing.T) {
	t.Parallel()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	role := p.Roles["platform-eng"]
	favoriteColor := role.FavoriteColor
	trueColor, err := p.RenderRoleTranscript("platform-eng", favoriteColor, RoleTranscriptOptions{
		Color: true, TrueColor: true, Expanded: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{strings.TrimSuffix(color.ANSI(favoriteColor, "role metadata", true), "\x1b[0m")}
	for _, personalityName := range role.Personalities {
		wants = append(wants, strings.TrimSuffix(color.ANSI(
			p.Personalities[personalityName].Color,
			"personality: "+personalityName,
			true,
		), "\x1b[0m"))
	}
	for _, want := range wants {
		if !strings.Contains(trueColor, want) {
			t.Errorf("truecolor transcript missing %q:\n%q", want, trueColor)
		}
	}
	fallback, err := p.RenderRoleTranscript("platform-eng", favoriteColor, RoleTranscriptOptions{
		Color: true, Expanded: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fallback, "\x1b[38;5;") {
		t.Fatalf("fallback transcript omitted 256-color ANSI:\n%q", fallback)
	}
}

func TestEmbeddedRoleMetadataCarriesEverySeat(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		favorite := role.FavoriteColor
		metadata, err := p.RenderRoleMetadata(roleName, favorite)
		if err != nil {
			t.Fatal(err)
		}
		identity := fmt.Sprintf("* Agent identity: `%s`", role.Identity.Name)
		if strings.Count(metadata, identity) != 1 {
			t.Errorf("role %q metadata identity count = %d:\n%s", roleName, strings.Count(metadata, identity), metadata)
		}
		for _, seat := range role.Seats {
			want := fmt.Sprintf("`%s`%s", seat.Selector(), seatRoutingSuffix(seat))
			if !strings.Contains(metadata, want) {
				t.Errorf("role %q metadata missing seat %q:\n%s", roleName, want, metadata)
			}
		}
	}
}

// Texture is the part only the summary shows, and the briefing restates a skill
// the agent loads anyway, so the long-form material rides --explain. See #323.
func TestRenderRoleTranscriptKeepsTheDefaultTerse(t *testing.T) {
	t.Parallel()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	terse, err := p.RenderRoleTranscript("platform-eng", "#90a66a", RoleTranscriptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	role := p.Roles["platform-eng"]
	briefingOpener := strings.SplitN(strings.TrimSpace(role.Briefing), "\n", 2)[0]
	for _, unwanted := range []string{
		"briefing:",
		briefingOpener,
		"renderer expressions:",
	} {
		if strings.Contains(terse, unwanted) {
			t.Errorf("terse transcript still carries %q:\n%s", unwanted, terse)
		}
	}
	first := p.Personalities[role.Personalities[0]]
	for _, want := range []string{
		"role metadata",
		"role: platform-eng",
		"personalities: " + strings.Join(role.Personalities, " // "),
		"personality metadata",
		first.Emblem.Name(),
		first.Motif,
		first.Skill,
	} {
		if !strings.Contains(terse, want) {
			t.Errorf("terse transcript missing %q:\n%s", want, terse)
		}
	}
	expanded, err := p.RenderRoleTranscript("platform-eng", "#90a66a", RoleTranscriptOptions{Expanded: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) <= len(terse) {
		t.Errorf("expanded transcript (%d) is not longer than terse (%d)", len(expanded), len(terse))
	}
	if !strings.Contains(expanded, briefingOpener) {
		t.Errorf("expanded transcript dropped the briefing:\n%s", expanded)
	}
}

// The load list read as optional because it was the only section with no
// content, so it now carries what each summary leaves out. See #303.
func TestIdentityCardNamesWhatTheLoadListOmits(t *testing.T) {
	t.Parallel()
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	role := p.Roles["platform-eng"]
	got, err := p.RenderRoleIdentityCard("platform-eng", "#90a66a", role.Boundaries)
	if err != nil {
		t.Fatal(err)
	}
	doctrine := got[strings.Index(got, "## Active doctrine"):]
	if !strings.Contains(doctrine, "summary is not the operative text") {
		t.Errorf("doctrine section does not say the summaries are not the doctrine:\n%s", doctrine)
	}
	named := append([]string{p.RoleSkillID("platform-eng")}, p.boundarySkillIDs(role.Boundaries)...)
	for _, name := range role.Personalities {
		named = append(named, p.Personalities[name].Skill)
	}
	sizes, total := p.skillBodySizes("platform-eng", named)
	if len(sizes) != len(named) {
		t.Fatalf("sized %d of %d named skills: %v", len(sizes), len(named), sizes)
	}
	if !strings.Contains(doctrine, thousands(total)+" bytes of doctrine") {
		t.Errorf("doctrine section omits the total %d:\n%s", total, doctrine)
	}
	for _, skill := range named {
		want := "`" + skill + "` - " + thousands(sizes[skill]) + " bytes"
		if !strings.Contains(doctrine, want) {
			t.Errorf("doctrine section missing %q:\n%s", want, doctrine)
		}
	}
}
