package roleslug

import "testing"

func TestCanonicalMapsARetiredSlugAndPassesAnythingElse(t *testing.T) {
	for in, want := range map[string]string{
		" science ":       "scientist",
		"scientist":       "scientist",
		"senior-sysadmin": "sysadmin-senior",
		"analyst":         "analyst",
		"":                "",
	} {
		if got := Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}
