package person

import (
	"testing"
	"testing/fstest"
)

// Negative control for validateNoPersonOrOwnerLeak: fails the moment a
// person-name line comes back, stays quiet without one. agent-compose#8039.
func TestScanForBannedIdentityTokensCatchesAReintroducedName(t *testing.T) {
	clean := fstest.MapFS{
		"roles/junior-sysadmin/SKILL.md": &fstest.MapFile{
			Data: []byte("Hand the human the forward line and the rollback line together.\n"),
		},
	}
	if violations, err := scanForBannedIdentityTokens(clean); err != nil {
		t.Fatalf("scan clean tree: %v", err)
	} else if len(violations) != 0 {
		t.Fatalf("clean tree flagged a violation: %v", violations)
	}

	leaked := fstest.MapFS{
		"roles/junior-sysadmin/SKILL.md": &fstest.MapFile{
			Data: []byte("Hand Kai the forward line and the rollback line together.\n"),
		},
	}
	violations, err := scanForBannedIdentityTokens(leaked)
	if err != nil {
		t.Fatalf("scan leaked tree: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want exactly one for the reintroduced line", violations)
	}
	if want := "roles/junior-sysadmin/SKILL.md:1: Kai"; violations[0] != want {
		t.Errorf("violation = %q, want %q", violations[0], want)
	}
}

// An owner token (a URL, a repo slug) is caught the same as a person name.
func TestScanForBannedIdentityTokensCatchesAnOwnerToken(t *testing.T) {
	leaked := fstest.MapFS{
		"boundaries/build-foundational-software/SKILL.md": &fstest.MapFile{
			Data: []byte("See forgejo.coilysiren.me/coilyco-bridge/agentic-os-kai for the doctrine.\n"),
		},
	}
	violations, err := scanForBannedIdentityTokens(leaked)
	if err != nil {
		t.Fatalf("scan leaked tree: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one violation for an owner-org URL, got none")
	}
}

// A deployment service name is caught the same as an owner token. The first
// regression was Agent Proxy named in a shared role scope.
func TestScanForBannedIdentityTokensCatchesADeploymentService(t *testing.T) {
	for _, line := range []string{
		"applying routes to the live Agent Proxy and LiteLLM service",
		"launch profiles, aosguard spec values, and ansible roles",
	} {
		leaked := fstest.MapFS{
			"roles/sysadmin-access/SKILL.md": &fstest.MapFile{Data: []byte(line + "\n")},
		}
		violations, err := scanForBannedIdentityTokens(leaked)
		if err != nil {
			t.Fatalf("scan %q: %v", line, err)
		}
		if len(violations) == 0 {
			t.Errorf("no violation for %q", line)
		}
	}
	generic := fstest.MapFS{
		"roles/sysadmin-access/SKILL.md": &fstest.MapFile{
			Data: []byte("permission specification values and the deployment's model gateway\n"),
		},
	}
	if violations, err := scanForBannedIdentityTokens(generic); err != nil || len(violations) != 0 {
		t.Fatalf("generic wording flagged: %v %v", violations, err)
	}
}
