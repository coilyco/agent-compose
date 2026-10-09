package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A deployment that supplies its own voice and run guidance asks the identity
// card to leave those sections out. The rest of the card must stay.

func TestCardOmitDropsOnlyTheNamedSections(t *testing.T) {
	whole := cardText(t, "card-whole.kdl", "prod-manager")
	trimmed := cardText(t, "card-omit.kdl", "prod-manager")
	for _, heading := range []string{"## Voice", "## Run"} {
		if !strings.Contains(whole, heading) {
			t.Fatalf("the unomitted card has no %q, so this test proves nothing", heading)
		}
		if strings.Contains(trimmed, heading) {
			t.Errorf("card-omit left %q in the card", heading)
		}
	}
	for _, kept := range []string{"## Personality meld", "## Boundaries", "## Active doctrine", "**Role skill //"} {
		if !strings.Contains(trimmed, kept) {
			t.Errorf("card-omit removed %q, which it did not name", kept)
		}
	}
	if len(trimmed) >= len(whole) {
		t.Errorf("trimmed card is %d bytes against %d whole", len(trimmed), len(whole))
	}
}

func cardText(t *testing.T, fixtureName, role string) string {
	t.Helper()
	result, err := Run(fixture(t, fixtureName), t.TempDir())
	if err != nil {
		t.Fatalf("compose %s: %v", fixtureName, err)
	}
	raw, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "content", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "`"+role+"`") {
		t.Fatalf("%s did not compose role %s", fixtureName, role)
	}
	return string(raw)
}
