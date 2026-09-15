package schema

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "contracts", name)
}

func TestParseRequestFixture(t *testing.T) {
	req, err := ParseRequest(fixture(t, "native.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	if req.Role != "platform" {
		t.Fatalf("unexpected identity: %+v", req)
	}
	if req.Delivery != DeliveryNativeSkills {
		t.Fatalf("unexpected delivery: %+v", req)
	}
	if req.ModelTier != ModelTierFrontier {
		t.Fatalf("unexpected default model tier: %+v", req)
	}
	if len(req.Sources) != 1 || req.Sources[0].ID != "aos-public" || !req.Sources[0].Required {
		t.Fatalf("unexpected sources: %+v", req.Sources)
	}
}

func TestParseRequestAcceptsCanonicalModelTiers(t *testing.T) {
	for _, tier := range ModelTiers() {
		t.Run(tier, func(t *testing.T) {
			path := writeRequest(t, `compose {
    role "platform"
    delivery "compiled"
    model-tier "`+tier+`"
}`)
			req, err := ParseRequest(path)
			if err != nil {
				t.Fatal(err)
			}
			if req.ModelTier != tier {
				t.Fatalf("model tier = %q, want %q", req.ModelTier, tier)
			}
		})
	}
}

func TestParseRequestAcceptsRelativePersonSource(t *testing.T) {
	req, err := ParseRequest(fixture(t, "custom-person.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	if req.PersonSource != "person-independent" {
		t.Fatalf("person source = %q", req.PersonSource)
	}
	if req.PersonPolicy != personpolicy.ExternalOnly {
		t.Fatalf("person policy = %q", req.PersonPolicy)
	}
}

func writeRequest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "request.kdl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseRequestAcceptsLegacyFullDensity(t *testing.T) {
	path := writeRequest(t, `compose {
    role "platform"
    delivery "compiled"
    density "full"
}`)
	req, err := ParseRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Role != "platform" || req.Delivery != DeliveryCompiled {
		t.Fatalf("unexpected legacy request: %+v", req)
	}
}

// A caller may name the seat it composes. See docs/person-contract.md for why
// that is identity rather than a role redefinition.
func TestParseRequestAcceptsASeatIdentityOverride(t *testing.T) {
	path := writeRequest(t, `compose {
    role "sysadmin"
    identity name="Echo" pronouns="it"
    delivery "native-skills"
}`)
	req, err := ParseRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Identity == nil {
		t.Fatal("identity override was dropped")
	}
	if req.Identity.Name != "Echo" || req.Identity.Pronouns != "it" {
		t.Fatalf("identity = %+v", req.Identity)
	}
}

// Absent stays absent, or every existing bundle silently gains an override
// nobody wrote.
func TestParseRequestLeavesIdentityUnsetWhenUnnamed(t *testing.T) {
	path := writeRequest(t, `compose {
    role "sysadmin"
    delivery "native-skills"
}`)
	req, err := ParseRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if req.Identity != nil {
		t.Fatalf("identity = %+v, want nil", req.Identity)
	}
}

func TestParseRequestFailsClosed(t *testing.T) {
	cases := map[string]string{
		"identity without pronouns": `compose {
    role "platform"
    delivery "native-skills"
    identity name="Echo"
}`,
		"identity without name": `compose {
    role "platform"
    delivery "native-skills"
    identity pronouns="it"
}`,
		"identity with a blank name": `compose {
    role "platform"
    delivery "native-skills"
    identity name="   " pronouns="it"
}`,
		"identity taking an argument": `compose {
    role "platform"
    delivery "native-skills"
    identity "Echo" pronouns="it"
}`,
		"duplicate identity": `compose {
    role "platform"
    delivery "native-skills"
    identity name="Echo" pronouns="it"
    identity name="Vera" pronouns="she"
}`,
		"unknown node": `compose {
    role "platform"
    delivery "native-skills"
    privacy-scope "public"
}`,
		"duplicate scalar": `compose {
    role "platform"
    role "designer"
    delivery "native-skills"
}`,
		"bad delivery": `compose {
    role "platform"
    delivery "carrier-pigeon"
}`,
		"removed model class": `compose {
    role "platform"
    delivery "native-skills"
    model-class "low-context"
}`,
		"bad model tier": `compose {
    role "platform"
    delivery "native-skills"
    model-tier "premium"
}`,
		"retired brief density": `compose {
    role "platform"
    delivery "compiled"
    density "brief"
}`,
		"retired personality selector": `compose {
    role "platform"
    personality "tenacious"
    delivery "native-skills"
}`,
		"source without declaration": `compose {
    role "platform"
    delivery "native-skills"
    source "aos-public"
}`,
		"source with declaration and root": `compose {
    role "platform"
    delivery "native-skills"
    source "aos-public" declaration="source.kdl" root="."
}`,
		"source with empty root": `compose {
    role "platform"
    delivery "native-skills"
    source "aos-public" root=""
}`,
		"absolute person source": `compose {
    person-source "/tmp/person"
    role "platform"
    delivery "native-skills"
}`,
		"external-only without person source": `compose {
    person-policy "external-only"
    role "platform"
    delivery "native-skills"
}`,
		"unknown person policy": `compose {
    person-policy "prefer-external"
    person-source "person"
    role "platform"
    delivery "native-skills"
}`,
		"invalid kdl": `compose { role "engineer`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseRequest(writeRequest(t, body)); err == nil {
				t.Fatal("expected parse failure")
			}
		})
	}
}

