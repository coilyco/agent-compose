// Package treehash digests a skill or source tree by content, so two copies of
// one commit compare equal. Extracted from the resolver when the catalogue
// superset needed the same rule to tell a duplicate from a collision.
package treehash

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strings"
)

// Digest hashes every file beneath root, path and length included, so a rename
// changes the result. A symlink is rejected rather than followed.
func Digest(files fs.FS, root string) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(files, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinks are invalid inside a source", p)
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, strings.TrimSuffix(root, "/")+"/")
		if rel == p {
			return fmt.Errorf("%s is not beneath %s", p, root)
		}
		raw, err := fs.ReadFile(files, p)
		if err != nil {
			return err
		}
		body := NormalizeEOL(raw)
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(body))
		h.Write(body)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// NormalizeEOL folds CRLF so two checkouts of one commit digest equal, and
// hashes raw when a NUL byte says the content is binary.
func NormalizeEOL(raw []byte) []byte {
	if bytes.IndexByte(raw, 0) >= 0 {
		return raw
	}
	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
}
