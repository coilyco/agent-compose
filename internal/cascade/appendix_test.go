package cascade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// appendixEnv writes one doctrine source and returns the config prefix that
// selects it, so each test states only the appendix it is about.
func appendixEnv(t *testing.T) (env, string) {
	t.Helper()
	e := newEnv(t)
	source := e.write(t, "doctrine/AGENTS.md", "# Doctrine\n\nBase body.\n")
	return e, "sources:\n  - " + source + "\n"
}

func composedBody(t *testing.T, e env) string {
	t.Helper()
	raw, err := os.ReadFile(e.paths.Composed)
	if err != nil {
		t.Fatalf("read composed output: %v", err)
	}
	return string(raw)
}

func TestInlineAppendixLandsAfterEverySource(t *testing.T) {
	e, sources := appendixEnv(t)
	e.config(t, sources+"appendix:\n  - text: |\n      ## Checkin\n\n      Open the dashboard first.\n")
	if code, _, errOut := e.run(t, false); code != 0 {
		t.Fatalf("run failed: %d %s", code, errOut)
	}
	body := composedBody(t, e)
	fence := "<!-- appendix: agent-compose.yaml entry 1 -->"
	if !strings.Contains(body, fence) {
		t.Fatalf("inline appendix missing its fence:\n%s", body)
	}
	if !strings.Contains(body, "Open the dashboard first.") {
		t.Fatalf("inline appendix body missing:\n%s", body)
	}
	if strings.Index(body, fence) < strings.Index(body, "Base body.") {
		t.Fatal("appendix composed before a source; it must carry the tail")
	}
}

func TestFileAppendixStripsFrontmatterAndAbsolutizesLinks(t *testing.T) {
	e, sources := appendixEnv(t)
	block := e.write(
		t,
		"appendix/dashboard.md",
		"---\nscopes: personal\n---\n# Dashboard\n\nSee [notes](notes.md).\n\n## See also\n\n* sibling\n",
	)
	e.config(t, sources+"appendix:\n  - path: "+block+"\n")
	if code, _, errOut := e.run(t, false); code != 0 {
		t.Fatalf("run failed: %d %s", code, errOut)
	}
	body := composedBody(t, e)
	if !strings.Contains(body, "<!-- appendix: "+block+" -->") {
		t.Fatalf("file appendix missing its fence:\n%s", body)
	}
	if strings.Contains(body, "scopes: personal") {
		t.Fatal("appendix frontmatter survived into the composed output")
	}
	if strings.Contains(body, "## See also") {
		t.Fatal("appendix navigation section survived into the composed output")
	}
	// absolutizeLinks resolves symlinks in the base, which on macOS turns the
	// temp dir's /var into /private/var; the assertion must follow it.
	resolved, err := filepath.EvalSymlinks(filepath.Dir(block))
	if err != nil {
		t.Fatal(err)
	}
	absolute := filepath.Join(resolved, "notes.md")
	if !strings.Contains(body, "]("+absolute+")") {
		t.Fatalf("appendix link was not absolutized to %s:\n%s", absolute, body)
	}
}

func TestRoleScopedAppendixSkipsTheRoleLessLoadPoint(t *testing.T) {
	e, sources := appendixEnv(t)
	e.config(t, sources+"appendix:\n  - text: |\n      Global block.\n  - roles: [platform-eng, sysadmin]\n    text: |\n      Platform block.\n")
	if code, _, errOut := e.run(t, false); code != 0 {
		t.Fatalf("run failed: %d %s", code, errOut)
	}
	body := composedBody(t, e)
	if !strings.Contains(body, "Global block.") {
		t.Fatalf("global appendix missing from the load point:\n%s", body)
	}
	if strings.Contains(body, "Platform block.") {
		t.Fatal("role-scoped appendix leaked into the role-less load point")
	}

	cfg, err := LoadConfig(e.paths.Config)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	base, err := OperatingBase(cfg, "claude", "platform-eng")
	if err != nil {
		t.Fatalf("operating base: %v", err)
	}
	if !strings.Contains(base, "Platform block.") || !strings.Contains(base, "Global block.") {
		t.Fatalf("platform-eng base missing a bound block:\n%s", base)
	}
	if !strings.Contains(base, "<!-- appendix: agent-compose.yaml entry 2 (roles: platform-eng, sysadmin) -->") {
		t.Fatalf("role-scoped fence does not name its roles:\n%s", base)
	}
	other, err := OperatingBase(cfg, "claude", "frontend-eng")
	if err != nil {
		t.Fatalf("operating base: %v", err)
	}
	if strings.Contains(other, "Platform block.") {
		t.Fatal("role-scoped appendix composed for an unbound role")
	}
}

