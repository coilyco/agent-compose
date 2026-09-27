package person

import (
	"fmt"
	"strings"
	"testing"
)

// Surfaces that exist on one estate and nowhere else. The failure this guards
// is agentic-os#1381, and the contract is docs/kdl-contracts.md.
var estateToolPrefixes = []string{"aosguard", "mcp__", "ward ", "acompose", "housecast"}

// The check #388 proposed as a grep. A rule an agent has to remember is weaker
// than a check that fails.
func TestCoreRosterNamesActsForEveryAttribute(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range p.roleOrder() {
		if len(p.Roles[roleName].Acts) == 0 {
			t.Errorf("core role %q names no acts", roleName)
		}
	}
	for _, name := range p.PersonalityOrder {
		if len(p.Personalities[name].Acts) == 0 {
			t.Errorf("core personality %q names no acts", name)
		}
	}
	for _, name := range p.BoundaryOrder {
		for _, side := range boundaryActSides {
			if len(p.Boundaries[name].ActsForSide(side)) == 0 {
				t.Errorf("core boundary %q %s side names no acts", name, side)
			}
		}
	}
}

// TestCoreRosterActsNameNoEstateOnlyTool keeps the shipped roster runnable by a
// stranger. See agent-compose#380.
func TestCoreRosterActsNameNoEstateOnlyTool(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	check := func(owner string, acts []Act) {
		for _, act := range acts {
			for _, prefix := range estateToolPrefixes {
				if strings.HasPrefix(act.Tool, prefix) {
					t.Errorf("%s names estate-only tool %q", owner, act.Tool)
				}
			}
		}
	}
	for _, roleName := range p.roleOrder() {
		check("core role "+roleName, p.Roles[roleName].Acts)
	}
	for _, name := range p.PersonalityOrder {
		check("core personality "+name, p.Personalities[name].Acts)
	}
	for _, name := range p.BoundaryOrder {
		check("core boundary "+name, p.Boundaries[name].Acts)
	}
}

// TestActsStayOptionalUntilOneIsDeclared keeps packages authored before acts
// loading, and makes the catalog complete the moment one attribute opts in.
func TestActsStayOptionalUntilOneIsDeclared(t *testing.T) {
	p := &Person{
		RoleOrder: []string{"builder", "reader"},
		Roles: map[string]Role{
			"builder": {},
			"reader":  {},
		},
	}
	if err := validateActCoverage(p); err != nil {
		t.Fatalf("a package declaring no acts failed: %v", err)
	}
	p.Roles["builder"] = Role{Acts: []Act{
		{Tool: "grep", Text: "grep it"},
		{Tool: "diff", Text: "diff it"},
		{Tool: "wc", Text: "wc it"},
	}}
	err := validateActCoverage(p)
	if err == nil || !strings.Contains(err.Error(), "role reader names no acts") {
		t.Fatalf("a half-covered catalog loaded, error = %v", err)
	}
}

// The load-bearing guarantee for the deferring side: a deferred boundary is a
// different act, never the owner's withheld.
func TestBoundaryActsFollowTheSideTheSeatHolds(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	boundary := p.Boundaries["seek-external-validation"]
	own := boundary.ActsForSide("own")
	deferred := boundary.ActsForSide("defer")
	if len(own) == 0 || len(deferred) == 0 {
		t.Fatal("seek-external-validation is missing an own or defer act list")
	}
	for _, ownAct := range own {
		for _, deferAct := range deferred {
			if ownAct.Text == deferAct.Text {
				t.Errorf("own and defer sides share the act %q", ownAct.Text)
			}
		}
	}
	card, err := p.RenderRoleIdentityCard("eng-platform", "#9c8b31", []string{"seek-external-validation"})
	if err != nil {
		t.Fatal(err)
	}
	// platform scopes this boundary, so the card owes it the scoped acts.
	for _, scoped := range boundary.ActsForSide("scoped") {
		if !strings.Contains(card, scoped.Text) {
			t.Errorf("platform card omits its scoped act %q", scoped.Text)
		}
	}
	for _, ownAct := range own {
		if strings.Contains(card, ownAct.Text) {
			t.Errorf("platform card carries the owning act %q", ownAct.Text)
		}
	}
}

// Kai, 2026-09-27: how many acts is a guideline, so only an empty list or side
// fails. See docs/kdl-contracts.md.
func TestActCountsAreGuidelinesButEmptyFails(t *testing.T) {
	acts := func(n int, side string) []Act {
		out := make([]Act, n)
		for i := range out {
			out[i] = Act{Side: side, Tool: "grep", Text: fmt.Sprintf("grep %s %d", side, i)}
		}
		return out
	}
	for _, n := range []int{1, 2, 12, 40} {
		if err := validateActs("role", acts(n, ""), false); err != nil {
			t.Errorf("a list of %d acts failed: %v", n, err)
		}
	}
	if err := validateActs("role", nil, false); err == nil {
		t.Error("an empty act list loaded")
	}
	sides := []Act{}
	for _, side := range boundaryActSides {
		sides = append(sides, acts(7, side)...)
	}
	if err := validateActs("boundary", sides, true); err != nil {
		t.Errorf("seven acts per boundary side failed: %v", err)
	}
	if err := validateActs("boundary", sides[7:], true); err == nil {
		t.Error("a boundary with an empty own side loaded")
	}
}
