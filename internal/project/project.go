// Package project places bundle content at harness load points. The harness
// names and paths come from the layouts table, so this package holds none.
package project

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/layouts"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

type LoadPoints = layouts.LoadPoints

type Layout = layouts.Layout

const (
	ScopeRepo = "repo"
	ScopeHome = "home"
)

// Registries splits the layouts table into its repo- and home-scoped views.
func Registries() (repo, home map[string]Layout, err error) {
	table, err := layouts.Load()
	if err != nil {
		return nil, nil, err
	}
	repo, home = map[string]Layout{}, map[string]Layout{}
	for name, harness := range table {
		repo[name] = harness.Repo
		home[name] = harness.Home
	}
	return repo, home, nil
}

const sidecarRel = ".agent-compose/projection.json"

// Projection records the immutable bundle and files owned at one target.
type Projection struct {
	Layout string   `json:"layout"`
	Bundle string   `json:"bundle"`
	Files  []string `json:"files"`
}

type Result struct {
	Layout string
	Files  []string
}

func Project(bundleDir, layoutName, targetDir string) (*Result, error) {
	return ProjectScoped(bundleDir, layoutName, targetDir, ScopeRepo)
}

// ProjectScoped selects repo-relative or home-relative load points; home
// scope treats targetDir as the home root.
func ProjectScoped(bundleDir, layoutName, targetDir, scope string) (*Result, error) {
	repo, home, err := Registries()
	if err != nil {
		return nil, err
	}
	registry := repo
	switch scope {
	case ScopeRepo:
	case ScopeHome:
		registry = home
	default:
		return nil, fmt.Errorf("unknown scope %q; scopes: %s, %s", scope, ScopeRepo, ScopeHome)
	}
	layout, ok := registry[layoutName]
	if !ok {
		return nil, fmt.Errorf("unknown layout %q; v0.1 layouts: %s", layoutName, strings.Join(registryNames(registry), ", "))
	}
	verification, err := bundle.Verify(bundleDir)
	if err != nil {
		return nil, err
	}
	if err := ensureSeparateTarget(bundleDir, targetDir); err != nil {
		return nil, err
	}
	manifest := verification.Manifest
	var points *LoadPoints
	switch manifest.Delivery.Mode {
	case schema.DeliveryNativeSkills:
		points = layout.Native
	case schema.DeliveryCompiled:
		points = layout.Compiled
	}
	if points == nil {
		return nil, fmt.Errorf("layout %q does not support bundles that deliver %s",
			layoutName, manifest.Delivery.Mode)
	}

	plan, err := buildPlan(bundleDir, manifest, *points)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for rel, srcAbs := range plan {
		raw, err := os.ReadFile(srcAbs)
		if err != nil {
			return nil, err
		}
		files[rel] = raw
	}
	release, err := lockTarget(targetDir)
	if err != nil {
		return nil, err
	}
	defer release()
	return applyOwned(targetDir, files, layoutName, bundleDir)
}

// ApplyOwned writes a file set under target with the projection ownership
// rules: sidecar-tracked, never clobbering foreign files, replacing stale.
func ApplyOwned(targetDir string, files map[string][]byte, label, origin string) (*Result, error) {
	release, err := lockTarget(targetDir)
	if err != nil {
		return nil, err
	}
	defer release()
	return applyOwned(targetDir, files, label, origin)
}

// buildPlan maps target-relative slash paths to absolute bundle sources.
func buildPlan(bundleDir string, manifest *bundle.Manifest, points LoadPoints) (map[string]string, error) {
	plan := map[string]string{}
	switch manifest.Delivery.Mode {
	case schema.DeliveryCompiled:
		if manifest.Delivery.CompiledContext == "" {
			return nil, fmt.Errorf("bundle manifest names no compiled_context entry point")
		}
		plan[points.Instructions] = filepath.Join(bundleDir, filepath.FromSlash(manifest.Delivery.CompiledContext))
	case schema.DeliveryNativeSkills:
		if manifest.Delivery.Instructions == "" || manifest.Delivery.SkillsRoot == "" {
			return nil, fmt.Errorf("bundle manifest names no native entry points")
		}
		plan[points.Instructions] = filepath.Join(bundleDir, filepath.FromSlash(manifest.Delivery.Instructions))
		skillsRoot := filepath.Join(bundleDir, filepath.FromSlash(manifest.Delivery.SkillsRoot))
		sourceDirs, err := os.ReadDir(skillsRoot)
		if err != nil {
			return nil, err
		}
		for _, sourceDir := range sourceDirs {
			if !sourceDir.IsDir() {
				continue
			}
			base := filepath.Join(skillsRoot, sourceDir.Name())
			err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				rel, err := filepath.Rel(base, path)
				if err != nil {
					return err
				}
				plan[points.SkillsDir+"/"+filepath.ToSlash(rel)] = path
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return plan, nil
}

// Validate reports whether the target holds a complete previous projection,
// so a failed refresh can fall back to it as last-known-good.
func Validate(targetDir string) error {
	s := ReadProjection(targetDir)
	if len(s.Files) == 0 {
		return fmt.Errorf("no projection recorded under %s", targetDir)
	}
	for _, rel := range s.Files {
		if _, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(rel))); err != nil {
			return fmt.Errorf("recorded projection is incomplete: %w", err)
		}
	}
	return nil
}

// ReadProjection loads the ownership sidecar at targetDir. An absent or
// malformed sidecar returns an empty projection for existing callers.
func ReadProjection(targetDir string) Projection {
	var s Projection
	raw, err := os.ReadFile(filepath.Join(targetDir, filepath.FromSlash(sidecarRel)))
	if err != nil {
		return s
	}
	if json.Unmarshal(raw, &s) != nil {
		return Projection{}
	}
	return s
}

// FindProjection walks from start toward the filesystem root and returns the
// nearest recorded projection plus the target directory that owns it.
func FindProjection(start string) (Projection, string) {
	candidate, err := filepath.Abs(start)
	if err != nil {
		return Projection{}, ""
	}
	if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
		candidate = filepath.Dir(candidate)
	}
	for {
		projection := ReadProjection(candidate)
		if projection.Bundle != "" && projection.Layout != "" {
			return projection, candidate
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return Projection{}, ""
		}
		candidate = parent
	}
}

func pruneEmptyDirs(dir, stop string) {
	for {
		if dir == stop || len(dir) <= len(stop) {
			return
		}
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func registryNames(registry map[string]Layout) []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
