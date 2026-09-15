// Package converge is bare `agent-compose compose`: refresh the roster
// artifact, then run the cascade - the whole host in one verb.
package converge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/cascade"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/catalogmanifest"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/roster"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/skillmount"
)

// Options controls host compose-layout reporting and forced application.
type Options struct {
	Reapply bool
	Verbose bool
}

// Run refreshes the roster from the selected person plus configured overlays,
// then cascades. Absent config stays the documented no-op.
func Run(paths cascade.Paths, opts Options, stdout, stderr io.Writer) int {
	cascadeOpts := cascade.RunOptions{
		Reapply: opts.Reapply,
		Verbose: opts.Verbose,
	}
	if _, err := os.Stat(paths.Config); err != nil {
		return cascade.Run(paths, cascadeOpts, stdout, stderr)
	}
	cfg, err := cascade.LoadConfig(paths.Config)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	var catalogs []skillmount.Catalog
	if cfg.SkillCatalogManifest != "" {
		manifestPath := cascade.ResolveConfiguredPath(
			cfg.SkillCatalogManifest,
			paths.Config,
			paths.Home,
		)
		local, err := catalogmanifest.Load(manifestPath)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
		for _, catalog := range local {
			catalogs = append(catalogs, skillmount.Catalog{
				Path:   catalog.Path,
				Source: catalog.Source.String(),
			})
		}
		if opts.Verbose {
			fmt.Fprintf(stdout, "catalog local=%d\n", len(local))
		}
	}

	p, err := person.Load()
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	if cfg.PersonSource != "" {
		sourcePath := cascade.ResolveConfiguredPath(cfg.PersonSource, paths.Config, paths.Home)
		p, err = person.LoadDirectory(sourcePath)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: %v\n", err)
			return 1
		}
	}
	var sources []*schema.Source
	for _, sourcePath := range cfg.RosterSources {
		src, err := schema.LoadSource(sourcePath)
		if err != nil {
			fmt.Fprintf(stderr, "agent-compose: warning: roster source %s: %v (skipped)\n", sourcePath, err)
			continue
		}
		sources = append(sources, src)
	}
	outDir := filepath.Join(filepath.Dir(paths.Config), "sources", "personality")
	files, err := roster.Render(p, sources, outDir)
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	result, err := project.ApplyOwned(outDir, files, "roster", p.ProviderID())
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	if opts.Verbose {
		fmt.Fprintf(stdout, "roster  %s (%d files)\n", outDir, len(result.Files))
	}
	catalogs = append(catalogs, skillmount.Catalog{
		Path: filepath.Join(outDir, ".agents", "skills"),
	})

	if code := cascade.Run(paths, cascadeOpts, stdout, stderr); code != 0 {
		return code
	}
	manifestPath := filepath.Join(filepath.Dir(paths.Composed), "repository-plan.yaml")
	skills, err := skillmount.ApplyWithCatalogs(
		manifestPath,
		cascade.ResolveSkillLoadPoints(cfg),
		filepath.Dir(paths.Config),
		catalogs,
	)
	for _, warning := range skills.Warnings {
		fmt.Fprintf(stderr, "agent-compose: warning: %s (skipped)\n", warning)
	}
	if err != nil {
		fmt.Fprintf(stderr, "agent-compose: %v\n", err)
		return 1
	}
	if opts.Verbose && skills.LoadPoints > 0 {
		fmt.Fprintf(stdout, "skills  managed=%d load-points=%d verified=%d linked=%d removed=%d preserved=%d\n",
			skills.Managed, skills.LoadPoints, skills.Verified, skills.Linked, skills.Removed, skills.Skipped)
	}
	return 0
}
