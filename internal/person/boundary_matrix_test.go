package person

import "testing"

// The matrix exists so the allocation is readable at a glance rather than
// reassembled from prose. agent-compose#325
func TestBoundaryMatrixGivesOneVerbPerCell(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	roles, entries := p.BoundaryMatrix()
	if len(entries) != len(p.Boundaries) {
		t.Fatalf("matrix has %d rows, roster has %d boundaries", len(entries), len(p.Boundaries))
	}
	valid := map[string]bool{"OWNS": true, "scope": true, "defers": true, "-": true}
	for _, entry := range entries {
		if len(entry.Verbs) != len(roles) {
			t.Errorf("%s covers %d roles, want %d", entry.Boundary, len(entry.Verbs), len(roles))
		}
		owners := 0
		for _, role := range roles {
			verb, ok := entry.Verbs[role]
			if !ok {
				t.Errorf("%s has no cell for role %s", entry.Boundary, role)
				continue
			}
			if !valid[verb] {
				t.Errorf("%s/%s verb %q is not a single known token", entry.Boundary, role, verb)
			}
			if verb == "OWNS" {
				owners++
			}
		}
		if want := 1 + len(entry.CoOwners); owners != want {
			t.Errorf("%s has %d owners, want the primary and %d co-owners", entry.Boundary, owners, len(entry.CoOwners))
		}
		for _, owner := range append([]string{entry.Owner}, entry.CoOwners...) {
			if entry.Verbs[owner] != "OWNS" {
				t.Errorf("%s names owner %q but its cell reads %q",
					entry.Boundary, owner, entry.Verbs[owner])
			}
		}
	}
}

// Boundary order must not depend on the role, or two runs cannot be lined up
// beside each other, which is the reason describe could not do this.
func TestBoundaryMatrixOrderIsIndependentOfRole(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	_, first := p.BoundaryMatrix()
	_, second := p.BoundaryMatrix()
	for i := range first {
		if first[i].Boundary != second[i].Boundary {
			t.Fatalf("row %d moved between calls: %q then %q", i, first[i].Boundary, second[i].Boundary)
		}
	}
	want := p.boundaryOrder()
	for i, entry := range first {
		if entry.Boundary != want[i] {
			t.Errorf("row %d is %q, want boundary order %q", i, entry.Boundary, want[i])
		}
	}
}
