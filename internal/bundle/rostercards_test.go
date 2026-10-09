package bundle

import (
	"os"
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/person"
	"github.com/coilyco/agent-compose/v2/internal/roster"
)

// The base's copies are dead weight on every turn. See docs/bundle-protocol.md.
func TestStripRosterCardsDropsEveryCardAndKeepsDoctrine(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.RoleOrder) < 2 {
		t.Fatalf("roster carries %d roles, want at least 2", len(p.RoleOrder))
	}

	var base strings.Builder
	base.WriteString("# Agent instructions\n\nkeep me\n\n")
	for _, roleName := range p.RoleOrder {
		card, err := p.RenderRoleIdentityCard(roleName, p.Roles[roleName].FavoriteColor, p.RoleActiveBoundaries(roleName))
		if err != nil {
			t.Fatal(err)
		}
		base.WriteString(card)
		base.WriteString("\n")
	}
	base.WriteString("# Agent seats\n\nkeep me too\n")

	stripped := stripRosterCards(base.String(), p)

	for _, roleName := range p.RoleOrder {
		heading := "# " + p.RoleDisplayName(roleName)
		if strings.Contains(stripped, heading) {
			t.Errorf("stripped base still carries %q", heading)
		}
	}
	for _, want := range []string{"# Agent instructions", "keep me", "# Agent seats", "keep me too"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("stripped base dropped doctrine %q", want)
		}
	}
	if before, after := len(base.String()), len(stripped); after >= before {
		t.Errorf("strip recovered nothing: %d -> %d bytes", before, after)
	} else {
		t.Logf("strip recovered %d of %d bytes", before-after, before)
	}
}

func TestStripRosterCardsLeavesACardlessBaseAlone(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	base := "# Agent instructions\n\nno cards here\n"
	if got := stripRosterCards(base, p); got != base {
		t.Errorf("cardless base changed:\n%q", got)
	}
	if got := stripRosterCards(base, nil); got != base {
		t.Errorf("nil person changed base:\n%q", got)
	}
}

func TestStripAssignedBaseDropsSwitchPolicyAndDuplicateBodies(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	files, err := roster.Render(p, nil, "/opt/artifact")
	if err != nil {
		t.Fatal(err)
	}
	base := string(files["AGENTS.COMPOSE.md"])
	invariant, err := os.ReadFile("../../seed/roster/data/invariant/INVARIANT.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(base, strings.TrimSpace(string(invariant))) {
		t.Fatal("rendered base no longer carries the invariant, so this test proves nothing")
	}

	stripped := stripAssignedBase(base, p, [][]byte{invariant})

	for _, heading := range switchPolicyHeadings {
		if !strings.Contains(base, heading) {
			t.Errorf("rendered base lacks %q, so the strip list drifted from the policy", heading)
		}
		if strings.Contains(stripped, heading) {
			t.Errorf("stripped base still carries %q", heading)
		}
	}
	for _, want := range []string{
		"# Agent seats",
		"#### Conditional evaluation fixture authority",
		"#### Humans grade evaluations",
	} {
		if !strings.Contains(stripped, want) {
			t.Errorf("stripped base dropped %q", want)
		}
	}
	if strings.Contains(stripped, strings.TrimSpace(string(invariant))) {
		t.Error("stripped base still carries the invariant the bundle appends")
	}
	t.Logf("assigned strip recovered %d of %d bytes", len(base)-len(stripped), len(base))
}
