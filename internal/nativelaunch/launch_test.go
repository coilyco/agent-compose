package nativelaunch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/agentid"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/describe"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/nativeui"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/repositoryplan"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

type testRoleProvider struct {
	Path       string
	Required   bool
	Skills     []string
	Name       string
	DeclaredBy string
}

type testRepositoryPlan struct {
	ProjectsRoot  string
	Defaults      []string
	Harnesses     map[string][]string
	RoleProviders map[string][]testRoleProvider
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProvider(t *testing.T, root string, withRole bool) {
	t.Helper()
	writeFile(
		t,
		filepath.Join(root, ".agents", "skills", "ordinary", "SKILL.md"),
		"---\nname: ordinary\ndescription: Ordinary test skill.\n---\n\n# Ordinary\n",
	)
	if !withRole {
		return
	}
	writeFile(
		t,
		filepath.Join(root, ".agents", "composed", "design-method", "COMPOSED.md"),
		"---\nname: design-method\ndescription: Role-composed design method.\n---\n\n# Design method\n",
	)
	writeFile(
		t,
		filepath.Join(root, ".agents", "roles.kdl"),
		"roles {\n    role frontend {\n        composed-skill design-method\n    }\n}\n",
	)
}

func writeOrdinarySkill(t *testing.T, root, name string) {
	t.Helper()
	writeFile(
		t,
		filepath.Join(root, ".agents", "skills", name, "SKILL.md"),
		"---\nname: "+name+"\ndescription: Role-only fixture skill.\n---\n\n# "+name+"\n",
	)
}

func writeEligibilityManifest(t *testing.T, path string, input testRepositoryPlan) {
	t.Helper()
	const defaultSource = "test/policy"
	basePaths := append([]string(nil), input.Defaults...)
	for _, paths := range input.Harnesses {
		basePaths = append(basePaths, paths...)
	}
	selection := func(path, scope, source, reason string) repositoryplan.Selection {
		identity, err := filepath.Rel(input.ProjectsRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		return repositoryplan.Selection{
			Identity: filepath.ToSlash(identity), Path: path,
			Source: source, Scope: scope, Reason: reason,
		}
	}
	inputs := map[string]bool{defaultSource: true}
	roles := map[string][]repositoryplan.Selection{}
	for _, role := range []string{"science", "advocate", "frontend", "gamedev", "platform", "sysadmin", "director"} {
		for _, path := range basePaths {
			roles[role] = append(roles[role], selection(path, "operating-context", defaultSource, "test operating context"))
		}
		for _, provider := range input.RoleProviders[role] {
			source := provider.DeclaredBy
			if source == "" {
				source = defaultSource
			}
			inputs[source] = true
			name := provider.Name
			if name == "" {
				name = filepath.Base(provider.Path)
			}
			reason := fmt.Sprintf("role %q uses a role provider", role)
			if provider.Name != "" && source != "" {
				reason = fmt.Sprintf("role %q -> provider %q declared by %s -> selected catalogue", role, provider.Name, source)
			}
			item := selection(provider.Path, "provider", source, reason)
			item.Required = provider.Required
			item.Skills = append([]string(nil), provider.Skills...)
			item.Name = name
			item.DeclaredBy = source
			roles[role] = append(roles[role], item)
		}
		sort.Slice(roles[role], func(i, j int) bool { return roles[role][i].Identity < roles[role][j].Identity })
	}
	resident := map[string]repositoryplan.Selection{}
	for _, selections := range roles {
		for _, item := range selections {
			item.Scope = "role-union"
			item.Reason = "test role union"
			resident[item.Identity] = item
		}
	}
	identities := make([]string, 0, len(resident))
	for identity := range resident {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	residency := make([]repositoryplan.Selection, 0, len(identities))
	for _, identity := range identities {
		residency = append(residency, resident[identity])
	}
	raw, err := repositoryplan.Marshal(repositoryplan.Plan{
		Format: repositoryplan.Format, ProjectsRoot: input.ProjectsRoot,
		Inputs: testPolicyInputs(inputs),
		Roles:  roles, Residency: residency,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(raw))
}

func testPolicyInputs(sources map[string]bool) []repositoryplan.Input {
	identities := make([]string, 0, len(sources))
	for identity := range sources {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	inputs := make([]repositoryplan.Input, 0, len(identities))
	for _, identity := range identities {
		inputs = append(inputs, repositoryplan.Input{
			Identity: identity,
			Revision: "0123456789012345678901234567890123456789",
			Policy: repositoryplan.PolicyInput{
				Path:   repositoryplan.PolicyPath,
				SHA256: "sha256:0123456789012345678901234567890123456789012345678901234567890123",
			},
		})
	}
	return inputs
}

func skillNames(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

func providerReport(t *testing.T, result *Result, source string) resolver.ProviderReport {
	t.Helper()
	for _, provider := range result.Composition.Resolution.Providers {
		if provider.Source == source {
			return provider
		}
	}
	t.Fatalf("provider report omitted %s: %+v", source, result.Composition.Resolution.Providers)
	return resolver.ProviderReport{}
}

func writeManifest(t *testing.T, path, projectsRoot, provider string) {
	t.Helper()
	writeEligibilityManifest(t, path, testRepositoryPlan{
		ProjectsRoot: projectsRoot,
		Defaults:     []string{provider},
		Harnesses:    map[string][]string{},
	})
}

func TestRefreshProjectsAssignedRoleBundleForEveryNativeHarness(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "coilyco-flight-deck", "agentic-os")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	out := filepath.Join(t.TempDir(), "bundles")
	profile, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	designer := profile.Roles["frontend"]

	cases := map[string]struct {
		instructions string
		skills       string
	}{
		"claude":   {"CLAUDE.md", ".claude/skills"},
		"codex":    {"AGENTS.md", ".agents/skills"},
		"goose":    {".goosehints", ".agents/skills"},
		"opencode": {"AGENTS.md", ".agents/skills"},
	}
	for harness, tc := range cases {
		t.Run(harness, func(t *testing.T) {
			target := t.TempDir()
			result, err := Refresh(Options{
				Role:      "frontend",
				Harness:   harness,
				CWD:       projects,
				TargetDir: target,
				PlanPath:  manifest,
				OutDir:    out,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ModelTier != schema.ModelTierFrontier {
				t.Fatalf("default model tier = %q, want frontier", result.ModelTier)
			}
			if result.Composition == nil ||
				result.Composition.Resolution.Request.Role != "frontend" {
				t.Fatalf("composition result does not retain the assigned role: %+v", result.Composition)
			}
			instructions, err := os.ReadFile(filepath.Join(target, tc.instructions))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(instructions), "assigned you the `frontend` role") {
				t.Fatalf("assigned role missing from instructions:\n%s", instructions)
			}
			skills := []string{
				"ordinary",
				"role-frontend",
				"design-method",
			}
			for _, name := range designer.Personalities {
				skills = append(skills, profile.Personalities[name].Skill)
			}
			for _, skill := range skills {
				if _, err := os.Stat(filepath.Join(target, tc.skills, skill, "SKILL.md")); err != nil {
					t.Errorf("selected skill %s missing: %v", skill, err)
				}
			}
			if _, err := os.Stat(filepath.Join(
				target,
				tc.skills,
				"role-platform",
				"SKILL.md",
			)); !os.IsNotExist(err) {
				t.Errorf("inactive role-platform entered the bundle: %v", err)
			}
		})
	}
}

// The seat name and the settings fragment are the two surfaces a Claude session
// receives as launch arguments, so Refresh resolves both without host mutation.
func TestRefreshEmitsClaudeLaunchIdentity(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "provider")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)

	result, err := Refresh(Options{
		Role:      "frontend",
		Harness:   "claude",
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SeatName != "Delphi [she] (Frontend Engineer)" {
		t.Errorf("seat name = %q, want Delphi [she] (Frontend Engineer)", result.SeatName)
	}
	want := filepath.Join(result.BundleDir, HarnessSettingsFile)
	if result.HarnessSettings != want {
		t.Fatalf("settings path = %q, want %q", result.HarnessSettings, want)
	}
	raw, err := os.ReadFile(result.HarnessSettings)
	if err != nil {
		t.Fatal(err)
	}
	var settings nativeui.Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Theme != "custom:aos-frontend" {
		t.Errorf("theme = %q, want custom:aos-frontend", settings.Theme)
	}
	if len(settings.SpinnerVerbs.Verbs) == 0 {
		t.Error("settings fragment carries no spinner verbs")
	}
	if len(settings.SpinnerTips.Tips) == 0 {
		t.Error("settings fragment carries no spinner tips")
	}
}

// Only Claude Code reads a settings fragment as a launch argument. Every other
// harness still resolves a seat name, since that is identity, not a file.
func TestRefreshWithholdsSettingsFromOtherHarnesses(t *testing.T) {
	t.Setenv(agentid.SessionEnv, "")
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "provider")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)

	for _, harness := range []string{"codex", "goose", "opencode"} {
		t.Run(harness, func(t *testing.T) {
			result, err := Refresh(Options{
				Role:      "frontend",
				Harness:   harness,
				CWD:       projects,
				TargetDir: t.TempDir(),
				PlanPath:  manifest,
				OutDir:    filepath.Join(t.TempDir(), "bundles"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.HarnessSettings != "" {
				t.Errorf("harness %q got settings %q, want none", harness, result.HarnessSettings)
			}
			if _, err := os.Stat(filepath.Join(
				result.BundleDir,
				HarnessSettingsFile,
			)); !os.IsNotExist(err) {
				t.Errorf("harness %q bundle holds a Claude settings fragment: %v", harness, err)
			}
			if result.SeatName != "Delphi [she] (Frontend Engineer)" {
				t.Errorf(
					"harness %q seat name = %q, want Delphi [she] (Frontend Engineer)",
					harness,
					result.SeatName,
				)
			}
		})
	}
}

func TestRefreshDefaultsToFrontierModelTier(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "provider")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)

	result, err := Refresh(Options{
		Role:      "frontend",
		Harness:   "goose",
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ModelTier != schema.ModelTierFrontier {
		t.Fatalf("default model tier = %q, want frontier", result.ModelTier)
	}
}

func TestRefreshAcceptsCanonicalModelTierAndRejectsUnknownTier(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "provider")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	options := Options{
		Role:      "frontend",
		Harness:   "codex",
		ModelTier: schema.ModelTierCommodity,
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	}

	result, err := Refresh(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.ModelTier != schema.ModelTierCommodity ||
		result.Composition.Resolution.Request.ModelTier != schema.ModelTierCommodity {
		t.Fatalf("commodity model tier not retained: %+v", result)
	}

	options.ModelTier = "budget"
	_, err = Refresh(options)
	if err == nil || !strings.Contains(err.Error(), "unsupported native model tier") {
		t.Fatalf("unknown model-tier error = %v", err)
	}
}

func TestRefreshRequiresRoleComposedProvider(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "example", "ordinary")
	writeProvider(t, provider, false)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)

	_, err := Refresh(Options{
		Role:      "frontend",
		Harness:   "codex",
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	})
	if err == nil || !strings.Contains(err.Error(), "needs an eligible provider") {
		t.Fatalf("role-provider error = %v", err)
	}
}

func TestRoleProvidersStayScopedAcrossNativeAndStagedHomes(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	base := filepath.Join(projects, "example", "base")
	infrastructure := filepath.Join(projects, "example", "infrastructure")
	deploy := filepath.Join(projects, "example", "deploy")
	missingOptional := filepath.Join(projects, "example", "optional")
	writeProvider(t, base, true)
	writeOrdinarySkill(t, base, "repo-infrastructure")
	writeOrdinarySkill(t, base, "repo-deploy")
	writeOrdinarySkill(t, infrastructure, "infrastructure-ops")
	writeOrdinarySkill(t, infrastructure, "repo-infrastructure")
	writeOrdinarySkill(t, deploy, "deploy-ops")
	writeOrdinarySkill(t, deploy, "repo-deploy")
	manifestPath := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeEligibilityManifest(t, manifestPath, testRepositoryPlan{
		ProjectsRoot: projects,
		Defaults:     []string{base},
		Harnesses:    map[string][]string{},
		RoleProviders: map[string][]testRoleProvider{
			"sysadmin": {
				{Path: infrastructure, Required: true, Name: "infrastructure", DeclaredBy: "example/aosk"},
				{Path: deploy, Required: true, Name: "deploy", DeclaredBy: "example/aosk"},
				{Path: missingOptional, Name: "optional", DeclaredBy: "example/aosk"},
			},
		},
	})
	out := filepath.Join(t.TempDir(), "bundles")
	results := map[string]*Result{}
	targets := map[string]string{}
	for _, role := range []string{"sysadmin", "platform"} {
		target := t.TempDir()
		targets[role] = target
		result, err := Refresh(Options{
			Role:      role,
			Harness:   "codex",
			CWD:       projects,
			TargetDir: target,
			PlanPath:  manifestPath,
			OutDir:    out,
		})
		if err != nil {
			t.Fatalf("refresh %s: %v", role, err)
		}
		results[role] = result
		for _, skill := range []string{"infrastructure-ops", "deploy-ops"} {
			_, err := os.Stat(filepath.Join(target, ".agents", "skills", skill, "SKILL.md"))
			if role == "sysadmin" && err != nil {
				t.Errorf("ops native bundle omitted %s: %v", skill, err)
			}
			if role == "platform" && !os.IsNotExist(err) {
				t.Errorf("engineer native bundle leaked %s: %v", skill, err)
			}
		}
	}

	for _, role := range []string{"sysadmin", "platform"} {
		staged := t.TempDir()
		if _, err := project.ProjectScoped(results[role].BundleDir, "claude", staged, project.ScopeHome); err != nil {
			t.Fatal(err)
		}
		for _, skill := range []string{"infrastructure-ops", "deploy-ops"} {
			_, err := os.Stat(filepath.Join(staged, ".claude", "skills", skill, "SKILL.md"))
			if role == "sysadmin" && err != nil {
				t.Errorf("staged Ops home omitted %s: %v", skill, err)
			}
			if role == "platform" && !os.IsNotExist(err) {
				t.Errorf("staged Engineer home leaked %s: %v", skill, err)
			}
		}
		nativeSkills := skillNames(t, filepath.Join(targets[role], ".agents", "skills"))
		stagedSkills := skillNames(t, filepath.Join(staged, ".claude", "skills"))
		if !slices.Equal(nativeSkills, stagedSkills) {
			t.Fatalf("native and staged %s skills differ: native=%v staged=%v", role, nativeSkills, stagedSkills)
		}
	}

	for _, result := range results {
		baseReport := providerReport(t, result, "example--base")
		if baseReport.Category != resolver.ProviderCategoryCatalogue ||
			baseReport.Scope != "default" ||
			baseReport.Outcome != resolver.OutcomeSelected ||
			baseReport.Skills == 0 || baseReport.ContextBytes == 0 {
			t.Fatalf("ordinary catalogue report = %+v", baseReport)
		}
	}
	for _, source := range []string{"example--infrastructure", "example--deploy"} {
		selected := providerReport(t, results["sysadmin"], source)
		if selected.Category != resolver.ProviderCategoryRole ||
			selected.Scope != "role" ||
			selected.Outcome != resolver.OutcomeSelected ||
			selected.Skills != 1 || selected.ContextBytes == 0 || selected.ApproximateTokens == 0 {
			t.Fatalf("selected role provider %s report = %+v", source, selected)
		}
		excluded := providerReport(t, results["platform"], source)
		if excluded.Category != resolver.ProviderCategoryRole ||
			excluded.Scope != "role" ||
			excluded.Outcome != resolver.OutcomeExcluded ||
			excluded.Skills != 0 || excluded.ContextBytes != 0 || excluded.ApproximateTokens != 0 {
			t.Fatalf("excluded role provider %s report = %+v", source, excluded)
		}
	}

	selectedWhy, err := describe.Why(results["sysadmin"].BundleDir, "skill:infrastructure-ops", describe.Options{})
	if err != nil || !strings.Contains(
		selectedWhy,
		"role \"sysadmin\" -> provider \"infrastructure\" declared by example/aosk -> selected catalogue",
	) {
		t.Fatalf("selected role-provider why = %q, err=%v", selectedWhy, err)
	}
	excludedWhy, err := describe.Why(results["platform"].BundleDir, "skill:infrastructure-ops", describe.Options{})
	if err != nil || !strings.Contains(excludedWhy, "not selected role \"platform\"") {
		t.Fatalf("excluded role-provider why = %q, err=%v", excludedWhy, err)
	}
	if !strings.Contains(excludedWhy, "provider: role-provider/role") ||
		!strings.Contains(excludedWhy, "context: 0 skills, 0 bytes, approximately 0 tokens") {
		t.Fatalf("excluded role-provider budget missing from why output: %q", excludedWhy)
	}
	optionalWhy, err := describe.Why(results["sysadmin"].BundleDir, "source:example--optional", describe.Options{})
	if err != nil || !strings.Contains(optionalWhy, "optional role provider") {
		t.Fatalf("optional missing provider why = %q, err=%v", optionalWhy, err)
	}
	pointerWhy, err := describe.Why(
		results["sysadmin"].BundleDir,
		"skill:repo-infrastructure",
		describe.Options{},
	)
	if err != nil ||
		!strings.Contains(pointerWhy, "provider: catalogue/default") ||
		!strings.Contains(pointerWhy, "provider: role-provider/role") ||
		!strings.Contains(pointerWhy, "outcome: shadowed") {
		t.Fatalf("ordinary pointer and role-provider provenance = %q, err=%v", pointerWhy, err)
	}
	described, err := describe.Bundle(results["platform"].BundleDir, describe.Options{All: true})
	if err != nil ||
		!strings.Contains(described, "example--infrastructure") ||
		!strings.Contains(described, "(role-provider/role)") ||
		!strings.Contains(described, "0 skills // 0 bytes // ~0 tokens") {
		t.Fatalf("provider-aware describe output = %q, err=%v", described, err)
	}
}

func TestRoleProviderSelectorsMatchAcrossNativeAndStagedHarnesses(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	base := filepath.Join(projects, "example", "base")
	hardware := filepath.Join(projects, "example", "hardware")
	writeProvider(t, base, true)
	writeProvider(t, hardware, false)
	for _, skill := range []string{
		"compute-stack",
		"home-power-strip",
		"machine-alpha",
		"machine-beta",
	} {
		writeOrdinarySkill(t, hardware, skill)
	}
	manifestPath := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeEligibilityManifest(t, manifestPath, testRepositoryPlan{
		ProjectsRoot: projects,
		Defaults:     []string{base},
		Harnesses:    map[string][]string{},
		RoleProviders: map[string][]testRoleProvider{
			"frontend": {{
				Path:     hardware,
				Required: true,
				Skills:   []string{"compute-stack", "machine-*"},
			}},
		},
	})

	cases := map[string]struct {
		nativeSkills string
		stagedSkills string
	}{
		"claude":   {".claude/skills", ".claude/skills"},
		"codex":    {".agents/skills", ".agents/skills"},
		"goose":    {".agents/skills", ".agents/skills"},
		"opencode": {".agents/skills", ".agents/skills"},
	}
	for harness, tc := range cases {
		t.Run(harness, func(t *testing.T) {
			target := t.TempDir()
			result, err := Refresh(Options{
				Role:      "frontend",
				Harness:   harness,
				CWD:       projects,
				TargetDir: target,
				PlanPath:  manifestPath,
				OutDir:    filepath.Join(t.TempDir(), "bundles"),
			})
			if err != nil {
				t.Fatal(err)
			}
			staged := t.TempDir()
			if _, err := project.ProjectScoped(result.BundleDir, harness, staged, project.ScopeHome); err != nil {
				t.Fatal(err)
			}
			for _, root := range []string{
				filepath.Join(target, tc.nativeSkills),
				filepath.Join(staged, tc.stagedSkills),
			} {
				for _, skill := range []string{"compute-stack", "machine-alpha", "machine-beta"} {
					if _, err := os.Stat(filepath.Join(root, skill, "SKILL.md")); err != nil {
						t.Errorf("%s omitted selected %s: %v", root, skill, err)
					}
				}
				for _, skill := range []string{"home-power-strip"} {
					if _, err := os.Stat(filepath.Join(root, skill, "SKILL.md")); !os.IsNotExist(err) {
						t.Errorf("%s leaked excluded %s: %v", root, skill, err)
					}
				}
			}
			report := providerReport(t, result, "example--hardware")
			if report.Skills != 3 || report.ContextBytes == 0 ||
				!strings.Contains(report.SelectorReason, "admitted 3 of 5 catalogue skills") {
				t.Fatalf("selected-slice provider report = %+v", report)
			}
			why, err := describe.Why(result.BundleDir, "skill:home-power-strip", describe.Options{})
			if err != nil || !strings.Contains(why, "matched no configured selector pattern") ||
				!strings.Contains(why, "selector: ordinary skill selector") ||
				!strings.Contains(why, "context: 3 skills") {
				t.Fatalf("selector exclusion why = %q, err=%v", why, err)
			}
		})
	}
}

func TestMissingRequiredRoleProviderFailsExplicitly(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	base := filepath.Join(projects, "example", "base")
	missing := filepath.Join(projects, "example", "required")
	writeProvider(t, base, true)
	manifestPath := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeEligibilityManifest(t, manifestPath, testRepositoryPlan{
		ProjectsRoot: projects,
		Defaults:     []string{base},
		Harnesses:    map[string][]string{},
		RoleProviders: map[string][]testRoleProvider{
			"sysadmin": {{Path: missing, Required: true}},
		},
	})
	_, err := Refresh(Options{
		Role:      "sysadmin",
		Harness:   "codex",
		CWD:       projects,
		TargetDir: t.TempDir(),
		PlanPath:  manifestPath,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	})
	if err == nil || !strings.Contains(err.Error(), "required role provider example/required") {
		t.Fatalf("required missing provider error = %v", err)
	}
}

// A staged session home owns its whole load-point surface, so the role must land
// at the harness global paths and leave the checkouts untouched.
func TestRefreshProjectsIntoRuntimeHomeAtGlobalLoadPoints(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "coilyco-flight-deck", "agentic-os")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	out := filepath.Join(t.TempDir(), "bundles")

	cases := map[string]struct {
		instructions string
		skills       string
	}{
		"claude":   {".claude/CLAUDE.md", ".claude/skills"},
		"codex":    {".codex/AGENTS.md", ".agents/skills"},
		"goose":    {".config/goose/.goosehints", ".agents/skills"},
		"opencode": {".config/opencode/AGENTS.md", ".agents/skills"},
	}
	for harness, tc := range cases {
		t.Run(harness, func(t *testing.T) {
			home, target := t.TempDir(), t.TempDir()
			result, err := Refresh(Options{
				Role:        "frontend",
				Harness:     harness,
				CWD:         projects,
				TargetDir:   target,
				RuntimeHome: home,
				PlanPath:    manifest,
				OutDir:      out,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Projected == 0 {
				t.Fatal("home projection reported no files")
			}
			instructions, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(tc.instructions)))
			if err != nil {
				t.Fatalf("instructions missing from the runtime home: %v", err)
			}
			if !strings.Contains(string(instructions), "assigned you the `frontend` role") {
				t.Fatalf("assigned role missing from instructions:\n%s", instructions)
			}
			skill := filepath.Join(home, filepath.FromSlash(tc.skills), "role-frontend", "SKILL.md")
			if _, err := os.Stat(skill); err != nil {
				t.Errorf("role skill missing from the runtime home: %v", err)
			}
			// The projects root is where the old repo-scope projection landed.
			entries, err := os.ReadDir(target)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("runtime-home projection also wrote into the target: %v", entries)
			}
		})
	}
}

// A session home replaces the host global load point, so the bundle has to carry
// the doctrine that file supplied rather than inherit it.
func TestRefreshLeadsInstructionsWithTheOperatingBase(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "coilyco-flight-deck", "agentic-os")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	home := t.TempDir()

	const base = "# Agent instructions\n\nPronouns stay she/her."
	if _, err := Refresh(Options{
		Role:          "frontend",
		Harness:       "claude",
		CWD:           projects,
		TargetDir:     t.TempDir(),
		RuntimeHome:   home,
		OperatingBase: base,
		PlanPath:      manifest,
		OutDir:        filepath.Join(t.TempDir(), "bundles"),
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	baseAt, roleAt := strings.Index(body, base), strings.Index(body, "# Role instructions")
	if baseAt < 0 {
		t.Fatalf("operating base missing from the projected instructions:\n%s", body)
	}
	if roleAt < 0 || baseAt > roleAt {
		t.Fatalf("operating base must lead the role card:\n%s", body)
	}
}

// stripRosterCards deletes a card to the next heading, so an appendix appended
// after the last card went with it. It travels separately now. #6987.
func TestRefreshKeepsTheAppendixBehindTheLastRosterCard(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	last := p.RoleOrder[len(p.RoleOrder)-1]
	card, err := p.RenderRoleIdentityCard(last, p.Roles[last].FavoriteColor, p.RoleActiveBoundaries(last))
	if err != nil {
		t.Fatal(err)
	}

	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "coilyco-flight-deck", "agentic-os")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	home := t.TempDir()

	base := "# Agent instructions\n\nPronouns stay she/her.\n\n" + card
	const tail = "<!-- appendix: agent-compose.yaml entry 1 -->\n## Checkin dashboard\n\nRead the dashboard first."
	if _, err := Refresh(Options{
		Role:              "frontend",
		Harness:           "claude",
		CWD:               projects,
		TargetDir:         t.TempDir(),
		RuntimeHome:       home,
		OperatingBase:     base,
		OperatingAppendix: tail,
		PlanPath:          manifest,
		OutDir:            filepath.Join(t.TempDir(), "bundles"),
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "## Checkin dashboard") {
		t.Fatalf("appendix missing from the projected instructions:\n%s", body)
	}
	if !strings.Contains(body, "Pronouns stay she/her.") {
		t.Errorf("operating base doctrine dropped:\n%s", body)
	}
	if heading := "\n# " + p.RoleDisplayName(last) + "\n"; strings.Contains(body, heading) {
		t.Errorf("base still carries the %q roster card", heading)
	}
}

// Without a session home the host load point still supplies the base, so
// repeating it in the bundle would double the doctrine.
func TestRefreshOmitsTheOperatingBaseWithoutARuntimeHome(t *testing.T) {
	projects := filepath.Join(t.TempDir(), "projects")
	provider := filepath.Join(projects, "coilyco-flight-deck", "agentic-os")
	writeProvider(t, provider, true)
	manifest := filepath.Join(t.TempDir(), "repository-plan.yaml")
	writeManifest(t, manifest, projects, provider)
	target := t.TempDir()

	if _, err := Refresh(Options{
		Role:      "frontend",
		Harness:   "claude",
		CWD:       projects,
		TargetDir: target,
		PlanPath:  manifest,
		OutDir:    filepath.Join(t.TempDir(), "bundles"),
	}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(target, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "# Role instructions") {
		t.Fatalf("repo-scope instructions must still start at the role card:\n%s", raw)
	}
}

func TestJoinTraitsKeepsTheSerialCommaOffATwoTraitMeld(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		traits []string
		want   string
	}{
		{"none", nil, ""},
		{"one", []string{"grounded"}, "grounded"},
		{"two", []string{"playful", "imaginative"}, "playful and imaginative"},
		{
			"three",
			[]string{"tenacious", "grounded", "protective"},
			"tenacious, grounded, and protective",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := joinTraits(tc.traits); got != tc.want {
				t.Fatalf("joinTraits(%v) = %q, want %q", tc.traits, got, tc.want)
			}
		})
	}
}
