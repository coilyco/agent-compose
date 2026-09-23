package schema

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// bindingCatalogue models the shape the consumer hit: one repository holding
// several categories, where a second role may only reach a subset.
var bindingCatalogue = []string{
	"lore-rule-commit",
	"lore-rule-voice",
	"lore-self-bio",
	"lore-self-portfolio",
	"lore-third-party-nda",
	"lore-third-party-partner",
}

func bindingProviderRoot(t *testing.T, graph string) string {
	t.Helper()
	root := t.TempDir()
	for _, id := range bindingCatalogue {
		dir := filepath.Join(root, ".agents", "skills", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+id+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(graph), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func loadBindingSource(t *testing.T, graph string) *Source {
	t.Helper()
	source, err := LoadSource(bindingProviderRoot(t, graph))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func selectedIDs(source *Source) []string {
	ids := make([]string, 0, len(source.Skills))
	for _, ref := range source.Skills {
		ids = append(ids, ref.ID)
	}
	return ids
}

const bindingGraph = `repositories {
    repository lore path="coilysiren/lore" {
        skill "lore-*"
    }
}

roles {
    role platform-eng {
        use-repository lore
    }
    role dev-advocate {
        use-repository lore {
            skill "lore-self-*"
            skill "lore-rule-*"
        }
    }
}
`

func TestBindingSelectorParsedOntoTheUse(t *testing.T) {
	source := loadBindingSource(t, bindingGraph)

	platform := source.RoleProviders["platform-eng"]
	if len(platform) != 1 || platform[0].Skills != nil {
		t.Fatalf("platform binding must carry no selector, got %+v", platform)
	}
	advocate := source.RoleProviders["dev-advocate"]
	if len(advocate) != 1 ||
		!slices.Equal(advocate[0].Skills, []string{"lore-self-*", "lore-rule-*"}) {
		t.Fatalf("advocate binding selector = %+v", advocate)
	}
}

func TestBindingSelectorNarrowsAndLeavesOtherRolesAlone(t *testing.T) {
	definition := []string{"lore-*"}

	narrowed := loadBindingSource(t, bindingGraph)
	binding := narrowed.RoleProviders["dev-advocate"][0].Skills
	if err := SelectOrdinarySkills(narrowed, definition, binding); err != nil {
		t.Fatal(err)
	}
	want := []string{"lore-rule-commit", "lore-rule-voice", "lore-self-bio", "lore-self-portfolio"}
	if got := selectedIDs(narrowed); !slices.Equal(got, want) {
		t.Fatalf("narrowed selection = %v, want %v", got, want)
	}
	for _, ref := range narrowed.Skills {
		if strings.HasPrefix(ref.ID, "lore-third-party-") {
			t.Fatalf("binding selector admitted excluded material %q", ref.ID)
		}
	}

	// The same definition mounted by a role that declared no binding selector
	// still receives everything the definition admits.
	whole := loadBindingSource(t, bindingGraph)
	if err := SelectOrdinarySkills(whole, definition, whole.RoleProviders["platform-eng"][0].Skills); err != nil {
		t.Fatal(err)
	}
	if got := selectedIDs(whole); !slices.Equal(got, bindingCatalogue) {
		t.Fatalf("unbound role selection = %v, want the whole catalogue", got)
	}
}

func TestBindingSelectorCannotWidenPastTheDefinition(t *testing.T) {
	source := loadBindingSource(t, bindingGraph)
	err := SelectOrdinarySkills(source, []string{"lore-self-*"}, []string{"lore-third-party-*"})
	if err == nil {
		t.Fatal("binding selector widened past the definition")
	}
	if !strings.Contains(err.Error(), `"lore-third-party-*" matches no ordinary provider skill`) {
		t.Fatalf("widening error = %v", err)
	}
}

func TestBindingSelectorFailsClosed(t *testing.T) {
	for name, testcase := range map[string]struct {
		definition, binding []string
		want                string
	}{
		"empty binding": {
			definition: []string{"lore-*"},
			binding:    []string{},
			want:       "skills selector is empty",
		},
		"unmatched after intersection": {
			definition: []string{"lore-self-*"},
			binding:    []string{"lore-rule-*"},
			want:       `"lore-rule-*" matches no ordinary provider skill`,
		},
		"overlapping binding patterns": {
			definition: []string{"lore-*"},
			binding:    []string{"lore-self-*", "*-bio"},
			want:       `overlap on skill "lore-self-bio"`,
		},
		"invalid binding pattern": {
			definition: []string{"lore-*"},
			binding:    []string{"["},
			want:       "is invalid",
		},
	} {
		t.Run(name, func(t *testing.T) {
			source := loadBindingSource(t, bindingGraph)
			err := SelectOrdinarySkills(source, testcase.definition, testcase.binding)
			if err == nil {
				t.Fatalf("selector %v passed", testcase.binding)
			}
			if !strings.Contains(err.Error(), testcase.want) {
				t.Fatalf("error = %v, want it to contain %q", err, testcase.want)
			}
		})
	}
}

func TestOmittedBindingSelectorMatchesDefinitionOnlyBehavior(t *testing.T) {
	definition := []string{"lore-self-*", "lore-rule-*"}

	before := loadBindingSource(t, bindingGraph)
	if err := SelectOrdinarySkills(before, definition, nil); err != nil {
		t.Fatal(err)
	}
	after := loadBindingSource(t, bindingGraph)
	if err := applySkillSelector(after, definition); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedIDs(before), selectedIDs(after)) {
		t.Fatalf("omitted binding changed selection: %v vs %v", selectedIDs(before), selectedIDs(after))
	}
	if before.SelectorReason != after.SelectorReason {
		t.Fatalf("omitted binding changed the trace reason:\n%s\n%s", before.SelectorReason, after.SelectorReason)
	}
}

func TestBindingSelectorRejectsUnknownChildNode(t *testing.T) {
	graph := `repositories {
    repository lore path="coilysiren/lore" {
        skill "lore-*"
    }
}

roles {
    role dev-advocate {
        use-repository lore {
            provider "lore-self-*"
        }
    }
}
`
	if _, err := LoadSource(bindingProviderRoot(t, graph)); err == nil {
		t.Fatal("unknown binding child node passed")
	}
}

func TestBindingSelectorOnUseProvider(t *testing.T) {
	graph := `providers {
    provider lore path="coilysiren/lore" {
        skill "lore-*"
    }
}

roles {
    role dev-advocate {
        use-provider lore required=#true {
            skill "lore-self-*"
        }
    }
}
`
	source := loadBindingSource(t, graph)
	uses := source.RoleProviders["dev-advocate"]
	if len(uses) != 1 || !uses[0].Required ||
		!slices.Equal(uses[0].Skills, []string{"lore-self-*"}) {
		t.Fatalf("use-provider binding = %+v", uses)
	}
	if err := SelectOrdinarySkills(source, source.Providers["lore"].Skills, uses[0].Skills); err != nil {
		t.Fatal(err)
	}
	want := []string{"lore-self-bio", "lore-self-portfolio"}
	if got := selectedIDs(source); !slices.Equal(got, want) {
		t.Fatalf("use-provider selection = %v, want %v", got, want)
	}
}
