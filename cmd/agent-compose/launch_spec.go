package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/launch"
	"github.com/coilyco/agent-compose/v2/internal/nativelaunch"
)

// launchSpecFormat versions the document `launch --spec-out` writes. It carries
// no harness flags, settings path, or MCP config: the launcher builds those.
const launchSpecFormat = "agent-compose.launch-spec.v1"

type launchSpec struct {
	Format       string            `json:"format"`
	Role         string            `json:"role"`
	Harness      string            `json:"harness"`
	ModelTier    string            `json:"model_tier"`
	SeatName     string            `json:"seat_name"`
	BundleDir    string            `json:"bundle_dir"`
	BundleReused bool              `json:"bundle_reused"`
	Projected    int               `json:"projected"`
	Warnings     []string          `json:"warnings"`
	RuntimeHome  string            `json:"runtime_home"`
	EnvSet       map[string]string `json:"env_set"`
	EnvUnset     []string          `json:"env_unset"`
}

// buildLaunchSpec records the environment the exec path would have applied, so
// a launcher reproduces it without reading agent-compose's source.
func buildLaunchSpec(role, harness, runtimeHome string, childDepth int, result *nativelaunch.Result) launchSpec {
	env := map[string]string{launch.EnvSentinel: "1"}
	for _, pair := range append(
		append(roleAttributionEnv(role), launch.DepthEnv(childDepth)),
		sessionBundleEnv(result.BundleDir, harness)...,
	) {
		if name, value, ok := strings.Cut(pair, "="); ok {
			env[name] = value
		}
	}
	warnings := []string{}
	if result.Composition != nil && result.Composition.Resolution != nil {
		warnings = append(warnings, result.Composition.Resolution.Warnings...)
	}
	return launchSpec{
		Format:       launchSpecFormat,
		Role:         role,
		Harness:      harness,
		ModelTier:    result.ModelTier,
		SeatName:     result.SeatName,
		BundleDir:    result.BundleDir,
		BundleReused: result.BundleReused,
		Projected:    result.Projected,
		Warnings:     warnings,
		RuntimeHome:  runtimeHome,
		EnvSet:       env,
		EnvUnset:     append([]string(nil), nativeLaunchSelectorEnv...),
	}
}

// writeLaunchSpec replaces the file atomically, so a launcher never reads half.
func writeLaunchSpec(path string, spec launchSpec) error {
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".launch-spec-*")
	if err != nil {
		return fmt.Errorf("write launch spec: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write launch spec: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write launch spec: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write launch spec: %w", err)
	}
	return nil
}
