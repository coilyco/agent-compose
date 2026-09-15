package catalogmanifest

import (
	"strings"
	"testing"
)

func TestParseSourceQualifiesAgainstTheDeclaredForge(t *testing.T) {
	for name, testCase := range map[string]struct {
		raw      string
		fallback string
		want     Source
		recorded string
	}{
		"bare takes the default": {
			raw:      "coilyco-gaming/enshrouded/.agents/skills@main",
			fallback: "forgejo.coilysiren.me",
			want: Source{
				Forge: "forgejo.coilysiren.me",
				Owner: "coilyco-gaming",
				Repo:  "enshrouded",
				Path:  ".agents/skills",
				Ref:   "main",
			},
			recorded: "forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main",
		},
		"its own forge beats the default": {
			raw:      "https://github.com/coilysiren/coilysiren/.agents/skills@main",
			fallback: "forgejo.coilysiren.me",
			want: Source{
				Forge: "github.com",
				Owner: "coilysiren",
				Repo:  "coilysiren",
				Path:  ".agents/skills",
				Ref:   "main",
			},
			recorded: "github.com/coilysiren/coilysiren/.agents/skills@main",
		},
		"no ref stays refless": {
			raw:      "https://forgejo.example.test/owner/repo/.agents/skills",
			fallback: "",
			want: Source{
				Forge: "forgejo.example.test",
				Owner: "owner",
				Repo:  "repo",
				Path:  ".agents/skills",
			},
			recorded: "forgejo.example.test/owner/repo/.agents/skills",
		},
		"a deeper catalogue path is kept whole": {
			raw:      "owner/repo/nested/.agents/skills@v1",
			fallback: "forgejo.example.test",
			want: Source{
				Forge: "forgejo.example.test",
				Owner: "owner",
				Repo:  "repo",
				Path:  "nested/.agents/skills",
				Ref:   "v1",
			},
			recorded: "forgejo.example.test/owner/repo/nested/.agents/skills@v1",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parseSource(testCase.raw, testCase.fallback)
			if err != nil {
				t.Fatal(err)
			}
			if got != testCase.want {
				t.Fatalf("parsed = %+v, want %+v", got, testCase.want)
			}
			if recorded := got.String(); recorded != testCase.recorded {
				t.Fatalf("recorded = %q, want %q", recorded, testCase.recorded)
			}
		})
	}
}

// The same bare source is real on two forges, so an unqualified one must not
// resolve at all rather than silently pick a side.
func TestParseSourceRefusesAnUnqualifiedSource(t *testing.T) {
	_, err := parseSource("coilyco-flight-deck/agentic-os/.agents/skills@main", "")
	if err == nil {
		t.Fatal("unqualified source resolved")
	}
	if !strings.Contains(err.Error(), "no forge") {
		t.Fatalf("error does not name the missing forge: %v", err)
	}
}

func TestParseSourceRejectsMalformedInput(t *testing.T) {
	for name, raw := range map[string]string{
		"empty":             "",
		"owner only":        "owner",
		"no catalogue path": "owner/repo",
		"trailing at":       "owner/repo/.agents/skills@",
		"scheme, no host":   "https:///owner/repo/.agents/skills@main",
		"empty segment":     "owner//repo/.agents/skills@main",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSource(raw, "forgejo.example.test"); err == nil {
				t.Fatalf("malformed source %q resolved", raw)
			}
		})
	}
}

func TestParseForgeAcceptsBothWrittenForms(t *testing.T) {
	for raw, want := range map[string]string{
		"https://forgejo.coilysiren.me": "forgejo.coilysiren.me",
		"forgejo.coilysiren.me":         "forgejo.coilysiren.me",
		"":                              "",
	} {
		got, err := parseForge(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("parseForge(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := parseForge("forgejo.example.test/owner"); err == nil {
		t.Fatal("a forge carrying a path was accepted")
	}
}
