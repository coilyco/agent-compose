package cascade

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/repositoryplan"
	"github.com/coilyco/agent-compose/v2/internal/schema"
)

// RenderRepositoryPlan compiles repository policy from trusted operating roots.
// It never infers availability from doctrine paths or harness load points.
func RenderRepositoryPlan(cfg *Config, projects string) (string, error) {
	canonicalProjects, err := canonicalPath(projects)
	if err != nil {
		return "", err
	}
	if len(cfg.OperatingContext) == 0 {
		return "", fmt.Errorf("agent-compose.yaml must declare operating_context repositories")
	}

	graphs := make([]trustedRoleGraph, 0, len(cfg.OperatingContext))
	for _, identity := range cfg.OperatingContext {
		root, err := resolveProviderPath(canonicalProjects, identity)
		if err != nil {
			return "", fmt.Errorf("operating context %q: %w", identity, err)
		}
		rolesPath := filepath.Join(root, ".agents", "roles.kdl")
		info, err := os.Stat(rolesPath)
		if err != nil {
			return "", fmt.Errorf("operating context %q needs %s: %w", identity, rolesPath, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("operating context role graph %s is not a regular file", rolesPath)
		}
		revision, err := sourceRevision(root)
		if err != nil {
			return "", fmt.Errorf("operating context %q: %w", identity, err)
		}
		digest, err := fileSHA256(rolesPath)
		if err != nil {
			return "", fmt.Errorf("operating context %q: %w", identity, err)
		}
		source, err := schema.LoadSource(root)
		if err != nil {
			return "", fmt.Errorf("load operating context %q: %w", identity, err)
		}
		graphs = append(graphs, trustedRoleGraph{
			root: root, relative: identity, source: source,
			input: repositoryplan.Input{
				Identity: identity,
				Revision: revision,
				Policy: repositoryplan.PolicyInput{
					Path: repositoryplan.PolicyPath, SHA256: digest,
				},
			},
		})
	}

	providerDefinitions := map[string]map[string]string{}
	claimedPaths := map[string]string{}
	repositoryDefinitions := map[string]map[string]string{}
	for _, graph := range graphs {
		providerDefinitions[graph.root] = map[string]string{}
		repositoryDefinitions[graph.root] = map[string]string{}
		for _, id := range sortedMapKeys(graph.source.Providers) {
			definition := graph.source.Providers[id]
			resolved, err := resolveProviderPath(canonicalProjects, definition.Path)
			if err != nil {
				return "", fmt.Errorf("provider %q declared by %s: %w", id, graph.relative, err)
			}
			if previous, exists := claimedPaths[definition.Path]; exists {
				return "", fmt.Errorf("repository path %q is declared by both %s and provider %q in %s", definition.Path, previous, id, graph.relative)
			}
			claimedPaths[definition.Path] = fmt.Sprintf("provider %q in %s", id, graph.relative)
			providerDefinitions[graph.root][id] = resolved
			if info, statErr := os.Stat(resolved); statErr == nil {
				if !info.IsDir() {
					return "", fmt.Errorf("provider %q path %s is not a directory", id, resolved)
				}
				providerSource, loadErr := schema.LoadSource(resolved)
				if loadErr != nil {
					return "", fmt.Errorf("load provider %q declared by %s: %w", id, graph.relative, loadErr)
				}
				if selectErr := schema.SelectOrdinarySkills(providerSource, definition.Skills, nil); selectErr != nil {
					return "", fmt.Errorf("select provider %q declared by %s: %w", id, graph.relative, selectErr)
				}
			} else if !os.IsNotExist(statErr) {
				return "", fmt.Errorf("inspect provider %q path %s: %w", id, resolved, statErr)
			}
		}
		for _, id := range sortedMapKeys(graph.source.Repositories) {
			definition := graph.source.Repositories[id]
			resolved, err := resolveProviderPath(canonicalProjects, definition.Path)
			if err != nil {
				return "", fmt.Errorf("repository %q declared by %s: %w", id, graph.relative, err)
			}
			if previous, exists := claimedPaths[definition.Path]; exists {
				unifiedProvider := definition.Skills != nil &&
					previous == fmt.Sprintf("provider %q in %s", id, graph.relative)
				if !unifiedProvider {
					return "", fmt.Errorf("repository path %q is declared by both %s and repository %q in %s", definition.Path, previous, id, graph.relative)
				}
			} else {
				claimedPaths[definition.Path] = fmt.Sprintf("repository %q in %s", id, graph.relative)
			}
			repositoryDefinitions[graph.root][id] = resolved
		}
	}
	if err := rejectProviderCycles(graphs, providerDefinitions); err != nil {
		return "", err
	}

	roles := map[string]bool{}
	for _, graph := range graphs {
		for role := range graph.source.RoleSkills {
			roles[role] = true
		}
		for role := range graph.source.RoleProviders {
			roles[role] = true
		}
		for role := range graph.source.RoleRepos {
			roles[role] = true
		}
	}
	if len(roles) == 0 {
		return "", fmt.Errorf("operating context role graphs define no roles")
	}

	plan := repositoryplan.Plan{
		Format: repositoryplan.Format, ProjectsRoot: canonicalProjects,
		Inputs: sortedInputs(graphs),
		Roles:  map[string][]repositoryplan.Selection{},
	}
	for _, role := range sortedBoolKeys(roles) {
		selected := map[string]repositoryplan.Selection{}
		add := func(selection repositoryplan.Selection) error {
			if previous, exists := selected[selection.Identity]; exists {
				return fmt.Errorf("role %q selects repository %q from both %s and %s", role, selection.Identity, previous.Source, selection.Source)
			}
			selected[selection.Identity] = selection
			return nil
		}
		for _, identity := range cfg.OperatingContext {
			if err := add(repositoryplan.Selection{
				Identity: identity,
				Path:     filepath.Join(canonicalProjects, filepath.FromSlash(identity)),
				Source:   identity, Scope: "operating-context",
				Reason: "host configuration selects this repository as role operating context",
			}); err != nil {
				return "", err
			}
		}
		for _, graph := range graphs {
			for _, use := range graph.source.GlobalRepos {
				definition := graph.source.Repositories[use.Repository]
				if err := add(repositorySelection(definition.Path, repositoryDefinitions[graph.root][use.Repository], graph.relative, "global", "repository policy makes this repository available to every role")); err != nil {
					return "", err
				}
			}
			for _, use := range graph.source.RoleRepos[role] {
				definition := graph.source.Repositories[use.Repository]
				selection := repositorySelection(definition.Path, repositoryDefinitions[graph.root][use.Repository], graph.relative, "role", fmt.Sprintf("repository policy makes this repository available to role %q", role))
				selection.BindingSkills = append([]string(nil), use.Skills...)
				if err := add(selection); err != nil {
					return "", err
				}
			}
			for _, use := range graph.source.RoleProviders[role] {
				definition := graph.source.Providers[use.Provider]
				if err := add(repositoryplan.Selection{
					Identity: definition.Path,
					Path:     providerDefinitions[graph.root][use.Provider],
					Source:   graph.relative, Scope: "provider", Required: use.Required,
					Skills: append([]string(nil), definition.Skills...), Name: use.Provider,
					BindingSkills: append([]string(nil), use.Skills...),
					DeclaredBy:    graph.relative,
					Reason:        fmt.Sprintf("role %q uses skill-provider repository %q declared by %s", role, use.Provider, graph.relative),
				}); err != nil {
					return "", err
				}
			}
		}
		plan.Roles[role] = sortedSelections(selected)
	}

	residency := map[string]repositoryplan.Selection{}
	for _, role := range sortedMapKeys(plan.Roles) {
		for _, selection := range plan.Roles[role] {
			if _, exists := residency[selection.Identity]; !exists {
				copy := selection
				copy.Scope = "role-union"
				copy.Reason = "repository is selected by at least one canonical role"
				// Residency is the union across roles, so one role's narrowing
				// must not read as a property of the repository itself.
				copy.BindingSkills = nil
				residency[selection.Identity] = copy
			}
		}
	}
	for _, graph := range graphs {
		for _, use := range graph.source.ResidentRepos {
			definition := graph.source.Repositories[use.Repository]
			if _, exists := residency[definition.Path]; exists {
				return "", fmt.Errorf("resident-only repository %q is already selected by a role", definition.Path)
			}
			residency[definition.Path] = repositorySelection(
				definition.Path,
				repositoryDefinitions[graph.root][use.Repository],
				graph.relative,
				"resident-only",
				"repository policy pins this checkout to host residency without granting it to a role",
			)
		}
	}
	plan.Residency = sortedSelections(residency)
	raw, err := repositoryplan.Marshal(plan)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func repositorySelection(identity, resolved, source, scope, reason string) repositoryplan.Selection {
	return repositoryplan.Selection{
		Identity: identity, Path: resolved, Source: source, Scope: scope, Reason: reason,
	}
}

func sortedSelections(values map[string]repositoryplan.Selection) []repositoryplan.Selection {
	identities := make([]string, 0, len(values))
	for identity := range values {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	result := make([]repositoryplan.Selection, 0, len(identities))
	for _, identity := range identities {
		result = append(result, values[identity])
	}
	return result
}

func sortedInputs(graphs []trustedRoleGraph) []repositoryplan.Input {
	inputs := make([]repositoryplan.Input, 0, len(graphs))
	for _, graph := range graphs {
		inputs = append(inputs, graph.input)
	}
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Identity < inputs[j].Identity
	})
	return inputs
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func planPath(stateDir string) string {
	return filepath.Join(strings.TrimSpace(stateDir), "repository-plan.yaml")
}

func fileSHA256(filename string) (string, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("read policy %s: %w", filename, err)
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum), nil
}

func sourceRevision(root string) (string, error) {
	output, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("read git revision for %s: %w", root, err)
	}
	return strings.TrimSpace(string(output)), nil
}
