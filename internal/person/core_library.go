package person

import (
	"fmt"
	"io/fs"
	"strings"
	"testing/fstest"

	"github.com/coilyco/agent-compose/v2/internal/schema"
)

// CoreLibraryRoot admits the mounted core personalities wherever a
// personality-library root is accepted. See docs/personality.md.
const CoreLibraryRoot = schema.CoreLibraryRoot

const coreLibraryLabel = "core personality library"

// IsCoreLibrary reports whether an admitted personality-library value names the
// mounted core library rather than a local directory, so callers skip resolution.
func IsCoreLibrary(root string) bool {
	return strings.TrimSpace(root) == CoreLibraryRoot
}

// coreLibrarySource projects the mounted personalities onto the library layout.
// Only shared disposition crosses. Roles, seats, identity, and the invariant do not.
func coreLibrarySource() (fs.FS, error) {
	seed, _, err := rosterSeed()
	if err != nil {
		return nil, err
	}
	projected, _, err := dataLayout(seed, coreLibraryLabel)
	if err != nil {
		return nil, err
	}
	manifest := fmt.Sprintf("library: %s\n", CoreLibraryRoot)
	library := fstest.MapFS{
		"library" + yamlFragmentExt: &fstest.MapFile{Data: []byte(manifest), Mode: 0o644},
	}
	walkErr := fs.WalkDir(projected, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !corePersonalityPath(path) {
			return err
		}
		raw, err := fs.ReadFile(projected, path)
		if err != nil {
			return err
		}
		library[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("%s: %w", coreLibraryLabel, walkErr)
	}
	if len(library) == 1 {
		return nil, fmt.Errorf("%s: the mounted roster declares no personalities", coreLibraryLabel)
	}
	return library, nil
}

func corePersonalityPath(path string) bool {
	return strings.HasPrefix(path, "personalities/") ||
		strings.HasPrefix(path, "definitions/skills/personality-")
}
