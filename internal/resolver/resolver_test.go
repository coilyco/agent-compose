package resolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func testPerson() *person.Person {
	p := &person.Person{
		Name: "kai",
		Roles: map[string]person.Role{
			"platform": {
				Purpose:       "Build.",
				Briefing:      "You are an engineer.\n\nFinish the complete repository workflow.",
				Personalities: []string{"tenacious", "grounded"},
			},
			"writer": {
				Purpose:       "Write.",
				Briefing:      "You are a writer.\n\nDeliver the finished text.",
				Personalities: []string{"grounded"},
			},
		},
		Personalities: map[string]person.Personality{
			"tenacious": {Skill: "personality-tenacious", Color: "#d98e48"},
			"grounded":  {Skill: "personality-grounded", Color: "#5fa87a"},
		},
		Raw: []byte("person \"kai\"\n"),
	}
	if err := p.ResolveFavoriteColors(); err != nil {
		panic(err)
	}
	return p
}

func testRequest(delivery string) *schema.Request {
	return &schema.Request{
		Role: "platform", Delivery: delivery,
	}
}

// makeSource lays out a synthetic source on disk: one instruction plus the
// named skills, each with a SKILL.md carrying the given body.
func makeSource(t *testing.T, id string, skillBodies map[string]string) *schema.Source {
	t.Helper()
	root := t.TempDir()
	src := &schema.Source{ID: id, Root: root, Declaration: []byte(id)}
	instr := filepath.Join(root, "foundation.md")
	if err := os.WriteFile(instr, []byte("# Foundation from "+id+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src.Instructions = []schema.ContentRef{{ID: "foundation-" + id, Path: "foundation.md"}}
	for name, body := range skillBodies {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		src.Skills = append(src.Skills, schema.ContentRef{ID: name, Path: name})
	}
	return src
}

func TestResolveSelectsPersonalityAndOrdinarySkills(t *testing.T) {
	src := makeSource(t, "aos", map[string]string{
		"fixture-review": "# Review\n",
	})
	res, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{src}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skills) != 4 ||
		res.Skills[0].ID != "role-platform" ||
		res.Skills[1].ID != "personality-tenacious" ||
		res.Skills[2].ID != "personality-grounded" ||
		res.Skills[3].ID != "fixture-review" {
		t.Fatalf("expected role skill, personalities, then ordinary skills, got %+v", res.Skills)
	}
	if res.FavoriteColor == "" {
		t.Fatal("expected a melded favorite color")
	}
	if res.RoleBriefing != testPerson().Roles["platform"].Briefing {
		t.Fatalf("role briefing = %q", res.RoleBriefing)
	}
	var selected, briefingSelected bool
	for _, d := range res.Decisions {
		if d.Subject == "skill:fixture-review" && d.Outcome == OutcomeSelected {
			selected = true
		}
		if d.Subject == "skill:role-platform" &&
			d.Source == "person:kai" &&
			d.Outcome == OutcomeSelected {
			briefingSelected = true
		}
	}
	if !selected {
		t.Fatalf("expected fixture-review selected, decisions: %+v", res.Decisions)
	}
	if !briefingSelected {
		t.Fatalf("expected role briefing selected, decisions: %+v", res.Decisions)
	}
}

func TestResolveIncludesSkillsRegardlessOfLegacyContextMetadata(t *testing.T) {
	src := makeSource(t, "aos", map[string]string{
		"coding-python": `---
name: coding-python
description: Core Python guidance.
low-context: required
---

# Python
`,
		"architecture": `---
name: architecture
description: High-end architecture guidance.
low-context: optional
---

# Architecture
`,
	})
	res, err := Resolve(
		testRequest(schema.DeliveryNativeSkills),
		testPerson(),
		[]*schema.Source{src},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{}
	for _, skill := range res.Skills {
		selected[skill.ID] = true
	}
	if !selected["coding-python"] || !selected["architecture"] {
		t.Fatalf("complete context selection = %v", selected)
	}
}

func TestResolveRejectsUnsupportedRoleModelTier(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	oss := &schema.Request{
		Role:      "director",
		Delivery:  schema.DeliveryNativeSkills,
		ModelTier: schema.ModelTierOSS,
	}
	if _, err := Resolve(oss, p, nil, nil); err == nil ||
		err.Error() != `role "director" does not support model tier "oss"` {
		t.Fatalf("OSS Executive Strategist error = %v", err)
	}

	frontier := *oss
	frontier.ModelTier = schema.ModelTierFrontier
	if _, err := Resolve(&frontier, p, nil, nil); err != nil {
		t.Fatalf("frontier Executive Strategist failed: %v", err)
	}
}

func TestResolveComposesOnlyTheActiveRolesSkills(t *testing.T) {
	src := makeSource(t, "aos", nil)
	for _, name := range []string{"coding-shape-cli", "design-system"} {
		dir := filepath.Join(src.Root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "COMPOSED.md"), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src.RoleSkills = map[string][]schema.ContentRef{
		"platform": {{
			ID: "coding-shape-cli", Path: "coding-shape-cli", EntryPoint: "COMPOSED.md",
		}},
		"writer": {{
			ID: "design-system", Path: "design-system", EntryPoint: "COMPOSED.md",
		}},
	}

	res, err := Resolve(
		testRequest(schema.DeliveryNativeSkills),
		testPerson(),
		[]*schema.Source{src},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, skill := range res.Skills {
		if skill.ID == "design-system" {
			t.Fatal("an inactive role's composed skill entered the resolution")
		}
		if skill.ID == "coding-shape-cli" {
			found = true
			if skill.EntryPoint != "COMPOSED.md" {
				t.Fatalf("composed entry point = %q", skill.EntryPoint)
			}
		}
	}
	if !found {
		t.Fatalf("active role composed skill missing from %+v", res.Skills)
	}
	for _, decision := range res.Decisions {
		if decision.Subject == "skill:design-system" {
			t.Fatalf("inactive role skill leaked into the decision trace: %+v", decision)
		}
	}
}

func TestResolveIgnoresComposedSkillsBoundToUndefinedRoles(t *testing.T) {
	src := makeSource(t, "aos", nil)
	for _, name := range []string{"coding-shape-cli", "tooling-ceo-platform-strategy"} {
		dir := filepath.Join(src.Root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "COMPOSED.md"), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// "director" is staged by the provider but absent from this roster.
	src.RoleSkills = map[string][]schema.ContentRef{
		"platform": {{
			ID: "coding-shape-cli", Path: "coding-shape-cli", EntryPoint: "COMPOSED.md",
		}},
		"director": {{
			ID:         "tooling-ceo-platform-strategy",
			Path:       "tooling-ceo-platform-strategy",
			EntryPoint: "COMPOSED.md",
		}},
	}

	res, err := Resolve(
		testRequest(schema.DeliveryNativeSkills),
		testPerson(),
		[]*schema.Source{src},
		nil,
	)
	if err != nil {
		t.Fatalf("a binding to an undefined role failed the whole resolution: %v", err)
	}
	var found bool
	for _, skill := range res.Skills {
		if skill.ID == "tooling-ceo-platform-strategy" {
			t.Fatal("an undefined role's composed skill entered the resolution")
		}
		if skill.ID == "coding-shape-cli" {
			found = true
		}
	}
	if !found {
		t.Fatalf("requested role composed skill missing from %+v", res.Skills)
	}
	for _, decision := range res.Decisions {
		if decision.Subject == "skill:tooling-ceo-platform-strategy" {
			t.Fatalf("undefined role skill leaked into the decision trace: %+v", decision)
		}
	}
}

func TestResolveStillRejectsAnUndefinedRequestedRole(t *testing.T) {
	src := makeSource(t, "aos", nil)
	req := &schema.Request{Role: "director", Delivery: schema.DeliveryNativeSkills}
	_, err := Resolve(req, testPerson(), []*schema.Source{src}, nil)
	if err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("expected the requested role to stay fail-closed, got %v", err)
	}
}

func TestResolveWarnsAndTracesOverlappingComposedSkillSelectors(t *testing.T) {
	src := makeSource(t, "aos", nil)
	skill := filepath.Join(src.Root, "writing-kai-voice")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "COMPOSED.md"), []byte("# Voice\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	selectors := []string{"*writing*", "*voice*"}
	src.RoleSkills = map[string][]schema.ContentRef{
		"platform": {{
			ID: "writing-kai-voice", Path: "writing-kai-voice",
			EntryPoint: "COMPOSED.md", Selectors: selectors,
		}},
	}
	src.SelectorOverlaps = []schema.SelectorOverlap{{
		Role: "platform", Skill: "writing-kai-voice", Selectors: selectors,
	}}

	res, err := Resolve(
		testRequest(schema.DeliveryNativeSkills),
		testPerson(),
		[]*schema.Source{src},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 ||
		!strings.Contains(res.Warnings[0], `selectors "*writing*", "*voice*", selected once`) {
		t.Fatalf("selector overlap warnings = %q", res.Warnings)
	}
	var overlapTraced, selectionTraced bool
	for _, decision := range res.Decisions {
		if decision.Subject == "selector:writing-kai-voice" &&
			decision.Kind == "selector" && decision.Outcome == OutcomeShadowed &&
			strings.Contains(decision.Reason, `selectors "*writing*", "*voice*", selected once`) {
			overlapTraced = true
		}
		if decision.Subject == "skill:writing-kai-voice" &&
			decision.Outcome == OutcomeSelected &&
			strings.Contains(decision.Reason, `selector(s) "*writing*", "*voice*"`) {
			selectionTraced = true
		}
	}
	if !overlapTraced || !selectionTraced {
		t.Fatalf("selector overlap provenance missing from decisions: %+v", res.Decisions)
	}
}

func TestEmbeddedRolePersonalitiesSelectBoundSkills(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range p.RoleOrder {
		t.Run(roleName, func(t *testing.T) {
			role := p.Roles[roleName]
			if role.Archived {
				t.Skip("archived roles are refused by Resolve, which TestResolveRefusesAnArchivedRole covers")
			}
			modelTier := schema.ModelTierFrontier
			if len(role.SupportedModelTiers) > 0 {
				modelTier = role.SupportedModelTiers[0]
			}
			res, err := Resolve(&schema.Request{
				Role:      roleName,
				Delivery:  schema.DeliveryNativeSkills,
				ModelTier: modelTier,
			}, p, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			rolePersonalities := role.Personalities
			wantCount := len(rolePersonalities) + len(p.RoleActiveBoundaries(roleName)) + len(role.Methods) + 1
			if role.Guardrail != "" {
				wantCount++
			}
			if len(res.Skills) != wantCount {
				t.Fatalf("selected %d skills, want %d: %+v", len(res.Skills), wantCount, res.Skills)
			}
			// Selection order is charter, shared doctrine, personalities, then
			// lazy methods, matching the order the identity card names them.
			if res.Skills[0].ID != p.RoleSkillID(roleName) {
				t.Fatalf("selected role skill = %q, want %q", res.Skills[0].ID, p.RoleSkillID(roleName))
			}
			activeBoundaries := p.RoleActiveBoundaries(roleName)
			for i, boundaryName := range activeBoundaries {
				if want := p.Boundaries[boundaryName].Skill; res.Skills[i+1].ID != want {
					t.Fatalf("selected boundary skill %d = %q, want %q", i+1, res.Skills[i+1].ID, want)
				}
			}
			for i, personalityName := range rolePersonalities {
				index := len(activeBoundaries) + i + 1
				if want := p.Personalities[personalityName].Skill; res.Skills[index].ID != want {
					t.Fatalf("selected skill %d = %q, want %q", index, res.Skills[index].ID, want)
				}
			}
			for i, method := range role.Methods {
				index := len(activeBoundaries) + len(rolePersonalities) + i + 1
				if res.Skills[index].ID != method {
					t.Fatalf("selected method skill %d = %q, want %q", index, res.Skills[index].ID, method)
				}
			}
		})
	}
}

func TestResolveValidationFailures(t *testing.T) {
	src := makeSource(t, "aos", nil)
	p := testPerson()

	if _, err := Resolve(&schema.Request{Role: "pilot", Delivery: schema.DeliveryNativeSkills}, p, []*schema.Source{src}, nil); err == nil || !strings.Contains(err.Error(), "not defined") {
		t.Fatalf("expected unknown role failure, got %v", err)
	}
	broken := testPerson()
	broken.Roles["platform"] = person.Role{
		Purpose: "Build.", Briefing: "Build from evidence.\n\nValidate the result.",
		Personalities: []string{"missing"},
	}
	if _, err := Resolve(testRequest(schema.DeliveryNativeSkills), broken, []*schema.Source{src}, nil); err == nil || !strings.Contains(err.Error(), "without a catalog binding") {
		t.Fatalf("expected missing catalog binding failure, got %v", err)
	}
	missingDefinition := testPerson()
	missingDefinition.Personalities["tenacious"] = person.Personality{
		Skill: "personality-absent", Color: "#d98e48",
	}
	if _, err := Resolve(testRequest(schema.DeliveryNativeSkills), missingDefinition, nil, nil); err == nil ||
		!strings.Contains(err.Error(), `skill "personality-absent"`) {
		t.Fatalf("expected missing selected-person definition failure, got %v", err)
	}
}

func TestResolveShadowsIdenticalAndFailsConflicts(t *testing.T) {
	legacy, err := person.Source(testPerson())
	if err != nil {
		t.Fatal(err)
	}
	legacy.ID = "aos"
	res, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{legacy}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skills[0].Source != "person:kai" || res.Skills[1].Source != "person:kai" {
		t.Fatalf("expected embedded copies selected, got %+v", res.Skills)
	}
	var shadowed bool
	for _, d := range res.Decisions {
		if d.Outcome == OutcomeShadowed && d.Source == legacy.ID {
			shadowed = true
		}
	}
	if !shadowed {
		t.Fatalf("expected the aos copy shadowed, decisions: %+v", res.Decisions)
	}

	conflict := makeSource(t, "aos", map[string]string{
		"personality-tenacious": "# Different\n",
	})
	if _, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{conflict}, nil); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected non-identical collision failure, got %v", err)
	}
}

func TestCompiledDeliveryUsesCanonicalSkillBodies(t *testing.T) {
	res, err := Resolve(testRequest(schema.DeliveryCompiled), testPerson(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.CompiledBodies) != 3 ||
		res.CompiledBodies[0].EntryPoint != "SKILL.md" ||
		res.CompiledBodies[1].EntryPoint != "SKILL.md" ||
		res.CompiledBodies[2].EntryPoint != "SKILL.md" {
		t.Fatalf("expected canonical SKILL.md for the role and every personality, got %v", res.CompiledBodies)
	}
}

func TestMissingOptionalSourceLandsInTrace(t *testing.T) {
	src := makeSource(t, "aos", nil)
	res, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{src},
		[]schema.MissingSource{{ID: "overlay", Reason: "optional source declaration overlay.kdl is absent"}})
	if err != nil {
		t.Fatal(err)
	}
	var noted bool
	for _, d := range res.Decisions {
		if d.Subject == "source:overlay" && d.Outcome == OutcomeExcluded {
			noted = true
		}
	}
	if !noted {
		t.Fatalf("expected the missing optional source excluded in trace, decisions: %+v", res.Decisions)
	}
}

func TestResolveShadowsCopiesThatDifferOnlyByLineEnding(t *testing.T) {
	unix := makeSource(t, "aos", map[string]string{"fixture-review": "# Review\n\nBody.\n"})
	windows := makeSource(t, "kai", map[string]string{"fixture-review": "# Review\r\n\r\nBody.\r\n"})
	res, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{unix, windows}, nil)
	if err != nil {
		t.Fatalf("CRLF copy should shadow, not conflict: %v", err)
	}
	var shadowed bool
	for _, d := range res.Decisions {
		if d.Subject == "skill:fixture-review" && d.Source == windows.ID && d.Outcome == OutcomeShadowed {
			shadowed = true
		}
	}
	if !shadowed {
		t.Fatalf("expected the CRLF copy shadowed, decisions: %+v", res.Decisions)
	}
}

func TestResolveRefusesAnArchivedRole(t *testing.T) {
	src := makeSource(t, "aos", nil)
	p := testPerson()
	role := p.Roles["platform"]
	role.Archived = true
	p.Roles["platform"] = role

	_, err := Resolve(testRequest(schema.DeliveryNativeSkills), p, []*schema.Source{src}, nil)
	if err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("expected an archived role to be refused, got %v", err)
	}
	// The roster still describes the seat, so the message must not read as absence.
	if strings.Contains(err.Error(), "not defined") {
		t.Fatalf("archived must not report as undefined, got %v", err)
	}
}

func TestResolveComposesARoleThatIsNotArchived(t *testing.T) {
	src := makeSource(t, "aos", nil)

	if _, err := Resolve(testRequest(schema.DeliveryNativeSkills), testPerson(), []*schema.Source{src}, nil); err != nil {
		t.Fatalf("a live role must still compose, got %v", err)
	}
}
