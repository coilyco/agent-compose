// Package catalogmanifest loads AOS-hydrated skill roots without granting
// Agent Compose ownership of Git, cache, or host convergence policy.
package catalogmanifest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const Format = "aos.catalogues.v1"

type document struct {
	Format     string  `json:"format"`
	Forge      string  `json:"forge"`
	Catalogues []entry `json:"catalogues"`
}

type entry struct {
	Source string `json:"source"`
	Forge  string `json:"forge"`
	Path   string `json:"path"`
	Commit string `json:"commit"`
}

// Catalog is one verified local skill root in declaration order. Source
// travels with Path so a downstream record can name where a skill came from.
type Catalog struct {
	Path   string
	Source Source
}

var fullCommit = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

// Load validates one AOS catalogue manifest and returns local roots only.
func Load(path string) ([]Catalog, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("skill catalogue manifest path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read skill catalogue manifest %s: %w", path, err)
	}
	defer file.Close()

	var manifest document
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parse skill catalogue manifest %s: %w", path, err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return nil, fmt.Errorf("parse skill catalogue manifest %s: %w", path, err)
	}
	if manifest.Format != Format {
		return nil, fmt.Errorf(
			"skill catalogue manifest %s has format %q, want %q",
			path,
			manifest.Format,
			Format,
		)
	}
	defaultForge, err := parseForge(manifest.Forge)
	if err != nil {
		return nil, fmt.Errorf("skill catalogue manifest %s: %w", path, err)
	}
	catalogs := make([]Catalog, 0, len(manifest.Catalogues))
	for index, item := range manifest.Catalogues {
		entryForge, err := parseForge(item.Forge)
		if err != nil {
			return nil, fmt.Errorf(
				"skill catalogue manifest %s entry %d: %w",
				path,
				index,
				err,
			)
		}
		fallback := entryForge
		if fallback == "" {
			fallback = defaultForge
		}
		source, err := ParseSource(item.Source, fallback)
		if err != nil {
			return nil, fmt.Errorf(
				"skill catalogue manifest %s entry %d %w",
				path,
				index,
				err,
			)
		}
		if !filepath.IsAbs(item.Path) {
			return nil, fmt.Errorf(
				"skill catalogue manifest %s entry %d path must be absolute",
				path,
				index,
			)
		}
		clean := filepath.Clean(item.Path)
		info, err := os.Stat(clean)
		if err != nil {
			return nil, fmt.Errorf(
				"inspect skill catalogue manifest %s entry %d: %w",
				path,
				index,
				err,
			)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf(
				"skill catalogue manifest %s entry %d path is not a directory",
				path,
				index,
			)
		}
		if !fullCommit.MatchString(item.Commit) {
			return nil, fmt.Errorf(
				"skill catalogue manifest %s entry %d commit is not a full Git object ID",
				path,
				index,
			)
		}
		catalogs = append(catalogs, Catalog{Path: clean, Source: source})
	}
	return catalogs, nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("contains trailing JSON")
}
