package person

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

func TestLoadEmbeddedRoster(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ai := p.Roles["science"]
	for _, method := range ai.Methods {
		if raw, ok := p.RoleMethodDefinition("science", method); !ok ||
			!strings.Contains(string(raw), "\nname: "+method+"\n") {
			t.Errorf("AI role method %q is missing or mismatched", method)
		}
	}
	identityNames := map[string]string{}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		if role.Skill != "role-"+roleName ||
			role.SkillSource == "" ||
			!strings.HasPrefix(role.SkillDigest, "sha256:") {
			t.Errorf("role %q skill provenance is incomplete: %+v", roleName, role)
		}
		if raw, ok := p.RoleSkillDefinition(roleName); !ok ||
			!strings.Contains(string(raw), "\nname: "+role.Skill+"\n") {
			t.Errorf("role %q canonical skill is missing or mismatched", roleName)
		}
		if got := briefingParagraphCount(role.Briefing); got < 3 {
			t.Errorf("role %q briefing has %d paragraphs, want at least three", roleName, got)
		}
		if len(role.Seats) < 2 {
			t.Errorf("role %q has %d seats, want at least claude and codex", roleName, len(role.Seats))
		}
		if role.Identity == nil || role.Identity.Name == "" || role.Identity.Pronouns == "" {
			t.Errorf("role %q identity is incomplete: %+v", roleName, role.Identity)
			continue
		}
		if prior, exists := identityNames[role.Identity.Name]; exists {
			t.Errorf("roles %q and %q share identity name %q", prior, roleName, role.Identity.Name)
		}
		identityNames[role.Identity.Name] = roleName
		seats := map[string]Seat{}
		for _, seat := range role.Seats {
			seats[seat.Harness] = seat
			if seat.Name != role.Identity.Name || seat.Pronouns != role.Identity.Pronouns {
				t.Errorf("role %q seat %q redefines identity: %+v", roleName, seat.Selector(), seat)
			}
		}
		for _, harness := range []string{"claude", "codex"} {
			seat, exists := seats[harness]
			if !exists || seat.Name != role.Identity.Name || seat.Pronouns != role.Identity.Pronouns {
				t.Errorf("role %q %s seat is incomplete: %+v", roleName, harness, seat)
			}
		}
		for _, name := range role.Personalities {
			binding, ok := p.Personalities[name]
			if !ok {
				t.Fatalf("role %q personality %q has no catalog binding", roleName, name)
			}
			if want := "personality-" + name; binding.Skill != want {
				t.Errorf("personality %q skill = %q, want %q", name, binding.Skill, want)
			}
		}
	}
}

func TestValidateCoreBoundariesRejectsUnbalancedRoster(t *testing.T) {
	t.Run("wrong slot count", func(t *testing.T) {
		p, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		role := p.Roles["platform"]
		role.Personalities = append(role.Personalities, "immersed")
		p.Roles["platform"] = role
		if err := validateCorePersonalityMelds(p); err == nil ||
			!strings.Contains(err.Error(), "want exactly 2") {
			t.Fatalf("slot-count validation error = %v", err)
		}
	})

	t.Run("overused personality", func(t *testing.T) {
		p, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		// grounded bonds three seats already, so a fourth binding tips it.
		role := p.Roles["frontend"]
		role.Personalities[1] = "grounded"
		p.Roles["frontend"] = role
		if err := validateCorePersonalityMelds(p); err == nil ||
			!strings.Contains(err.Error(), "want at most three") {
			t.Fatalf("usage validation error = %v", err)
		}
	})

	t.Run("favorite colors too close", func(t *testing.T) {
		p, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		// Two roles melding the same personalities land on the same anchor, so
		// the spread step can only separate them by twice the drift cap.
		role := p.Roles["director"]
		role.Personalities = append([]string(nil), p.Roles["advocate"].Personalities...)
		p.Roles["director"] = role
		if err := p.ResolveFavoriteColors(); err != nil {
			t.Fatal(err)
		}
		if err := validateCorePersonalityMelds(p); err == nil ||
			!strings.Contains(err.Error(), "apart at their closest") {
			t.Fatalf("favorite-color separation error = %v", err)
		}
	})
}

