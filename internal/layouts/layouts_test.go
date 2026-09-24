package layouts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `
alpha:
  cascade: true
  repo:
    native: {instructions: ALPHA.md, skills: .alpha/skills}
  home:
    native: {instructions: .alpha/ALPHA.md, skills: .alpha/skills}
    compiled: {instructions: .alpha/ALPHA.md}
`

func TestDefaultTableLoads(t *testing.T) {
	t.Setenv(Env, "")
	table, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(table) == 0 {
		t.Fatal("default table declares no harness")
	}
}

func TestEnvReplacesTheDefaultWhole(t *testing.T) {
	file := filepath.Join(t.TempDir(), "layouts.yaml")
	if err := os.WriteFile(file, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(Env, file)
	table, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := Names(table); len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("names = %v, want only the replacement's", got)
	}
	if table["alpha"].Repo.Compiled != nil {
		t.Fatal("an omitted mode must stay unsupported")
	}
}

func TestEnvNamingAMissingFileFails(t *testing.T) {
	t.Setenv(Env, filepath.Join(t.TempDir(), "absent.yaml"))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), Env) {
		t.Fatalf("err = %v, want a diagnostic naming %s", err, Env)
	}
}

func TestParseRejects(t *testing.T) {
	for name, body := range map[string]string{
		"empty":            "{}\n",
		"unknown field":    strings.Replace(fixture, "cascade: true", "cascade: true\n  model: x", 1),
		"absolute path":    strings.Replace(fixture, "instructions: ALPHA.md", "instructions: /etc/ALPHA.md", 1),
		"escaping path":    strings.Replace(fixture, "skills: .alpha/skills}\n  home", "skills: ../skills}\n  home", 1),
		"no instructions":  strings.Replace(fixture, "instructions: ALPHA.md, ", "", 1),
		"no mode":          "alpha:\n  repo: {}\n  home:\n    native: {instructions: A.md}\n",
		"cascade no skill": "alpha:\n  cascade: true\n  repo:\n    native: {instructions: A.md}\n  home:\n    native: {instructions: A.md}\n",
		"two documents":    fixture + "---\n" + fixture,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(body), "fixture"); err == nil {
				t.Fatal("malformed table parsed")
			}
		})
	}
}
