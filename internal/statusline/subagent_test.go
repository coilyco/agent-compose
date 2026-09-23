package statusline

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
)

// writeProjectedRole stages a projection whose bundle names one seat, which
// is all the row renderer reads.
func writeProjectedRole(t *testing.T, target, layout, role, seat string) {
	t.Helper()
	bundleDir := t.TempDir()
	writeJSON(t, filepath.Join(target, ".agent-compose", "projection.json"), project.Projection{
		Layout: layout, Bundle: bundleDir,
	})
	writeJSON(t, filepath.Join(bundleDir, "manifest.json"), bundle.Manifest{
		Format: "agent-compose.bundle", Role: role, ModelTier: "frontier",
		Color: "#b39258",
		Identity: bundle.RoleIdentity{
			Person: "core",
			Seats: []person.Seat{
				{Key: layout, Name: seat},
			},
			Personalities: []bundle.IdentityPersonality{
				{Name: "tenacious", Color: "#d98e48", Emblem: person.Emblem{Emoji: "🧭"}},
				{Name: "tenacious", Color: "#8f8c47", Emblem: person.Emblem{Emoji: "⛏️"}},
			},
		},
	})
	writeJSON(t, filepath.Join(bundleDir, "trace.json"), bundle.Trace{
		Format:    "agent-compose.trace",
		Providers: []resolver.ProviderReport{{Outcome: resolver.OutcomeSelected, Skills: 3}},
	})
}

func decodeRows(t *testing.T, rendered string) []SubagentRow {
	t.Helper()
	var rows []SubagentRow
	for _, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if line == "" {
			continue
		}
		var row SubagentRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("row %q is not JSON: %v", line, err)
		}
		if row.ID == "" || row.Content == "" {
			t.Fatalf("row %q has an empty field, which the harness rejects", line)
		}
		rows = append(rows, row)
	}
	return rows
}

func TestRenderSubagentsDecoratesEveryRowFromItsOwnDirectory(t *testing.T) {
	engineer := t.TempDir()
	writeProjectedRole(t, engineer, "claude", "platform-eng", "Angie")
	qa := t.TempDir()
	writeProjectedRole(t, qa, "claude", "qa", "Quail")

	request, err := json.Marshal(SubagentRequest{
		Columns: 80,
		Tasks: []SubagentTask{
			{ID: "task-1", Type: "local_agent", Status: "running", CWD: engineer},
			{ID: "task-2", Type: "local_agent", Status: "running", CWD: qa},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := RenderSubagents(strings.NewReader(string(request)), Options{})
	if err != nil {
		t.Fatal(err)
	}
	rows := decodeRows(t, rendered)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if want := "🧭 ⛏️ Angie // platform-eng@claude"; rows[0].Content != want {
		t.Errorf("row 1 content = %q, want %q", rows[0].Content, want)
	}
	if want := "🧭 ⛏️ Quail // qa@claude"; rows[1].Content != want {
		t.Errorf("row 2 content = %q, want %q", rows[1].Content, want)
	}
}

func TestRenderSubagentsSkipsRowsWithoutAProjection(t *testing.T) {
	projected := t.TempDir()
	writeProjectedRole(t, projected, "claude", "platform-eng", "Angie")

	request, err := json.Marshal(SubagentRequest{Tasks: []SubagentTask{
		{ID: "task-1", CWD: projected},
		{ID: "task-2", CWD: t.TempDir()},
		{ID: "", CWD: projected},
	}})
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := RenderSubagents(strings.NewReader(string(request)), Options{})
	if err != nil {
		t.Fatal(err)
	}
	rows := decodeRows(t, rendered)
	if len(rows) != 1 || rows[0].ID != "task-1" {
		t.Fatalf("rows = %+v, want only task-1", rows)
	}
}

func TestRenderSubagentsFallsBackToTargetWhenRowHasNoDirectory(t *testing.T) {
	projected := t.TempDir()
	writeProjectedRole(t, projected, "codex", "sysadmin", "Vera")

	request, err := json.Marshal(SubagentRequest{Tasks: []SubagentTask{{ID: "task-1"}}})
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := RenderSubagents(
		strings.NewReader(string(request)),
		Options{Target: projected},
	)
	if err != nil {
		t.Fatal(err)
	}
	rows := decodeRows(t, rendered)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if want := "🧭 ⛏️ Vera // sysadmin@codex"; rows[0].Content != want {
		t.Errorf("content = %q, want %q", rows[0].Content, want)
	}
}

func TestRenderSubagentsReportsAnUnreadableCompositionPerRow(t *testing.T) {
	target := t.TempDir()
	writeJSON(t, filepath.Join(target, ".agent-compose", "projection.json"), project.Projection{
		Layout: "claude", Bundle: t.TempDir(),
	})

	request, err := json.Marshal(SubagentRequest{Tasks: []SubagentTask{{ID: "task-1", CWD: target}}})
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := RenderSubagents(strings.NewReader(string(request)), Options{})
	if err != nil {
		t.Fatal(err)
	}
	rows := decodeRows(t, rendered)
	if len(rows) != 1 || rows[0].Content != "⚠ composition unreadable" {
		t.Fatalf("rows = %+v, want one warning row", rows)
	}
}

func TestRenderSubagentsPaintsOnlyWhenColorIsRequested(t *testing.T) {
	target := t.TempDir()
	writeProjectedRole(t, target, "claude", "platform-eng", "Angie")
	request, err := json.Marshal(SubagentRequest{Tasks: []SubagentTask{{ID: "task-1", CWD: target}}})
	if err != nil {
		t.Fatal(err)
	}

	plain, err := RenderSubagents(strings.NewReader(string(request)), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("plain row carries ANSI: %q", plain)
	}
	painted, err := RenderSubagents(
		strings.NewReader(string(request)),
		Options{Color: true, TrueColor: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(painted, "38;2;179;146;88") {
		t.Errorf("painted row omits the role color: %q", painted)
	}
}

func TestRenderSubagentsRejectsAMalformedRequest(t *testing.T) {
	if _, err := RenderSubagents(strings.NewReader("not json"), Options{}); err == nil {
		t.Fatal("malformed request did not fail")
	}
}

func TestRenderSubagentsEmitsNothingForAnEmptyTickList(t *testing.T) {
	rendered, err := RenderSubagents(strings.NewReader(`{"columns":80,"tasks":[]}`), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rendered != "" {
		t.Fatalf("empty tick rendered %q", rendered)
	}
}

// The renderer must tolerate the fields it does not read, since the harness
// sends more than this package declares.
func TestRenderSubagentsIgnoresUnknownTickFields(t *testing.T) {
	target := t.TempDir()
	writeProjectedRole(t, target, "claude", "platform-eng", "Angie")
	raw := `{"columns":80,"version":"2.1.221","tasks":[{"id":"task-1","cwd":` +
		mustJSON(t, target) +
		`,"model":"opus","effort":"high","tokenSamples":[1,2],"unknown":{"a":1}}]}`

	rendered, err := RenderSubagents(strings.NewReader(raw), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rows := decodeRows(t, rendered); len(rows) != 1 {
		t.Fatalf("rows = %+v, want 1", rows)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
