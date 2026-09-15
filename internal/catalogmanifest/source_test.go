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
			got, err := ParseSource(testCase.raw, testCase.fallback)
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
	_, err := ParseSource("coilyco-flight-deck/agentic-os/.agents/skills@main", "")
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
			if _, err := ParseSource(raw, "forgejo.example.test"); err == nil {
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

// A private source redacts everywhere it renders, because String is what
// reaches a log line, a warning, and a published artifact.
func TestPrivateSourceRedactsByDefault(t *testing.T) {
	source, err := ParseSource(
		"https://forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main", "")
	if err != nil {
		t.Fatal(err)
	}
	source.Private = true
	source.Index = 3

	if got := source.String(); got != "private catalogue 3" {
		t.Fatalf("String = %q", got)
	}
	// Only the source half redacts: the skill name ships regardless, so a skill
	// named after its private repo is an authoring choice, not a leak to close.
	address := source.SkillAddress("sirens-game-enshrouded")
	prefix, name, found := strings.Cut(address, "/sirens-game-enshrouded")
	if !found || name != "" {
		t.Fatalf("SkillAddress = %q, want it to end in the skill name", address)
	}
	for _, leak := range []string{"coilyco-gaming", "forgejo.coilysiren.me", "/enshrouded"} {
		if strings.Contains(prefix, leak) {
			t.Fatalf("SkillAddress source half %q leaks %q", prefix, leak)
		}
	}
	if prefix != "private catalogue 3" {
		t.Fatalf("SkillAddress source half = %q", prefix)
	}
	if source.Reveal() != "forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main" {
		t.Fatalf("Reveal = %q", source.Reveal())
	}
}

func TestPublicSourceIsUnchangedByTheRedactionPath(t *testing.T) {
	source, err := ParseSource("https://forgejo.example.test/org/repo/.agents/skills@main", "")
	if err != nil {
		t.Fatal(err)
	}
	source.Index = 2
	if source.String() != source.Reveal() {
		t.Fatalf("public source redacted: %q", source.String())
	}
}