func TestLoadRoleSkillsRejectsMissingAndMalformedDefinitions(t *testing.T) {
	for name, files := range map[string]fstest.MapFS{
		"missing": {},
		"mismatched frontmatter": {
			"roles/builder/SKILL.md": {
				Data: []byte("---\nname: role-other\ndescription: Wrong.\n---\n\nOne.\n\nTwo.\n\nThree.\n"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := &Person{
				Name:      "fixture",
				RoleOrder: []string{"builder"},
				Roles: map[string]Role{
					"builder": {Skill: "role-builder"},
				},
			}
			if err := loadRoleSkills(files, p); err == nil {
				t.Fatal("invalid role skill must fail")
			}
		})
	}
}

func TestLoadRoleSkillsEnforcesAuthoredBodyWordLimit(t *testing.T) {
	for name, words := range map[string]int{
		"at limit":   maxRoleSkillBodyWords,
		"over limit": maxRoleSkillBodyWords + 1,
	} {
		t.Run(name, func(t *testing.T) {
			files := fstest.MapFS{
				"roles/builder/SKILL.md": {
					Data: roleSkillWordLimitFixture(words),
				},
			}
			p := &Person{
				Name:      "fixture",
				RoleOrder: []string{"builder"},
				Roles: map[string]Role{
					"builder": {Skill: "role-builder"},
				},
			}
			err := loadRoleSkills(files, p)
			if words == maxRoleSkillBodyWords && err != nil {
				t.Fatalf("role skill at word limit failed: %v", err)
			}
			if words > maxRoleSkillBodyWords &&
				(err == nil || !strings.Contains(err.Error(), "maximum is 1200")) {
				t.Fatalf("role skill over word limit error = %v", err)
			}
		})
	}
}

// TestBoundaryBodiesDoNotConsumeTheRoleWordBudget is the load-bearing guarantee:
// shared doctrine must not compete with the charter for the role's word budget.
func TestBoundaryBodiesDoNotConsumeTheRoleWordBudget(t *testing.T) {
	files := fstest.MapFS{
		"roles/builder/SKILL.md": {
			Data: roleSkillWordLimitFixture(maxRoleSkillBodyWords),
		},
		"definitions/skills/boundary-shared/SKILL.md": {
			Data: boundaryWordLimitFixture(maxBoundarySkillBodyWords),
		},
	}
	p := &Person{
		Name:      "fixture",
		RoleOrder: []string{"builder"},
		Roles: map[string]Role{
			"builder": {Skill: "role-builder", Boundaries: []string{"shared"}},
		},
		BoundaryOrder: []string{"shared"},
		Boundaries: map[string]Boundary{
			"shared": {Skill: "boundary-shared", Summary: "shared fixture doctrine"},
		},
	}
	if err := loadRoleSkills(files, p); err != nil {
		t.Fatalf("role skill at its word limit failed alongside a boundary: %v", err)
	}
	if err := loadBoundarySkills(files, p); err != nil {
		t.Fatalf("boundary at its own word limit failed: %v", err)
	}
	// Role at its cap plus boundary at its own proves the boundary is additive.
	if got := roleSkillBodyWordCount(p.Roles["builder"].Briefing); got != maxRoleSkillBodyWords {
		t.Fatalf("role briefing word count = %d, want %d", got, maxRoleSkillBodyWords)
	}
	if _, ok := p.BoundarySkillDefinition("shared"); !ok {
		t.Fatal("boundary body did not load")
	}

	over := fstest.MapFS{
		"definitions/skills/boundary-shared/SKILL.md": {
			Data: boundaryWordLimitFixture(maxBoundarySkillBodyWords + 1),
		},
	}
	err := loadBoundarySkills(over, p)
	if err == nil || !strings.Contains(err.Error(), "maximum is 200") {
		t.Fatalf("boundary over its own word limit error = %v", err)
	}
}

func TestValidateBoundaryOwnersRejectsIncoherentPairs(t *testing.T) {
	for name, p := range map[string]*Person{
		"missing owner": {
			RoleOrder: []string{"builder"},
			Roles: map[string]Role{
				"builder": {Skill: "role-builder", Boundaries: []string{"shared"}},
			},
			BoundaryOrder: []string{"shared"},
			Boundaries: map[string]Boundary{
				"shared": {Skill: "boundary-shared"},
			},
		},
		"unknown owner": {
			RoleOrder: []string{"builder"},
			Roles: map[string]Role{
				"builder": {Skill: "role-builder", Boundaries: []string{"shared"}},
			},
			BoundaryOrder: []string{"shared"},
			Boundaries: map[string]Boundary{
				"shared": {Skill: "boundary-shared", Owner: "absent"},
			},
		},
		"owner declares the boundary": {
			RoleOrder: []string{"builder", "mirror"},
			Roles: map[string]Role{
				"builder": {Skill: "role-builder", Boundaries: []string{"shared"}},
				"mirror":  {Skill: "role-mirror", Boundaries: []string{"shared"}},
			},
			BoundaryOrder: []string{"shared"},
			Boundaries: map[string]Boundary{
				"shared": {Skill: "boundary-shared", Owner: "mirror"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateBoundaryOwners(p); err == nil {
				t.Fatal("incoherent boundary owner passed validation")
			}
		})
	}

	coherent := &Person{
		RoleOrder: []string{"builder", "mirror"},
		Roles: map[string]Role{
			"builder": {Skill: "role-builder", Boundaries: []string{"shared"}},
			"mirror":  {Skill: "role-mirror"},
		},
		BoundaryOrder: []string{"shared"},
		Boundaries: map[string]Boundary{
			"shared": {Skill: "boundary-shared", Owner: "mirror"},
		},
	}
	if err := validateBoundaryOwners(coherent); err != nil {
		t.Fatalf("coherent boundary owner rejected: %v", err)
	}
}

func TestValidateRoleAdjacentsRejectsIncompleteGraphs(t *testing.T) {
	for name, p := range map[string]*Person{
		"out-degree below the fixed width": {
			RoleOrder: []string{"builder", "mirror"},
			Roles: map[string]Role{
				"builder": {
					Skill:     "role-builder",
					Adjacents: []Adjacent{{Role: "mirror", Reason: "absorbs review"}},
				},
				"mirror": {
					Skill: "role-mirror",
					Adjacents: []Adjacent{
						{Role: "builder", Reason: "absorbs build"},
						{Role: "builder", Reason: "duplicate is caught at parse"},
					},
				},
			},
		},
		"partial roster leaves a role undeclared": {
			RoleOrder: []string{"builder", "mirror", "scribe"},
			Roles: map[string]Role{
				"builder": {
					Skill: "role-builder",
					Adjacents: []Adjacent{
						{Role: "mirror", Reason: "absorbs review"},
						{Role: "scribe", Reason: "absorbs the record"},
					},
				},
				"mirror": {
					Skill: "role-mirror",
					Adjacents: []Adjacent{
						{Role: "builder", Reason: "absorbs build"},
						{Role: "scribe", Reason: "absorbs the record"},
					},
				},
				"scribe": {Skill: "role-scribe"},
			},
		},
		"unknown adjacent target": {
			RoleOrder: []string{"builder", "mirror"},
			Roles: map[string]Role{
				"builder": {
					Skill: "role-builder",
					Adjacents: []Adjacent{
						{Role: "mirror", Reason: "absorbs review"},
						{Role: "absent", Reason: "names nothing"},
					},
				},
				"mirror": {
					Skill: "role-mirror",
					Adjacents: []Adjacent{
						{Role: "builder", Reason: "absorbs build"},
						{Role: "builder", Reason: "duplicate is caught at parse"},
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateRoleAdjacents(p); err == nil {
				t.Fatal("incomplete adjacency graph passed validation")
			}
		})
	}

	complete := &Person{
		RoleOrder: []string{"builder", "mirror", "builder2"},
		Roles: map[string]Role{
			"builder": {
				Skill: "role-builder",
				Adjacents: []Adjacent{
					{Role: "mirror", Reason: "absorbs review"},
					{Role: "builder2", Reason: "absorbs the second thing"},
				},
			},
			"mirror": {
				Skill: "role-mirror",
				Adjacents: []Adjacent{
					{Role: "builder", Reason: "absorbs build"},
					{Role: "builder2", Reason: "absorbs the second thing"},
				},
			},
			"builder2": {
				Skill: "role-builder2",
				Adjacents: []Adjacent{
					{Role: "builder", Reason: "absorbs build"},
					{Role: "mirror", Reason: "absorbs review"},
				},
			},
		},
	}
	if err := validateRoleAdjacents(complete); err != nil {
		t.Fatalf("complete adjacency graph rejected: %v", err)
	}
}

// An external person package authored before the adjacency axis keeps loading.
// Adjacency is all-or-nothing across a roster rather than required of every one.
func TestValidateRoleAdjacentsSkipsRostersWithoutAdjacency(t *testing.T) {
	p := &Person{
		RoleOrder: []string{"builder", "mirror"},
		Roles: map[string]Role{
			"builder": {Skill: "role-builder"},
			"mirror":  {Skill: "role-mirror"},
		},
	}
	if err := validateRoleAdjacents(p); err != nil {
		t.Fatalf("roster without adjacency rejected: %v", err)
	}
}

// The core roster is the thing the evaluation board reads adjacency from, so a
// missing edge there is a silently thinner board rather than a load failure.
func TestCoreRosterDeclaresACompleteAdjacencyGraph(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range p.roleOrder() {
		role := p.Roles[roleName]
		if len(role.Adjacents) != adjacentsPerRole {
			t.Fatalf(
				"core role %q declares %d adjacent roles, want %d",
				roleName,
				len(role.Adjacents),
				adjacentsPerRole,
			)
		}
		for _, adjacent := range role.Adjacents {
			if adjacent.Role == roleName {
				t.Fatalf("core role %q names itself adjacent", roleName)
			}
			if _, ok := p.Roles[adjacent.Role]; !ok {
				t.Fatalf(
					"core role %q names unknown adjacent %q", roleName, adjacent.Role,
				)
			}
			if strings.TrimSpace(adjacent.Reason) == "" {
				t.Fatalf(
					"core role %q adjacent %q carries no reason",
					roleName,
					adjacent.Role,
				)
			}
		}
	}
}

func boundaryWordLimitFixture(words int) []byte {
	body := strings.TrimSpace(strings.Repeat("word ", words))
	return []byte(
		"---\nname: boundary-shared\ndescription: Shared fixture doctrine.\n---\n\n" +
			boundaryOwnHeading + "\n\nowner side.\n\n" +
			boundaryDeferHeading + "\n\n" + body + "\n",
	)
}

func roleSkillWordLimitFixture(words int) []byte {
	body := strings.TrimSpace(strings.Repeat("word ", words-2))
	return []byte(
		"---\nname: role-builder\ndescription: Build fixtures.\n---\n\n" +
			"# Builder\n\n" + body + "\n\nSecond.\n\nThird.\n",
	)
}

func TestLoadRoleMethodsRejectsIncompleteOrExtraDefinitions(t *testing.T) {
	valid := []byte("---\nname: eval-fixture\ndescription: Evaluate the fixture.\n---\n\n# Fixture\n")
	for name, files := range map[string]fstest.MapFS{
		"missing directory": {},
		"mismatched frontmatter": {
			"roles/builder/skills/eval-fixture/SKILL.md": {
				Data: []byte("---\nname: eval-other\ndescription: Wrong.\n---\n\n# Wrong\n"),
			},
		},
		"extra file": {
			"roles/builder/skills/eval-fixture/SKILL.md":  {Data: valid},
			"roles/builder/skills/eval-fixture/README.md": {Data: []byte("extra")},
		},
		"undeclared skill": {
			"roles/builder/skills/eval-fixture/SKILL.md": {Data: valid},
			"roles/builder/skills/eval-other/SKILL.md":   {Data: valid},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := &Person{
				Name:      "fixture",
				RoleOrder: []string{"builder"},
				Roles: map[string]Role{
					"builder": {Skill: "role-builder", Methods: []string{"eval-fixture"}},
				},
			}
			if err := loadRoleMethods(files, p); err == nil {
				t.Fatal("invalid role method layout must fail")
			}
		})
	}
}

func TestPersonSourceBindsAIOnlyRoleMethods(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := Source(p)
	if err != nil {
		t.Fatal(err)
	}
	// Methods and a guardrail belong to one role and boundaries are shared, so
	// every count derives from the model rather than a second copy of the policy.
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		want := 1 + len(role.Methods) + len(p.RoleActiveBoundaries(roleName))
		if role.Guardrail != "" {
			want++
		}
		if got := len(src.RoleSkills[roleName]); got != want {
			t.Errorf("role %q selected %d person skills, want %d", roleName, got, want)
		}
	}
	for roleName, role := range p.Roles {
		if roleName != "science" && len(role.Methods) != 0 {
			t.Errorf("role %q owns methods, which only the AI role may declare", roleName)
		}
	}
}

func TestLookupCueUsesDeclaredAliasesAndPreservesAmbiguity(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		cue  string
		want []string
	}{
		{cue: "calm", want: []string{"grounded"}},
		{cue: " welcoming ", want: []string{"warm"}},
		{cue: "grounded", want: []string{"grounded"}},
	} {
		got, lookupErr := p.LookupCue(test.cue)
		if lookupErr != nil {
			t.Fatalf("lookup %q: %v", test.cue, lookupErr)
		}
		if strings.Join(got, ",") != strings.Join(test.want, ",") {
			t.Errorf("lookup %q = %v, want %v", test.cue, got, test.want)
		}
	}
	// Negative control: the positive cases above also pass against a lookup
	// that ignores its argument, which is how the silent empty result shipped.
	missed, err := p.LookupCue("zzznotapersonality")
	if err != nil {
		t.Fatalf("unmatched cue must resolve to nothing, not error: %v", err)
	}
	if len(missed) != 0 {
		t.Errorf("unmatched cue resolved to %v, want no names", missed)
	}
	if _, err := NormalizeCue("bad\x00cue"); err == nil {
		t.Fatal("control characters must be rejected")
	}
}

func TestLoadDirectoryKeepsExternalPersonIndependent(t *testing.T) {
	p, err := LoadDirectory(filepath.Join("..", "..", "testdata", "contracts", "person-independent"))
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "workbench" || len(p.Roles) != 1 {
		t.Fatalf("external person identity = %q roles=%d", p.Name, len(p.Roles))
	}
	if _, ok := p.Roles["builder"]; !ok {
		t.Fatalf("external person omitted builder role: %+v", p.RoleOrder)
	}
	if _, inherited := p.Roles["platform"]; inherited {
		t.Fatal("external person inherited the embedded engineer role")
	}
	src, err := Source(p)
	if err != nil {
		t.Fatal(err)
	}
	if src.ID != "person:workbench" || len(src.Skills) != 2 {
		t.Fatalf("external person source = %q skills=%d", src.ID, len(src.Skills))
	}
}

func TestExternalProfileComposesLibrariesSeatsAndCopyContracts(t *testing.T) {
	profile := filepath.Join("..", "..", "examples", "person-profile")
	library := filepath.Join("..", "..", "examples", "shared-personality-library")
	p, err := LoadDirectoryWithLibraries(profile, library)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "example" ||
		len(p.Roles["caption-review"].Personalities) != 1 ||
		strings.Join(p.Roles["bulk-captioner"].Personalities, ",") != "local-guide,shared-care" {
		t.Fatalf("external effective graph is wrong: %+v", p.Roles)
	}
	role := p.Roles["bulk-captioner"]
	if role.Seats[0].Selector() != "chatbot-sonnet-low" ||
		role.Seats[0].Channel != "chatbot" ||
		role.Seats[0].Tier != schema.ModelTierCommodity ||
		role.Seats[0].Pronouns != "they" {
		t.Fatalf("generalized seat was not preserved: %+v", role.Seats)
	}
	if strings.Join(role.SupportedModelTiers, ",") != "frontier,commodity" {
		t.Fatalf("role model tiers = %q", role.SupportedModelTiers)
	}
	if role.CopyContract == nil ||
		role.CopyContract.Source != "person:example:role:bulk-captioner:copy-contract" ||
		!strings.HasPrefix(role.CopyContract.Digest, "sha256:") {
		t.Fatalf("copy contract provenance is incomplete: %+v", role.CopyContract)
	}
	personalities, err := p.PersonalityCatalog(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(personalities) != 3 ||
		personalities[0].Description == "" ||
		personalities[0].SourceLibrary == "" ||
		!strings.HasPrefix(personalities[0].Digest, "sha256:") {
		t.Fatalf("personality catalogue is incomplete: %+v", personalities)
	}
	var unusedFound bool
	for _, entry := range personalities {
		if entry.Slug == "unused-spark" {
			unusedFound = true
			if len(entry.Affinities) != 0 {
				t.Fatalf("unused personality acquired affinities: %+v", entry.Affinities)
			}
		}
	}
	if !unusedFound {
		t.Fatal("effective catalogue omitted unused personality")
	}
	roles, err := p.RoleCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 ||
		roles[0].Skill == "" ||
		roles[0].SkillSource == "" ||
		roles[0].FavoriteColor == "" {
		t.Fatalf("role catalogue is incomplete: %+v", roles)
	}
	seats, err := p.SeatCatalog("bulk-captioner")
	if err != nil {
		t.Fatal(err)
	}
	if len(seats) != 1 || seats[0].Role != "bulk-captioner" ||
		seats[0].Seat.Selector() != "chatbot-sonnet-low" {
		t.Fatalf("seat catalogue is incomplete: %+v", seats)
	}
}

func TestExternalLibraryRejectsSymlinks(t *testing.T) {
	link := filepath.Join(t.TempDir(), "linked-library")
	target, err := filepath.Abs(filepath.Join("..", "..", "examples", "shared-personality-library"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err = LoadDirectoryWithLibraries(
		filepath.Join("..", "..", "examples", "person-profile"),
		link,
	)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked personality library passed: %v", err)
	}
}

func TestExternalLibraryConflictDiagnostics(t *testing.T) {
	profile := filepath.Join("..", "..", "examples", "person-profile")
	library := filepath.Join("..", "..", "examples", "shared-personality-library")
	if _, err := LoadDirectoryWithLibraries(profile, filepath.Join(t.TempDir(), "missing")); err == nil ||
		!strings.Contains(err.Error(), "inspect personality library") {
		t.Fatalf("missing library diagnostic = %v", err)
	}
	if _, err := LoadDirectoryWithLibraries(profile, library, library); err == nil ||
		!strings.Contains(err.Error(), "admitted more than once") {
		t.Fatalf("duplicate library diagnostic = %v", err)
	}

	invalid := filepath.Join(t.TempDir(), "invalid")
	copyTestTree(t, library, invalid)
	if err := os.WriteFile(filepath.Join(invalid, "library.yaml"), []byte("library: Bad ID\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDirectoryWithLibraries(profile, invalid); err == nil ||
		!strings.Contains(err.Error(), "stable logical id") {
		t.Fatalf("invalid library id diagnostic = %v", err)
	}

	divergent := filepath.Join(t.TempDir(), "divergent")
	copyTestTree(t, library, divergent)
	if err := os.WriteFile(filepath.Join(divergent, "library.yaml"), []byte("library: shared-example-copy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(divergent, "definitions", "skills", "personality-shared-care", "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("\nDivergent copy.\n")...)
	if err := os.WriteFile(skillPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDirectoryWithLibraries(profile, library, divergent); err == nil ||
		!strings.Contains(err.Error(), "definition") ||
		!strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("divergent definition diagnostic = %v", err)
	}
}

func copyTestTree(t *testing.T, source, target string) {
	t.Helper()
	if err := fs.WalkDir(os.DirFS(source), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		destination := filepath.Join(target, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		raw, err := fs.ReadFile(os.DirFS(source), path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedPersonSourceOwnsEveryPersonalityDefinition(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := Source(p)
	if err != nil {
		t.Fatal(err)
	}
	if src.ID != "roster:core" {
		t.Fatalf("source id = %q, want roster:core", src.ID)
	}
	if len(src.Instructions) != 1 || src.Instructions[0].ID != "personality-invariant" {
		t.Fatalf("unexpected embedded instructions: %+v", src.Instructions)
	}
	if len(src.Skills) != len(p.Personalities) {
		t.Fatalf("embedded skills = %d, catalog bindings = %d", len(src.Skills), len(p.Personalities))
	}
	for _, skill := range src.Skills {
		if _, err := fs.Stat(src.FileSystem(), skill.Path+"/SKILL.md"); err != nil {
			t.Fatalf("definition %q is not embedded: %v", skill.ID, err)
		}
	}
}

func completePersonFixture() string {
	return `person "fixture" {
    role "builder" {
        purpose "Build."
        briefing "Build independently.\n\nVerify the important paths.\n\nFinish the complete handoff."
        personality "bright" "steady"
    }
    personality "bright" skill="personality-bright" color="#d98e48" motif="sunbeam" geometry="open-rays" {
        emblem { name "lantern" "beacon"; emoji "🏮" }
        body {
            archetype "small upright body, thin limbs, a glass lantern housing for a chest"
            attachment "the flame sits inside its chest, casting rays out through the glass"
        }
        sound-mark { timbre "bell"; contour "rising"; pulse "triplet" }
    }
    personality "steady" skill="personality-steady" color="#5fa87a" motif="stone" geometry="stacked-rounds" {
        emblem { name "anchor" "cairn"; emoji "⚓" }
        body {
            archetype "low and rounded, a body of stacked stones settled into one another"
            attachment "one small cairn standing at its foot, the same stone as its body"
        }
        sound-mark { timbre "wood-block"; contour "returning"; pulse "steady-pair" }
    }
}`
}

func TestRosterProseFloorsRejectAThinnedEntry(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	source := p.source
	role := p.Roles["platform"]
	role.Briefing = "Too short."
	p.Roles["platform"] = role
	if err := validateRosterProseFloors(source, p); err == nil ||
		!strings.Contains(err.Error(), "minimum is") {
		t.Fatalf("thinned role body error = %v", err)
	}
}

func TestRosterProseCeilingsAndFloorsBoundEveryShippedEntry(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// The shipped roster must sit inside both bounds, which is what makes the
	// floors a live gate rather than a constant nobody reaches.
	for _, roleName := range p.RoleOrder {
		words := roleSkillBodyWordCount(p.Roles[roleName].Briefing)
		if words < minRoleSkillBodyWords || words > maxRoleSkillBodyWords {
			t.Errorf("role %q body has %d words, bounds are %d..%d",
				roleName, words, minRoleSkillBodyWords, maxRoleSkillBodyWords)
		}
	}
}
