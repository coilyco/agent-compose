// Package person loads the mounted roster: roles, agent seats, personality
// definitions, and the invariant. The binary embeds none of it.
package person

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing/fstest"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

const (
	minRoleSkillBodyWords = 140
	maxRoleSkillBodyWords = 1200
)

// Personality and boundary prose carry their own bounds. A floor keeps an entry
// from thinning into a label. See docs/science-context-budget.md.
const (
	minPersonalitySkillBodyWords = 120
	maxPersonalitySkillBodyWords = 320
	minBoundarySideWords         = 80
)

// maxBoundarySkillBodyWords bounds each side of a boundary separately from the
// role charter that declares it. See docs/role-boundaries.md.
const maxBoundarySkillBodyWords = 200

// adjacentsPerRole fixes the out-degree of the role adjacency graph, so the
// roster chooses its sharpest confusions. See docs/role-boundaries.md.
const adjacentsPerRole = 2

// Three is the authored number rather than a measured optimum, and
// housecast#6 is where measuring it belongs. See docs/kdl-contracts.md.
const actsPerAttribute = 3

// The scoped side is required even though its prose section is optional. Why:
// docs/kdl-contracts.md.
var boundaryActSides = []string{"own", "scoped", "defer"}

// Both sides of a boundary live in one body under conditional headings, so the
// reader self-selects. See docs/ownership.md.
const (
	boundaryOwnHeading    = "## If you own this boundary"
	boundaryScopedHeading = "## If you hold this boundary within a scope"
	boundaryDeferHeading  = "## If you defer this boundary"
)

var personSections = []struct {
	directory string
	node      string
}{
	{directory: "roles", node: "role"},
	{directory: "boundaries", node: "boundary"},
	{directory: "personalities", node: "personality"},
	{directory: "guardrails", node: "guardrail"},
}

var librarySections = []struct {
	directory string
	node      string
}{
	{directory: "boundaries", node: "boundary"},
	{directory: "personalities", node: "personality"},
	{directory: "guardrails", node: "guardrail"},
}

// Seat is one named agent identity within a role. The harness joins the
// launcher's own catalog, while the name remains opaque here.
type Seat struct {
	// Key is the stable profile-owned seat selector. Harness remains populated
	// for legacy agent entries and v1 result compatibility.
	Key      string `json:"key,omitempty" yaml:"key,omitempty"`
	Harness  string `json:"harness" yaml:"harness"`
	Name     string `json:"name" yaml:"name"`
	Pronouns string `json:"pronouns" yaml:"pronouns"`
	Channel  string `json:"channel,omitempty" yaml:"channel,omitempty"`
	Tier     string `json:"tier,omitempty" yaml:"tier,omitempty"`
	// LegalName is the seat's own product name, authored rather than derived.
	// Absent stays absent: identity describes and grants nothing (#396).
	LegalName string `json:"legal_name,omitempty" yaml:"legal_name,omitempty"`
}

// AgentIdentity is the role-owned name and pronoun pair shared by every seat.
// Legacy external packages may continue to own identity on individual seats.
type AgentIdentity struct {
	Name     string `json:"name" yaml:"name"`
	Pronouns string `json:"pronouns" yaml:"pronouns"`
}

type Role struct {
	DisplayName      string           `json:"display_name"`
	Purpose          string           `json:"purpose"`
	Skill            string           `json:"skill"`
	SkillSource      string           `json:"skill_source"`
	SkillDigest      string           `json:"skill_digest"`
	Methods          []string         `json:"methods,omitempty"`
	Boundaries       []string         `json:"boundaries,omitempty"`
	ScopedBoundaries []ScopedBoundary `json:"scoped_boundaries,omitempty"`
	Adjacents        []Adjacent       `json:"adjacents,omitempty"`
	Acts             []Act            `json:"acts,omitempty"`
	Briefing         string           `json:"briefing"`
	Stance           string           `json:"stance,omitempty"`
	Voice            *Voice           `json:"voice,omitempty"`
	Outro            *Outro           `json:"outro,omitempty"`
	Creature         string           `json:"creature,omitempty"`
	// Element selects the animal lineage a seat is drawn from, ATLA-style.
	// Kai's assignment, mirrored by the renderer. See docs/identity.md.
	Element string `json:"element,omitempty"`
	// Guardrail names this role's guardrail element, or is empty. Four roles
	// carry one by Kai's scope decision, so absence is a decision.
	Guardrail           string         `json:"guardrail,omitempty"`
	Personalities       []string       `json:"personalities"`
	FavoriteColor       string         `json:"favorite_color,omitempty"`
	Background          string         `json:"background,omitempty"`
	Identity            *AgentIdentity `json:"identity,omitempty"`
	Seats               []Seat         `json:"seats"`
	SupportedModelTiers []string       `json:"supported_model_tiers,omitempty"`
	CopyContract        *CopyContract  `json:"copy_contract,omitempty"`
	// Retired from selection, kept whole and still a valid adjacency target.
	// See docs/role-selection.md.
	Archived bool `json:"archived,omitempty"`
}

// ScopedBoundary is a bounded grant, not an absence: the scope text is the
// content, so the flat declared list cannot hold it. See docs/role-boundaries.md.
type ScopedBoundary struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

// Adjacent names one role whose work this role most risks absorbing. The reason
// is generator input rather than commentary. See docs/role-boundaries.md.
type Adjacent struct {
	Role   string `json:"role"`
	Reason string `json:"reason"`
}

// Act is one thing an attribute requires a seat to actually run.
// Field-by-field: docs/kdl-contracts.md.
type Act struct {
	Tool string `json:"tool"`
	Text string `json:"text"`
	Side string `json:"side,omitempty"`
}

// SupportsModelTier keeps packages authored before the tier axis compatible.
// First-party roles declare their supported tiers explicitly.
func (r Role) SupportsModelTier(modelTier string) bool {
	if len(r.SupportedModelTiers) == 0 {
		return true
	}
	for _, supported := range r.SupportedModelTiers {
		if modelTier == supported {
			return true
		}
	}
	return false
}

type CopyContract struct {
	Scope  string     `json:"scope"`
	Rules  []CopyRule `json:"rules"`
	Source string     `json:"source"`
	Digest string     `json:"digest"`
}

type CopyRule struct {
	Forbid string `json:"forbid"`
	Prefer string `json:"prefer"`
}

// Emblem gives renderers equivalent plain-text and rich-text marks. Names are
// ordered widest-reading first. See docs/overlay.md.
type Emblem struct {
	Names []string `json:"names"`
	Emoji string   `json:"emoji"`
}

// Name is the single mark a renderer with no room for the full list shows.
func (e Emblem) Name() string {
	if len(e.Names) == 0 {
		return ""
	}
	return e.Names[0]
}

// Body is the creature the emblem belongs to. Anatomy leads and the object
// follows, for the reason in docs/overlay.md.
type Body struct {
	Archetype  string `json:"archetype"`
	Attachment string `json:"attachment"`
}

// SoundMark describes a short identity cue without prescribing audio files or
// playback behavior.
type SoundMark struct {
	Timbre  string `json:"timbre"`
	Contour string `json:"contour"`
	Pulse   string `json:"pulse"`
}

// Outro is what a session says as it closes. Role only, not melded, for the
// reason in docs/overlay.md.
type Outro struct {
	Clean   string `json:"clean,omitempty"`
	Failure string `json:"failure,omitempty"`
}

// Voice melds like the creature does: role authors the baseline, personalities
// modulate. Banks rather than prose because a bank is checkable. See #378.
type Voice struct {
	Summary string   `json:"summary,omitempty"`
	Person  string   `json:"person,omitempty"`
	Cadence string   `json:"cadence,omitempty"`
	Prefer  []string `json:"prefer,omitempty"`
	Avoid   []string `json:"avoid,omitempty"`
	Tell    string   `json:"tell,omitempty"`
}

