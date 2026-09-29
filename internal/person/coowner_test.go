package person

import (
	"strings"
	"testing"
)

// A boundary keeps one primary owner and may share the owner side with
// co-owners. Every case names the rule it protects.

func coOwnedPerson() *Person {
	senior, access, other := validRole(), validRole(), validRole()
	access.Derives = "senior"
	return &Person{
		RoleOrder:     []string{"senior", "access", "other"},
		Roles:         map[string]Role{"senior": senior, "access": access, "other": other},
		BoundaryOrder: []string{"shared-thing"},
		Boundaries: map[string]Boundary{"shared-thing": {
			Skill: "boundary-shared-thing", Owner: "senior", CoOwners: []string{"access"},
			Summary: "Senior and Access change it, other roles hand it over",
		}},
	}
}

func TestOwnedByCoversPrimaryAndCoOwners(t *testing.T) {
	b := coOwnedPerson().Boundaries["shared-thing"]
	for role, want := range map[string]bool{"senior": true, "access": true, "other": false, "": false} {
		if got := b.OwnedBy(role); got != want {
			t.Errorf("OwnedBy(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestCoOwnerReceivesTheBodyWithoutDeclaringIt(t *testing.T) {
	p := coOwnedPerson()
	if owned := p.RoleOwnedBoundaries("access"); len(owned) != 1 || owned[0] != "shared-thing" {
		t.Fatalf("co-owner owns %v, want the shared boundary", owned)
	}
	if owned := p.RoleOwnedBoundaries("other"); len(owned) != 0 {
		t.Fatalf("a role outside the owners owns %v", owned)
	}
}

func TestValidateBoundaryOwnersRulesForCoOwners(t *testing.T) {
	cases := []struct {
		name string
		want string
		edit func(*Person)
	}{
		{"co-owner repeats the primary", "twice",
			func(p *Person) {
				b := p.Boundaries["shared-thing"]
				b.CoOwners = []string{"senior"}
				p.Boundaries["shared-thing"] = b
			}},
		{"co-owner listed twice", "twice",
			func(p *Person) {
				b := p.Boundaries["shared-thing"]
				b.CoOwners = []string{"access", "access"}
				p.Boundaries["shared-thing"] = b
			}},
		{"unknown co-owner", "unknown owner",
			func(p *Person) {
				b := p.Boundaries["shared-thing"]
				b.CoOwners = []string{"ghost"}
				p.Boundaries["shared-thing"] = b
			}},
		{"co-owner also declares it", "also declares it",
			func(p *Person) {
				r := p.Roles["access"]
				r.Boundaries = []string{"shared-thing"}
				p.Roles["access"] = r
			}},
		{"co-owner also scopes it", "also scopes it",
			func(p *Person) {
				r := p.Roles["access"]
				r.ScopedBoundaries = []ScopedBoundary{{Name: "shared-thing", Scope: "anything"}}
				p.Roles["access"] = r
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := coOwnedPerson()
			tc.edit(p)
			err := validateBoundaryOwners(p)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
	if err := validateBoundaryOwners(coOwnedPerson()); err != nil {
		t.Fatalf("a well-formed co-owned boundary was refused: %v", err)
	}
}

// A derived role narrows its parent, so it may share the parent's own
// boundary and nothing else.
func TestDerivedRoleMayCoOwnOnlyItsParentsBoundary(t *testing.T) {
	if err := validateDecodedPerson(coOwnedPerson()); err != nil {
		t.Fatalf("a derived co-owner of its parent's boundary was refused: %v", err)
	}
	p := coOwnedPerson()
	b := p.Boundaries["shared-thing"]
	b.Owner = "other"
	p.Boundaries["shared-thing"] = b
	err := validateDecodedPerson(p)
	if err == nil || !strings.Contains(err.Error(), "does not own it") {
		t.Fatalf("err = %v, want a derived co-owner of a foreign boundary refused", err)
	}
}

func TestBoundaryMatrixMarksCoOwnersAsOwners(t *testing.T) {
	p := coOwnedPerson()
	_, entries := p.BoundaryMatrix()
	if len(entries) != 1 {
		t.Fatalf("matrix rows = %d, want 1", len(entries))
	}
	for _, role := range []string{"senior", "access"} {
		if got := entries[0].Verbs[role]; got != "OWNS" {
			t.Errorf("%s reads %q, want OWNS", role, got)
		}
	}
	if got := entries[0].Verbs["other"]; got == "OWNS" {
		t.Errorf("a non-owner reads OWNS")
	}
}
