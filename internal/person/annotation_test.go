package person

import "testing"

func TestSeatAnnotation(t *testing.T) {
	cases := map[string]struct {
		name        string
		displayName string
		want        string
	}{
		"complete":             {"Angie", "Engineer", "Angie (Engineer)"},
		"authored label wins":  {"Quail", "QA", "Quail (QA)"},
		"missing display name": {"Angie", "", "Angie"},
		"missing name":         {"", "Engineer", ""},
		"padded input":         {" Angie ", " Engineer ", "Angie (Engineer)"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			got := SeatAnnotation(test.name, test.displayName)
			if got != test.want {
				t.Errorf("SeatAnnotation = %q, want %q", got, test.want)
			}
		})
	}
}

// Appended, never interleaved: every documented form stays a prefix of the
// annotated one, so a surface with no id renders exactly what it did before.
func TestWithShortID(t *testing.T) {
	for name, test := range map[string]struct {
		display string
		shortID string
		want    string
	}{
		"annotated":     {"Angie (Engineer)", "uz86", "Angie (Engineer) uz86"},
		"label only":    {"Angie", "uz86", "Angie uz86"},
		"role fallback": {"platform-eng", "uz86", "platform-eng uz86"},
		"no id":         {"Angie", "", "Angie"},
		"no display":    {"", "uz86", ""},
		"neither":       {"", "", ""},
		"padded input":  {" Angie ", " uz86 ", "Angie uz86"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := WithShortID(test.display, test.shortID); got != test.want {
				t.Errorf("WithShortID = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSeatLabelStopsShortOfTheRole(t *testing.T) {
	if got := SeatLabel("Angie"); got != "Angie" {
		t.Errorf("SeatLabel = %q, want Angie", got)
	}
	if got := SeatLabel(""); got != "" {
		t.Errorf("SeatLabel without a name = %q, want empty", got)
	}
	if got := SeatLabel(" Angie "); got != "Angie" {
		t.Errorf("SeatLabel with padded input = %q, want Angie", got)
	}
}

// TestShippedRolesAnnotate keeps the roster honest: every seat the shipped
// person package offers has to render a complete annotation.
func TestShippedRolesAnnotate(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range p.Roles {
		displayName := p.RoleDisplayName(name)
		for _, seat := range role.Seats {
			annotation := SeatAnnotation(seat.Name, displayName)
			if annotation == "" {
				t.Errorf("role %q seat %q renders no annotation", name, seat.Selector())
				continue
			}
			if displayName == "" {
				t.Errorf("role %q has no display name", name)
			}
		}
	}
}
