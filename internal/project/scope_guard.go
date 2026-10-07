package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// repoScopeSearchDepth reaches <projects>/<org>/<repo>, the deepest shelf a
// directory of repositories takes on a fleet host.
const repoScopeSearchDepth = 2

// refuseAboveRepositories keeps a repo-scope projection out of a directory that
// holds repositories. A repository nesting others is a valid target (docs/projection.md).
func refuseAboveRepositories(targetDir string) error {
	if isRepository(targetDir) {
		return nil
	}
	found, err := findRepository(targetDir, repoScopeSearchDepth)
	if err != nil {
		return err
	}
	if found == "" {
		return nil
	}
	return fmt.Errorf(
		"refusing to project the role into %s, which holds the repository %s, so every session beneath it would inherit the role: project into a repository, or use home scope",
		targetDir,
		found,
	)
}

// findRepository returns the first repository within depth levels of dir, following
// symlinked children since retired org directories are symlinks.
func findRepository(dir string, depth int) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return "", nil
		}
		return "", err
	}
	for _, entry := range entries {
		child := filepath.Join(dir, entry.Name())
		if isRepository(child) {
			return child, nil
		}
		if depth > 1 {
			if info, err := os.Stat(child); err != nil || !info.IsDir() {
				continue
			}
			if found, err := findRepository(child, depth-1); err != nil || found != "" {
				return found, err
			}
		}
	}
	return "", nil
}

// isRepository reports a .git entry, a directory in a clone and a file in a worktree.
func isRepository(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}
