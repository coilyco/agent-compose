package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/launch"
)

// The flag is accepted and ignored, so what matters is that it never reaches
// the harness as an argument. agent-compose#403
func TestSplitNativeLaunchFlags(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in       []string
		wantRest []string
	}{
		"no flag":          {[]string{"platform-eng", "claude"}, []string{"platform-eng", "claude"}},
		"nested":           {[]string{"--nested", "scientist", "claude"}, []string{"scientist", "claude"}},
		"harness dash":     {[]string{"platform-eng", "claude", "--nested"}, []string{"platform-eng", "claude", "--nested"}},
		"nothing at all":   {nil, nil},
		"harness flag arg": {[]string{"scientist", "claude", "-p", "go"}, []string{"scientist", "claude", "-p", "go"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if rest := splitNativeLaunchFlags(tc.in); !reflect.DeepEqual(rest, tc.wantRest) {
				t.Fatalf("rest = %#v, want %#v", rest, tc.wantRest)
			}
		})
	}
}

func writeProjectionSidecar(t *testing.T, target, bundleDir string) {
	t.Helper()
	dir := filepath.Join(target, ".agent-compose")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"layout":"claude","bundle":"` + bundleDir + `","files":["CLAUDE.md"]}`
	if err := os.WriteFile(filepath.Join(dir, "projection.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRefuseOwnProjectionReplacementAcceptsAnUnclaimedTarget(t *testing.T) {
	target := t.TempDir()
	t.Setenv(launch.SessionBundleEnv, "/bundles/parent")
	if err := refuseOwnProjectionReplacement(target); err != nil {
		t.Fatalf("empty target refused: %v", err)
	}
}

func TestRefuseOwnProjectionReplacementAcceptsAnotherSessionsProjection(t *testing.T) {
	target := t.TempDir()
	writeProjectionSidecar(t, target, "/bundles/stale")
	t.Setenv(launch.SessionBundleEnv, "/bundles/parent")
	if err := refuseOwnProjectionReplacement(target); err != nil {
		t.Fatalf("a projection this session does not run on was refused: %v", err)
	}
}

func TestRefuseOwnProjectionReplacementRefusesThisSessionsProjection(t *testing.T) {
	target := t.TempDir()
	writeProjectionSidecar(t, target, "/bundles/parent")
	t.Setenv(launch.SessionBundleEnv, "/bundles/parent")
	if err := refuseOwnProjectionReplacement(target); err == nil {
		t.Fatal("a nested launch was allowed to replace its own session's load points")
	}
}

func TestRefuseOwnProjectionReplacementFailsClosedWithoutASessionMarker(t *testing.T) {
	target := t.TempDir()
	writeProjectionSidecar(t, target, "/bundles/unknown")
	t.Setenv(launch.SessionBundleEnv, "")
	if err := refuseOwnProjectionReplacement(target); err == nil {
		t.Fatal("an unattributable projection was accepted as safe to replace")
	}
}

// One sentinel, two call sites. They said different things and did different
// things, and only the saying was ever checked. agent-compose#348
func TestBothSentinelCallSitesSkipTheirStep(t *testing.T) {
	t.Parallel()
	if nestedLaunchSkipsConverge(0) {
		t.Fatal("a top-level launch must converge")
	}
	if !nestedLaunchSkipsConverge(1) {
		t.Fatal("a nested launch must skip the converge its parent already ran")
	}
	for step, want := range map[string]string{
		"refresh":  "agent-compose: nested launch detected; skipping refresh",
		"converge": "agent-compose: nested launch detected; skipping converge",
	} {
		if got := nestedLaunchNotice(step); got != want {
			t.Fatalf("notice(%q) = %q, want %q", step, got, want)
		}
	}
}
