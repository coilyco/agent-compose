package schema

import (
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	kdl "github.com/calico32/kdl-go"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/skillselector"
)

const (
	DeliveryNativeSkills = "native-skills"
	DeliveryCompiled     = "compiled"
	ModelTierFrontier    = "frontier"
	ModelTierCommodity   = "commodity"
	ModelTierOSS         = "oss"
	legacyDensityFull    = "full"

	providerSkillsPath    = ".agents/skills"
	providerComposedPath  = ".agents/composed"
	providerRolesPath     = ".agents/roles.kdl"
	providerInvariantID   = "personality-invariant"
	providerInvariantPath = ".agents/skills/personality-shared/INVARIANT.md"

	ProviderScopePerson  = "person"
	ProviderScopeRequest = "request"
	ProviderScopeDefault = "default"
	ProviderScopeHarness = "harness"
	ProviderScopeRole    = "role"

	// CoreLibraryRoot names the embedded personality library wherever a
	// personality-library root is accepted. See docs/personality.md.
	CoreLibraryRoot = "roster:core"
)

type Request struct {
	PersonPolicy         string
	PersonSource         string
	PersonalityLibraries []string
	Role                 string
	Delivery             string
	ModelTier            string
	Identity             *IdentityOverride
	Sources              []SourceLocator
	Repositories         []RepositorySelection
	// BoundaryOmissions names defer-side boundaries this deployment does not
	// compose. See docs/ownership.md.
	BoundaryOmissions []string
}

// IdentityOverride renames the seat a request composes. It says who is
// speaking, never what the role is. See docs/person-contract.md.
type IdentityOverride struct {
	Name     string
	Pronouns string
}

// IsModelTier reports whether value belongs to the complete stable model-tier
// vocabulary. Model identity and provider routing remain caller-owned facts.
func IsModelTier(value string) bool {
	switch value {
	case ModelTierFrontier, ModelTierCommodity, ModelTierOSS:
		return true
	default:
		return false
	}
}

// ModelTiers returns the canonical presentation order for model tiers.
func ModelTiers() []string {
	return []string{ModelTierFrontier, ModelTierCommodity, ModelTierOSS}
}

type SourceLocator struct {
	ID          string
	Declaration string
	Root        string
	Required    bool
}

type ContentRef struct {
	ID         string
	Path       string
	EntryPoint string
	Selectors  []string
}

// SelectorOverlap records one composed skill admitted by multiple selectors
// while retaining their complete diagnostic and trace provenance.
type SelectorOverlap struct {
	Role      string
	Skill     string
	Selectors []string
}

// ProviderDefinition names one document-local ordinary-skill provider. Path
// remains logical until cascade resolves it beneath projects_root.
type ProviderDefinition struct {
	ID     string
	Path   string
	Skills []string
}

// ProviderUse binds one document-local skill-provider repository to a role.
// Skills narrows within the definition's selector for this role alone.
type ProviderUse struct {
	Provider string
	Required bool
	Skills   []string
}

// RepositoryDefinition names one logical repository policy target. Optional
// Skills turns the repository into a bounded ordinary-skill provider.
type RepositoryDefinition struct {
	ID     string
	Path   string
	Skills []string
}

// RepositoryUse is one reference to a document-local repository definition.
// Skills narrows within the definition's selector for this role alone.
type RepositoryUse struct {
	Repository string
	Skills     []string
}

// OrgDefinition is one whole-forge-owner provider. It names an owner rather
// than a path: the catalogues are discovered, not listed. See docs/skill-catalogues.md.
type OrgDefinition struct {
	ID     string
	Owner  string
	Skills []string
}

// OrgUse is one role's reference to a declared org.
type OrgUse struct {
	Org string
}

// RepositorySelection carries one immutable repository identity and its
// compiler decision provenance into a bundle.
type RepositorySelection struct {
	Identity string `json:"identity"`
	Source   string `json:"source"`
	Scope    string `json:"scope"`
	Reason   string `json:"reason"`
}

type Source struct {
	ID               string
	Root             string
	Files            fs.FS
	Declaration      []byte
	Instructions     []ContentRef
	Skills           []ContentRef
	RoleSkills       map[string][]ContentRef
	SelectorOverlaps []SelectorOverlap
	Providers        map[string]ProviderDefinition
	RoleProviders    map[string][]ProviderUse
	Repositories     map[string]RepositoryDefinition
	Orgs             map[string]OrgDefinition
	RoleOrgs         map[string][]OrgUse
	GlobalRepos      []RepositoryUse
	RoleRepos        map[string][]RepositoryUse
	ResidentRepos    []RepositoryUse
	AdmissionReason  string
	ProviderScope    string
	ExcludedSkills   []ContentRef
	SelectorReason   string
}

