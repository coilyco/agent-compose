// Package nativelaunch builds and projects one caller-assigned role bundle for
// a native harness session.
package nativelaunch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/agentid"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/compose"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/nativeui"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/project"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/repositoryplan"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

const (
	// EnvModelTier carries the launch consumer's runtime model-tier selection.
	EnvModelTier = "AGENT_COMPOSE_MODEL_TIER"
	// EnvRuntimeHome selects a session-scoped home for the harness process.
	EnvRuntimeHome = "AGENT_COMPOSE_RUNTIME_HOME"
	// EnvVerbose restores the routine composition status a launch otherwise
	// keeps off screen. See docs/native-role-launch.md.
	EnvVerbose = "AGENT_COMPOSE_VERBOSE"
)

// Options supplies the host-owned selection and filesystem anchors for one
// native role launch.
type Options struct {
	Role      string
	Harness   string
	ModelTier string
	CWD       string
	TargetDir string
	// RuntimeHome is the session-scoped home a launch consumer staged. When set,
	// projection uses the harness global load points; see docs/projection.md.
	RuntimeHome string
	// OperatingBase leads the composed instructions, so a session home that
	// replaces the host load point keeps the doctrine that file carried.
	OperatingBase string
	// OperatingAppendix travels apart from the base because the base is rewritten
	// before rendering; see cascade.ComposeParts. agent-compose#6987.
	OperatingAppendix string
	PlanPath          string
	OutDir            string
	PersonSelection   compose.Options
	SkipProjection    bool
}

// Result records the immutable bundle and projected load points selected for
// the native session.
type Result struct {
	Composition  *compose.Result
	BundleDir    string
	BundleReused bool
	Projected    int
	ModelTier    string
	Sources      []compose.RootSource
	// SeatName is the resolved display name for the selected seat. The launch
	// path hands it to a harness that can show a session name.
	SeatName string
	// HarnessSettings is the emitted settings fragment for harnesses that read
	// one as a launch argument. Empty when the harness has no such surface.
	HarnessSettings string
	// Introduction names who this session is before it asks for anything, so a
	// session list does not show identical openings. See docs/native-role-launch.md.
	Introduction string
}

// HarnessSettingsFile is the bundle-relative fragment the Claude launch path
// passes as --settings. See docs/claude-launch-identity.md.
const HarnessSettingsFile = "claude-settings.json"

type repository struct {
	relative  string
	selection repositoryplan.Selection
}

// name is the message spelling of a selected repository. filepath.Rel yields
// backslashes on Windows, and these strings name a logical owner/repository.
func (r repository) name() string { return filepath.ToSlash(r.relative) }

// Refresh resolves eligible providers, composes the complete role boundary, and
// transactionally projects the result at the selected harness load points.
func Refresh(opts Options) (*Result, error) {
	if err := validateHarness(opts.Harness); err != nil {
		return nil, err
	}
	modelTier, err := normalizeModelTier(opts.ModelTier)
	if err != nil {
		return nil, err
	}
	plan, err := repositoryplan.Load(opts.PlanPath)
	if err != nil {
		return nil, err
	}
	roots, missing, repositories, err := resolveRoots(plan, opts.Role, opts.CWD)
	if err != nil {
		return nil, err
	}
	request := &schema.Request{
		Role:         strings.TrimSpace(opts.Role),
		Delivery:     schema.DeliveryNativeSkills,
		ModelTier:    modelTier,
		Repositories: repositories,
	}
	selection := opts.PersonSelection
	selection.OperatingBase = opts.OperatingBase
	selection.OperatingAppendix = opts.OperatingAppendix
	composed, err := compose.RunRootsWithMissing(
		request,
		roots,
		missing,
		opts.OutDir,
		selection,
	)
	if err != nil {
		return nil, fmt.Errorf("compose native role %q: %w", opts.Role, err)
	}
	// A staged session home owns its whole load-point surface, so the role lands
	// at the harness global paths rather than beside the checkouts.
	target, scope := opts.TargetDir, project.ScopeRepo
	if target == "" {
		target = opts.CWD
	}
	if home := strings.TrimSpace(opts.RuntimeHome); home != "" {
		target, scope = home, project.ScopeHome
	}
	projectedCount := 0
	if !opts.SkipProjection {
		projected, err := project.ProjectScoped(composed.Bundle.Dir, opts.Harness, target, scope)
		if err != nil {
			return nil, fmt.Errorf(
				"project native role %q for %s into %s (%s scope): %w",
				opts.Role,
				opts.Harness,
				target,
				scope,
				err,
			)
		}
		projectedCount = len(projected.Files)
	}
	settings, err := emitHarnessSettings(composed, opts.Harness, opts.Role)
	if err != nil {
		return nil, err
	}
	return &Result{
		Composition:     composed,
		BundleDir:       composed.Bundle.Dir,
		BundleReused:    composed.Bundle.Reused,
		Projected:       projectedCount,
		ModelTier:       modelTier,
		Sources:         roots,
		SeatName:        seatName(composed, opts.Harness, opts.Role),
		Introduction:    introduction(composed, opts.Harness, opts.Role),
		HarnessSettings: settings,
	}, nil
}