// Boundary binds one shared doctrine body that any number of roles may activate.
// Its optional owner is described in docs/ownership.md.
type Boundary struct {
	Skill   string `json:"skill"`
	Summary string `json:"summary"`
	Owner   string `json:"owner,omitempty"`
	Source  string `json:"source,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Acts    []Act  `json:"acts,omitempty"`
}

// Guardrail is one strong course correction bound to a single role. Card is
// eager, so the correction fires without the seat choosing to load its skill.
type Guardrail struct {
	Skill    string            `json:"skill"`
	Role     string            `json:"role"`
	Card     string            `json:"card"`
	Detector string            `json:"detector,omitempty"`
	Attests  *GuardrailAttests `json:"attests,omitempty"`
	Source   string            `json:"source,omitempty"`
	Digest   string            `json:"digest,omitempty"`
}

// GuardrailAttests carries the two eval targets an attested guardrail authors.
// A detector generates both instead, so exactly one of the pair is ever set.
type GuardrailAttests struct {
	In  string `json:"in"`
	Out string `json:"out"`
}

// ActsForSide returns the acts a seat holding this boundary on one side runs.
// A deferred boundary is a different action, never the owner's withheld.
func (b Boundary) ActsForSide(side string) []Act {
	matched := []Act{}
	for _, act := range b.Acts {
		if act.Side == side {
			matched = append(matched, act)
		}
	}
	return matched
}

// Personality binds the definition, visual and sensory identity primitives,
// and favorite color for one canonical personality.
type Personality struct {
	Skill     string    `json:"skill"`
	Color     string    `json:"color"`
	Motif     string    `json:"motif"`
	Geometry  string    `json:"geometry"`
	Emblem    Emblem    `json:"emblem"`
	Body      Body      `json:"body"`
	SoundMark SoundMark `json:"sound_mark"`
	Aliases   []string  `json:"aliases,omitempty"`
	Verbs     []string  `json:"verbs,omitempty"`
	Voice     *Voice    `json:"voice,omitempty"`
	Acts      []Act     `json:"acts,omitempty"`
}

// Selector returns the stable key used by all new commands and artifacts.
func (s Seat) Selector() string {
	if s.Key != "" {
		return s.Key
	}
	return s.Harness
}

// NormalizeCue turns a user-facing personality cue into a stable lookup form.
func NormalizeCue(cue string) (string, error) {
	cue = strings.TrimSpace(norm.NFKC.String(cue))
	if cue == "" {
		return "", fmt.Errorf("cue is empty")
	}
	var out strings.Builder
	separator := false
	for _, r := range strings.ToLower(cue) {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("cue contains a control character")
		}
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			separator = out.Len() > 0
			continue
		}
		if separator {
			out.WriteByte('-')
			separator = false
		}
		out.WriteRune(r)
	}
	if separator || out.Len() == 0 {
		return "", fmt.Errorf("cue has a leading or trailing separator")
	}
	return out.String(), nil
}

// LookupCue preserves catalogue order and never hides an ambiguous alias.
// A canonical slug is an exact match and therefore wins over aliases.
func (p *Person) LookupCue(cue string) ([]string, error) {
	normalized, err := NormalizeCue(cue)
	if err != nil {
		return nil, err
	}
	for _, name := range p.personalityOrder() {
		slug, err := NormalizeCue(name)
		if err != nil {
			return nil, fmt.Errorf("normalize personality slug %q: %w", name, err)
		}
		if slug == normalized {
			return []string{name}, nil
		}
	}
	var matches []string
	for _, name := range p.personalityOrder() {
		for _, alias := range p.Personalities[name].Aliases {
			normalizedAlias, err := NormalizeCue(alias)
			if err != nil {
				return nil, fmt.Errorf("normalize alias %q for personality %q: %w", alias, name, err)
			}
			if normalizedAlias == normalized {
				matches = append(matches, name)
				break
			}
		}
	}
	return matches, nil
}

func (p *Person) personalityOrder() []string {
	if len(p.PersonalityOrder) == len(p.Personalities) {
		return append([]string(nil), p.PersonalityOrder...)
	}
	names := make([]string, 0, len(p.Personalities))
	for name := range p.Personalities {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *Person) boundaryOrder() []string {
	if len(p.BoundaryOrder) == len(p.Boundaries) {
		return append([]string(nil), p.BoundaryOrder...)
	}
	names := make([]string, 0, len(p.Boundaries))
	for name := range p.Boundaries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *Person) roleOrder() []string {
	if len(p.RoleOrder) == len(p.Roles) {
		return append([]string(nil), p.RoleOrder...)
	}
	names := make([]string, 0, len(p.Roles))
	for name := range p.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var expressionVocabulary = [...]string{
	"available",
	"listening",
	"thinking",
	"acting",
	"waiting-for-human",
	"blocked",
	"completed",
	"failed",
	"offline",
}

// ExpressionVocabulary returns the complete stable renderer state vocabulary.
func ExpressionVocabulary() []string {
	return append([]string(nil), expressionVocabulary[:]...)
}

type Person struct {
	ProviderKind         string                 `json:"provider_kind"`
	Name                 string                 `json:"person"`
	Roles                map[string]Role        `json:"roles"`
	RoleOrder            []string               `json:"role_order"`
	Boundaries           map[string]Boundary    `json:"boundaries,omitempty"`
	BoundaryOrder        []string               `json:"boundary_order,omitempty"`
	Personalities        map[string]Personality `json:"personalities"`
	PersonalityOrder     []string               `json:"personality_order"`
	Guardrails           map[string]Guardrail   `json:"guardrails,omitempty"`
	GuardrailOrder       []string               `json:"guardrail_order,omitempty"`
	Raw                  []byte                 `json:"-"`
	Libraries            map[string]string      `json:"-"`
	PersonalityLibraries map[string]string      `json:"-"`
	roleSkills           map[string][]byte
	roleMethods          map[string]map[string][]byte
	boundarySkills       map[string][]byte
	guardrailSkills      map[string][]byte
	source               fs.FS
}

// ProviderID returns the package identity. Legacy in-memory fixtures default
// to person so external package callers retain their v1 identity.
func (p *Person) ProviderID() string {
	return p.providerKind() + ":" + p.Name
}

func (p *Person) localSourceID() string {
	return p.ProviderID() + ":local"
}

func (p *Person) providerKind() string {
	if p.ProviderKind == "" {
		return "person"
	}
	return p.ProviderKind
}

// RoleDisplayName returns the authored product label for a role.
func (p *Person) RoleDisplayName(roleName string) string {
	if role, ok := p.Roles[roleName]; ok && role.DisplayName != "" {
		return role.DisplayName
	}
	return displaySlug(roleName)
}

// Load returns the shipped roster:core package.
func Load() (*Person, error) {
	source, label, err := rosterSeed()
	if err != nil {
		return nil, err
	}
	p, err := loadSource(source, label)
	if err != nil {
		return nil, err
	}
	if err := resolveAndValidatePerson(p); err != nil {
		return nil, err
	}
	return p, nil
}

// LoadDirectory reads one complete external person package using the default layout.
// An external package replaces the mounted roster rather than extending it.
func LoadDirectory(root string) (*Person, error) {
	return LoadDirectoryWithLibraries(root)
}

// LoadDirectoryWithLibraries loads a compatible profile root, its lexical
// package-local libraries, and explicitly admitted local library roots.
func LoadDirectoryWithLibraries(root string, libraries ...string) (*Person, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("external person source path is empty")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve external person source %s: %w", root, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("inspect external person source %s: %w", absolute, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("external person source %s is not a directory", absolute)
	}
	if err := filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("external person source contains symlink %s", path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("inspect external person source tree %s: %w", absolute, err)
	}
	p, err := loadSource(os.DirFS(absolute), "external person source "+absolute)
	if err != nil {
		return nil, err
	}
	local, err := discoverLibraries(absolute)
	if err != nil {
		return nil, err
	}
	return mergeLibraries(p, append(local, libraries...))
}

func discoverLibraries(root string) ([]string, error) {
	dir := filepath.Join(root, "libraries")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profile libraries %s: %w", dir, err)
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf("profile libraries has non-directory entry %q", entry.Name())
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

func mergeLibraries(p *Person, roots []string) (*Person, error) {
	if len(roots) == 0 {
		return p, resolveAndValidatePerson(p)
	}
	if p.Libraries == nil {
		p.Libraries = map[string]string{p.localSourceID(): "profile-local"}
	}
	if p.PersonalityLibraries == nil {
		p.PersonalityLibraries = map[string]string{}
		for name := range p.Personalities {
			p.PersonalityLibraries[name] = p.localSourceID()
		}
	}
	overlay, err := definitionOverlay(p.source, p.Personalities, p.Boundaries)
	if err != nil {
		return nil, err
	}
	for _, root := range roots {
		library, id, librarySource, err := loadLibrary(root)
		if err != nil {
			return nil, err
		}
		if err := mergeLoadedLibraryWithOverlay(p, overlay, library, id, librarySource); err != nil {
			return nil, err
		}
	}
	p.source = overlay
	return p, resolveAndValidatePerson(p)
}

func mergeLoadedLibrary(p *Person, library *Person, id string, source fs.FS) error {
	overlay, err := definitionOverlay(p.source, p.Personalities, p.Boundaries)
	if err != nil {
		return err
	}
	if err := mergeLoadedLibraryWithOverlay(p, overlay, library, id, source); err != nil {
		return err
	}
	p.source = overlay
	return nil
}

func mergeLoadedLibraryWithOverlay(p *Person, overlay fstest.MapFS, library *Person, id string, librarySource fs.FS) error {
	if p.Libraries == nil {
		p.Libraries = map[string]string{p.localSourceID(): "profile-local"}
	}
	if p.PersonalityLibraries == nil {
		p.PersonalityLibraries = map[string]string{}
	}
	if _, exists := p.Libraries[id]; exists {
		return fmt.Errorf("personality library %q is admitted more than once", id)
	}
	for _, name := range library.personalityOrder() {
		binding := library.Personalities[name]
		if existing, exists := p.Personalities[name]; exists {
			if !equalPersonality(existing, binding) {
				return fmt.Errorf("personality %q conflicts between profile libraries", name)
			}
			continue
		}
		for otherName, other := range p.Personalities {
			if other.Skill == binding.Skill && otherName != name {
				return fmt.Errorf("personality skill %q conflicts between %q and %q", binding.Skill, otherName, name)
			}
		}
		p.Personalities[name] = binding
		p.PersonalityLibraries[name] = id
		p.PersonalityOrder = append(p.PersonalityOrder, name)
	}
	for _, name := range library.boundaryOrder() {
		binding := library.Boundaries[name]
		if existing, exists := p.Boundaries[name]; exists {
			if existing.Skill != binding.Skill || existing.Summary != binding.Summary {
				return fmt.Errorf("boundary %q conflicts between profile libraries", name)
			}
			continue
		}
		for otherName, other := range p.Boundaries {
			if other.Skill == binding.Skill && otherName != name {
				return fmt.Errorf("boundary skill %q conflicts between %q and %q", binding.Skill, otherName, name)
			}
		}
		p.Boundaries[name] = binding
		p.BoundaryOrder = append(p.BoundaryOrder, name)
		if raw, ok := library.boundarySkills[name]; ok {
			p.boundarySkills[name] = append([]byte(nil), raw...)
		}
	}
	if err := appendDefinitions(overlay, librarySource, library.Personalities, library.Boundaries); err != nil {
		return err
	}
	p.Libraries[id] = "admitted-local"
	return nil
}

// ResolveFavoriteColors derives every role's favorite together. Load applies
// it; a hand-built Person must call it before rendering any role color.
func (p *Person) ResolveFavoriteColors() error {
	order := p.roleOrder()
	groups := make([][]string, 0, len(order))
	for _, roleName := range order {
		components := make([]string, 0, len(p.Roles[roleName].Personalities))
		for _, name := range p.Roles[roleName].Personalities {
			binding, ok := p.Personalities[name]
			if !ok {
				return fmt.Errorf("role %q: personality %q has no catalog binding", roleName, name)
			}
			components = append(components, binding.Color)
		}
		if len(components) == 0 {
			return fmt.Errorf("role %q has no personalities to derive a favorite color from", roleName)
		}
		groups = append(groups, components)
	}
	favorites, err := color.Favorites(groups)
	if err != nil {
		return fmt.Errorf("derive role favorite colors: %w", err)
	}
	for index, roleName := range order {
		role := p.Roles[roleName]
		role.FavoriteColor = favorites[index]
		p.Roles[roleName] = role
	}
	return nil
}

// ResolveBackgrounds derives one window background per role from the resolved
// accents, solved across the whole roster. See internal/palette/role-palette.txt.
func (p *Person) ResolveBackgrounds() error {
	order := p.roleOrder()
	accents := make([]string, 0, len(order))
	for _, roleName := range order {
		accent := p.Roles[roleName].FavoriteColor
		if accent == "" {
			return fmt.Errorf("role %q has no favorite color to derive a background from", roleName)
		}
		accents = append(accents, accent)
	}
	backgrounds, err := color.Backgrounds(accents)
	if err != nil {
		return fmt.Errorf("derive role backgrounds: %w", err)
	}
	for index, roleName := range order {
		role := p.Roles[roleName]
		role.Background = backgrounds[index]
		p.Roles[roleName] = role
	}
	return nil
}

func resolveAndValidatePerson(p *Person) error {
	if err := p.ResolveFavoriteColors(); err != nil {
		return err
	}
	if err := p.ResolveBackgrounds(); err != nil {
		return err
	}
	for _, roleName := range p.roleOrder() {
		for _, personalityName := range p.Roles[roleName].Personalities {
			if _, ok := p.Personalities[personalityName]; !ok {
				return fmt.Errorf("role %q: personality %q has no catalog binding", roleName, personalityName)
			}
		}
		for _, boundaryName := range p.Roles[roleName].Boundaries {
			if _, ok := p.Boundaries[boundaryName]; !ok {
				return fmt.Errorf("role %q: boundary %q has no catalog binding", roleName, boundaryName)
			}
		}
	}
	return nil
}

// validateNoUnusedPersonalities derives garbage collection from the effective
// Core Roster graph without maintaining a second inventory.
func validateNoUnusedPersonalities(p *Person) error {
	used := map[string]bool{}
	for _, roleName := range p.RoleOrder {
		for _, personalityName := range p.Roles[roleName].Personalities {
			used[personalityName] = true
		}
	}
	var unused []string
	for _, personalityName := range p.personalityOrder() {
		if !used[personalityName] {
			unused = append(unused, personalityName)
		}
	}
	if len(unused) != 0 {
		return fmt.Errorf("core roster has unused personalities: %s", strings.Join(unused, ", "))
	}
	return nil
}

// validateNoUnusedBoundaries keeps shared doctrine anchored to at least one role, so
// a boundary cannot linger after the last role drops its reference.
func validateNoUnusedBoundaries(p *Person) error {
	used := map[string]bool{}
	for _, roleName := range p.RoleOrder {
		for _, boundaryName := range p.Roles[roleName].Boundaries {
			used[boundaryName] = true
		}
		for _, scoped := range p.Roles[roleName].ScopedBoundaries {
			used[scoped.Name] = true
		}
	}
	var unused []string
	for _, boundaryName := range p.boundaryOrder() {
		if !used[boundaryName] {
			unused = append(unused, boundaryName)
		}
	}
	if len(unused) != 0 {
		return fmt.Errorf("core roster has unused boundaries: %s", strings.Join(unused, ", "))
	}
	return nil
}

// validateBoundaryBodySides requires both sides and bounds each separately, so
// one long half cannot crowd out the other.
func validateBoundaryBodySides(boundaryName, body string) error {
	own := strings.Index(body, boundaryOwnHeading)
	defer_ := strings.Index(body, boundaryDeferHeading)
	if own < 0 || defer_ < 0 {
		return fmt.Errorf(
			"boundary %q skill body needs both %q and %q sections",
			boundaryName, boundaryOwnHeading, boundaryDeferHeading,
		)
	}
	if own > defer_ {
		return fmt.Errorf("boundary %q skill body states the defer side before the own side", boundaryName)
	}
	sections := map[string]string{"own": body[own:defer_], "defer": body[defer_:]}
	// The scoped side stays optional so a boundary nobody scopes needs no
	// third section, which keeps packages authored before the axis loading.
	if scoped := strings.Index(body, boundaryScopedHeading); scoped >= 0 {
		if scoped < own || scoped > defer_ {
			return fmt.Errorf(
				"boundary %q skill body states the scoped side outside own..defer order", boundaryName,
			)
		}
		sections["own"] = body[own:scoped]
		sections["scoped"] = body[scoped:defer_]
	}
	for label, section := range sections {
		_, prose, _ := strings.Cut(section, "\n")
		words := roleSkillBodyWordCount(prose)
		if words > maxBoundarySkillBodyWords {
			return fmt.Errorf(
				"boundary %q %s side has %d words, maximum is %d",
				boundaryName, label, words, maxBoundarySkillBodyWords,
			)
		}
	}
	return nil
}

// validateBoundaryOwners keeps a two-sided boundary coherent. An owner receives
// the body by owning it, never by declaring it.
func validateBoundaryOwners(p *Person) error {
	for _, boundaryName := range p.boundaryOrder() {
		owner := p.Boundaries[boundaryName].Owner
		if owner == "" {
			return fmt.Errorf("boundary %q has no owner", boundaryName)
		}
		role, ok := p.Roles[owner]
		if !ok {
			return fmt.Errorf("boundary %q names unknown owner %q", boundaryName, owner)
		}
		for _, declared := range role.Boundaries {
			if declared == boundaryName {
				return fmt.Errorf(
					"boundary %q owner %q also declares it",
					boundaryName,
					owner,
				)
			}
		}
		for _, scoped := range role.ScopedBoundaries {
			if scoped.Name == boundaryName {
				return fmt.Errorf(
					"boundary %q owner %q also scopes it",
					boundaryName,
					owner,
				)
			}
		}
	}
	return nil
}

// validateScopedBoundaryBodies requires the scoped side exactly where a role
// scopes the boundary, so a grant never arrives without its instructions.
func validateScopedBoundaryBodies(p *Person) error {
	for _, roleName := range p.roleOrder() {
		for _, scoped := range p.Roles[roleName].ScopedBoundaries {
			binding, defined := p.Boundaries[scoped.Name]
			if !defined {
				return fmt.Errorf("role %q scopes unknown boundary %q", roleName, scoped.Name)
			}
			raw, ok := p.BoundarySkillDefinition(scoped.Name)
			if !ok {
				continue
			}
			body, err := skillBody(raw)
			if err != nil {
				return err
			}
			if !strings.Contains(body, boundaryScopedHeading) {
				return fmt.Errorf(
					"role %q scopes boundary %q, whose skill %q has no %q section",
					roleName, scoped.Name, binding.Skill, boundaryScopedHeading,
				)
			}
		}
	}
	return nil
}

// validateActCoverage follows validateRoleAdjacents: optional until one
// attribute declares, then required everywhere. docs/kdl-contracts.md.
func validateActCoverage(p *Person) error {
	declared := false
	for _, roleName := range p.roleOrder() {
		if len(p.Roles[roleName].Acts) > 0 {
			declared = true
		}
	}
	for _, name := range p.PersonalityOrder {
		if len(p.Personalities[name].Acts) > 0 {
			declared = true
		}
	}
	for _, name := range p.BoundaryOrder {
		if len(p.Boundaries[name].Acts) > 0 {
			declared = true
		}
	}
	if !declared {
		return nil
	}
	for _, roleName := range p.roleOrder() {
		if err := validateActs("role "+roleName, p.Roles[roleName].Acts, false); err != nil {
			return err
		}
	}
	for _, name := range p.PersonalityOrder {
		if err := validateActs("personality "+name, p.Personalities[name].Acts, false); err != nil {
			return err
		}
	}
	for _, name := range p.BoundaryOrder {
		if err := validateActs("boundary "+name, p.Boundaries[name].Acts, true); err != nil {
			return err
		}
	}
	return nil
}

// validateRoleAdjacents keeps the graph complete. Adjacency is deliberately
// directed, so do not add a symmetry check. See docs/role-boundaries.md.
func validateRoleAdjacents(p *Person) error {
	declared := false
	for _, roleName := range p.roleOrder() {
		if len(p.Roles[roleName].Adjacents) > 0 {
			declared = true
			break
		}
	}
	if !declared {
		return nil
	}
	for _, roleName := range p.roleOrder() {
		adjacents := p.Roles[roleName].Adjacents
		if len(adjacents) != adjacentsPerRole {
			return fmt.Errorf(
				"role %q declares %d adjacent roles, needs exactly %d",
				roleName,
				len(adjacents),
				adjacentsPerRole,
			)
		}
		for _, adjacent := range adjacents {
			if _, ok := p.Roles[adjacent.Role]; !ok {
				return fmt.Errorf(
					"role %q names unknown adjacent role %q", roleName, adjacent.Role,
				)
			}
		}
	}
	return nil
}

// Identical melds derive barely 0.06 apart, and the roster measures 0.1590, so
// two-component blends leave this floor slack. See docs/personality.md.
const minFavoriteSeparation = 0.08

// The background floor is lower because the whole set sits at one lightness
// and chroma, so hue is the only axis it has.
const minBackgroundSeparation = 0.030

// personalitiesPerRole is one signature trait plus one bond shared with a
// sibling seat. See docs/personality.md.
const personalitiesPerRole = 2

func validateCorePersonalityMelds(p *Person) error {
	usage := map[string]int{}
	colors := map[string]string{}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		if len(role.Personalities) != personalitiesPerRole {
			return fmt.Errorf(
				"core role %q has %d personalities, want exactly %d",
				roleName,
				len(role.Personalities),
				personalitiesPerRole,
			)
		}
		for _, name := range role.Personalities {
			usage[name]++
		}
		favorite := role.FavoriteColor
		if err := color.Legible(favorite); err != nil {
			return fmt.Errorf("core role %q favorite color: %w", roleName, err)
		}
		if existing, ok := colors[favorite]; ok {
			return fmt.Errorf(
				"core roles %q and %q share melded favorite color %q",
				existing,
				roleName,
				favorite,
			)
		}
		colors[favorite] = roleName
	}
	favorites := make([]string, 0, len(p.RoleOrder))
	for _, roleName := range p.RoleOrder {
		favorites = append(favorites, p.Roles[roleName].FavoriteColor)
	}
	separation, err := color.MinSeparation(favorites)
	if err != nil {
		return fmt.Errorf("core role favorite colors: %w", err)
	}
	if separation < minFavoriteSeparation {
		return fmt.Errorf(
			"core role favorite colors are %.4f apart at their closest, want at least %.4f: "+
				"two roles meld nearly the same personalities",
			separation,
			minFavoriteSeparation,
		)
	}
	backgrounds := make([]string, 0, len(p.RoleOrder))
	for _, roleName := range p.RoleOrder {
		backgrounds = append(backgrounds, p.Roles[roleName].Background)
	}
	backgroundSeparation, err := color.MinSeparation(backgrounds)
	if err != nil {
		return fmt.Errorf("core role backgrounds: %w", err)
	}
	if backgroundSeparation < minBackgroundSeparation {
		return fmt.Errorf(
			"core role backgrounds are %.4f apart at their closest, want at least %.4f: "+
				"the roster has outgrown the hue circle at this lightness",
			backgroundSeparation,
			minBackgroundSeparation,
		)
	}
	for name, count := range usage {
		if count > 3 {
			return fmt.Errorf(
				"core personality %q appears in %d roles, want at most three",
				name,
				count,
			)
		}
	}
	return nil
}

func loadLibrary(root string) (*Person, string, fs.FS, error) {
	if IsCoreLibrary(root) {
		source, err := coreLibrarySource()
		if err != nil {
			return nil, "", nil, err
		}
		library, id, err := loadLibrarySource(source, coreLibraryLabel)
		if err != nil {
			return nil, "", nil, err
		}
		return library, id, source, nil
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, "", nil, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, "", nil, fmt.Errorf("inspect personality library %s: %w", absolute, err)
	}
	if !info.IsDir() {
		return nil, "", nil, fmt.Errorf("personality library %s is not a directory", absolute)
	}
	if err := filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("personality library contains symlink %s", path)
		}
		return nil
	}); err != nil {
		return nil, "", nil, fmt.Errorf("inspect personality library tree %s: %w", absolute, err)
	}
	source := os.DirFS(absolute)
	library, id, err := loadLibrarySource(source, "personality library "+absolute)
	if err != nil {
		return nil, "", nil, err
	}
	return library, id, source, nil
}

func loadLibrarySource(source fs.FS, label string) (*Person, string, error) {
	raw, err := fs.ReadFile(source, "library"+yamlFragmentExt)
	if err != nil {
		return nil, "", fmt.Errorf("%s: read manifest: %w", label, err)
	}
	node, id, err := yamlManifestName(raw)
	if err != nil || node != "library" {
		return nil, "", fmt.Errorf("%s: needs one library name", label)
	}
	if !validLogicalID(id) {
		return nil, "", fmt.Errorf("%s: library name %q is not a stable logical id", label, id)
	}
	library, err := decodeLibrarySource(source, id)
	if err != nil {
		return nil, "", err
	}
	// A library may publish shared doctrine, so its boundary bodies are read and
	// bounded here rather than only in the consuming package.
	if err := loadGuardrailSkills(source, library); err != nil {
		return nil, "", fmt.Errorf("personality library %q: %w", id, err)
	}
	if err := loadBoundarySkills(source, library); err != nil {
		return nil, "", fmt.Errorf("personality library %q: %w", id, err)
	}
	return library, id, nil
}

func equalPersonality(left, right Personality) bool {
	// %#v renders a pointer field as its address, so two structurally identical
	// copies parsed separately would compare unequal once Voice arrived.
	return reflect.DeepEqual(left, right)
}

func definitionOverlay(
	base fs.FS, personalities map[string]Personality, boundaries map[string]Boundary,
) (fstest.MapFS, error) {
	overlay := fstest.MapFS{}
	if err := appendDefinitions(overlay, base, personalities, boundaries); err != nil {
		return nil, err
	}
	return overlay, nil
}

func appendDefinitions(
	overlay fstest.MapFS,
	source fs.FS,
	personalities map[string]Personality,
	boundaries map[string]Boundary,
) error {
	for _, binding := range personalities {
		path := "definitions/skills/" + binding.Skill + "/SKILL.md"
		raw, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("personality skill %q: read definition: %w", binding.Skill, err)
		}
		if existing, ok := overlay[path]; ok && !bytes.Equal(existing.Data, raw) {
			return fmt.Errorf("personality definition %q conflicts between local libraries", binding.Skill)
		}
		overlay[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
	}
	for _, binding := range boundaries {
		path := "definitions/skills/" + binding.Skill + "/SKILL.md"
		raw, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("boundary skill %q: read definition: %w", binding.Skill, err)
		}
		if existing, ok := overlay[path]; ok && !bytes.Equal(existing.Data, raw) {
			return fmt.Errorf("boundary definition %q conflicts between local libraries", binding.Skill)
		}
		overlay[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
	}
	if _, exists := overlay["definitions/INVARIANT.md"]; !exists {
		raw, err := fs.ReadFile(source, "definitions/INVARIANT.md")
		if err != nil {
			return fmt.Errorf("read profile invariant: %w", err)
		}
		overlay["definitions/INVARIANT.md"] = &fstest.MapFile{Data: raw, Mode: 0o644}
	}
	return nil
}

// validatePersonalityBodies bounds each personality body. External libraries
// supply their own definitions, so a missing one is not an error here.
func validatePersonalityBodies(source fs.FS, p *Person) error {
	for name, binding := range p.Personalities {
		raw, err := fs.ReadFile(source, "definitions/skills/"+binding.Skill+"/SKILL.md")
		if err != nil {
			continue
		}
		body, err := skillBody(raw)
		if err != nil {
			return fmt.Errorf("personality %q skill %q: %w", name, binding.Skill, err)
		}
		words := roleSkillBodyWordCount(body)
		if words > maxPersonalitySkillBodyWords {
			return fmt.Errorf(
				"personality %q skill body has %d words, maximum is %d",
				name, words, maxPersonalitySkillBodyWords,
			)
		}
	}
	return nil
}

// validateRosterProseFloors keeps a shipped entry from thinning into a label.
// The ceilings bind every package, the floors bind the roster this repo ships.
func validateRosterProseFloors(source fs.FS, p *Person) error {
	for _, roleName := range p.roleOrder() {
		words := roleSkillBodyWordCount(p.Roles[roleName].Briefing)
		if words < minRoleSkillBodyWords {
			return fmt.Errorf("role %q body has %d words, minimum is %d", roleName, words, minRoleSkillBodyWords)
		}
	}
	for name, binding := range p.Personalities {
		raw, err := fs.ReadFile(source, "definitions/skills/"+binding.Skill+"/SKILL.md")
		if err != nil {
			continue
		}
		body, err := skillBody(raw)
		if err != nil {
			return err
		}
		if words := roleSkillBodyWordCount(body); words < minPersonalitySkillBodyWords {
			return fmt.Errorf("personality %q body has %d words, minimum is %d", name, words, minPersonalitySkillBodyWords)
		}
	}
	for _, boundaryName := range p.boundaryOrder() {
		raw, ok := p.BoundarySkillDefinition(boundaryName)
		if !ok {
			continue
		}
		body, err := skillBody(raw)
		if err != nil {
			return err
		}
		own := strings.Index(body, boundaryOwnHeading)
		defer_ := strings.Index(body, boundaryDeferHeading)
		if own < 0 || defer_ < 0 {
			continue
		}
		sections := map[string]string{"own": body[own:defer_], "defer": body[defer_:]}
		if scoped := strings.Index(body, boundaryScopedHeading); scoped >= 0 {
			sections["own"] = body[own:scoped]
			sections["scoped"] = body[scoped:defer_]
		}
		for label, section := range sections {
			_, prose, _ := strings.Cut(section, "\n")
			if words := roleSkillBodyWordCount(prose); words < minBoundarySideWords {
				return fmt.Errorf(
					"boundary %q %s side has %d words, minimum is %d",
					boundaryName, label, words, minBoundarySideWords,
				)
			}
		}
	}
	return nil
}

func loadSource(source fs.FS, label string) (*Person, error) {
	source, _, err := dataLayout(source, label)
	if err != nil {
		return nil, err
	}
	p, err := decodePersonSource(source, label)
	if err != nil {
		return nil, err
	}
	if err := loadRoleSkills(source, p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := loadRoleMethods(source, p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := loadGuardrailSkills(source, p); err != nil {
		return nil, err
	}
	if err := loadBoundarySkills(source, p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := validatePersonalityBodies(source, p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := addCopyContractProvenance(p); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	p.source = source
	return p, nil
}

func addCopyContractProvenance(p *Person) error {
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		if role.CopyContract == nil {
			continue
		}
		canonical := struct {
			Scope string     `json:"scope"`
			Rules []CopyRule `json:"rules"`
		}{
			Scope: role.CopyContract.Scope,
			Rules: role.CopyContract.Rules,
		}
		raw, err := json.Marshal(canonical)
		if err != nil {
			return fmt.Errorf("role %q copy contract: %w", roleName, err)
		}
		digest := sha256.Sum256(raw)
		role.CopyContract.Source = p.ProviderID() + ":role:" + roleName + ":copy-contract"
		role.CopyContract.Digest = fmt.Sprintf("sha256:%x", digest)
		p.Roles[roleName] = role
	}
	return nil
}

func loadRoleSkills(source fs.FS, p *Person) error {
	p.roleSkills = map[string][]byte{}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		if role.Briefing != "" {
			raw := []byte(fmt.Sprintf(
				"---\nname: %s\ndescription: Adopt the %s charter. Use when the session activates the %s role.\n---\n\n# %s\n\n%s\n",
				role.Skill,
				roleName,
				roleName,
				roleName,
				role.Briefing,
			))
			p.roleSkills[roleName] = raw
			role.SkillSource = p.ProviderID() + ":legacy-role:" + roleName
			digest := sha256.Sum256(raw)
			role.SkillDigest = fmt.Sprintf("sha256:%x", digest)
			p.Roles[roleName] = role
			continue
		}
		if role.Skill != "role-"+roleName {
			return fmt.Errorf("role %q skill %q must be role-%s", roleName, role.Skill, roleName)
		}
		raw, err := fs.ReadFile(source, "roles/"+roleName+"/SKILL.md")
		if err != nil {
			return fmt.Errorf("role %q skill %q: %w", roleName, role.Skill, err)
		}
		if err := validateSkillDefinition(role.Skill, raw); err != nil {
			return err
		}
		body, err := skillBody(raw)
		if err != nil {
			return fmt.Errorf("role %q skill %q: %w", roleName, role.Skill, err)
		}
		if words := roleSkillBodyWordCount(body); words > maxRoleSkillBodyWords {
			return fmt.Errorf(
				"role %q skill body has %d words, maximum is %d",
				roleName,
				words,
				maxRoleSkillBodyWords,
			)
		}
		if paragraphs := briefingParagraphCount(body); paragraphs < 3 {
			return fmt.Errorf("role %q skill needs at least three paragraphs, got %d", roleName, paragraphs)
		}
		role.Briefing = body
		role.SkillSource = p.ProviderID() + ":role:" + roleName
		digest := sha256.Sum256(raw)
		role.SkillDigest = fmt.Sprintf("sha256:%x", digest)
		p.roleSkills[roleName] = append([]byte(nil), raw...)
		p.Roles[roleName] = role
	}
	return nil
}

// loadBoundarySkills reads each shared doctrine body and bounds it against the boundary
// budget. The body never enters Role.Briefing, so it spends no role budget.
func loadBoundarySkills(source fs.FS, p *Person) error {
	p.boundarySkills = map[string][]byte{}
	for _, boundaryName := range p.BoundaryOrder {
		boundary := p.Boundaries[boundaryName]
		path := "definitions/skills/" + boundary.Skill + "/SKILL.md"
		raw, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("boundary %q skill %q: read definition: %w", boundaryName, boundary.Skill, err)
		}
		if err := validateSkillDefinition(boundary.Skill, raw); err != nil {
			return err
		}
		body, err := skillBody(raw)
		if err != nil {
			return fmt.Errorf("boundary %q skill %q: %w", boundaryName, boundary.Skill, err)
		}
		if err := validateBoundaryBodySides(boundaryName, body); err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		boundary.Source = p.ProviderID() + ":boundary:" + boundaryName
		boundary.Digest = fmt.Sprintf("sha256:%x", digest)
		p.Boundaries[boundaryName] = boundary
		p.boundarySkills[boundaryName] = append([]byte(nil), raw...)
	}
	return nil
}

// loadGuardrailSkills reads each guardrail body. The card half is eager and
// lives on the element, so only the procedure is read here.
func loadGuardrailSkills(source fs.FS, p *Person) error {
	p.guardrailSkills = map[string][]byte{}
	for _, name := range p.GuardrailOrder {
		rail := p.Guardrails[name]
		path := "definitions/skills/" + rail.Skill + "/SKILL.md"
		raw, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("guardrail %q skill %q: read definition: %w", name, rail.Skill, err)
		}
		if err := validateSkillDefinition(rail.Skill, raw); err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		rail.Source = p.ProviderID() + ":guardrail:" + name
		rail.Digest = fmt.Sprintf("sha256:%x", digest)
		p.Guardrails[name] = rail
		p.guardrailSkills[name] = append([]byte(nil), raw...)
	}
	return nil
}

// GuardrailSkillDefinition returns the raw procedure skill for one guardrail.
func (p *Person) GuardrailSkillDefinition(name string) ([]byte, bool) {
	raw, ok := p.guardrailSkills[name]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), raw...), true
}

// BoundarySkillDefinition returns the raw shared doctrine skill for one boundary.
func (p *Person) BoundarySkillDefinition(boundaryName string) ([]byte, bool) {
	raw, ok := p.boundarySkills[boundaryName]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), raw...), true
}

// RoleOwnedBoundaries returns the boundaries this role owns. An owner receives
// the body without declaring it. See docs/ownership.md.
func (p *Person) RoleOwnedBoundaries(roleName string) []string {
	owned := make([]string, 0, 1)
	for _, name := range p.boundaryOrder() {
		if p.Boundaries[name].Owner == roleName {
			owned = append(owned, name)
		}
	}
	return owned
}

// RoleActiveBoundaries returns every boundary whose body this role receives,
// declared first and then owned.
func (p *Person) RoleActiveBoundaries(roleName string) []string {
	role, ok := p.Roles[roleName]
	if !ok {
		return nil
	}
	active := append([]string(nil), role.Boundaries...)
	for _, scoped := range role.ScopedBoundaries {
		active = append(active, scoped.Name)
	}
	return append(active, p.RoleOwnedBoundaries(roleName)...)
}

// RoleBoundarySkillIDs returns the boundary skill ids one role activates, in
// declared then owned order, so callers can name them without loading bodies.
func (p *Person) RoleBoundarySkillIDs(roleName string) []string {
	if _, ok := p.Roles[roleName]; !ok {
		return nil
	}
	active := p.RoleActiveBoundaries(roleName)
	ids := make([]string, 0, len(active))
	for _, boundaryName := range active {
		binding, exists := p.Boundaries[boundaryName]
		if !exists {
			continue
		}
		ids = append(ids, binding.Skill)
	}
	return ids
}

func loadRoleMethods(source fs.FS, p *Person) error {
	p.roleMethods = map[string]map[string][]byte{}
	claimed := map[string]string{}
	for _, roleName := range p.RoleOrder {
		role := p.Roles[roleName]
		methodsRoot := "roles/" + roleName + "/skills"
		if len(role.Methods) == 0 {
			if _, err := fs.Stat(source, methodsRoot); err == nil {
				return fmt.Errorf("role %q has an undeclared skills directory", roleName)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("role %q skills: %w", roleName, err)
			}
			continue
		}

		expected := make(map[string]bool, len(role.Methods))
		for _, method := range role.Methods {
			if owner, duplicate := claimed[method]; duplicate {
				return fmt.Errorf("roles %q and %q bind the same method skill %q", owner, roleName, method)
			}
			claimed[method] = roleName
			expected[method] = true
		}
		entries, err := fs.ReadDir(source, methodsRoot)
		if err != nil {
			return fmt.Errorf("role %q method skills: %w", roleName, err)
		}
		if len(entries) != len(expected) {
			return fmt.Errorf(
				"role %q binds %d method skills but its directory contains %d entries",
				roleName,
				len(expected),
				len(entries),
			)
		}
		p.roleMethods[roleName] = map[string][]byte{}
		for _, entry := range entries {
			method := entry.Name()
			if !entry.IsDir() || !expected[method] {
				return fmt.Errorf("role %q method skills has unexpected entry %q", roleName, method)
			}
			methodRoot := methodsRoot + "/" + method
			methodEntries, err := fs.ReadDir(source, methodRoot)
			if err != nil {
				return fmt.Errorf("role %q method skill %q: %w", roleName, method, err)
			}
			if len(methodEntries) != 1 || methodEntries[0].IsDir() || methodEntries[0].Name() != "SKILL.md" {
				return fmt.Errorf("role %q method skill %q must contain only SKILL.md", roleName, method)
			}
			raw, err := fs.ReadFile(source, methodRoot+"/SKILL.md")
			if err != nil {
				return fmt.Errorf("role %q method skill %q: %w", roleName, method, err)
			}
			if err := validateSkillDefinition(method, raw); err != nil {
				return err
			}
			p.roleMethods[roleName][method] = append([]byte(nil), raw...)
		}
	}
	return nil
}

func skillBody(raw []byte) (string, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", fmt.Errorf("missing YAML frontmatter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return "", fmt.Errorf("unterminated YAML frontmatter")
	}
	return strings.TrimSpace(text[end+9:]), nil
}

// RoleSkillDefinition returns the canonical role skill selected by the profile.
func (p *Person) RoleSkillDefinition(roleName string) ([]byte, bool) {
	raw, ok := p.roleSkills[roleName]
	if !ok {
		role, exists := p.Roles[roleName]
		if !exists || strings.TrimSpace(role.Briefing) == "" {
			return nil, false
		}
		skill := role.Skill
		if skill == "" {
			skill = "role-" + roleName
		}
		raw = []byte(fmt.Sprintf(
			"---\nname: %s\ndescription: Adopt the %s charter. Use when the session activates the %s role.\n---\n\n# %s\n\n%s\n",
			skill,
			roleName,
			roleName,
			roleName,
			role.Briefing,
		))
		ok = true
	}
	return append([]byte(nil), raw...), ok
}

// RoleSkillID returns the stable skill binding, including the v1.x legacy
// adapter for callers that construct an inline role in memory.
func (p *Person) RoleSkillID(roleName string) string {
	if role, ok := p.Roles[roleName]; ok && role.Skill != "" {
		return role.Skill
	}
	return "role-" + roleName
}

// RoleMethodDefinition returns one role-bound progressive-disclosure skill.
func (p *Person) RoleMethodDefinition(roleName, method string) ([]byte, bool) {
	methods, ok := p.roleMethods[roleName]
	if !ok {
		return nil, false
	}
	raw, ok := methods[method]
	return append([]byte(nil), raw...), ok
}

func pRoleSkillDirectory(name string) (string, bool) {
	if !validSemanticToken(name) {
		return "", false
	}
	return name, true
}

func personFragmentSlug(name string) (string, bool) {
	extension, ok := fragmentExtension(name)
	if !ok ||
		len(name) < len("00-a")+len(extension) ||
		name[0] < '0' || name[0] > '9' ||
		name[1] < '0' || name[1] > '9' ||
		name[2] != '-' {
		return "", false
	}
	return strings.TrimSuffix(name[3:], extension), true
}

// A package authors each fragment as KDL or YAML, and may hold both while it
// converts. See docs/person-packages.md.
func fragmentExtension(name string) (string, bool) {
	for _, extension := range []string{yamlFragmentExt} {
		if strings.HasSuffix(name, extension) {
			return extension, true
		}
	}
	return "", false
}

func isFragmentFile(name string) bool {
	_, ok := fragmentExtension(name)
	return ok
}

// fragmentKDL trims one KDL fragment for the assembler. A YAML package never
// reaches here: decodePersonSource routes it to the native decoder instead.

// readManifest accepts either manifest spelling, preferring YAML so a converted
// package is not shadowed by a stale KDL manifest beside it.
func readManifest(source fs.FS, stem string) ([]byte, error) {
	raw, err := fs.ReadFile(source, stem+yamlFragmentExt)
	if err != nil {
		return nil, err
	}
	node, name, err := yamlManifestName(raw)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%s %q", node, name)), nil
}

// decodeLibrarySource builds a library the same way, from its own two sections.
func decodeLibrarySource(source fs.FS, id string) (*Person, error) {
	label := "personality library " + id
	files, err := readSectionFiles(source, label, librarySections)
	if err != nil {
		return nil, err
	}
	isYAML, err := yamlSectionFiles(label, files)
	if err != nil {
		return nil, err
	}
	if !isYAML {
		return nil, fmt.Errorf("%s: fragments must be YAML", label)
	}
	return buildYAMLPerson("person", id, files, label)
}

// decodePersonSource builds the person from whichever format the package uses.
func decodePersonSource(source fs.FS, label string) (*Person, error) {
	files, err := readSectionFiles(source, label, personSections)
	if err != nil {
		return nil, err
	}
	isYAML, err := yamlSectionFiles(label, files)
	if err != nil {
		return nil, err
	}
	if !isYAML {
		return nil, fmt.Errorf("%s: person fragments must be YAML", label)
	}
	manifest, err := readManifest(source, "person")
	if err != nil {
		return nil, fmt.Errorf("%s: read person manifest: %w", label, err)
	}
	kind, name, err := manifestParts(manifest)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return buildYAMLPerson(kind, name, files, label)
}

// manifestParts reads the node and name back out of the normalized manifest.
func manifestParts(manifest []byte) (string, string, error) {
	fields := strings.SplitN(strings.TrimSpace(string(manifest)), " ", 2)
	if len(fields) != 2 {
		return "", "", fmt.Errorf("manifest needs one name argument")
	}
	kind := fields[0]
	if kind != "person" && kind != "roster" {
		return "", "", fmt.Errorf("manifest needs exactly one person or roster node")
	}
	return kind, strings.Trim(fields[1], `"`), nil
}

