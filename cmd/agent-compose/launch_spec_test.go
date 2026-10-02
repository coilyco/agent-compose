package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/compose"
	"github.com/coilyco/agent-compose/v2/internal/launch"
	"github.com/coilyco/agent-compose/v2/internal/nativelaunch"
	"github.com/coilyco/agent-compose/v2/internal/resolver"
)

func TestParseNativeLaunchFlagsReadsSpecOut(t *testing.T) {
	t.Parallel()
	for name, args := range map[string][]string{
		"separate": {"--spec-out", "/tmp/spec.json", "scientist", "claude"},
		"inline":   {"--spec-out=/tmp/spec.json", "scientist", "claude"},
		"after":    {"--nested", "--spec-out", "/tmp/spec.json", "scientist", "claude"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rest, specOut, _, err := parseNativeLaunchFlags(args)
			if err != nil || specOut != "/tmp/spec.json" || !reflect.DeepEqual(rest, []string{"scientist", "claude"}) {
				t.Fatalf("rest=%v specOut=%q err=%v", rest, specOut, err)
			}
		})
	}
	for name, args := range map[string][]string{
		"missing": {"--spec-out"},
		"empty":   {"--spec-out=", "scientist", "claude"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, _, _, err := parseNativeLaunchFlags(args); err == nil {
				t.Fatal("a spec-out with no path parsed")
			}
		})
	}
}

// The spec must carry exactly the environment the exec path applies, and
// nothing a launcher would have to build per harness.
func TestLaunchSpecCarriesTheExecEnvironment(t *testing.T) {
	result := &nativelaunch.Result{
		Composition:  &compose.Result{Resolution: &resolver.Resolution{Warnings: []string{"w1"}}},
		BundleDir:    "/bundles/abc",
		BundleReused: true,
		Projected:    3,
		ModelTier:    "frontier",
		SeatName:     "Frog-Ox",
	}
	spec := buildLaunchSpec("scientist", "codex", "/session/home", 1, result)
	want := map[string]string{
		launch.EnvSentinel:        "1",
		launch.EnvDepth:           "1",
		launch.AttributionRoleEnv: "scientist",
		launch.SessionBundleEnv:   "/bundles/abc",
		launch.SessionLayoutEnv:   "codex",
	}
	if !reflect.DeepEqual(spec.EnvSet, want) {
		t.Fatalf("env_set = %v, want %v", spec.EnvSet, want)
	}
	if !reflect.DeepEqual(spec.EnvUnset, nativeLaunchSelectorEnv) {
		t.Fatalf("env_unset = %v, want %v", spec.EnvUnset, nativeLaunchSelectorEnv)
	}

	path := filepath.Join(t.TempDir(), "spec.json")
	if err := writeLaunchSpec(path, spec); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]any{
		"format": launchSpecFormat, "role": "scientist", "harness": "codex",
		"model_tier": "frontier", "seat_name": "Frog-Ox", "bundle_dir": "/bundles/abc",
		"bundle_reused": true, "projected": float64(3), "runtime_home": "/session/home",
	} {
		if decoded[key] != value {
			t.Errorf("%s = %#v, want %#v", key, decoded[key], value)
		}
	}
	if warnings, _ := decoded["warnings"].([]any); len(warnings) != 1 || warnings[0] != "w1" {
		t.Errorf("warnings = %#v", decoded["warnings"])
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("atomic write left %d entries, want only the spec", len(entries))
	}
}

func TestLaunchSpecWarningsIsNeverNull(t *testing.T) {
	t.Parallel()
	spec := buildLaunchSpec("scientist", "claude", "", 0, &nativelaunch.Result{})
	raw, _ := json.Marshal(spec)
	var decoded map[string]any
	_ = json.Unmarshal(raw, &decoded)
	if _, ok := decoded["warnings"].([]any); !ok {
		t.Fatalf("warnings = %#v, want an empty array", decoded["warnings"])
	}
}

func TestParseNativeLaunchFlagsReadsNoPause(t *testing.T) {
	t.Parallel()
	rest, _, noPause, err := parseNativeLaunchFlags([]string{"--no-pause", "--nested", "scientist", "claude", "--no-pause"})
	if err != nil || !noPause || !reflect.DeepEqual(rest, []string{"scientist", "claude", "--no-pause"}) {
		t.Fatalf("rest=%v noPause=%v err=%v, want the head flag read and the harness's own left alone", rest, noPause, err)
	}
	if _, _, noPause, _ := parseNativeLaunchFlags([]string{"scientist", "claude"}); noPause {
		t.Fatal("no flag must mean the gate still pauses")
	}
}