// SelectOrdinarySkills applies a definition's selector, then a role binding's
// over the result so it narrows only. See docs/kdl-contracts.md.
func SelectOrdinarySkills(source *Source, definition, binding []string) error {
	if err := applySkillSelector(source, definition); err != nil {
		return err
	}
	return applySkillSelector(source, binding)
}

func applySkillSelector(source *Source, patterns []string) error {
	if patterns == nil {
		return nil
	}
	ids := make([]string, 0, len(source.Skills))
	refs := make(map[string]ContentRef, len(source.Skills))
	for _, ref := range source.Skills {
		ids = append(ids, ref.ID)
		refs[ref.ID] = ref
	}
	selected, excluded, err := skillselector.Select(patterns, ids)
	if err != nil {
		return err
	}
	source.Skills = source.Skills[:0]
	for _, id := range selected {
		source.Skills = append(source.Skills, refs[id])
	}
	for _, id := range excluded {
		source.ExcludedSkills = append(source.ExcludedSkills, refs[id])
	}
	reason := fmt.Sprintf(
		"ordinary skill selector %s admitted %d of %d catalogue skills",
		strings.Join(patterns, ", "),
		len(selected),
		len(ids),
	)
	// A binding selector runs as a second pass, so both narrowings stay legible
	// in the trace rather than the last one overwriting the first.
	if source.SelectorReason != "" {
		reason = source.SelectorReason + " // " + reason
	}
	source.SelectorReason = reason
	return nil
}

// FileSystem returns embedded content for shipped sources and disk content for
// declared or inferred providers.
func (s *Source) FileSystem() fs.FS {
	if s.Files != nil {
		return s.Files
	}
	return os.DirFS(s.Root)
}

func (s *Source) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(s.FileSystem(), filepath.ToSlash(name))
}

// MissingSource records an optional source whose declaration was absent, so
// the resolver can note the exclusion in the trace.
type MissingSource struct {
	ID            string
	Reason        string
	Skills        []string
	ProviderScope string
	Warning       bool
}

