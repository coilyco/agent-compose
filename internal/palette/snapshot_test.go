package palette

import (
	"os"
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/person"
)

func TestRenderSnapshotMatchesTheCommittedRecord(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	// The path is repo-root relative and this test runs in the package dir.
	if err := CheckSnapshot("role-palette.txt", rendered); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestRenderSnapshotCoversEveryRoleAndIsStable(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	first, err := RenderSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := RenderSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(again) {
		t.Fatal("snapshot rendering is not stable")
	}
	for _, name := range p.RoleOrder {
		line := name + " // " + p.Roles[name].FavoriteColor + " //"
		if !strings.Contains(string(first), line) {
			t.Fatalf("snapshot omits %q", line)
		}
	}
}

func TestCheckSnapshotReportsDrift(t *testing.T) {
	path := t.TempDir() + "/role-palette.txt"
	if err := WriteSnapshot(path, []byte("stale\n")); err != nil {
		t.Fatal(err)
	}
	err := CheckSnapshot(path, []byte("fresh\n"))
	if err == nil || !strings.Contains(err.Error(), "is stale") {
		t.Fatalf("drift error = %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("check must not rewrite the file: %v", statErr)
	}
}
