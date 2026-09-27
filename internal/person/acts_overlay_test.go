package person

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

// loadWithActsOverlay layers an overlay carrying acts.yaml over the seed
// roster and loads it the way Load does.
func loadWithActsOverlay(t *testing.T, acts string) (*Person, error) {
	t.Helper()
	top := fstest.MapFS{
		overlayMarker:   {Data: []byte("layers_over: core\n")},
		actsOverlayFile: {Data: []byte(acts)},
	}
	layered, err := layerRoster(top, os.DirFS(os.Getenv(RosterEnv)))
	if err != nil {
		t.Fatal(err)
	}
	p, err := loadSource(layered, "test overlay")
	if err != nil {
		t.Fatal(err)
	}
	if err := applyActsOverlay(layered, p); err != nil {
		return nil, err
	}
	return p, nil
}

func TestActsOverlayAppendsAfterTheShippedActs(t *testing.T) {
	base, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var roleActs strings.Builder
	for i := range 12 {
		fmt.Fprintf(&roleActs, "      - {tool: signoz, text: \"signoz query %d before claiming the error rate\"}\n", i)
	}
	p, err := loadWithActsOverlay(t, "overlay: estate\nroles:\n  eng-platform:\n"+roleActs.String()+
		"personalities:\n  grounded:\n    - {tool: node-stats, text: \"node-stats snapshot of the host you are on\"}\n"+
		"boundaries:\n  seek-external-validation:\n    scoped:\n"+
		"      - {tool: glama, text: \"glama search before calling a server new\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	baseRole, role := base.Roles["eng-platform"].Acts, p.Roles["eng-platform"].Acts
	if len(role) != len(baseRole)+12 {
		t.Fatalf("eng-platform has %d acts, want %d shipped plus 12 appended", len(role), len(baseRole))
	}
	for i, act := range baseRole {
		if role[i] != act {
			t.Errorf("shipped act %d moved: got %+v, want %+v", i, role[i], act)
		}
	}
	if got := len(p.Personalities["grounded"].Acts); got != len(base.Personalities["grounded"].Acts)+1 {
		t.Errorf("grounded has %d acts, want one appended", got)
	}
	scoped := p.Boundaries["seek-external-validation"].ActsForSide("scoped")
	if last := scoped[len(scoped)-1]; last.Tool != "glama" || last.Side != "scoped" {
		t.Errorf("scoped side did not receive the appended act: %+v", last)
	}
	if got, want := len(p.Boundaries["seek-external-validation"].ActsForSide("own")),
		len(base.Boundaries["seek-external-validation"].ActsForSide("own")); got != want {
		t.Errorf("own side has %d acts, want it untouched at %d", got, want)
	}
	card, err := p.RenderRoleIdentityCard("eng-platform", "#9c8b31", []string{"seek-external-validation"})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"signoz query 0 before claiming", "glama search before calling a server new"} {
		if !strings.Contains(card, text) {
			t.Errorf("identity card omits appended act %q", text)
		}
	}
}

func TestActsOverlayRefusesWhatWouldReachNobody(t *testing.T) {
	for _, test := range []struct {
		name, acts, want string
	}{
		{"unnamed overlay", "roles: {}\n", "overlay must name itself"},
		{"unknown role", "overlay: e\nroles:\n  not-a-role:\n    - {tool: ssh, text: ssh in}\n", "names no role"},
		{"unknown personality", "overlay: e\npersonalities:\n  not-a-personality:\n    - {tool: ssh, text: ssh in}\n", "names no personality"},
		{"unknown boundary", "overlay: e\nboundaries:\n  nope:\n    own:\n      - {tool: ssh, text: ssh in}\n", "names no boundary"},
		{"unknown side", "overlay: e\nboundaries:\n  modify-live-backend:\n    sideways:\n      - {tool: ssh, text: ssh in}\n", "is not one of"},
		{"unknown key", "overlay: e\npurpose: x\n", "field purpose not found"},
		{"unknown act key", "overlay: e\nroles:\n  scientist:\n    - {tool: ssh, text: ssh in, side: own}\n", "field side not found"},
		{"tool missing from text", "overlay: e\nroles:\n  scientist:\n    - {tool: ssh, text: log in first}\n", "does not name its tool"},
		{"empty act", "overlay: e\nroles:\n  scientist:\n    - {tool: ssh}\n", "without a tool or text"},
		{"repeat", "overlay: e\nroles:\n  scientist:\n    - {tool: ssh, text: ssh in}\n    - {tool: ssh, text: ssh in}\n", "repeats act"},
	} {
		_, err := loadWithActsOverlay(t, test.acts)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: err = %v, want it to contain %q", test.name, err, test.want)
		}
	}
}

func TestLoadAppliesTheMountedActsOverlay(t *testing.T) {
	// Mounted the way a host mounts it: the override names the overlay, and the
	// plain roster under it is the next search location, ~/.agent-compose/roster.
	seed := os.Getenv(RosterEnv)
	home := t.TempDir()
	if err := os.MkdirAll(home+"/.agent-compose", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(seed, home+"/.agent-compose/roster"); err != nil {
		t.Fatal(err)
	}
	overlay := t.TempDir()
	writeTree(t, overlay, map[string]string{
		overlayMarker:   "layers_over: core\n",
		actsOverlayFile: "overlay: estate\nroles:\n  scientist:\n    - {tool: housecast, text: \"housecast grade export the run\"}\n",
	})
	t.Setenv("HOME", home)
	t.Setenv(RosterEnv, overlay)
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	acts := p.Roles["scientist"].Acts
	if last := acts[len(acts)-1]; last.Tool != "housecast" {
		t.Errorf("Load did not apply the mounted overlay: last scientist act %+v", last)
	}
}