// emitHarnessSettings writes the role's native UI fragment beside the bundle so
// the launch path can hand it over as an argument. Only Claude Code reads one.
func emitHarnessSettings(composed *compose.Result, harness, role string) (string, error) {
	if strings.TrimSpace(harness) != "claude" {
		return "", nil
	}
	p := composed.Resolution.Person
	if p == nil {
		return "", nil
	}
	built, err := nativeui.BuildRole(p, strings.TrimSpace(role), nativeui.Options{})
	if err != nil {
		return "", fmt.Errorf("emit native UI settings for role %q: %w", role, err)
	}
	raw, err := json.MarshalIndent(built.Settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode native UI settings for role %q: %w", role, err)
	}
	path := filepath.Join(composed.Bundle.Dir, HarnessSettingsFile)
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write native UI settings for role %q: %w", role, err)
	}
	return path, nil
}

// introduction renders the identity-led opener, falling back to the role
// alone when no seat name resolves. See docs/native-role-launch.md.
func introduction(composed *compose.Result, harness, role string) string {
	p := composed.Resolution.Person
	if p == nil {
		return ""
	}
	selected, ok := p.Roles[role]
	if !ok {
		return ""
	}
	displayName := p.RoleDisplayName(role)
	traits := joinTraits(selected.Personalities)
	subject := displayName
	if traits != "" {
		subject = traits + " " + displayName
	}
	name := ""
	for _, seat := range selected.Seats {
		if seat.Selector() == strings.TrimSpace(harness) && seat.Name != "" {
			name = seat.Name
			break
		}
	}
	if name == "" && selected.Identity != nil {
		name = selected.Identity.Name
	}
	if name == "" {
		return "You are the " + subject + "."
	}
	return name + ", you are the " + subject + "."
}

// joinTraits reads a meld as English. A two-trait meld takes a bare "and",
// where the serial comma every longer list wants would be a comma splice.
func joinTraits(traits []string) string {
	switch len(traits) {
	case 0:
		return ""
	case 1:
		return traits[0]
	case 2:
		return traits[0] + " and " + traits[1]
	default:
		last := len(traits) - 1
		return strings.Join(traits[:last], ", ") + ", and " + traits[last]
	}
}

// seatName resolves the launch annotation: the selected seat's own name wins,
// then the role-owned agent identity, then the role. See docs/overlay.md.
func seatName(composed *compose.Result, harness, role string) string {
	role = strings.TrimSpace(role)
	p := composed.Resolution.Person
	if p == nil {
		return role
	}
	selected, ok := p.Roles[role]
	if !ok {
		return role
	}
	displayName := p.RoleDisplayName(role)
	// Becomes the harness `--name` flag, which is ephemeral per launch rather
	// than a bundle artifact a later session reads. See docs/identity.md.
	shortID := agentid.FromEnv()
	for _, seat := range selected.Seats {
		if seat.Selector() == strings.TrimSpace(harness) && seat.Name != "" {
			return person.WithShortID(
				person.SeatAnnotation(seat.Name, seat.Pronouns, displayName),
				shortID,
			)
		}
	}
	if selected.Identity != nil && selected.Identity.Name != "" {
		return person.WithShortID(
			person.SeatAnnotation(
				selected.Identity.Name,
				selected.Identity.Pronouns,
				displayName,
			),
			shortID,
		)
	}
	return person.WithShortID(role, shortID)
}