func TestLoadConfigRejectsMalformedAppendixEntries(t *testing.T) {
	e := newEnv(t)
	for name, body := range map[string]string{
		"both text and path": "appendix:\n  - text: hi\n    path: ./block.md\n",
		"neither":            "appendix:\n  - roles: [platform-eng]\n",
		"blank text":         "appendix:\n  - text: \"   \"\n",
		"upper-case role":    "appendix:\n  - text: hi\n    roles: [Platform]\n",
		"empty role":         "appendix:\n  - text: hi\n    roles: [\"\"]\n",
		"repeated role":      "appendix:\n  - text: hi\n    roles: [platform-eng, platform-eng]\n",
		"unknown key":        "appendix:\n  - text: hi\n    harnesses: [claude]\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(e.write(t, name+".yaml", body)); err == nil {
				t.Fatal("malformed appendix entry passed strict config loading")
			}
		})
	}
}

func TestMissingAppendixPathWarnsOnRunAndFailsOnCheck(t *testing.T) {
	e, sources := appendixEnv(t)
	absent := filepath.Join(e.dir, "appendix", "absent.md")
	e.config(t, sources+"appendix:\n  - path: "+absent+"\n")
	code, _, errOut := e.run(t, false)
	if code != 0 {
		t.Fatalf("a missing appendix must not brick convergence: %d %s", code, errOut)
	}
	if !strings.Contains(errOut, "appendix path not found: "+absent) {
		t.Fatalf("run did not warn about the missing appendix:\n%s", errOut)
	}
	checkCode, _, checkErr := e.check(t)
	if checkCode == 0 {
		t.Fatal("check accepted a missing appendix path")
	}
	if !strings.Contains(checkErr, "appendix path not found: "+absent) {
		t.Fatalf("check did not name the missing appendix:\n%s", checkErr)
	}
}

func TestVerboseLayoutNamesEveryAppendixDestination(t *testing.T) {
	e, sources := appendixEnv(t)
	e.config(t, sources+"appendix:\n  - text: |\n      Global block.\n  - roles: [platform-eng]\n    text: |\n      Platform block.\n")
	code, out, errOut := e.run(t, false)
	if code != 0 {
		t.Fatalf("run failed: %d %s", code, errOut)
	}
	for _, want := range []string{
		"layout  <!-- appendix: agent-compose.yaml entry 1 --> => every composed output",
		"layout  <!-- appendix: agent-compose.yaml entry 2 (roles: platform-eng) --> => role bundles: platform-eng",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("verbose layout missing %q:\n%s", want, out)
		}
	}
}

func TestLoadConfigRejectsMalformedTelemetry(t *testing.T) {
	e := newEnv(t)
	for name, body := range map[string]string{
		"bare host":      "telemetry:\n  otlp_metrics_endpoint: collector.example:4318\n",
		"bad protocol":   "telemetry:\n  otlp_metrics_endpoint: http://collector.example:4318/v1/metrics\n  protocol: udp\n",
		"logs requested": "telemetry:\n  otlp_logs_endpoint: http://collector.example:4318/v1/logs\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(e.write(t, name+".yaml", body)); err == nil {
				t.Fatal("malformed telemetry block passed strict config loading")
			}
		})
	}
}
