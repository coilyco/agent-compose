package nativelaunch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/agentid"
)

func shortIDRefresh(t *testing.T) *Result {
	t.Helper()
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "provider")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)

	result, err := Refresh(Options{
		Role:      "frontend-eng",
		Harness:   "claude",
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// The seat name becomes Claude Code's `--name`, the launch-time CLI surface.
func TestSeatNameCarriesTheShortID(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	result := shortIDRefresh(t)
	if result.SeatName != "Imp-Dragonfly (Frontend Engineer) uz86" {
		t.Errorf("seat name = %q, want it annotated with uz86", result.SeatName)
	}
}

// A bundle is read by later sessions, so a baked id would name the wrong agent
// and fork the cache. See docs/whoami.md.
func TestShortIDNeverReachesPersistedBundleArtifacts(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "uz86")
	result := shortIDRefresh(t)

	if !strings.Contains(result.SeatName, "uz86") {
		t.Fatal("fixture did not resolve a session id, so this guard proves nothing")
	}

	var checked int
	err := filepath.WalkDir(result.BundleDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		checked++
		if strings.Contains(string(raw), "uz86") {
			rel, _ := filepath.Rel(result.BundleDir, path)
			t.Errorf("bundle artifact %s carries the session id; a later session would read the wrong name", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("walked no bundle files, so this guard proves nothing")
	}
}