func validateHarness(harness string) error {
	switch strings.TrimSpace(harness) {
	case "claude", "codex", "goose", "opencode":
		return nil
	default:
		return fmt.Errorf(
			"unsupported native harness %q: want claude, codex, goose, or opencode",
			harness,
		)
	}
}

func normalizeModelTier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return schema.ModelTierFrontier, nil
	}
	if schema.IsModelTier(value) {
		return value, nil
	}
	return "", fmt.Errorf("unsupported native model tier %q", value)
}

func resolveRoots(
	plan repositoryplan.Plan,
	role string,
	cwd string,
) ([]compose.RootSource, []schema.MissingSource, []schema.RepositorySelection, error) {
	selections, err := plan.ForRole(strings.TrimSpace(role))
	if err != nil {
		return nil, nil, nil, err
	}
	if len(selections) == 0 {
		return nil, nil, nil, fmt.Errorf("repository plan selects no repositories for role %q", role)
	}
	relative := make([]repository, 0, len(selections))
	repositories := make([]schema.RepositorySelection, 0, len(selections))
	for _, selection := range selections {
		rel, err := filepath.Rel(plan.ProjectsRoot, selection.Path)
		if err != nil || rel == "." || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, nil, nil, fmt.Errorf(
				"selected repository %s is outside projects_root %s",
				selection.Path,
				plan.ProjectsRoot,
			)
		}
		relative = append(relative, repository{relative: rel, selection: selection})
		repositories = append(repositories, schema.RepositorySelection{
			Identity: selection.Identity, Source: selection.Source,
			Scope: selection.Scope, Reason: selection.Reason,
		})
	}
	projectsRoot := resolveProjectsRoot(cwd, plan.ProjectsRoot, relative)
	roots := make([]compose.RootSource, 0, len(relative))
	var missing []schema.MissingSource
	hasRoleProvider := false
	sourceIDs := map[string]bool{}
	selectedProviderIDs := map[string]bool{}
	for _, repo := range relative {
		root := filepath.Join(projectsRoot, repo.relative)
		id := sourceID(repo.relative)
		selectedProviderIDs[id] = true
		if info, err := os.Stat(filepath.Join(root, ".agents", "skills")); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if repo.selection.Scope == "provider" {
					reason := fmt.Sprintf(
						"optional role provider %s for role %q is unavailable",
						repo.name(),
						role,
					)
					if repo.selection.Required {
						return nil, nil, nil, fmt.Errorf(
							"required role provider %s for role %q is unavailable beneath %s",
							repo.name(),
							role,
							projectsRoot,
						)
					}
					missing = append(missing, schema.MissingSource{
						ID: id, Reason: reason, ProviderScope: schema.ProviderScopeRole,
						Warning: true,
					})
				}
				continue
			}
			return nil, nil, nil, fmt.Errorf("inspect selected repository %s: %w", root, err)
		} else if !info.IsDir() {
			if repo.selection.Scope == "provider" {
				reason := fmt.Sprintf(
					"optional role provider %s for role %q has no .agents/skills directory",
					repo.name(),
					role,
				)
				if repo.selection.Required {
					return nil, nil, nil, fmt.Errorf(
						"required role provider %s for role %q has no .agents/skills directory",
						repo.name(),
						role,
					)
				}
				missing = append(missing, schema.MissingSource{
					ID: id, Reason: reason, ProviderScope: schema.ProviderScopeRole,
					Warning: true,
				})
			}
			continue
		}
		if info, err := os.Stat(filepath.Join(root, ".agents", "roles.kdl")); err == nil &&
			info.Mode().IsRegular() {
			hasRoleProvider = true
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, fmt.Errorf("inspect role bindings in %s: %w", root, err)
		}
		if sourceIDs[id] {
			return nil, nil, nil, fmt.Errorf(
				"eligible providers produce duplicate source id %q",
				id,
			)
		}
		sourceIDs[id] = true
		roots = append(roots, compose.RootSource{
			ID:            id,
			Root:          root,
			Reason:        repo.selection.Reason,
			Scope:         providerScope(repo.selection.Scope),
			Skills:        append([]string(nil), repo.selection.Skills...),
			BindingSkills: append([]string(nil), repo.selection.BindingSkills...),
		})
	}
	if len(roots) == 0 {
		return nil, nil, nil, fmt.Errorf(
			"no eligible skill providers are available beneath %s",
			projectsRoot,
		)
	}
	if !hasRoleProvider {
		return nil, nil, nil, fmt.Errorf(
			"native role launch needs an eligible provider with .agents/roles.kdl beneath %s",
			projectsRoot,
		)
	}
	missing = append(missing, excludedRoleProviders(
		plan,
		strings.TrimSpace(role),
		projectsRoot,
		selectedProviderIDs,
	)...)
	return roots, missing, repositories, nil
}

