package person

import (
	"strings"
	"testing"
)

// A stamp has to say exactly who sent a message, so no two roles may answer to
// the same name, and no role's later instances may take another role's name.
func TestShippedRosterNamesAreUniqueAcrossRoles(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}
	holder := map[string]string{}
	for _, role := range p.roleOrder() {
		for _, name := range p.RoleIdentityNames(role) {
			if other, taken := holder[name]; taken {
				t.Errorf("%q is a name of both %s and %s", name, other, role)
			}
			holder[name] = role
		}
	}
}

// Every role that launches gets a second name, so two live instances of it
// differ. The archived roles keep the single name they have.
func TestEveryLaunchableRoleHasMoreThanOneName(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}
	for _, role := range p.roleOrder() {
		if p.Roles[role].Archived {
			continue
		}
		if names := p.RoleIdentityNames(role); len(names) < 2 {
			t.Errorf("role %s has only %v, so a second instance would share it", role, names)
		}
	}
}

func TestAlternatesKeepTheFamilyWord(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}
	for _, role := range p.roleOrder() {
		names := p.RoleIdentityNames(role)
		family := names[0][strings.LastIndex(names[0], "-")+1:]
		for _, name := range names[1:] {
			if !strings.HasSuffix(name, "-"+family) {
				t.Errorf("role %s: %q does not keep the family word %q of %q", role, name, family, names[0])
			}
		}
	}
}

func TestChooseRoleIdentityTakesOnlyTheRolesOwnNames(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatalf("load the roster: %v", err)
	}
	names := p.RoleIdentityNames("sysadmin-senior")
	if err := p.ChooseRoleIdentity("sysadmin-senior", names[1]); err != nil {
		t.Fatalf("choose %q: %v", names[1], err)
	}
	if got := p.RoleIdentityName("sysadmin-senior"); got != names[1] {
		t.Errorf("identity is %q, want %q", got, names[1])
	}
	for _, seat := range p.Roles["sysadmin-senior"].Seats {
		if seat.Name != names[1] {
			t.Errorf("seat %s kept %q, want %q", seat.Key, seat.Name, names[1])
		}
	}
	after := p.RoleIdentityNames("sysadmin-senior")
	if after[0] != names[1] || len(after) != len(names) {
		t.Errorf("pool is %v after choosing %q from %v", after, names[1], names)
	}
	seen := map[string]bool{}
	for _, name := range after {
		if seen[name] {
			t.Errorf("%q is in the pool twice: %v", name, after)
		}
		seen[name] = true
	}
	for _, name := range names {
		if !seen[name] {
			t.Errorf("%q left the pool: %v", name, after)
		}
	}
	if err := p.ChooseRoleIdentity("sysadmin-senior", "Griffin-Goose"); err == nil ||
		!strings.Contains(err.Error(), "not one of sysadmin-senior's names") {
		t.Errorf("another role's name was accepted: %v", err)
	}
	if err := p.ChooseRoleIdentity("nope", "Echo"); err == nil {
		t.Error("a missing role was accepted")
	}
}

func TestAlternatesMustBeSetAndDistinct(t *testing.T) {
	for _, test := range []struct {
		name       string
		identity   AgentIdentity
		wantInText string
	}{
		{"empty", AgentIdentity{Name: "A-Ox", Alternates: []string{"B-Ox", " "}}, "is empty"},
		{"repeat", AgentIdentity{Name: "A-Ox", Alternates: []string{"B-Ox", "B-Ox"}}, "appears twice"},
		{"primary again", AgentIdentity{Name: "A-Ox", Alternates: []string{"A-Ox"}}, "appears twice"},
	} {
		identity := test.identity
		err := validateAlternates("demo", &identity)
		if err == nil || !strings.Contains(err.Error(), test.wantInText) {
			t.Errorf("%s: got %v, want an error containing %q", test.name, err, test.wantInText)
		}
	}
	ok := AgentIdentity{Name: "A-Ox", Alternates: []string{" B-Ox ", "C-Ox"}}
	if err := validateAlternates("demo", &ok); err != nil {
		t.Fatalf("a good pool failed: %v", err)
	}
	if ok.Alternates[0] != "B-Ox" {
		t.Errorf("alternates were not trimmed: %q", ok.Alternates[0])
	}
}
