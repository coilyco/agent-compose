package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/bundle"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "contracts", name)
}

func readManifest(t *testing.T, dir string) bundle.Manifest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m bundle.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestComposeAllFixtures(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	wantPersonalities := p.Roles["platform-eng"].Personalities
	wantColor := p.Roles["platform-eng"].FavoriteColor
	cases := map[string]string{
		"native.kdl":   "native-skills",
		"compiled.kdl": "compiled",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			out := t.TempDir()
			result, err := Run(fixture(t, name), out)
			if err != nil {
				t.Fatal(err)
			}
			m := readManifest(t, result.Bundle.Dir)
			if m.Format != "agent-compose.bundle" || m.Role != "platform-eng" ||
				m.ModelTier != schema.ModelTierFrontier ||
				!slices.Equal(m.Personalities, wantPersonalities) ||
				m.Delivery.Mode != want {
				t.Fatalf("unexpected manifest: %+v", m)
			}
			if len(m.Sources) != 2 || m.Sources[0] != "roster:core" || m.Sources[1] != "aos-public" {
				t.Fatalf("unexpected sources: %+v", m.Sources)
			}
			if m.Color != wantColor {
				t.Fatalf("manifest color = %q, want melded %q", m.Color, wantColor)
			}
			instructionsPath := filepath.Join(
				result.Bundle.Dir,
				"content",
				"instructions.md",
			)
			instructions, err := os.ReadFile(instructionsPath)
			if err != nil {
				t.Fatal(err)
			}
			instructionText := string(instructions)
			wantCard, err := p.RenderRoleIdentityCard("platform-eng", wantColor, p.RoleActiveBoundaries("platform-eng"))
			if err != nil {
				t.Fatal(err)
			}
			for _, selected := range []string{
				"# Role instructions",
				"Agent-compose assigned you the `platform-eng` role from the caller's compose request.",
				"Treat it as authoritative and fixed for this session.",
				wantCard,
				"**Role skill // `role-platform-eng`**",
				"# Fixture foundation",
			} {
				if !strings.Contains(instructionText, selected) {
					t.Fatalf("instructions missing %q:\n%s", selected, instructionText)
				}
			}
			if strings.Contains(instructionText, p.Roles["platform-eng"].Briefing) {
				t.Fatalf("native startup instructions eagerly embedded the role skill body:\n%s", instructionText)
			}
			for _, roleName := range p.RoleOrder {
				if roleName == "platform-eng" {
					continue
				}
				if strings.Contains(instructionText, p.Roles[roleName].Briefing) {
					t.Fatalf("inactive role %q briefing entered the bundle:\n%s", roleName, instructionText)
				}
			}
			mustExist(t, result.Bundle.Dir, "trace.json")
			mustExist(t, result.Bundle.Dir, "content/skills/roster%3Acore/role-platform-eng/SKILL.md")
			for _, personalityName := range wantPersonalities {
				skillPath := "content/skills/roster%3Acore/personality-" + personalityName + "/SKILL.md"
				mustExist(t, result.Bundle.Dir, skillPath)
			}
			if want == "compiled" {
				if m.Delivery.CompiledContext != "delivery/compiled.md" || m.Delivery.SkillsRoot != "" {
					t.Fatalf("unexpected compiled delivery: %+v", m.Delivery)
				}
				mustExist(t, result.Bundle.Dir, "delivery/compiled.md")
				compiled, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "delivery", "compiled.md"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(compiled), p.Roles["platform-eng"].Briefing) ||
					!strings.Contains(string(compiled), wantCard) {
					t.Fatalf("compiled context omitted role card or skill body:\n%s", compiled)
				}
			} else {
				if m.Delivery.SkillsRoot != "content/skills" || m.Delivery.CompiledContext != "" {
					t.Fatalf("unexpected native delivery: %+v", m.Delivery)
				}
			}
		})
	}
}

func TestEvalRoleMethodsMatchNativeAndCompiledDelivery(t *testing.T) {
	for _, delivery := range []string{schema.DeliveryNativeSkills, schema.DeliveryCompiled} {
		t.Run(delivery, func(t *testing.T) {
			result, err := RunRoots(
				&schema.Request{
					Role:      "scientist",
					ModelTier: schema.ModelTierFrontier,
					Delivery:  delivery,
				},
				nil,
				t.TempDir(),
				Options{},
			)
			if err != nil {
				t.Fatal(err)
			}
			for _, boundary := range []string{"boundary-modify-live-backend", "boundary-seek-external-validation"} {
				mustExist(t, result.Bundle.Dir, "content/skills/roster%3Acore/"+boundary+"/SKILL.md")
			}
			if delivery == schema.DeliveryCompiled {
				raw, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "delivery", "compiled.md"))
				if err != nil {
					t.Fatal(err)
				}
				for _, heading := range []string{"# Boundary: modify live backend", "# Boundary: seek external validation"} {
					if !strings.Contains(string(raw), heading) {
						t.Errorf("compiled science context omitted %q", heading)
					}
				}
			}
		})
	}
}