func TestLoadSourcesRequiredVersusOptional(t *testing.T) {
	required := writeRequest(t, `compose {
    role "platform"
    delivery "native-skills"
    source "ghost" declaration="ghost.kdl" required=#true
}`)
	req, err := ParseRequest(required)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadSources(req, required); err == nil {
		t.Fatal("expected a required missing source to fail")
	}

	optional := writeRequest(t, `compose {
    role "platform"
    delivery "native-skills"
    source "ghost" declaration="ghost.kdl"
}`)
	req, err = ParseRequest(optional)
	if err != nil {
		t.Fatal(err)
	}
	sources, missing, err := LoadSources(req, optional)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 || len(missing) != 1 || missing[0].ID != "ghost" {
		t.Fatalf("expected one missing optional source, got %+v / %+v", sources, missing)
	}
}

func TestLoadSourcesRejectsEscapingPaths(t *testing.T) {
	path := writeRequest(t, `compose {
    role "platform"
    delivery "native-skills"
    source "evil" declaration="../evil.kdl" required=#true
}`)
	req, err := ParseRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadSources(req, path)
	if err == nil || !strings.Contains(err.Error(), "relative and clean") {
		t.Fatalf("expected escaping declaration to fail, got %v", err)
	}
}

func TestParseSourceFixture(t *testing.T) {
	req, err := ParseRequest(fixture(t, "native.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	sources, missing, err := LoadSources(req, fixture(t, "native.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 || len(sources) != 1 {
		t.Fatalf("unexpected load result: %+v / %+v", sources, missing)
	}
	src := sources[0]
	if len(src.Instructions) != 1 || src.Instructions[0].ID != "foundation" {
		t.Fatalf("unexpected instructions: %+v", src.Instructions)
	}
	if len(src.Skills) != 1 || src.Skills[0].ID != "fixture-review" {
		t.Fatalf("expected only the ordinary fixture skill, got %+v", src.Skills)
	}
}

func TestLoadInferredProviderRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "public-provider")
	for _, name := range []string{"personality-bright", "personality-calm", "coding-go"} {
		dir := filepath.Join(root, ".agents", "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	shared := filepath.Join(root, ".agents", "skills", "personality-shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "INVARIANT.md"), []byte("# Invariant\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"coding-aws", "coding-shape-cli", "design-system"} {
		dir := filepath.Join(root, ".agents", "composed", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "COMPOSED.md"), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role platform {
        composed-skill "coding-*"
    }
    role designer {
        composed-skill design-system
    }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	request := filepath.Join(root, "request.kdl")
	if err := os.WriteFile(request, []byte(`compose {
    role "platform"
    delivery "native-skills"
    source "aos-public" root="." required=#true
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	req, err := ParseRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	sources, missing, err := LoadSources(req, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 || len(sources) != 1 {
		t.Fatalf("unexpected inferred load result: %+v / %+v", sources, missing)
	}
	src := sources[0]
	if src.ID != "aos-public" ||
		len(src.Instructions) != 1 ||
		src.Instructions[0].ID != "personality-invariant" {
		t.Fatalf("unexpected inferred source: %+v", src)
	}
	if len(src.Skills) != 3 ||
		src.Skills[0].ID != "coding-go" ||
		src.Skills[1].ID != "personality-bright" ||
		src.Skills[2].ID != "personality-calm" {
		t.Fatalf("inferred ordinary skills must be sorted: %+v", src.Skills)
	}
	if len(src.RoleSkills["platform"]) != 2 ||
		src.RoleSkills["platform"][0].ID != "coding-aws" ||
		src.RoleSkills["platform"][1].ID != "coding-shape-cli" ||
		src.RoleSkills["platform"][0].EntryPoint != "COMPOSED.md" ||
		len(src.RoleSkills["designer"]) != 1 ||
		src.RoleSkills["designer"][0].ID != "design-system" {
		t.Fatalf("unexpected composed role skills: %+v", src.RoleSkills)
	}
	direct, err := LoadSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if direct.ID != filepath.Base(root) ||
		len(direct.Skills) != len(src.Skills) ||
		len(direct.RoleSkills) != len(src.RoleSkills) {
		t.Fatalf("direct provider load differs: %+v", direct)
	}
}

func TestLoadInferredProviderParsesRepositorySkillProviderGraph(t *testing.T) {
	root := filepath.Join(t.TempDir(), "aosk")
	skill := filepath.Join(root, ".agents", "skills", "repo-aosk")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# AOSK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`repositories {
    repository hardware path="coilyco-bridge/agentic-os-hardware" {
        skill "compute-stack"
        skill "machine-*"
    }
    repository infrastructure path="coilyco-flight-deck/infrastructure" {
        skill "*"
    }
}

roles {
    role platform {
        use-repository hardware
    }
    role sysadmin {
        use-repository hardware
        use-repository infrastructure
    }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := LoadSource(root)
	if err != nil {
		t.Fatal(err)
	}
	hardware := source.Providers["hardware"]
	if hardware.Path != "coilyco-bridge/agentic-os-hardware" ||
		!slices.Equal(hardware.Skills, []string{"compute-stack", "machine-*"}) {
		t.Fatalf("hardware provider = %+v", hardware)
	}
	if got := source.RoleProviders["sysadmin"]; len(got) != 2 ||
		got[0].Provider != "hardware" || !got[0].Required ||
		got[1].Provider != "infrastructure" || !got[1].Required {
		t.Fatalf("ops provider uses = %+v", got)
	}
}

func TestLoadInferredProviderRejectsUnsafeComposedLayouts(t *testing.T) {
	makeProvider := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		shared := filepath.Join(root, ".agents", "skills", "personality-shared")
		ordinary := filepath.Join(root, ".agents", "skills", "coding-go")
		if err := os.MkdirAll(shared, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(ordinary, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(shared, "INVARIANT.md"), []byte("# Invariant\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ordinary, "SKILL.md"), []byte("# Go\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	t.Run("provider graph without composed root", func(t *testing.T) {
		root := makeProvider(t)
		if err := os.WriteFile(
			filepath.Join(root, ".agents", "roles.kdl"),
			[]byte(`repositories {
    repository hardware path="example/hardware" {
        skill "machine-*"
    }
}
roles {
    role platform {
        use-repository hardware
    }
}
`),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
		if source, err := LoadSource(root); err != nil || len(source.RoleProviders["platform"]) != 1 {
			t.Fatalf("provider-only graph must load, source=%+v err=%v", source, err)
		}
	})

	// Multi-line because KDL takes a newline as the node terminator: as
	// one-liners four of these five were rejected before reaching validation.
	for name, graph := range map[string]string{
		"undeclared repository": "roles {\n role platform {\n use-repository missing\n }\n}\n",
		"invalid selector": "repositories {\n repository hardware path=\"example/hardware\" {\n" +
			" skill \"[\"\n }\n}\nroles {\n}\n",
		"duplicate path": "repositories {\n repository one path=\"example/hardware\"\n" +
			" repository two path=\"example/hardware\"\n}\nroles {\n}\n",
		"unsafe path": "repositories {\n repository hardware path=\"../hardware\"\n}\nroles {\n}\n",
		"unknown use property": "repositories {\n repository hardware path=\"example/hardware\" {\n" +
			" skill \"*\"\n }\n}\nroles {\n role platform {\n use-repository hardware required=#true\n }\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := makeProvider(t)
			if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(graph), 0o644); err != nil {
				t.Fatal(err)
			}
			err := LoadSourceErr(root)
			if err == nil {
				t.Fatalf("invalid unified graph passed: %s", graph)
			}
			if strings.Contains(err.Error(), "parse error at") {
				t.Fatalf("fixture is malformed KDL rather than an invalid graph: %v", err)
			}
		})
	}

	t.Run("SKILL.md under composed", func(t *testing.T) {
		root := makeProvider(t)
		composed := filepath.Join(root, ".agents", "composed", "coding-shape-cli")
		if err := os.MkdirAll(composed, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(composed, "COMPOSED.md"), []byte("# CLI\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(composed, "SKILL.md"), []byte("# Leaked\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSource(root); err == nil || !strings.Contains(err.Error(), "contains SKILL.md") {
			t.Fatalf("composed SKILL.md needs an actionable failure, got %v", err)
		}
	})

	t.Run("ordinary name collision", func(t *testing.T) {
		root := makeProvider(t)
		composed := filepath.Join(root, ".agents", "composed", "coding-go")
		if err := os.MkdirAll(composed, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(composed, "COMPOSED.md"), []byte("# Go\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSource(root); err == nil || !strings.Contains(err.Error(), "exists in both") {
			t.Fatalf("ordinary/composed collisions must fail closed, got %v", err)
		}
	})

	t.Run("missing role binding target", func(t *testing.T) {
		root := makeProvider(t)
		if err := os.MkdirAll(filepath.Join(root, ".agents", "composed"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role "platform" {
        composed-skill "missing"
    }
}
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSource(root); err == nil || !strings.Contains(err.Error(), "missing composed skill") {
			t.Fatalf("missing composed binding targets must fail closed, got %v", err)
		}
	})

	t.Run("unmatched role binding wildcard", func(t *testing.T) {
		root := makeProvider(t)
		if err := os.MkdirAll(filepath.Join(root, ".agents", "composed"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role "platform" {
        composed-skill "coding-*"
    }
}
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSource(root); err == nil || !strings.Contains(err.Error(), "matches no skills") {
			t.Fatalf("unmatched composed-skill wildcard must fail closed, got %v", err)
		}
	})

	t.Run("malformed role binding wildcard", func(t *testing.T) {
		root := makeProvider(t)
		composed := filepath.Join(root, ".agents", "composed", "coding-shape-cli")
		if err := os.MkdirAll(composed, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(composed, "COMPOSED.md"), []byte("# CLI\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role "platform" {
        composed-skill "coding-["
    }
}
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSource(root); err == nil || !strings.Contains(err.Error(), "pattern \"coding-[\" is invalid") {
			t.Fatalf("malformed composed-skill wildcard must fail closed, got %v", err)
		}
	})

	t.Run("overlapping role binding patterns", func(t *testing.T) {
		root := makeProvider(t)
		composed := filepath.Join(root, ".agents", "composed", "coding-shape-cli")
		if err := os.MkdirAll(composed, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(composed, "COMPOSED.md"), []byte("# CLI\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role "platform" {
        composed-skill "coding-*"
        composed-skill "coding-shape-cli"
    }
}
`), 0o644); err != nil {
			t.Fatal(err)
		}
		source, err := LoadSource(root)
		if err != nil {
			t.Fatal(err)
		}
		roleSkills := source.RoleSkills["platform"]
		if len(roleSkills) != 1 || roleSkills[0].ID != "coding-shape-cli" ||
			!slices.Equal(roleSkills[0].Selectors, []string{"coding-*", "coding-shape-cli"}) {
			t.Fatalf("overlapping composed-skill patterns must select one skill with complete provenance, got %+v", roleSkills)
		}
		wantOverlap := SelectorOverlap{
			Role: "platform", Skill: "coding-shape-cli",
			Selectors: []string{"coding-*", "coding-shape-cli"},
		}
		if len(source.SelectorOverlaps) != 1 ||
			source.SelectorOverlaps[0].Role != wantOverlap.Role ||
			source.SelectorOverlaps[0].Skill != wantOverlap.Skill ||
			!slices.Equal(source.SelectorOverlaps[0].Selectors, wantOverlap.Selectors) {
			t.Fatalf("overlapping composed-skill warning provenance = %+v, want %+v", source.SelectorOverlaps, wantOverlap)
		}
	})

}

func TestLoadSourceRepositoryPolicy(t *testing.T) {
	root := t.TempDir()
	roles := filepath.Join(root, ".agents", "roles.kdl")
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, ".agents", "skills", "fixture", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skill), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skill, []byte("# Fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	graph := `repositories {
    repository lore path="owner/lore"
    repository voice path="owner/voice"
    repository resident path="owner/resident"
    global lore
    resident-only resident
}
roles {
    role platform { use-repository voice }
    role science {}
}
`
	if err := os.WriteFile(roles, []byte(graph), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := LoadSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if source.Repositories["voice"].Path != "owner/voice" ||
		len(source.GlobalRepos) != 1 || source.GlobalRepos[0].Repository != "lore" ||
		len(source.ResidentRepos) != 1 || source.ResidentRepos[0].Repository != "resident" ||
		len(source.RoleRepos["platform"]) != 1 || source.RoleRepos["platform"][0].Repository != "voice" {
		t.Fatalf("repository policy = %+v", source)
	}
	if _, exists := source.RoleRepos["science"]; !exists {
		t.Fatal("empty canonical role was omitted from repository policy")
	}
}

func TestLoadSourceRepositoryPolicyFailsClosed(t *testing.T) {
	for name, graph := range map[string]string{
		"undeclared global":          `repositories { global missing } roles { role platform {} }`,
		"undeclared role repository": `repositories {} roles { role platform { use-repository missing } }`,
		"duplicate repository path":  `repositories { repository one path="owner/repo"; repository two path="owner/repo" } roles { role platform {} }`,
		"unsafe repository path":     `repositories { repository one path="../repo" } roles { role platform {} }`,
		"duplicate global":           `repositories { repository one path="owner/one"; global one; global one } roles { role platform {} }`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			roles := filepath.Join(root, ".agents", "roles.kdl")
			if err := os.MkdirAll(filepath.Join(root, ".agents", "skills"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(roles, []byte(graph+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadSource(root); err == nil {
				t.Fatalf("invalid repository policy passed: %s", graph)
			}
		})
	}
}

func TestLoadInferredProviderAllowsMissingInvariant(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ordinary-provider")
	skill := filepath.Join(root, ".agents", "skills", "personality-bright")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# Bright\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := LoadSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Instructions) != 0 || len(src.Skills) != 1 {
		t.Fatalf("provider without invariant loaded incorrectly: %+v", src)
	}
}

func makeOrgProvider(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ordinary := filepath.Join(root, ".agents", "skills", "coding-go")
	if err := os.MkdirAll(ordinary, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ordinary, "SKILL.md"), []byte("# Go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadProviderParsesOrgDeclarationAndBinding(t *testing.T) {
	root := makeOrgProvider(t)
	if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`repositories {
    repository lore path="coilyco-bridge/lore" {
        skill "repo-lore"
    }
    org gaming owner="coilyco-gaming" {
        skill "repo-*"
        skill "sirens-game-*"
    }
}

roles {
    role gamedev {
        use-repository lore
        use-org gaming
    }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	source, err := LoadSource(root)
	if err != nil {
		t.Fatal(err)
	}
	definition, declared := source.Orgs["gaming"]
	if !declared || definition.Owner != "coilyco-gaming" || len(definition.Skills) != 2 {
		t.Fatalf("org definition = %+v", definition)
	}
	uses := source.RoleOrgs["gamedev"]
	if len(uses) != 1 || uses[0].Org != "gaming" {
		t.Fatalf("role org uses = %+v", uses)
	}
	// An org sits beside repositories rather than replacing them.
	if len(source.RoleRepos["gamedev"]) != 1 && len(source.RoleProviders["gamedev"]) != 1 {
		t.Fatal("the repository binding beside the org was lost")
	}
}

func TestLoadProviderRejectsUnusableOrgGraphs(t *testing.T) {
	for name, graph := range map[string]string{
		"undeclared org": "roles {\n role gamedev {\n use-org missing\n }\n}\n",
		"no selector": "repositories {\n org gaming owner=\"coilyco-gaming\"\n}\n" +
			"roles {\n role gamedev {\n use-org gaming\n }\n}\n",
		"owner is a path": "repositories {\n org gaming owner=\"coilyco-gaming/enshrouded\" {\n" +
			" skill \"*\"\n }\n}\nroles {\n}\n",
		"owner missing": "repositories {\n org gaming {\n skill \"*\"\n }\n}\nroles {\n}\n",
		"unknown property": "repositories {\n org gaming owner=\"coilyco-gaming\" path=\"x\" {\n" +
			" skill \"*\"\n }\n}\nroles {\n}\n",
		"invalid selector": "repositories {\n org gaming owner=\"coilyco-gaming\" {\n" +
			" skill \"[\"\n }\n}\nroles {\n}\n",
		"duplicate owner": "repositories {\n org one owner=\"coilyco-gaming\" {\n skill \"*\"\n }\n" +
			" org two owner=\"coilyco-gaming\" {\n skill \"a-*\"\n }\n}\nroles {\n}\n",
		"id collides with repository": "repositories {\n repository gaming path=\"example/gaming\"\n" +
			" org gaming owner=\"coilyco-gaming\" {\n skill \"*\"\n }\n}\nroles {\n}\n",
		"repeated use": "repositories {\n org gaming owner=\"coilyco-gaming\" {\n skill \"*\"\n }\n}\n" +
			"roles {\n role gamedev {\n use-org gaming\n use-org gaming\n }\n}\n",
		"use carries children": "repositories {\n org gaming owner=\"coilyco-gaming\" {\n skill \"*\"\n }\n}\n" +
			"roles {\n role gamedev {\n use-org gaming {\n skill \"x\"\n }\n }\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := makeOrgProvider(t)
			if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(graph), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadSource(root)
			if err == nil {
				t.Fatalf("invalid org graph passed: %s", graph)
			}
			// A KDL syntax error would reject every fixture here without
			// reaching the validation each one is named after.
			if strings.Contains(err.Error(), "parse error at") {
				t.Fatalf("fixture is malformed KDL rather than an invalid graph: %v", err)
			}
		})
	}
}

// LoadSourceErr is LoadSource's error alone, for tables that assert refusal.
func LoadSourceErr(root string) error {
	_, err := LoadSource(root)
	return err
}