func providerScope(scope string) string {
	if scope == "provider" {
		return schema.ProviderScopeRole
	}
	return schema.ProviderScopeDefault
}

func excludedRoleProviders(
	plan repositoryplan.Plan,
	selectedRole string,
	projectsRoot string,
	selectedIDs map[string]bool,
) []schema.MissingSource {
	rolesByPath := map[string][]string{}
	roles := make([]string, 0, len(plan.Roles))
	for role := range plan.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		if role == selectedRole {
			continue
		}
		for _, provider := range plan.Roles[role] {
			if provider.Scope != "provider" {
				continue
			}
			rel, err := filepath.Rel(plan.ProjectsRoot, provider.Path)
			if err != nil || rel == "." || rel == ".." ||
				strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
			id := sourceID(rel)
			if selectedIDs[id] {
				continue
			}
			rolesByPath[rel] = append(rolesByPath[rel], role)
		}
	}
	paths := make([]string, 0, len(rolesByPath))
	for path := range rolesByPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	missing := make([]schema.MissingSource, 0, len(paths))
	for _, rel := range paths {
		providerRoles := rolesByPath[rel]
		reason := fmt.Sprintf(
			"role provider is scoped to role(s) %s, not selected role %q",
			strings.Join(providerRoles, ", "),
			selectedRole,
		)
		entry := schema.MissingSource{
			ID: sourceID(rel), Reason: reason, ProviderScope: schema.ProviderScopeRole,
		}
		if source, err := schema.LoadSource(filepath.Join(projectsRoot, rel)); err == nil {
			seen := map[string]bool{}
			for _, skill := range source.Skills {
				if !seen[skill.ID] {
					entry.Skills = append(entry.Skills, skill.ID)
					seen[skill.ID] = true
				}
			}
			for _, roleSkills := range source.RoleSkills {
				for _, skill := range roleSkills {
					if !seen[skill.ID] {
						entry.Skills = append(entry.Skills, skill.ID)
						seen[skill.ID] = true
					}
				}
			}
			sort.Strings(entry.Skills)
		}
		missing = append(missing, entry)
	}
	return missing
}

func resolveProjectsRoot(cwd, configured string, repositories []repository) string {
	candidate, err := filepath.Abs(cwd)
	if err == nil {
		for {
			if providerCount(candidate, repositories) > 0 {
				return candidate
			}
			parent := filepath.Dir(candidate)
			if parent == candidate {
				break
			}
			candidate = parent
		}
	}
	return configured
}

func providerCount(root string, repositories []repository) int {
	count := 0
	for _, repo := range repositories {
		if info, err := os.Stat(filepath.Join(
			root,
			repo.relative,
			".agents",
			"skills",
		)); err == nil && info.IsDir() {
			count++
		}
	}
	return count
}

func sourceID(relative string) string {
	slash := filepath.ToSlash(relative)
	return strings.NewReplacer("/", "--", "\\", "--").Replace(slash)
}