func TestComposeExternalPersonDoesNotInheritEmbeddedRoster(t *testing.T) {
	result, err := Run(fixture(t, "custom-person.kdl"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !result.ExternalOnly {
		t.Fatal("portable external-only request lost its policy")
	}
	manifest := readManifest(t, result.Bundle.Dir)
	if manifest.Role != "builder" ||
		!slices.Equal(manifest.Personalities, []string{"bright", "steady"}) {
		t.Fatalf("external manifest identity = %+v", manifest)
	}
	if !slices.Equal(manifest.Sources, []string{"person:workbench", "aos-public"}) {
		t.Fatalf("external manifest sources = %+v", manifest.Sources)
	}
	if slices.Contains(manifest.Sources, "roster:core") {
		t.Fatal("external composition inherited roster:core")
	}
	for _, rel := range []string{
		"content/skills/person%3Aworkbench/personality-bright/SKILL.md",
		"content/skills/person%3Aworkbench/personality-steady/SKILL.md",
	} {
		mustExist(t, result.Bundle.Dir, rel)
	}
	instructions, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "content", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(instructions), "# Workbench invariant") ||
		strings.Contains(string(instructions), "# Personality invariant") {
		t.Fatalf("external instructions crossed person boundaries:\n%s", instructions)
	}
}

func TestComposeHostExternalOnlySelection(t *testing.T) {
	personRoot, err := filepath.Abs(fixture(t, "person-independent"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunWithOptions(
		fixture(t, "custom-person-inherited.kdl"),
		t.TempDir(),
		Options{
			PersonPolicy: personpolicy.ExternalOnly,
			PersonSource: personRoot,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	manifest := readManifest(t, result.Bundle.Dir)
	if !result.ExternalOnly ||
		!slices.Equal(manifest.Sources, []string{"person:workbench", "aos-public"}) {
		t.Fatalf("guarded host selection = %+v / %+v", result, manifest.Sources)
	}
}

func TestComposeHostExternalOnlyRequiresSource(t *testing.T) {
	_, err := RunWithOptions(
		fixture(t, "native.kdl"),
		t.TempDir(),
		Options{PersonPolicy: personpolicy.ExternalOnly},
	)
	if err == nil || !IsExternalOnlyError(err) ||
		!strings.Contains(err.Error(), "requires person_source") {
		t.Fatalf("external-only missing source error = %v", err)
	}
}

func TestComposeInferredProviderRoot(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	ordinary := filepath.Join(root, ".agents", "skills", "coding-go")
	if err := os.MkdirAll(ordinary, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ordinary, "SKILL.md"), []byte("# Go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"coding-shape-cli": "# CLI foundation\n",
		"design-system":    "# Design system\n",
	} {
		dir := filepath.Join(root, ".agents", "composed", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "COMPOSED.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "roles.kdl"), []byte(`roles {
    role "platform-eng" {
        composed-skill "coding-shape-cli"
    }
    role "frontend-eng" {
        composed-skill "design-system"
    }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	request := filepath.Join(root, "request.kdl")
	if err := os.WriteFile(request, []byte(`compose {
    role "platform-eng"
    delivery "native-skills"
    source "aos-public" root="." required=#true
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(request, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manifest := readManifest(t, result.Bundle.Dir)
	if len(manifest.Personalities) != len(p.Roles["platform-eng"].Personalities) {
		t.Fatalf("inferred provider selected the wrong personalities: %+v", manifest.Personalities)
	}
	instructions, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "content", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(instructions), "# Personality invariant") {
		t.Fatalf("composition omitted the embedded personality invariant:\n%s", instructions)
	}
	for _, rel := range []string{
		"content/skills/roster%3Acore/personality-tenacious/SKILL.md",
		"content/skills/aos-public/coding-go/SKILL.md",
		"content/skills/aos-public/coding-shape-cli/SKILL.md",
	} {
		mustExist(t, result.Bundle.Dir, rel)
	}
	composedSource := filepath.Join(
		result.Bundle.Dir,
		"content",
		"skills",
		"aos-public",
		"coding-shape-cli",
		"COMPOSED.md",
	)
	if _, err := os.Stat(composedSource); !os.IsNotExist(err) {
		t.Fatalf("source-only COMPOSED.md leaked into the bundle: %v", err)
	}
	inactive := filepath.Join(result.Bundle.Dir, "content", "skills", "aos-public", "design-system")
	if _, err := os.Stat(inactive); !os.IsNotExist(err) {
		t.Fatalf("inactive role skill leaked into the bundle: %v", err)
	}
}

func TestCompiledDeliveryUsesCanonicalProse(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	result, err := Run(fixture(t, "compiled.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "delivery", "compiled.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, personalityName := range p.Roles["platform-eng"].Personalities {
		heading := "# " + strings.ToUpper(personalityName[:1]) + personalityName[1:]
		if !strings.Contains(string(body), heading) {
			t.Fatalf("compiled prose missing %q:\n%s", heading, body)
		}
	}
}

func TestDesignerPageExperienceBoundaryMatchesNativeAndCompiledDelivery(t *testing.T) {
	provider := t.TempDir()
	skillDir := filepath.Join(provider, ".agents", "skills", "fixture")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: fixture\ndescription: Fixture capability.\n---\n\n# Fixture\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := p.Roles["frontend-eng"].Briefing
	for _, delivery := range []string{
		schema.DeliveryNativeSkills,
		schema.DeliveryCompiled,
	} {
		t.Run(delivery, func(t *testing.T) {
			result, err := RunRoots(
				&schema.Request{
					Role:     "frontend-eng",
					Delivery: delivery,
				},
				[]RootSource{{ID: "fixture", Root: provider}},
				t.TempDir(),
				Options{},
			)
			if err != nil {
				t.Fatal(err)
			}
			var path string
			if delivery == schema.DeliveryCompiled {
				path = filepath.Join(result.Bundle.Dir, "delivery", "compiled.md")
			} else {
				path = filepath.Join(
					result.Bundle.Dir,
					"content",
					"skills",
					"roster%3Acore",
					"role-frontend-eng",
					"SKILL.md",
				)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), want) {
				t.Fatalf("%s delivery omitted the canonical Designer page-experience boundary:\n%s",
					delivery, raw)
			}
		})
	}
}

func TestRepeatedRunsReuseWithoutRewriting(t *testing.T) {
	out := t.TempDir()
	first, err := Run(fixture(t, "native.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	if first.Bundle.Reused {
		t.Fatal("first run must materialize, not reuse")
	}
	manifest := filepath.Join(first.Bundle.Dir, "manifest.json")
	before, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Run(fixture(t, "native.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Bundle.Reused || second.Bundle.Dir != first.Bundle.Dir || second.Bundle.Key != first.Bundle.Key {
		t.Fatalf("expected cache reuse, got %+v then %+v", first.Bundle, second.Bundle)
	}
	after, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("reuse must not rewrite bundle files")
	}
}

// The override has to reach the rendered card, the seats, and the bundle key.
// Parsing it and dropping it downstream is the failure worth a test.
func TestSeatIdentityOverrideReachesTheRenderedBundle(t *testing.T) {
	dir := t.TempDir()
	request := filepath.Join(dir, "request.kdl")
	if err := os.WriteFile(request, []byte(`compose {
    role "sysadmin-senior"
    identity name="Echo"
    delivery "native-skills"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	renamed, err := Run(request, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	instructions, err := os.ReadFile(
		filepath.Join(renamed.Bundle.Dir, "content", "instructions.md"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(instructions), "**Agent // Echo**") {
		t.Fatalf("identity card kept the role's own seat:\n%s", instructions)
	}
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	// The role's own creature must be gone rather than merely joined.
	creature := p.RoleCreature("sysadmin-senior")
	if strings.Contains(string(instructions), creature) {
		t.Fatalf("identity card carries both the override and the role's own creature %q", creature)
	}
	// Seats back the statusline, overlay, and manifest, so they move together.
	for _, seat := range readManifest(t, renamed.Bundle.Dir).Identity.Seats {
		if seat.Name != "Echo" {
			t.Fatalf("manifest seat %q = %s", seat.Key, seat.Name)
		}
	}

	baseline := filepath.Join(dir, "baseline.kdl")
	if err := os.WriteFile(baseline, []byte(`compose {
    role "sysadmin-senior"
    delivery "native-skills"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plain, err := Run(baseline, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if plain.Bundle.Key == renamed.Bundle.Key {
		t.Fatal("a renamed seat must not reuse the unrenamed bundle")
	}
}

func TestDifferentDeliveriesGetDifferentBundles(t *testing.T) {
	out := t.TempDir()
	a, err := Run(fixture(t, "native.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Run(fixture(t, "compiled.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	if a.Bundle.Key == b.Bundle.Key {
		t.Fatal("different deliveries must produce different bundle keys")
	}
}

func TestModelTiersGetDistinctBundlesWithIdenticalContext(t *testing.T) {
	provider := t.TempDir()
	skillDir := filepath.Join(provider, ".agents", "skills", "fixture")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: fixture\ndescription: Fixture capability.\n---\n\n# Fixture\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	frontierRequest := &schema.Request{
		Role:      "platform-eng",
		Delivery:  schema.DeliveryNativeSkills,
		ModelTier: schema.ModelTierFrontier,
	}
	frontier, err := RunRoots(
		frontierRequest,
		[]RootSource{{ID: "fixture", Root: provider}},
		out,
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	commodityRequest := *frontierRequest
	commodityRequest.ModelTier = schema.ModelTierCommodity
	commodity, err := RunRoots(
		&commodityRequest,
		[]RootSource{{ID: "fixture", Root: provider}},
		out,
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if frontier.Bundle.Key == commodity.Bundle.Key {
		t.Fatal("different model tiers must produce different bundle keys")
	}
	frontierManifest := readManifest(t, frontier.Bundle.Dir)
	commodityManifest := readManifest(t, commodity.Bundle.Dir)
	if commodityManifest.ModelTier != schema.ModelTierCommodity {
		t.Fatal("commodity bundle manifest lost its model tier")
	}
	if !slices.Equal(frontierManifest.Content, commodityManifest.Content) {
		t.Fatal("model tier changed logical context content")
	}
	if len(frontier.Resolution.Skills) != len(commodity.Resolution.Skills) {
		t.Fatal("model tier changed selected skill count")
	}
	for index, frontierSkill := range frontier.Resolution.Skills {
		commoditySkill := commodity.Resolution.Skills[index]
		if frontierSkill.ID != commoditySkill.ID ||
			frontierSkill.Source != commoditySkill.Source ||
			frontierSkill.EntryPoint != commoditySkill.EntryPoint {
			t.Fatalf("model tier changed selected skill %d: %+v versus %+v", index, frontierSkill, commoditySkill)
		}
	}
}

func TestFailedFinalizeLeavesNoPartialBundle(t *testing.T) {
	out := t.TempDir()
	probe, err := Run(fixture(t, "native.kdl"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(out, probe.Bundle.Key)
	if err := os.WriteFile(blocker, []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(fixture(t, "native.kdl"), out); err == nil {
		t.Fatal("expected finalize to fail when the target path is blocked")
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".stage-") {
			t.Fatalf("staging directory leaked: %s", e.Name())
		}
	}
	raw, err := os.ReadFile(blocker)
	if err != nil || string(raw) != "occupied" {
		t.Fatalf("blocked target must be untouched, got %q / %v", raw, err)
	}
}

func TestTraceRecordsDecisions(t *testing.T) {
	out := t.TempDir()
	result, err := Run(fixture(t, "native.kdl"), out)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	var trace bundle.Trace
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.Format != "agent-compose.trace" || len(trace.Decisions) == 0 || len(trace.Providers) == 0 {
		t.Fatalf("unexpected trace: %+v", trace)
	}
	outcomes := map[string]bool{}
	for _, d := range trace.Decisions {
		if d.Reason == "" {
			t.Fatalf("decision without a reason: %+v", d)
		}
		outcomes[d.Outcome] = true
	}
	for _, want := range []string{"selected", "delivered"} {
		if !outcomes[want] {
			t.Fatalf("trace missing %q outcome: %+v", want, trace.Decisions)
		}
	}
	for _, provider := range trace.Providers {
		if provider.Reason == "" || provider.Category == "" || provider.Scope == "" {
			t.Fatalf("provider report is incomplete: %+v", provider)
		}
		if provider.Outcome == resolver.OutcomeExcluded &&
			(provider.Skills != 0 || provider.ContextBytes != 0 || provider.ApproximateTokens != 0) {
			t.Fatalf("excluded provider contributes context: %+v", provider)
		}
	}
}

func mustExist(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		t.Fatalf("expected %s in bundle: %v", rel, err)
	}
}

// The composed body is what a consumer pays for on every turn, so its size is
// recorded rather than left to be measured downstream. See #275.
func TestTheManifestRecordsTheComposedBodySize(t *testing.T) {
	for _, tc := range []struct{ request, file string }{
		{"compiled.kdl", "delivery/compiled.md"},
		{"native.kdl", "content/instructions.md"},
	} {
		result, err := Run(fixture(t, tc.request), t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		manifest := readManifest(t, result.Bundle.Dir)
		body, err := os.ReadFile(filepath.Join(result.Bundle.Dir, tc.file))
		if err != nil {
			t.Fatal(err)
		}
		if manifest.Delivery.BodyBytes != len(body) {
			t.Errorf(
				"%s: manifest body_bytes = %d, %s is %d bytes",
				tc.request, manifest.Delivery.BodyBytes, tc.file, len(body),
			)
		}
		if manifest.Delivery.BodyBytes == 0 {
			t.Errorf("%s: composed body size is zero", tc.request)
		}
	}
}
