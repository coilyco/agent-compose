package resolver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

const (
	OutcomeSelected  = "selected"
	OutcomeExcluded  = "excluded"
	OutcomeShadowed  = "shadowed"
	OutcomeDelivered = "delivered"

	ProviderCategoryPerson    = "person-package"
	ProviderCategoryCatalogue = "catalogue"
	ProviderCategoryRole      = "role-provider"
)

// composedBoundaries is the role's active set minus what this deployment
// omitted. See docs/boundary-omission.md.
func composedBoundaries(p *person.Person, req *schema.Request) ([]string, error) {
	active := p.RoleActiveBoundaries(req.Role)
	if len(req.BoundaryOmissions) == 0 {
		return active, nil
	}
	inActive := map[string]bool{}
	for _, name := range active {
		inActive[name] = true
	}
	omitted := map[string]bool{}
	for _, name := range req.BoundaryOmissions {
		binding, defined := p.Boundaries[name]
		if !defined {
			return nil, fmt.Errorf("boundary-omit names unknown boundary %q", name)
		}
		// An owner losing its own boundary is a larger claim than a deferrer
		// losing one, and nothing here is allowed to make it.
		if binding.Owner == req.Role {
			return nil, fmt.Errorf("boundary-omit %q is owned by role %q and cannot be omitted", name, req.Role)
		}
		// A scoped grant is a bounded permission rather than a deferral, so
		// dropping it widens the role instead of narrowing the deployment.
		for _, scoped := range p.Roles[req.Role].ScopedBoundaries {
			if scoped.Name == name {
				return nil, fmt.Errorf(
					"boundary-omit %q is held within a scope by role %q and cannot be omitted",
					name, req.Role,
				)
			}
		}
		if !inActive[name] {
			return nil, fmt.Errorf("boundary-omit %q is not active for role %q", name, req.Role)
		}
		omitted[name] = true
	}
	kept := make([]string, 0, len(active))
	for _, name := range active {
		if !omitted[name] {
			kept = append(kept, name)
		}
	}
	return kept, nil
}

