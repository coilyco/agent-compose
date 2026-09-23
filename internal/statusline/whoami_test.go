package statusline

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/agentid"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/launch"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

func TestWhoamiNamesTheSeatAndSession(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	got, err := Whoami(Options{Target: shortIDFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Angie uz86" {
		t.Errorf("Whoami = %q, want Angie uz86", got)
	}
}

// Whoami and the status-line row must agree, or two surfaces name the same
// session differently and the hook stops being authoritative.
func TestWhoamiAgreesWithTheStatusLineRow(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	target := shortIDFixture(t)
	name, err := Whoami(Options{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	row, err := Render(Options{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	if name == "" || !strings.Contains(row, name) {
		t.Errorf("row %q does not carry the whoami name %q", row, name)
	}
}

// Silence rather than a guess. A session with no composition has no composed
// name, and inventing one is what the retired local computation did wrong.
func TestWhoamiSuppressesWithoutAProjection(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	got, err := Whoami(Options{Target: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("Whoami = %q, want empty without a projection", got)
	}
}

func TestWhoamiWithoutASessionStillNamesTheSeat(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	got, err := Whoami(Options{Target: shortIDFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Angie" {
		t.Errorf("Whoami = %q, want Angie", got)
	}
}

// seatBundleFixture writes one bundle carrying a single named seat, so a test
// can bind two sessions to two different compositions.
func seatBundleFixture(t *testing.T, role, name string) string {
	t.Helper()
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "manifest.json"), bundle.Manifest{
		Format: "agent-compose.bundle", Role: role, ModelTier: "frontier",
		Identity: bundle.RoleIdentity{
			Person: "core",
			Seats:  []person.Seat{{Key: "claude", Name: name}},
		},
	})
	return dir
}

// The failure this binding exists to fix: a native session shadow has no
// projection above its cwd, so the walk found nothing and whoami said nothing.
func TestWhoamiAnswersFromTheSessionBundleWithNoProjectionInReach(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "kj58")
	t.Setenv(launch.SessionBundleEnv, seatBundleFixture(t, "platform-eng", "Angie"))
	t.Setenv(launch.SessionLayoutEnv, "claude")
	got, err := Whoami(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Angie kj58" {
		t.Errorf("Whoami = %q, want Angie kj58", got)
	}
}

// Two roles on one host resolved to whichever projection converged last.
func TestWhoamiKeepsConcurrentSessionsApart(t *testing.T) {
	platform := seatBundleFixture(t, "platform-eng", "Angie")
	director := seatBundleFixture(t, "prod-director", "Portia")
	for _, tc := range []struct{ bundleDir, want string }{
		{platform, "Angie"},
		{director, "Portia"},
	} {
		t.Setenv(agentid.SessionEnv, "")
		t.Setenv(launch.SessionBundleEnv, tc.bundleDir)
		t.Setenv(launch.SessionLayoutEnv, "claude")
		got, err := Whoami(Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("Whoami = %q, want %q", got, tc.want)
		}
	}
}

// --target is an inspection request, so it still reports what sits at that path.
func TestWhoamiTargetStillInspectsAProjection(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	t.Setenv(launch.SessionBundleEnv, seatBundleFixture(t, "prod-director", "Portia"))
	t.Setenv(launch.SessionLayoutEnv, "claude")
	got, err := Whoami(Options{Target: shortIDFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Angie" {
		t.Errorf("Whoami = %q, want the targeted projection's seat Angie", got)
	}
}

// A half-set binding is not a binding, so the walk still runs.
func TestWhoamiIgnoresAnIncompleteBinding(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	t.Setenv(launch.SessionBundleEnv, seatBundleFixture(t, "prod-director", "Portia"))
	t.Setenv(launch.SessionLayoutEnv, "")
	got, err := Whoami(Options{Target: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("Whoami = %q, want empty when the binding is incomplete", got)
	}
}

// The row and the hook resolve the same way, so a bound session with no
// projection in reach cannot have one surface name it and the other stay blank.
func TestTheRowAgreesWithWhoamiUnderASessionBinding(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "kj58")
	t.Setenv(launch.SessionBundleEnv, shortIDBundleFixture(t))
	t.Setenv(launch.SessionLayoutEnv, "claude")
	name, err := Whoami(Options{})
	if err != nil {
		t.Fatal(err)
	}
	row, err := Render(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if name == "" || !strings.Contains(row, name) {
		t.Errorf("row %q does not carry the whoami name %q", row, name)
	}
}

// A transcript that names a bundle is only useful if the name is stable and
// discriminating. agent-compose#350
func TestWhoamiJSONCarriesTheBundleFingerprint(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	got, err := WhoamiJSON(Options{Target: shortIDFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal([]byte(got), &rec); err != nil {
		t.Fatalf("WhoamiJSON emitted invalid JSON %q: %v", got, err)
	}
	if rec.Format != "agent-compose.whoami.v1" {
		t.Errorf("format = %q", rec.Format)
	}
	if rec.Seat != "Angie uz86" {
		t.Errorf("seat = %q, want Angie uz86", rec.Seat)
	}
	if !strings.HasPrefix(rec.Bundle, "sha256:") || len(rec.Bundle) != 71 {
		t.Errorf("bundle = %q, want a sha256: fingerprint", rec.Bundle)
	}
}

// The plain and machine-readable forms must name one session, or the hook and
// a human reading the row disagree about which seat ran.
func TestWhoamiJSONAgreesWithThePlainForm(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	target := shortIDFixture(t)
	plain, err := Whoami(Options{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := WhoamiJSON(Options{Target: target})
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Seat != plain {
		t.Errorf("seat %q disagrees with whoami %q", rec.Seat, plain)
	}
}

// Silence rather than a synthesised identity, matching the plain form. A
// session with no composition must record no bundle rather than an empty one.
func TestWhoamiJSONSuppressesWithoutAProjection(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	got, err := WhoamiJSON(Options{Target: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("WhoamiJSON = %q, want empty", got)
	}
}

// The fingerprint has to discriminate, or a transcript join cannot tell one
// side of a roster change from the other, which is the whole point.
func TestFingerprintChangesWithTheComposition(t *testing.T) {
	base := bundle.Manifest{
		Role: "scientist", RoleSkill: "role-scientist", RoleSkillDigest: "sha256:aa",
		ModelTier: "frontier", Personalities: []string{"empirical"},
		Content: []bundle.ContentDigest{{ID: "roster:core", Digest: "sha256:bb"}},
	}
	first := bundle.Fingerprint(base)
	if first != bundle.Fingerprint(base) {
		t.Fatal("fingerprint is not deterministic")
	}
	for name, mutate := range map[string]func(*bundle.Manifest){
		"role":        func(m *bundle.Manifest) { m.Role = "platform-eng" },
		"role skill":  func(m *bundle.Manifest) { m.RoleSkillDigest = "sha256:cc" },
		"model tier":  func(m *bundle.Manifest) { m.ModelTier = "commodity" },
		"personality": func(m *bundle.Manifest) { m.Personalities = []string{"grounded"} },
		"boundary":    func(m *bundle.Manifest) { m.Boundaries = []string{"modify-live-backend"} },
		"content":     func(m *bundle.Manifest) { m.Content[0].Digest = "sha256:dd" },
	} {
		changed := base
		changed.Personalities = append([]string(nil), base.Personalities...)
		changed.Content = append([]bundle.ContentDigest(nil), base.Content...)
		mutate(&changed)
		if bundle.Fingerprint(changed) == first {
			t.Errorf("changing the %s did not change the fingerprint", name)
		}
	}
}
