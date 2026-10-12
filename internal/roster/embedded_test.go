package roster

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/person"
	"github.com/coilyco/agent-compose/v2/internal/schema"
)

func TestEmbeddedRosterRendersEveryCanonicalSeat(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	source, err := person.Source(p)
	if err != nil {
		t.Fatal(err)
	}
	files, err := Render(p, []*schema.Source{source}, "/opt/artifact")
	if err != nil {
		t.Fatal(err)
	}
	table := string(files["AGENTS.COMPOSE.md"])
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		if !strings.Contains(table, "**Role skill // `"+p.RoleSkillID(roleName)+"`**") ||
			!strings.Contains(table, role.Purpose) {
			t.Errorf("roster omitted role %q", roleName)
		}
		identity := fmt.Sprintf("**Agent // %s**", role.Identity.Name)
		// Every role answers to its own name, a color twin included, so each
		// card names its role once and no stamp is ambiguous.
		if got := strings.Count(table, identity); got != 1 {
			t.Errorf("roster identity count for role %q = %d, want 1", roleName, got)
		}
		for _, seat := range role.Seats {
			want := "// " + seat.Selector()
			if !strings.Contains(table, want) {
				t.Errorf("roster omitted seat %q for role %q", seat.Harness, roleName)
			}
		}
		if _, ok := files[".agents/skills/"+p.RoleSkillID(roleName)+"/SKILL.md"]; !ok {
			t.Errorf("roster omitted role skill %q", roleName)
		}
		for _, method := range role.Methods {
			if _, ok := files[".agents/skills/"+method+"/SKILL.md"]; !ok {
				t.Errorf("roster omitted method skill %q for role %q", method, roleName)
			}
		}
	}
}
