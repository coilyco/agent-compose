package bundle_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/coilyco/agent-compose/v2/internal/bundle"
)

func TestExportIsReproducibleAndPortable(t *testing.T) {
	dir := composeBundle(t, "native.kdl")
	first := filepath.Join(t.TempDir(), "first.tar.gz")
	second := filepath.Join(t.TempDir(), "second.tar.gz")
	if err := bundle.Export(dir, first); err != nil {
		t.Fatal(err)
	}
	if err := bundle.Export(dir, second); err != nil {
		t.Fatal(err)
	}
	firstRaw, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstRaw, secondRaw) {
		t.Fatal("identical verified input produced different archives")
	}

	file, err := os.Open(first)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	if (!gz.ModTime.IsZero() && !gz.ModTime.Equal(time.Unix(0, 0))) || gz.OS != 255 {
		t.Fatalf("gzip metadata is not normalized: time=%s os=%d", gz.ModTime, gz.OS)
	}
	reader := tar.NewReader(gz)
	var names []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
		if strings.Contains(header.Name, `\`) ||
			strings.HasPrefix(header.Name, "/") ||
			strings.Contains(header.Name, "../") ||
			!header.ModTime.Equal(time.Unix(0, 0)) ||
			header.Mode != 0o644 {
			t.Fatalf("archive header is not portable: %+v", header)
		}
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("archive paths are not sorted: %v", names)
	}
	if !containsName(names, "manifest.json") {
		t.Fatalf("archive omitted verified manifest: %v", names)
	}
}

func TestExportVerificationFailureCreatesNoArchive(t *testing.T) {
	dir := copyBundle(t, composeBundle(t, "native.kdl"))
	if err := os.Remove(filepath.Join(dir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "invalid.tar.gz")
	if err := bundle.Export(dir, output); err == nil ||
		!strings.Contains(err.Error(), "verify bundle before export") {
		t.Fatalf("invalid bundle export error = %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("verification failure created archive: %v", err)
	}
}

func TestExportRejectsSymlinkBeforeOpeningOutput(t *testing.T) {
	dir := copyBundle(t, composeBundle(t, "native.kdl"))
	link := filepath.Join(dir, "linked")
	if err := os.Symlink(filepath.Join(dir, "manifest.json"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	output := filepath.Join(t.TempDir(), "linked.tar.gz")
	if err := bundle.Export(dir, output); err == nil ||
		!strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked bundle export error = %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("symlink failure created archive: %v", err)
	}
}

func containsName(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}