func ParseRequest(path string) (*Request, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read request %s: %w", path, err)
	}
	doc, err := kdl.ParseString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse request %s: %w", path, err)
	}
	if len(doc.Nodes) != 1 || doc.Nodes[0].Name() != "compose" {
		return nil, fmt.Errorf("request %s: expected exactly one top-level compose node", path)
	}

	req := &Request{}
	seen := map[string]bool{}
	for _, n := range doc.Nodes[0].Children().Nodes {
		switch n.Name() {
		case "personality-library":
			v, err := oneStringArg(n)
			if err != nil {
				return nil, fmt.Errorf("request %s: %w", path, err)
			}
			// The reserved embedded root is a logical name, not a path to resolve.
			if v != CoreLibraryRoot &&
				(filepath.IsAbs(v) || strings.HasPrefix(v, "/") || strings.HasPrefix(v, `\`)) {
				return nil, fmt.Errorf("request %s: personality-library path %q must be relative and clean", path, v)
			}
			req.PersonalityLibraries = append(req.PersonalityLibraries, v)
		case "person-policy", "person-source", "role", "delivery", "model-tier":
			if seen[n.Name()] {
				return nil, fmt.Errorf("request %s: duplicate %s node", path, n.Name())
			}
			seen[n.Name()] = true
			v, err := oneStringArg(n)
			if err != nil {
				return nil, fmt.Errorf("request %s: %w", path, err)
			}
			switch n.Name() {
			case "person-policy":
				req.PersonPolicy = v
			case "person-source":
				if strings.Contains(v, "..") || filepath.IsAbs(v) ||
					strings.HasPrefix(v, "/") || strings.HasPrefix(v, `\`) {
					return nil, fmt.Errorf(
						"request %s: person-source path %q must be relative and clean",
						path,
						v,
					)
				}
				req.PersonSource = v
			case "role":
				req.Role = v
			case "delivery":
				req.Delivery = v
			case "model-tier":
				req.ModelTier = v
			}
		case "density":
			if seen[n.Name()] {
				return nil, fmt.Errorf("request %s: duplicate density node", path)
			}
			seen[n.Name()] = true
			v, err := oneStringArg(n)
			if err != nil {
				return nil, fmt.Errorf("request %s: %w", path, err)
			}
			if v != legacyDensityFull {
				return nil, fmt.Errorf(
					"request %s: density is removed; delete the node (only legacy %q is accepted)",
					path,
					legacyDensityFull,
				)
			}
		case "identity":
			// Deliberately the same shape as a role's own identity node in a
			// person package, so one grammar covers both. See person.go.
			if seen[n.Name()] {
				return nil, fmt.Errorf("request %s: duplicate identity node", path)
			}
			seen[n.Name()] = true
			if len(n.Arguments()) != 0 {
				return nil, fmt.Errorf("request %s: identity takes name and pronouns properties", path)
			}
			identityName := n.Prop("name")
			pronouns := n.Prop("pronouns")
			if !identityName.IsValid() || strings.TrimSpace(identityName.String()) == "" {
				return nil, fmt.Errorf("request %s: identity needs a name property", path)
			}
			if !pronouns.IsValid() || strings.TrimSpace(pronouns.String()) == "" {
				return nil, fmt.Errorf("request %s: identity needs a pronouns property", path)
			}
			req.Identity = &IdentityOverride{
				Name:     strings.TrimSpace(identityName.String()),
				Pronouns: strings.TrimSpace(pronouns.String()),
			}
		case "boundary-omit":
			if len(n.Arguments()) == 0 {
				return nil, fmt.Errorf("request %s: boundary-omit needs at least one boundary name", path)
			}
			for _, arg := range n.Arguments() {
				name := strings.TrimSpace(arg.String())
				if name == "" {
					return nil, fmt.Errorf("request %s: boundary-omit name cannot be empty", path)
				}
				for _, other := range req.BoundaryOmissions {
					if other == name {
						return nil, fmt.Errorf("request %s: duplicate boundary-omit %q", path, name)
					}
				}
				req.BoundaryOmissions = append(req.BoundaryOmissions, name)
			}
		case "source":
			id, err := oneStringArg(n)
			if err != nil {
				return nil, fmt.Errorf("request %s: %w", path, err)
			}
			decl := n.Prop("declaration")
			root := n.Prop("root")
			if decl.IsValid() == root.IsValid() {
				return nil, fmt.Errorf("request %s: source %q needs exactly one of declaration or root", path, id)
			}
			loc := SourceLocator{ID: id}
			if decl.IsValid() {
				loc.Declaration = decl.String()
				if loc.Declaration == "" {
					return nil, fmt.Errorf("request %s: source %q declaration cannot be empty", path, id)
				}
			} else {
				loc.Root = root.String()
				if loc.Root == "" {
					return nil, fmt.Errorf("request %s: source %q root cannot be empty", path, id)
				}
			}
			if p := n.Prop("required"); p.IsValid() {
				loc.Required = p.Bool()
			}
			for _, other := range req.Sources {
				if other.ID == loc.ID {
					return nil, fmt.Errorf("request %s: duplicate source id %q", path, loc.ID)
				}
			}
			req.Sources = append(req.Sources, loc)
		default:
			return nil, fmt.Errorf("request %s: unknown node %q", path, n.Name())
		}
	}

	for _, field := range []struct{ name, value string }{
		{"role", req.Role}, {"delivery", req.Delivery},
	} {
		if field.value == "" {
			return nil, fmt.Errorf("request %s: missing %s", path, field.name)
		}
	}
	if req.Delivery != DeliveryNativeSkills && req.Delivery != DeliveryCompiled {
		return nil, fmt.Errorf("request %s: delivery must be %q or %q, got %q",
			path, DeliveryNativeSkills, DeliveryCompiled, req.Delivery)
	}
	if req.ModelTier == "" {
		req.ModelTier = ModelTierFrontier
	}
	if !IsModelTier(req.ModelTier) {
		return nil, fmt.Errorf(
			"request %s: model-tier must be %q, %q, or %q, got %q",
			path,
			ModelTierFrontier,
			ModelTierCommodity,
			ModelTierOSS,
			req.ModelTier,
		)
	}
	if err := personpolicy.Validate(req.PersonPolicy, req.PersonSource); err != nil {
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	return req, nil
}

// LoadSources reads each declared source relative to the request file. A
// missing required source fails; a missing optional one returns for the trace.
func LoadSources(req *Request, requestPath string) ([]*Source, []MissingSource, error) {
	baseDir := filepath.Dir(requestPath)
	var sources []*Source
	var missing []MissingSource
	for _, loc := range req.Sources {
		sourcePath := loc.Declaration
		kind := "declaration"
		if loc.Root != "" {
			sourcePath = loc.Root
			kind = "root"
		}
		if strings.Contains(sourcePath, "..") || filepath.IsAbs(sourcePath) {
			return nil, nil, fmt.Errorf("source %q: %s path %q must be relative and clean", loc.ID, kind, sourcePath)
		}
		resolvedPath := filepath.Join(baseDir, sourcePath)
		var src *Source
		var err error
		if loc.Root != "" {
			src, err = inferProvider(loc.ID, resolvedPath)
		} else {
			src, err = parseSource(resolvedPath)
		}
		if err != nil {
			_, rootErr := os.Stat(resolvedPath)
			if os.IsNotExist(rootErr) && !loc.Required {
				missing = append(missing, MissingSource{
					ID:      loc.ID,
					Reason:  fmt.Sprintf("optional source %s %s is absent", kind, sourcePath),
					Warning: true,
				})
				continue
			}
			return nil, nil, fmt.Errorf("source %q: %w", loc.ID, err)
		}
		if loc.Declaration != "" && src.ID != loc.ID {
			return nil, nil, fmt.Errorf("source %q: declaration %s declares id %q", loc.ID, loc.Declaration, src.ID)
		}
		sources = append(sources, src)
	}
	return sources, missing, nil
}

// LoadSource reads one explicit declaration or infers an AOS knowledge
// provider root for direct consumers such as roster.
func LoadSource(sourcePath string) (*Source, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		root, err := filepath.Abs(sourcePath)
		if err != nil {
			return nil, err
		}
		return inferProvider(filepath.Base(root), root)
	}
	return parseSource(sourcePath)
}

// inferProvider applies the AOS knowledge-provider filesystem convention.
// ReadDir returns names in lexical order, keeping source decisions stable.
func inferProvider(id, root string) (*Source, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("provider root %s is not a directory", root)
	}

	skillsRoot := filepath.Join(root, filepath.FromSlash(providerSkillsPath))
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("provider skills %s: %w", providerSkillsPath, err)
	}
	src := &Source{
		ID:   id,
		Root: root,
	}
	invariant := filepath.Join(root, filepath.FromSlash(providerInvariantPath))
	if info, err := os.Stat(invariant); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("provider invariant %s is not a regular file", providerInvariantPath)
		}
		src.Instructions = append(src.Instructions, ContentRef{
			ID:   providerInvariantID,
			Path: providerInvariantPath,
		})
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("provider invariant %s: %w", providerInvariantPath, err)
	}
	ordinary := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || name == "personality-shared" {
			continue
		}
		skillPath := filepath.Join(skillsRoot, name, "SKILL.md")
		if info, err := os.Stat(skillPath); err != nil {
			return nil, fmt.Errorf("provider skill %s: %w", name, err)
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("provider skill %s SKILL.md is not a regular file", name)
		}
		src.Skills = append(src.Skills, ContentRef{
			ID:         name,
			Path:       filepath.ToSlash(filepath.Join(providerSkillsPath, name)),
			EntryPoint: "SKILL.md",
		})
		ordinary[name] = true
	}
	if len(src.Skills) == 0 {
		return nil, fmt.Errorf("provider root %s has no skills under %s", root, providerSkillsPath)
	}

	composed := map[string]string{}
	composedRoot := filepath.Join(root, filepath.FromSlash(providerComposedPath))
	if _, err := os.Stat(composedRoot); err == nil {
		composed, err = inspectComposedSkills(composedRoot, ordinary)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("provider composed skills %s: %w", providerComposedPath, err)
	}

	rolesPath := filepath.Join(root, filepath.FromSlash(providerRolesPath))
	if _, err := os.Stat(rolesPath); err == nil {
		graph, err := parseRoleGraph(rolesPath, composed)
		if err != nil {
			return nil, err
		}
		src.RoleSkills = graph.RoleSkills
		src.SelectorOverlaps = graph.Overlaps
		src.Providers = graph.Providers
		src.RoleProviders = graph.RoleProviders
		src.Repositories = graph.Repositories
		src.GlobalRepos = graph.GlobalRepos
		src.RoleRepos = graph.RoleRepos
		src.ResidentRepos = graph.ResidentRepos
		src.Orgs = graph.Orgs
		src.RoleOrgs = graph.RoleOrgs
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
	} else if len(composed) > 0 {
		return nil, fmt.Errorf(
			"provider composed skills %s exist without role bindings %s",
			providerComposedPath,
			providerRolesPath,
		)
	}
	return src, nil
}

func inspectComposedSkills(root string, ordinary map[string]bool) (map[string]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("provider composed skills %s: %w", providerComposedPath, err)
	}
	composed := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("provider composed entry %s must be a directory", name)
		}
		if ordinary[name] {
			return nil, fmt.Errorf("provider skill %q exists in both %s and %s",
				name, providerSkillsPath, providerComposedPath)
		}
		skillRoot := filepath.Join(root, name)
		if err := filepath.WalkDir(skillRoot, func(path string, item os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if item.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s: symlinks are invalid inside a composed skill", path)
			}
			if !item.IsDir() && item.Name() == "SKILL.md" {
				return fmt.Errorf("provider composed skill %q contains SKILL.md; source entry points must be COMPOSED.md", name)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		entryPoint := filepath.Join(skillRoot, "COMPOSED.md")
		if info, err := os.Stat(entryPoint); err != nil {
			return nil, fmt.Errorf("provider composed skill %q: %w", name, err)
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("provider composed skill %q COMPOSED.md is not a regular file", name)
		}
		composed[name] = filepath.ToSlash(filepath.Join(providerComposedPath, name))
	}
	return composed, nil
}

// roleGraph is everything one role-bindings document declares. It is a struct
// rather than a return list because the list had reached eight.
type roleGraph struct {
	RoleSkills    map[string][]ContentRef
	Overlaps      []SelectorOverlap
	Providers     map[string]ProviderDefinition
	RoleProviders map[string][]ProviderUse
	Repositories  map[string]RepositoryDefinition
	GlobalRepos   []RepositoryUse
	RoleRepos     map[string][]RepositoryUse
	ResidentRepos []RepositoryUse
	Orgs          map[string]OrgDefinition
	RoleOrgs      map[string][]OrgUse
}

func parseRoleGraph(path string, composed map[string]string) (roleGraph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
	}
	doc, err := kdl.ParseString(string(raw))
	if err != nil {
		return roleGraph{}, fmt.Errorf("parse provider role bindings %s: %w", providerRolesPath, err)
	}
	if len(doc.Nodes) == 0 {
		return roleGraph{}, fmt.Errorf("provider role bindings %s: expected a top-level roles node", providerRolesPath)
	}

	var rolesNode *kdl.Node
	seenProvidersNode := false
	seenRepositoriesNode := false
	providers := map[string]ProviderDefinition{}
	repositories := map[string]RepositoryDefinition{}
	orgs := map[string]OrgDefinition{}
	roleOrgs := map[string][]OrgUse{}
	var globalRepos []RepositoryUse
	var residentRepos []RepositoryUse
	paths := map[string]string{}
	for _, node := range doc.Nodes {
		switch node.Name() {
		case "repositories":
			if seenRepositoriesNode {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate repositories node", providerRolesPath)
			}
			seenRepositoriesNode = true
			if len(node.Arguments()) > 0 || len(node.Properties()) > 0 {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: repositories node accepts only children", providerRolesPath)
			}
			seenGlobal := map[string]bool{}
			seenResident := map[string]bool{}
			for _, repositoryNode := range node.Children().Nodes {
				switch repositoryNode.Name() {
				case "repository":
					id, err := oneStringArg(repositoryNode)
					if err != nil {
						return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
					}
					if _, exists := repositories[id]; exists {
						return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate repository %q", providerRolesPath, id)
					}
					for property := range repositoryNode.Properties() {
						if property != "path" {
							return roleGraph{}, fmt.Errorf("repository %q: unknown property %q", id, property)
						}
					}
					pathValue := repositoryNode.Prop("path")
					if !pathValue.IsValid() || pathValue.Kind() != kdl.String {
						return roleGraph{}, fmt.Errorf("repository %q: path must be a string property", id)
					}
					logicalPath := pathValue.String()
					if err := validateLogicalProviderPath(logicalPath); err != nil {
						return roleGraph{}, fmt.Errorf("repository %q: %w", id, err)
					}
					if previous, exists := paths[logicalPath]; exists {
						return roleGraph{}, fmt.Errorf("repository declaration %q duplicates path %q already named by %s", id, logicalPath, previous)
					}
					skills, err := parseSkillSelectorChildren(fmt.Sprintf("repository %q", id), repositoryNode.Children().Nodes)
					if err != nil {
						return roleGraph{}, err
					}
					definition := RepositoryDefinition{ID: id, Path: logicalPath, Skills: skills}
					repositories[id] = definition
					if definition.Skills != nil {
						if _, exists := providers[id]; exists {
							return roleGraph{}, fmt.Errorf("repository %q: skill-provider id duplicates provider %q", id, id)
						}
						providers[id] = ProviderDefinition{
							ID: id, Path: logicalPath, Skills: append([]string(nil), definition.Skills...),
						}
					}
					paths[logicalPath] = fmt.Sprintf("repository %q", id)
				case "org":
					id, err := oneStringArg(repositoryNode)
					if err != nil {
						return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
					}
					if _, exists := orgs[id]; exists {
						return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate org %q", providerRolesPath, id)
					}
					if _, exists := repositories[id]; exists {
						return roleGraph{}, fmt.Errorf("org %q: id duplicates repository %q", id, id)
					}
					for property := range repositoryNode.Properties() {
						if property != "owner" {
							return roleGraph{}, fmt.Errorf("org %q: unknown property %q", id, property)
						}
					}
					ownerValue := repositoryNode.Prop("owner")
					if !ownerValue.IsValid() || ownerValue.Kind() != kdl.String {
						return roleGraph{}, fmt.Errorf("org %q: owner must be a string property", id)
					}
					owner := ownerValue.String()
					if err := validateOrgOwner(owner); err != nil {
						return roleGraph{}, fmt.Errorf("org %q: %w", id, err)
					}
					for existing, definition := range orgs {
						if definition.Owner == owner {
							return roleGraph{}, fmt.Errorf("org %q duplicates owner %q already named by org %q", id, owner, existing)
						}
					}
					skills, err := parseSkillSelectorChildren(fmt.Sprintf("org %q", id), repositoryNode.Children().Nodes)
					if err != nil {
						return roleGraph{}, err
					}
					// The grant is open by ruling, so the selector is the whole
					// review surface and an absent one bounds nothing.
					if skills == nil {
						return roleGraph{}, fmt.Errorf(
							"org %q: declare at least one skill selector, because an org grant admits repositories added later",
							id,
						)
					}
					orgs[id] = OrgDefinition{ID: id, Owner: owner, Skills: skills}
				case "global", "resident-only":
					id, err := oneStringArg(repositoryNode)
					if err != nil {
						return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
					}
					if len(repositoryNode.Properties()) > 0 || len(repositoryNode.Children().Nodes) > 0 {
						return roleGraph{}, fmt.Errorf("repository policy %q accepts only one repository argument", repositoryNode.Name())
					}
					seen := seenGlobal
					if repositoryNode.Name() == "resident-only" {
						seen = seenResident
					}
					if seen[id] {
						return roleGraph{}, fmt.Errorf("repository policy %q repeats repository %q", repositoryNode.Name(), id)
					}
					seen[id] = true
					use := RepositoryUse{Repository: id}
					if repositoryNode.Name() == "global" {
						globalRepos = append(globalRepos, use)
					} else {
						residentRepos = append(residentRepos, use)
					}
				default:
					return roleGraph{}, fmt.Errorf("provider role bindings %s: repositories has unknown node %q", providerRolesPath, repositoryNode.Name())
				}
			}
		case "providers":
			if seenProvidersNode {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate providers node", providerRolesPath)
			}
			seenProvidersNode = true
			if len(node.Arguments()) > 0 || len(node.Properties()) > 0 {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: providers node accepts only children", providerRolesPath)
			}
			for _, providerNode := range node.Children().Nodes {
				if providerNode.Name() != "provider" {
					return roleGraph{}, fmt.Errorf("provider role bindings %s: providers has unknown node %q", providerRolesPath, providerNode.Name())
				}
				id, err := oneStringArg(providerNode)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
				}
				if _, exists := providers[id]; exists {
					return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate provider %q", providerRolesPath, id)
				}
				for property := range providerNode.Properties() {
					if property != "path" {
						return roleGraph{}, fmt.Errorf("provider %q: unknown property %q", id, property)
					}
				}
				pathValue := providerNode.Prop("path")
				if !pathValue.IsValid() || pathValue.Kind() != kdl.String {
					return roleGraph{}, fmt.Errorf("provider %q: path must be a string property", id)
				}
				logicalPath := pathValue.String()
				if err := validateLogicalProviderPath(logicalPath); err != nil {
					return roleGraph{}, fmt.Errorf("provider %q: %w", id, err)
				}
				if previous, exists := paths[logicalPath]; exists {
					return roleGraph{}, fmt.Errorf("providers %q and %q name the same repository path %q", previous, id, logicalPath)
				}
				skills, err := parseSkillSelectorChildren(fmt.Sprintf("provider %q", id), providerNode.Children().Nodes)
				if err != nil {
					return roleGraph{}, err
				}
				definition := ProviderDefinition{ID: id, Path: logicalPath, Skills: skills}
				providers[id] = definition
				paths[logicalPath] = fmt.Sprintf("provider %q", id)
			}
		case "roles":
			if rolesNode != nil {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate roles node", providerRolesPath)
			}
			if len(node.Arguments()) > 0 || len(node.Properties()) > 0 {
				return roleGraph{}, fmt.Errorf("provider role bindings %s: roles node accepts only children", providerRolesPath)
			}
			rolesNode = node
		default:
			return roleGraph{}, fmt.Errorf("provider role bindings %s: unknown top-level node %q", providerRolesPath, node.Name())
		}
	}
	if rolesNode == nil {
		return roleGraph{}, fmt.Errorf("provider role bindings %s: missing top-level roles node", providerRolesPath)
	}
	for _, policy := range []struct {
		name string
		uses []RepositoryUse
	}{
		{name: "global", uses: globalRepos},
		{name: "resident-only", uses: residentRepos},
	} {
		for _, use := range policy.uses {
			if _, declared := repositories[use.Repository]; !declared {
				return roleGraph{}, fmt.Errorf(
					"repository policy %q references undeclared repository %q",
					policy.name,
					use.Repository,
				)
			}
		}
	}

	refs := map[string][]ContentRef{}
	uses := map[string][]ProviderUse{}
	roleRepos := map[string][]RepositoryUse{}
	seenRoles := map[string]bool{}
	for _, roleNode := range rolesNode.Children().Nodes {
		if roleNode.Name() != "role" {
			return roleGraph{}, fmt.Errorf("provider role bindings %s: unknown node %q", providerRolesPath, roleNode.Name())
		}
		role, err := oneStringArg(roleNode)
		if err != nil {
			return roleGraph{}, fmt.Errorf("provider role bindings %s: %w", providerRolesPath, err)
		}
		if len(roleNode.Properties()) > 0 {
			return roleGraph{}, fmt.Errorf("provider role %q: role accepts no properties", role)
		}
		if seenRoles[role] {
			return roleGraph{}, fmt.Errorf("provider role bindings %s: duplicate role %q", providerRolesPath, role)
		}
		seenRoles[role] = true
		if _, exists := refs[role]; !exists {
			refs[role] = nil
		}
		if _, exists := roleRepos[role]; !exists {
			roleRepos[role] = nil
		}
		seenSkills := map[string]int{}
		seenProviders := map[string]bool{}
		seenRepositories := map[string]bool{}
		seenOrgs := map[string]bool{}
		for _, child := range roleNode.Children().Nodes {
			switch child.Name() {
			case "use-org":
				orgID, err := oneStringArg(child)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role %q: %w", role, err)
				}
				if _, declared := orgs[orgID]; !declared {
					return roleGraph{}, fmt.Errorf("provider role %q references undeclared org %q", role, orgID)
				}
				if seenOrgs[orgID] {
					return roleGraph{}, fmt.Errorf("provider role %q repeats org %q", role, orgID)
				}
				if len(child.Properties()) > 0 || len(child.Children().Nodes) > 0 {
					return roleGraph{}, fmt.Errorf("provider role %q org %q: use-org accepts only one argument", role, orgID)
				}
				seenOrgs[orgID] = true
				roleOrgs[role] = append(roleOrgs[role], OrgUse{Org: orgID})
			case "composed-skill":
				if len(child.Properties()) > 0 || len(child.Children().Nodes) > 0 {
					return roleGraph{}, fmt.Errorf("provider role %q: composed-skill accepts only one pattern argument", role)
				}
				pattern, err := oneStringArg(child)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role %q: %w", role, err)
				}
				skills, err := expandComposedSkillPattern(pattern, composed)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role %q: %w", role, err)
				}
				for _, skill := range skills {
					if index, seen := seenSkills[skill]; seen {
						roleRefs := refs[role]
						roleRefs[index].Selectors = append(roleRefs[index].Selectors, pattern)
						refs[role] = roleRefs
						continue
					}
					seenSkills[skill] = len(refs[role])
					refs[role] = append(refs[role], ContentRef{
						ID:         skill,
						Path:       composed[skill],
						EntryPoint: "COMPOSED.md",
						Selectors:  []string{pattern},
					})
				}
			case "use-provider":
				providerID, err := oneStringArg(child)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role %q: %w", role, err)
				}
				_, declared := providers[providerID]
				if !declared {
					return roleGraph{}, fmt.Errorf("provider role %q references undeclared provider %q", role, providerID)
				}
				if seenProviders[providerID] {
					return roleGraph{}, fmt.Errorf("provider role %q repeats provider %q", role, providerID)
				}
				seenProviders[providerID] = true
				for property := range child.Properties() {
					if property != "required" {
						return roleGraph{}, fmt.Errorf("provider role %q provider %q: unknown property %q", role, providerID, property)
					}
				}
				required := false
				if value := child.Prop("required"); value.IsValid() {
					if value.Kind() != kdl.Bool {
						return roleGraph{}, fmt.Errorf("provider role %q provider %q: required must be boolean", role, providerID)
					}
					required = value.Bool()
				}
				bindingSkills, err := parseSkillSelectorChildren(
					fmt.Sprintf("provider role %q provider %q", role, providerID),
					child.Children().Nodes,
				)
				if err != nil {
					return roleGraph{}, err
				}
				uses[role] = append(uses[role], ProviderUse{
					Provider: providerID, Required: required, Skills: bindingSkills,
				})
			case "use-repository":
				repositoryID, err := oneStringArg(child)
				if err != nil {
					return roleGraph{}, fmt.Errorf("provider role %q: %w", role, err)
				}
				definition, declared := repositories[repositoryID]
				if !declared {
					return roleGraph{}, fmt.Errorf("provider role %q references undeclared repository %q", role, repositoryID)
				}
				if seenRepositories[repositoryID] {
					return roleGraph{}, fmt.Errorf("provider role %q repeats repository %q", role, repositoryID)
				}
				if len(child.Properties()) > 0 {
					return roleGraph{}, fmt.Errorf("provider role %q repository %q: use-repository accepts only one argument", role, repositoryID)
				}
				bindingSkills, err := parseSkillSelectorChildren(
					fmt.Sprintf("provider role %q repository %q", role, repositoryID),
					child.Children().Nodes,
				)
				if err != nil {
					return roleGraph{}, err
				}
				seenRepositories[repositoryID] = true
				if definition.Skills != nil {
					if seenProviders[repositoryID] {
						return roleGraph{}, fmt.Errorf("provider role %q repeats provider %q", role, repositoryID)
					}
					seenProviders[repositoryID] = true
					uses[role] = append(uses[role], ProviderUse{
						Provider: repositoryID, Required: true, Skills: bindingSkills,
					})
				} else {
					roleRepos[role] = append(roleRepos[role], RepositoryUse{
						Repository: repositoryID, Skills: bindingSkills,
					})
				}
			default:
				return roleGraph{}, fmt.Errorf("provider role %q: unknown node %q", role, child.Name())
			}
		}
	}
	var overlaps []SelectorOverlap
	roles := make([]string, 0, len(refs))
	for role := range refs {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		for _, ref := range refs[role] {
			if len(ref.Selectors) < 2 {
				continue
			}
			overlaps = append(overlaps, SelectorOverlap{
				Role: role, Skill: ref.ID, Selectors: append([]string(nil), ref.Selectors...),
			})
		}
	}
	return roleGraph{
		RoleSkills:    refs,
		Overlaps:      overlaps,
		Providers:     providers,
		RoleProviders: uses,
		Repositories:  repositories,
		GlobalRepos:   globalRepos,
		RoleRepos:     roleRepos,
		ResidentRepos: residentRepos,
		Orgs:          orgs,
		RoleOrgs:      roleOrgs,
	}, nil
}

func parseSkillSelectorChildren(owner string, nodes []*kdl.Node) ([]string, error) {
	var skills []string
	for _, skillNode := range nodes {
		if skillNode.Name() != "skill" {
			return nil, fmt.Errorf("%s: unknown node %q", owner, skillNode.Name())
		}
		if len(skillNode.Properties()) > 0 || len(skillNode.Children().Nodes) > 0 {
			return nil, fmt.Errorf("%s: skill accepts only one pattern argument", owner)
		}
		pattern, err := oneStringArg(skillNode)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", owner, err)
		}
		skills = append(skills, pattern)
	}
	if skills != nil {
		if err := skillselector.Validate(skills); err != nil {
			return nil, fmt.Errorf("%s: %w", owner, err)
		}
	}
	return skills, nil
}

// validateOrgOwner accepts one forge owner segment. An org names an owner and
// never a path, because which repositories it holds is the manifest's answer.
func validateOrgOwner(value string) error {
	if value == "" || strings.ContainsAny(value, `/\@`) || strings.TrimSpace(value) != value {
		return fmt.Errorf("owner %q must be one forge owner segment", value)
	}
	return nil
}

func validateLogicalProviderPath(value string) error {
	if value == "" || strings.Contains(value, `\`) || pathpkg.IsAbs(value) ||
		pathpkg.Clean(value) != value || strings.HasPrefix(value, "../") {
		return fmt.Errorf("path %q must be a clean relative repository path", value)
	}
	if parts := strings.Split(value, "/"); len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("path %q must have owner/repository form", value)
	}
	return nil
}

func expandComposedSkillPattern(pattern string, composed map[string]string) ([]string, error) {
	if _, err := pathpkg.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("composed skill pattern %q is invalid: %w", pattern, err)
	}
	names := make([]string, 0, len(composed))
	for name := range composed {
		names = append(names, name)
	}
	sort.Strings(names)

	matches := make([]string, 0, len(names))
	for _, name := range names {
		matched, _ := pathpkg.Match(pattern, name)
		if matched {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		if strings.ContainsAny(pattern, "*?[") {
			return nil, fmt.Errorf("composed skill pattern %q matches no skills", pattern)
		}
		return nil, fmt.Errorf("names missing composed skill %q", pattern)
	}
	return matches, nil
}

func parseSource(declPath string) (*Source, error) {
	raw, err := os.ReadFile(declPath)
	if err != nil {
		return nil, err
	}
	doc, err := kdl.ParseString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", declPath, err)
	}
	if len(doc.Nodes) != 1 || doc.Nodes[0].Name() != "source" {
		return nil, fmt.Errorf("%s: expected exactly one top-level source node", declPath)
	}
	root := doc.Nodes[0]
	id, err := oneStringArg(root)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", declPath, err)
	}
	src := &Source{ID: id, Root: filepath.Dir(declPath), Declaration: raw}
	for _, n := range root.Children().Nodes {
		switch n.Name() {
		case "instruction", "skill":
			cid, err := oneStringArg(n)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", declPath, err)
			}
			p := n.Prop("path")
			if !p.IsValid() {
				return nil, fmt.Errorf("%s: %s %q needs a path property", declPath, n.Name(), cid)
			}
			rel := p.String()
			if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
				return nil, fmt.Errorf("%s: %s %q path %q must stay beneath the source root", declPath, n.Name(), cid, rel)
			}
			ref := ContentRef{ID: cid, Path: rel}
			if n.Name() == "instruction" {
				src.Instructions = append(src.Instructions, ref)
			} else {
				ref.EntryPoint = "SKILL.md"
				src.Skills = append(src.Skills, ref)
			}
		default:
			return nil, fmt.Errorf("%s: unknown node %q", declPath, n.Name())
		}
	}
	return src, nil
}

func oneStringArg(n *kdl.Node) (string, error) {
	args := n.Arguments()
	if len(args) != 1 {
		return "", fmt.Errorf("node %q: expected exactly one argument", n.Name())
	}
	v := args[0].String()
	if v == "" {
		return "", fmt.Errorf("node %q: argument must be a non-empty string", n.Name())
	}
	return v, nil
}
