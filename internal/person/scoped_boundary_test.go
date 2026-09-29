package person

import "testing"

// Every boundary reaches every seat with one primary owner plus any co-owners.
// The scoped/deferring split moves as seats are added, so it is not asserted.
func TestEveryBoundaryAllocatesTheWholeRoster(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, boundaryName := range p.BoundaryOrder {
		owner, scoped, deferring := 0, 0, 0
		for _, roleName := range p.RoleOrder {
			role := p.Roles[roleName]
			if p.Boundaries[boundaryName].OwnedBy(roleName) {
				owner++
			}
			for _, entry := range role.ScopedBoundaries {
				if entry.Name == boundaryName {
					scoped++
					if entry.Scope == "" {
						t.Errorf("role %q scopes %q with no limit text", roleName, boundaryName)
					}
				}
			}
			for _, declared := range role.Boundaries {
				if declared == boundaryName {
					deferring++
				}
			}
		}
		wantOwners := 1 + len(p.Boundaries[boundaryName].CoOwners)
		if owner != wantOwners || owner+scoped+deferring != len(p.RoleOrder) {
			t.Errorf(
				"boundary %q reaches %d owners, %d scoped, %d deferring, want %d owners and %d seats total",
				boundaryName, owner, scoped, deferring, wantOwners, len(p.RoleOrder),
			)
		}
	}
}

func TestScopedBoundaryJoinsTheActiveSet(t *testing.T) {
	p := &Person{
		RoleOrder:  []string{"builder"},
		Roles:      map[string]Role{"builder": {ScopedBoundaries: []ScopedBoundary{{Name: "modify-live-backend", Scope: "local worlds only"}}}},
		Boundaries: map[string]Boundary{"modify-live-backend": {Owner: "operator", Skill: "boundary-modify-live-backend"}},
	}
	active := p.RoleActiveBoundaries("builder")
	if len(active) != 1 || active[0] != "modify-live-backend" {
		t.Fatalf("active = %v, want the scoped boundary to be delivered", active)
	}
}

func TestScopedBoundaryCountsAsUse(t *testing.T) {
	p := &Person{
		RoleOrder:  []string{"builder"},
		Roles:      map[string]Role{"builder": {ScopedBoundaries: []ScopedBoundary{{Name: "modify-live-backend", Scope: "local worlds only"}}}},
		Boundaries: map[string]Boundary{"modify-live-backend": {Owner: "operator"}},
	}
	if err := validateNoUnusedBoundaries(p); err != nil {
		t.Fatalf("a scoped declaration should count as a use: %v", err)
	}
}

// An owner already receives the body by owning it. Scoping its own boundary
// would hand it two contradictory sides.
func TestOwnerMayNotScopeItsOwnBoundary(t *testing.T) {
	p := &Person{
		RoleOrder:  []string{"operator"},
		Roles:      map[string]Role{"operator": {ScopedBoundaries: []ScopedBoundary{{Name: "modify-live-backend", Scope: "anything"}}}},
		Boundaries: map[string]Boundary{"modify-live-backend": {Owner: "operator"}},
	}
	if err := validateBoundaryOwners(p); err == nil {
		t.Fatal("owner scoping its own boundary should fail")
	}
}

func TestScopedBoundaryBodyOrdering(t *testing.T) {
	for name, body := range map[string]string{
		"scoped before own":  boundaryScopedHeading + "\na\n" + boundaryOwnHeading + "\nb\n" + boundaryDeferHeading + "\nc\n",
		"scoped after defer": boundaryOwnHeading + "\na\n" + boundaryDeferHeading + "\nb\n" + boundaryScopedHeading + "\nc\n",
	} {
		if err := validateBoundaryBodySides("b", body); err == nil {
			t.Errorf("%s: want an ordering failure", name)
		}
	}
	ok := boundaryOwnHeading + "\na\n" + boundaryScopedHeading + "\nb\n" + boundaryDeferHeading + "\nc\n"
	if err := validateBoundaryBodySides("b", ok); err != nil {
		t.Errorf("own..scoped..defer should validate: %v", err)
	}
}

// A two-sided body is still valid. Only a boundary somebody scopes needs three.
func TestTwoSidedBodyStillValidates(t *testing.T) {
	body := boundaryOwnHeading + "\na\n" + boundaryDeferHeading + "\nb\n"
	if err := validateBoundaryBodySides("b", body); err != nil {
		t.Fatalf("a boundary nobody scopes needs no scoped section: %v", err)
	}
}

func TestRoleMayNotBothDeferAndScope(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		for _, scoped := range role.ScopedBoundaries {
			for _, declared := range role.Boundaries {
				if declared == scoped.Name {
					t.Errorf("role %q both defers and scopes %q", roleName, scoped.Name)
				}
			}
		}
	}
}
