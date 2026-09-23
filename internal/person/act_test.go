package person

import (
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
		if got := len(p.Roles[roleName].Acts); got != actsPerAttribute {
			t.Errorf("core role %q names %d acts, want %d", roleName, got, actsPerAttribute)
		}
	}
	for _, name := range p.PersonalityOrder {
		if got := len(p.Personalities[name].Acts); got != actsPerAttribute {
			t.Errorf("core personality %q names %d acts, want %d", name, got, actsPerAttribute)
		}
	}
	for _, name := range p.BoundaryOrder {
		for _, side := range boundaryActSides {
			if got := len(p.Boundaries[name].ActsForSide(side)); got != actsPerAttribute {
				t.Errorf(
					"core boundary %q %s side names %d acts, want %d",
					name, side, got, actsPerAttribute,
				)
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
	if err == nil || !strings.Contains(err.Error(), "role reader names 0 acts") {
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
	card, err := p.RenderRoleIdentityCard("platform-eng", "#9c8b31", []string{"seek-external-validation"})
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