type Decision struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	Source  string `json:"source,omitempty"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

type Selected struct {
	ID         string
	Source     string
	Files      fs.FS
	Path       string
	EntryPoint string
}

// ProviderReport retains one provider outcome and its canonical selected-skill
// byte budget, which stays deterministic across native and staged projection.
type ProviderReport struct {
	Source            string `json:"source"`
	Category          string `json:"category"`
	Scope             string `json:"scope"`
	Outcome           string `json:"outcome"`
	Reason            string `json:"reason"`
	Warning           bool   `json:"warning,omitempty"`
	Skills            int    `json:"skills"`
	ContextBytes      int64  `json:"context_bytes"`
	ApproximateTokens int64  `json:"approximate_tokens"`
	SelectorReason    string `json:"selector_reason,omitempty"`
}

// Resolution is the full composition plan: what was selected, how it is
// delivered, and the trace built while those choices were made.
type Resolution struct {
	Request           *schema.Request
	Person            *person.Person
	Personalities     []string
	Boundaries        []string
	RolePurpose       string
	RoleBriefing      string
	OperatingBase     string
	OperatingAppendix string
	Instructions      []Selected
	Skills            []Selected
	CompiledBodies    []Selected
	FavoriteColor     string
	Warnings          []string
	Decisions         []Decision
	Providers         []ProviderReport
	SourceIDs         []string
	Repositories      []schema.RepositorySelection
}

func Resolve(req *schema.Request, p *person.Person, sources []*schema.Source, missing []schema.MissingSource) (*Resolution, error) {
	if req.ModelTier == "" {
		req.ModelTier = schema.ModelTierFrontier
	}
	if !schema.IsModelTier(req.ModelTier) {
		return nil, fmt.Errorf("unsupported model tier %q", req.ModelTier)
	}
	personSource, err := person.Source(p)
	if err != nil {
		return nil, err
	}
	personSource.ProviderScope = schema.ProviderScopePerson
	for _, src := range sources {
		if src.ProviderScope == "" {
			src.ProviderScope = schema.ProviderScopeRequest
		}
	}
	sources = append([]*schema.Source{personSource}, sources...)

	role, ok := p.Roles[req.Role]
	if !ok {
		return nil, fmt.Errorf("role %q is not defined by person %q; defined roles: %s",
			req.Role, p.Name, strings.Join(sortedKeys(p.Roles), ", "))
	}
	if !role.SupportsModelTier(req.ModelTier) {
		return nil, fmt.Errorf(
			"role %q does not support model tier %q",
			req.Role,
			req.ModelTier,
		)
	}
	if len(role.Personalities) == 0 {
		return nil, fmt.Errorf("role %q defines no personalities", req.Role)
	}
	// Only the requested role's bindings are read, so a source may stage a
	// role this roster does not define. See docs/kdl-contracts.md.
	roleSkill := p.RoleSkillID(req.Role)

	activeBySkill := make(map[string]string, len(role.Personalities))
	personalityBySkill := make(map[string]string, len(p.Personalities))
	activeSkills := make([]string, 0, len(role.Personalities))
	colors := make([]string, 0, len(role.Personalities))
	for name, binding := range p.Personalities {
		personalityBySkill[binding.Skill] = name
	}
	for _, name := range role.Personalities {
		binding, ok := p.Personalities[name]
		if !ok {
			return nil, fmt.Errorf("role %q names personality %q without a catalog binding", req.Role, name)
		}
		if prior, duplicate := activeBySkill[binding.Skill]; duplicate {
			return nil, fmt.Errorf("role %q personalities %q and %q bind the same skill %q",
				req.Role, prior, name, binding.Skill)
		}
		activeBySkill[binding.Skill] = name
		activeSkills = append(activeSkills, binding.Skill)
		colors = append(colors, binding.Color)
	}
	favorite := role.FavoriteColor

	boundaries, err := composedBoundaries(p, req)
	if err != nil {
		return nil, err
	}
	omittedSkills := map[string]bool{}
	for _, name := range req.BoundaryOmissions {
		omittedSkills[p.Boundaries[name].Skill] = true
	}

	res := &Resolution{
		Request:       req,
		Person:        p,
		Personalities: append([]string(nil), role.Personalities...),
		Boundaries:    append([]string(nil), boundaries...),
		RolePurpose:   role.Purpose,
		RoleBriefing:  role.Briefing,
		FavoriteColor: favorite,
		Repositories:  append([]schema.RepositorySelection(nil), req.Repositories...),
	}
	priorRepository := ""
	for _, repository := range res.Repositories {
		if repository.Identity == "" || repository.Source == "" || repository.Scope == "" || repository.Reason == "" {
			return nil, fmt.Errorf("repository selections need identity, source, scope, and reason provenance")
		}
		if repository.Identity <= priorRepository {
			return nil, fmt.Errorf("repository selections must be strictly sorted and deduplicated")
		}
		priorRepository = repository.Identity
		res.decide(Decision{
			Subject: "repository:" + repository.Identity,
			Kind:    "repository", Source: repository.Source,
			Outcome: OutcomeSelected, Reason: repository.Reason,
		})
	}
	for _, src := range sources {
		res.SourceIDs = append(res.SourceIDs, src.ID)
	}
	res.decide(Decision{
		Subject: "role:" + req.Role, Kind: "profile", Source: p.ProviderID(),
		Outcome: OutcomeSelected,
		Reason:  fmt.Sprintf("%s defines this role: %s", p.ProviderID(), role.Purpose),
	})
	for _, name := range role.Personalities {
		res.decide(Decision{
			Subject: "personality:" + name, Kind: "profile", Source: p.ProviderID(),
			Outcome: OutcomeSelected,
			Reason:  fmt.Sprintf("role %q activates its full personality set: %s", req.Role, strings.Join(role.Personalities, ", ")),
		})
	}
	for _, name := range req.BoundaryOmissions {
		res.decide(Decision{
			Subject: "boundary:" + name, Kind: "profile", Source: p.ProviderID(),
			Outcome: OutcomeExcluded,
			Reason: fmt.Sprintf(
				"request omits boundary %q, whose owner %q is not a seat in this deployment",
				name, p.Boundaries[name].Owner),
		})
	}
	for _, name := range boundaries {
		// The trace names which of the three sides the role received, because
		// a scoped grant reading as a deferral hides a permission.
		relationship := "defers"
		if p.Boundaries[name].Owner == req.Role {
			relationship = "owns"
		}
		for _, scoped := range p.Roles[req.Role].ScopedBoundaries {
			if scoped.Name == name {
				relationship = "holds within a scope"
				break
			}
		}
		res.decide(Decision{
			Subject: "boundary:" + name, Kind: "profile", Source: p.ProviderID(),
			Outcome: OutcomeSelected,
			Reason: fmt.Sprintf(
				"role %q %s boundary %q, whose body is identical on every side",
				req.Role, relationship, name),
		})
	}
	for _, m := range missing {
		if m.ProviderScope == "" {
			m.ProviderScope = schema.ProviderScopeRequest
		}
		res.decide(Decision{
			Subject: "source:" + m.ID, Kind: "source", Source: m.ID,
			Outcome: OutcomeExcluded, Reason: m.Reason,
		})
		for _, skill := range m.Skills {
			res.decide(Decision{
				Subject: "skill:" + skill, Kind: "skill", Source: m.ID,
				Outcome: OutcomeExcluded, Reason: m.Reason,
			})
		}
	}
	for _, src := range sources {
		reason := providerAdmissionReason(src, req.Role)
		res.decide(Decision{
			Subject: "source:" + src.ID, Kind: "source", Source: src.ID,
			Outcome: OutcomeSelected, Reason: reason,
		})
	}
	for _, src := range sources {
		for _, overlap := range src.SelectorOverlaps {
			reason := selectorOverlapReason(overlap)
			res.Warnings = append(res.Warnings, fmt.Sprintf("source %q %s", src.ID, reason))
			res.decide(Decision{
				Subject: "selector:" + overlap.Skill,
				Kind:    "selector",
				Source:  src.ID,
				Outcome: OutcomeShadowed,
				Reason:  reason,
			})
		}
	}
	for _, src := range sources {
		for _, ref := range src.ExcludedSkills {
			res.decide(Decision{
				Subject: "skill:" + ref.ID, Kind: "skill", Source: src.ID,
				Outcome: OutcomeExcluded,
				Reason:  src.SelectorReason + "; this skill matched no configured selector pattern",
			})
		}
	}

	instructionBytes := map[string][]byte{}
	instructionOwner := map[string]string{}
	selectedBySkill := map[string]Selected{}
	skillDigests := map[string]string{}
	considerSkill := func(src *schema.Source, ref schema.ContentRef, reason string) error {
		digest, err := treeDigest(src.FileSystem(), ref.Path)
		if err != nil {
			return fmt.Errorf("source %q skill %q: %w", src.ID, ref.ID, err)
		}
		if selected, found := selectedBySkill[ref.ID]; found {
			if digest == skillDigests[ref.ID] && entryPoint(ref) == selected.EntryPoint {
				res.decide(Decision{
					Subject: "skill:" + ref.ID, Kind: "skill", Source: src.ID,
					Outcome: OutcomeShadowed,
					Reason:  fmt.Sprintf("identical to the copy already selected from %s", selected.Source),
				})
				return nil
			}
			return fmt.Errorf("skill %q from %s conflicts with a different copy from %s; v0.1 fails non-identical collisions",
				ref.ID, src.ID, selected.Source)
		}
		selectedBySkill[ref.ID] = Selected{
			ID: ref.ID, Source: src.ID, Files: src.FileSystem(), Path: ref.Path, EntryPoint: entryPoint(ref),
		}
		skillDigests[ref.ID] = digest
		res.decide(Decision{
			Subject: "skill:" + ref.ID, Kind: "skill", Source: src.ID,
			Outcome: OutcomeSelected, Reason: reason,
		})
		return nil
	}
	for _, src := range sources {
		for _, ref := range src.Instructions {
			raw, err := src.ReadFile(ref.Path)
			if err != nil {
				return nil, fmt.Errorf("source %q instruction %q: %w", src.ID, ref.ID, err)
			}
			body := normalizeEOL(raw)
			if prior, dup := instructionBytes[ref.ID]; dup {
				if string(prior) == string(body) {
					res.decide(Decision{
						Subject: "instruction:" + ref.ID, Kind: "instruction", Source: src.ID,
						Outcome: OutcomeShadowed,
						Reason:  fmt.Sprintf("identical to the copy already selected from %s", instructionOwner[ref.ID]),
					})
					continue
				}
				return nil, fmt.Errorf("instruction %q from %s conflicts with a different copy from %s; v0.1 fails non-identical collisions",
					ref.ID, src.ID, instructionOwner[ref.ID])
			}
			instructionBytes[ref.ID] = body
			instructionOwner[ref.ID] = src.ID
			res.Instructions = append(res.Instructions, Selected{
				ID: ref.ID, Source: src.ID, Files: src.FileSystem(), Path: ref.Path,
			})
			res.decide(Decision{
				Subject: "instruction:" + ref.ID, Kind: "instruction", Source: src.ID,
				Outcome: OutcomeSelected, Reason: "instructions from admitted sources are always selected",
			})
		}
		for _, ref := range src.Skills {
			_, isPersonality := personalityBySkill[ref.ID]
			activePersonality, active := activeBySkill[ref.ID]
			if isPersonality && !active {
				res.decide(Decision{
					Subject: "skill:" + ref.ID, Kind: "skill", Source: src.ID,
					Outcome: OutcomeExcluded,
					Reason: fmt.Sprintf("role %q activates personalities %s, which bind skills %s",
						req.Role, strings.Join(role.Personalities, ", "), strings.Join(activeSkills, ", ")),
				})
				continue
			}
			reason := "ordinary provider skills are discoverable for every role"
			if src.AdmissionReason != "" {
				reason = src.AdmissionReason + ". Ordinary skills from the admitted provider are discoverable"
			}
			if src.SelectorReason != "" {
				reason += ". " + src.SelectorReason
			}
			if isPersonality {
				reason = fmt.Sprintf("active personality %q binds this skill", activePersonality)
			}
			if err := considerSkill(src, ref, reason); err != nil {
				return nil, err
			}
		}
		for _, ref := range src.RoleSkills[req.Role] {
			if omittedSkills[ref.ID] {
				continue
			}
			if err := considerSkill(
				src,
				ref,
				roleSkillReason(req.Role, ref),
			); err != nil {
				return nil, err
			}
		}
	}

	added := map[string]bool{}
	if selected, found := selectedBySkill[roleSkill]; found {
		res.Skills = append(res.Skills, selected)
		added[roleSkill] = true
	} else {
		return nil, fmt.Errorf("role %q binds skill %q, but no admitted source provides it", req.Role, roleSkill)
	}
	for _, name := range boundaries {
		binding, ok := p.Boundaries[name]
		if !ok {
			return nil, fmt.Errorf("role %q names boundary %q without a catalog binding", req.Role, name)
		}
		selected, found := selectedBySkill[binding.Skill]
		if !found {
			return nil, fmt.Errorf("boundary %q binds skill %q, but no admitted source provides it", name, binding.Skill)
		}
		if added[binding.Skill] {
			continue
		}
		res.Skills = append(res.Skills, selected)
		added[binding.Skill] = true
	}
	for _, name := range role.Personalities {
		boundSkill := p.Personalities[name].Skill
		selected, found := selectedBySkill[boundSkill]
		if !found {
			return nil, fmt.Errorf("personality %q binds skill %q, but no admitted source provides it", name, boundSkill)
		}
		res.Skills = append(res.Skills, selected)
		added[boundSkill] = true
	}
	for _, src := range sources {
		refs := append([]schema.ContentRef{}, src.Skills...)
		refs = append(refs, src.RoleSkills[req.Role]...)
		for _, ref := range refs {
			if added[ref.ID] {
				continue
			}
			selected, found := selectedBySkill[ref.ID]
			if !found {
				continue
			}
			res.Skills = append(res.Skills, selected)
			added[ref.ID] = true
		}
	}
	if err := res.buildProviderReports(sources, missing); err != nil {
		return nil, err
	}

	if err := res.planDelivery(); err != nil {
		return nil, err
	}
	return res, nil
}

func roleSkillReason(role string, ref schema.ContentRef) string {
	if len(ref.Selectors) == 0 {
		return fmt.Sprintf("role %q composes this skill", role)
	}
	return fmt.Sprintf(
		"role %q composes this skill through selector(s) %s",
		role,
		quotedSelectors(ref.Selectors),
	)
}

func selectorOverlapReason(overlap schema.SelectorOverlap) string {
	return fmt.Sprintf(
		"provider role %q matched composed skill %q through selectors %s, selected once",
		overlap.Role,
		overlap.Skill,
		quotedSelectors(overlap.Selectors),
	)
}

func quotedSelectors(selectors []string) string {
	quoted := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		quoted = append(quoted, fmt.Sprintf("%q", selector))
	}
	return strings.Join(quoted, ", ")
}

func providerAdmissionReason(src *schema.Source, role string) string {
	if src.AdmissionReason != "" {
		return src.AdmissionReason
	}
	if src.ProviderScope == schema.ProviderScopePerson {
		return fmt.Sprintf("selected person package defines role %q and its active personalities", role)
	}
	return "capability provider admitted by the compose request"
}

func providerCategory(scope string) string {
	switch scope {
	case schema.ProviderScopePerson:
		return ProviderCategoryPerson
	case schema.ProviderScopeRole:
		return ProviderCategoryRole
	default:
		return ProviderCategoryCatalogue
	}
}

func (r *Resolution) buildProviderReports(
	sources []*schema.Source,
	missing []schema.MissingSource,
) error {
	type contribution struct {
		skills int
		bytes  int64
	}
	contributions := map[string]contribution{}
	for _, skill := range r.Skills {
		bytes, err := treeBytes(skill.Files, skill.Path)
		if err != nil {
			return fmt.Errorf("measure source %q skill %q context: %w", skill.Source, skill.ID, err)
		}
		current := contributions[skill.Source]
		current.skills++
		current.bytes += bytes
		contributions[skill.Source] = current
	}
	for _, src := range sources {
		contribution := contributions[src.ID]
		r.Providers = append(r.Providers, ProviderReport{
			Source:            src.ID,
			Category:          providerCategory(src.ProviderScope),
			Scope:             src.ProviderScope,
			Outcome:           OutcomeSelected,
			Reason:            providerAdmissionReason(src, r.Request.Role),
			Skills:            contribution.skills,
			ContextBytes:      contribution.bytes,
			ApproximateTokens: (contribution.bytes + 3) / 4,
			SelectorReason:    src.SelectorReason,
		})
	}
	for _, source := range missing {
		scope := source.ProviderScope
		if scope == "" {
			scope = schema.ProviderScopeRequest
		}
		r.Providers = append(r.Providers, ProviderReport{
			Source:   source.ID,
			Category: providerCategory(scope),
			Scope:    scope,
			Outcome:  OutcomeExcluded,
			Reason:   source.Reason,
			Warning:  source.Warning,
		})
	}
	return nil
}

// planDelivery chooses the compiled body for each skill.
func (r *Resolution) planDelivery() error {
	r.decide(Decision{
		Subject: "content/instructions.md", Kind: "delivery",
		Outcome: OutcomeDelivered, Reason: "canonical selected instructions",
	})
	r.decide(Decision{
		Subject: "content/skills", Kind: "delivery",
		Outcome: OutcomeDelivered, Reason: "canonical selected skill trees",
	})
	if r.Request.Delivery != schema.DeliveryCompiled {
		return nil
	}
	for _, skill := range r.Skills {
		body := path.Join(skill.Path, skill.EntryPoint)
		if _, err := fs.Stat(skill.Files, body); err != nil {
			return fmt.Errorf("skill %q: compiled delivery needs %s: %w", skill.ID, path.Base(body), err)
		}
		r.CompiledBodies = append(r.CompiledBodies, skill)
	}
	r.decide(Decision{
		Subject: "delivery/compiled.md", Kind: "delivery",
		Outcome: OutcomeDelivered, Reason: "selected instructions and skill prose compiled into one context document",
	})
	return nil
}

func entryPoint(ref schema.ContentRef) string {
	if ref.EntryPoint != "" {
		return ref.EntryPoint
	}
	return "SKILL.md"
}

func (r *Resolution) decide(d Decision) {
	r.Decisions = append(r.Decisions, d)
}

func treeDigest(files fs.FS, root string) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(files, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinks are invalid inside a source", p)
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, strings.TrimSuffix(root, "/")+"/")
		if rel == p {
			return fmt.Errorf("%s is not beneath %s", p, root)
		}
		raw, err := fs.ReadFile(files, p)
		if err != nil {
			return err
		}
		body := normalizeEOL(raw)
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(body))
		h.Write(body)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// normalizeEOL folds CRLF so two checkouts of one commit digest equal, and
// hashes raw when a NUL byte says the content is binary.
func normalizeEOL(raw []byte) []byte {
	if bytes.IndexByte(raw, 0) >= 0 {
		return raw
	}
	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
}

func treeBytes(files fs.FS, root string) (int64, error) {
	var total int64
	err := fs.WalkDir(files, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinks are invalid inside a source", p)
		}
		if d.IsDir() {
			return nil
		}
		raw, err := fs.ReadFile(files, p)
		if err != nil {
			return err
		}
		total += int64(len(raw))
		return nil
	})
	return total, err
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