// Source returns the selected person's content provider. The role catalog,
// invariant, and every bound personality definition come from the same package.
func Source(p *Person) (*schema.Source, error) {
	if p == nil {
		return nil, fmt.Errorf("person is required")
	}
	source := p.source
	strictDefinitions := true
	if source == nil {
		seed, label, err := rosterSeed()
		if err != nil {
			return nil, err
		}
		projected, _, err := dataLayout(seed, label)
		if err != nil {
			return nil, err
		}
		overlay, err := definitionOverlay(projected, p.Personalities, p.Boundaries)
		if err != nil {
			return nil, err
		}
		source = overlay
		strictDefinitions = false
	}
	definitions, err := fs.Sub(source, "definitions")
	if err != nil {
		return nil, fmt.Errorf("open person %q personality definitions: %w", p.Name, err)
	}
	files := fstest.MapFS{}
	if err := fs.WalkDir(definitions, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		raw, readErr := fs.ReadFile(definitions, path)
		if readErr != nil {
			return readErr
		}
		files[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("copy person %q definitions: %w", p.Name, err)
	}
	invariant, err := fs.ReadFile(files, "INVARIANT.md")
	if err != nil {
		return nil, fmt.Errorf("read person %q personality invariant: %w", p.Name, err)
	}
	if len(bytes.TrimSpace(invariant)) == 0 {
		return nil, fmt.Errorf("person %q personality invariant is empty", p.Name)
	}

	// The definitions tree carries personality, boundary and guardrail bodies
	// alike, so the canonical set spans all three catalogs.
	total := len(p.Personalities) + len(p.Boundaries) + len(p.Guardrails)
	expected := make(map[string]bool, total)
	canonicalSkills := make([]string, 0, total)
	for _, binding := range p.Personalities {
		expected[binding.Skill] = true
		canonicalSkills = append(canonicalSkills, binding.Skill)
	}
	boundarySkills := make(map[string]bool, len(p.Boundaries))
	for _, binding := range p.Boundaries {
		expected[binding.Skill] = true
		boundarySkills[binding.Skill] = true
		canonicalSkills = append(canonicalSkills, binding.Skill)
	}
	for _, binding := range p.Guardrails {
		expected[binding.Skill] = true
		canonicalSkills = append(canonicalSkills, binding.Skill)
	}
	sort.Strings(canonicalSkills)

	selected := make(map[string]bool, len(p.Personalities))
	for name, binding := range p.Personalities {
		if binding.Skill == "" {
			return nil, fmt.Errorf("personality %q has no skill binding", name)
		}
		if !expected[binding.Skill] {
			return nil, fmt.Errorf("personality %q binds unknown skill %q", name, binding.Skill)
		}
		if selected[binding.Skill] {
			return nil, fmt.Errorf("personality skill %q is bound more than once", binding.Skill)
		}
		selected[binding.Skill] = true
	}

	entries, err := fs.ReadDir(files, "skills")
	if err != nil {
		return nil, fmt.Errorf("read person %q personality skills: %w", p.Name, err)
	}
	if strictDefinitions && len(entries) != len(canonicalSkills) {
		return nil, fmt.Errorf(
			"person %q personality skills: catalog binds %d skills but definitions contain %d entries",
			p.Name,
			len(canonicalSkills),
			len(entries),
		)
	}

	src := &schema.Source{
		ID:    p.ProviderID(),
		Files: files,
		Instructions: []schema.ContentRef{{
			ID:   "personality-invariant",
			Path: "INVARIANT.md",
		}},
	}
	for _, entry := range entries {
		if !entry.IsDir() || strictDefinitions && !expected[entry.Name()] {
			return nil, fmt.Errorf("person %q personality skills: unexpected entry %q", p.Name, entry.Name())
		}
	}
	for _, skill := range canonicalSkills {
		skillPath := "skills/" + skill
		raw, err := fs.ReadFile(files, skillPath+"/SKILL.md")
		if err != nil {
			return nil, fmt.Errorf("read person %q skill %q: %w", p.Name, skill, err)
		}
		if err := validateSkillDefinition(skill, raw); err != nil {
			return nil, err
		}
		if !selected[skill] {
			continue
		}
		src.Skills = append(src.Skills, schema.ContentRef{
			ID:         skill,
			Path:       skillPath,
			EntryPoint: "SKILL.md",
		})
	}
	src.RoleSkills = map[string][]schema.ContentRef{}
	for _, roleName := range p.roleOrder() {
		roleSkill := p.RoleSkillID(roleName)
		raw, ok := p.RoleSkillDefinition(roleName)
		if !ok {
			return nil, fmt.Errorf("person %q role %q has no skill definition", p.Name, roleName)
		}
		path := "skills/" + roleSkill + "/SKILL.md"
		files[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
		src.RoleSkills[roleName] = []schema.ContentRef{{
			ID:         roleSkill,
			Path:       "skills/" + roleSkill,
			EntryPoint: "SKILL.md",
		}}
		for _, method := range p.Roles[roleName].Methods {
			raw, ok := p.RoleMethodDefinition(roleName, method)
			if !ok {
				return nil, fmt.Errorf("person %q role %q has no method skill %q", p.Name, roleName, method)
			}
			path := "skills/" + method + "/SKILL.md"
			if _, collision := files[path]; collision {
				return nil, fmt.Errorf("person %q role %q method skill %q collides with another person skill", p.Name, roleName, method)
			}
			files[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
			src.RoleSkills[roleName] = append(src.RoleSkills[roleName], schema.ContentRef{
				ID:         method,
				Path:       "skills/" + method,
				EntryPoint: "SKILL.md",
			})
		}
		for _, boundaryName := range p.RoleActiveBoundaries(roleName) {
			binding, ok := p.Boundaries[boundaryName]
			if !ok {
				return nil, fmt.Errorf("person %q role %q has no boundary binding %q", p.Name, roleName, boundaryName)
			}
			raw, ok := p.BoundarySkillDefinition(boundaryName)
			if !ok {
				return nil, fmt.Errorf("person %q role %q has no boundary skill %q", p.Name, roleName, boundaryName)
			}
			path := "skills/" + binding.Skill + "/SKILL.md"
			// Several roles share one boundary body, so an identical repeat is the
			// expected case and only a differing body is a real collision.
			if existing, collision := files[path]; collision && !bytes.Equal(existing.Data, raw) {
				return nil, fmt.Errorf(
					"person %q role %q boundary skill %q collides with another person skill",
					p.Name, roleName, binding.Skill,
				)
			}
			files[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
			src.RoleSkills[roleName] = append(src.RoleSkills[roleName], schema.ContentRef{
				ID:         binding.Skill,
				Path:       "skills/" + binding.Skill,
				EntryPoint: "SKILL.md",
			})
		}
		// A guardrail binds to exactly one role, so it rides here rather than in
		// the shared skill list, the same way a boundary body does.
		if rail := p.Roles[roleName].Guardrail; rail != "" {
			binding, ok := p.Guardrails[rail]
			if !ok {
				return nil, fmt.Errorf("person %q role %q has no guardrail binding %q", p.Name, roleName, rail)
			}
			raw, ok := p.GuardrailSkillDefinition(rail)
			if !ok {
				return nil, fmt.Errorf("person %q role %q has no guardrail skill %q", p.Name, roleName, rail)
			}
			path := "skills/" + binding.Skill + "/SKILL.md"
			if existing, collision := files[path]; collision && !bytes.Equal(existing.Data, raw) {
				return nil, fmt.Errorf(
					"person %q role %q guardrail skill %q collides with another person skill",
					p.Name, roleName, binding.Skill,
				)
			}
			files[path] = &fstest.MapFile{Data: raw, Mode: 0o644}
			src.RoleSkills[roleName] = append(src.RoleSkills[roleName], schema.ContentRef{
				ID:         binding.Skill,
				Path:       "skills/" + binding.Skill,
				EntryPoint: "SKILL.md",
			})
		}
	}
	return src, nil
}

func validateSkillDefinition(skill string, raw []byte) error {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Errorf("person skill %q: SKILL.md needs YAML frontmatter", skill)
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return fmt.Errorf("person skill %q: SKILL.md has unterminated frontmatter", skill)
	}
	end += 4
	frontmatter := "\n" + text[4:end] + "\n"
	if !strings.Contains(frontmatter, "\nname: "+skill+"\n") {
		return fmt.Errorf("person skill %q: frontmatter name does not match", skill)
	}
	if !strings.Contains(frontmatter, "\ndescription: ") {
		return fmt.Errorf("person skill %q: frontmatter needs a description", skill)
	}
	if strings.TrimSpace(text[end+5:]) == "" {
		return fmt.Errorf("person skill %q: SKILL.md body is empty", skill)
	}
	return nil
}

func briefingParagraphCount(briefing string) int {
	normalized := strings.ReplaceAll(briefing, "\r\n", "\n")
	count := 0
	for _, paragraph := range strings.Split(normalized, "\n\n") {
		if strings.TrimSpace(paragraph) != "" {
			count++
		}
	}
	return count
}

func roleSkillBodyWordCount(body string) int {
	content := strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	firstLine, remainder, found := strings.Cut(content, "\n")
	if found && strings.HasPrefix(strings.TrimSpace(firstLine), "# ") {
		content = remainder
	}
	return len(strings.Fields(content))
}

func validLogicalID(value string) bool {
	parts := strings.Split(value, ":")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !validSemanticToken(part) {
			return false
		}
	}
	return true
}

func validSemanticToken(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			continue
		}
		if char == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}

func validateIdentityCatalog(catalog map[string]Personality) error {
	emblems := map[string]string{}
	emojis := map[string]string{}
	motifs := map[string]string{}
	for name, personality := range catalog {
		for value, owners := range map[string]map[string]string{
			personality.Emblem.Emoji: emojis,
			personality.Motif:        motifs,
		} {
			if owner, duplicate := owners[value]; duplicate {
				return fmt.Errorf("personalities %q and %q share identity value %q", owner, name, value)
			}
			owners[value] = name
		}
		// Every name is a lookup key a human may say out loud, so the whole
		// list is unique across the roster rather than only the first entry.
		for _, value := range personality.Emblem.Names {
			if owner, duplicate := emblems[value]; duplicate {
				return fmt.Errorf("personalities %q and %q share identity value %q", owner, name, value)
			}
			emblems[value] = name
		}
	}
	return nil
}
