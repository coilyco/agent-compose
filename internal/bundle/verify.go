package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

type Identity struct {
	Source string
	Skill  string
}

// Verification is the stable, read-only result of checking a bundle.
type Verification struct {
	Manifest   *Manifest
	Files      int
	Identities []Identity
}

// Verify checks the consumer contract, rejecting unsafe or incomplete trees
// and identity content that disagrees with the retained selection trace.
func Verify(dir string) (*Verification, error) {
	files, err := verifyTree(dir)
	if err != nil {
		return nil, err
	}
	manifest, err := ReadManifest(dir)
	if err != nil {
		return nil, err
	}
	if err := verifyManifest(dir, manifest); err != nil {
		return nil, err
	}
	trace, err := ReadTrace(dir)
	if err != nil {
		return nil, err
	}
	if err := verifyTraceProfiles(trace, manifest); err != nil {
		return nil, err
	}
	if err := verifyRepositoryDecisions(trace, manifest); err != nil {
		return nil, err
	}
	identities, err := selectedIdentities(trace)
	if err != nil {
		return nil, err
	}
	for _, identity := range identities {
		if !slices.Contains(manifest.Sources, identity.Source) {
			return nil, fmt.Errorf("selected identity source %q is absent from manifest sources", identity.Source)
		}
	}
	if err := verifyIdentityTree(dir, identities); err != nil {
		return nil, err
	}
	if len(trace.Providers) > 0 {
		if err := verifyProviderReports(dir, trace, manifest, identities); err != nil {
			return nil, err
		}
	}
	return &Verification{
		Manifest:   manifest,
		Files:      files,
		Identities: identities,
	}, nil
}

func verifyProviderReports(
	dir string,
	trace *Trace,
	manifest *Manifest,
	identities []Identity,
) error {
	type contribution struct {
		skills int
		bytes  int64
	}
	expected := map[string]contribution{}
	for _, identity := range identities {
		root := filepath.Join(
			dir,
			"content",
			"skills",
			sourceSegment(identity.Source),
			identity.Skill,
		)
		var bytes int64
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			bytes += info.Size()
			return nil
		}); err != nil {
			return fmt.Errorf("measure provider %q skill %q: %w", identity.Source, identity.Skill, err)
		}
		current := expected[identity.Source]
		current.skills++
		current.bytes += bytes
		expected[identity.Source] = current
	}

	selectedSources := map[string]bool{}
	for _, source := range manifest.Sources {
		selectedSources[source] = true
	}
	seen := map[string]bool{}
	for _, provider := range trace.Providers {
		if provider.Source == "" || seen[provider.Source] {
			return fmt.Errorf("bundle trace has empty or duplicate provider %q", provider.Source)
		}
		seen[provider.Source] = true
		if provider.Reason == "" {
			return fmt.Errorf("bundle trace provider %q has no reason", provider.Source)
		}
		wantCategory := resolver.ProviderCategoryCatalogue
		switch provider.Scope {
		case schema.ProviderScopePerson:
			wantCategory = resolver.ProviderCategoryPerson
		case schema.ProviderScopeRole:
			wantCategory = resolver.ProviderCategoryRole
		case schema.ProviderScopeRequest, schema.ProviderScopeDefault,
			schema.ProviderScopeHarness, schema.ProviderScopeOrg:
		default:
			return fmt.Errorf("bundle trace provider %q has unknown scope %q", provider.Source, provider.Scope)
		}
		if provider.Category != wantCategory {
			return fmt.Errorf(
				"bundle trace provider %q category %q does not match scope %q",
				provider.Source,
				provider.Category,
				provider.Scope,
			)
		}
		if provider.ApproximateTokens != (provider.ContextBytes+3)/4 {
			return fmt.Errorf("bundle trace provider %q has an invalid token estimate", provider.Source)
		}
		switch provider.Outcome {
		case resolver.OutcomeSelected:
			if !selectedSources[provider.Source] {
				return fmt.Errorf("selected provider %q is absent from manifest sources", provider.Source)
			}
			contribution := expected[provider.Source]
			if provider.Skills != contribution.skills || provider.ContextBytes != contribution.bytes {
				return fmt.Errorf(
					"provider %q budget is %d skills and %d bytes, expected %d skills and %d bytes",
					provider.Source,
					provider.Skills,
					provider.ContextBytes,
					contribution.skills,
					contribution.bytes,
				)
			}
		case resolver.OutcomeExcluded:
			if selectedSources[provider.Source] {
				return fmt.Errorf("excluded provider %q appears in manifest sources", provider.Source)
			}
			if provider.Skills != 0 || provider.ContextBytes != 0 || provider.ApproximateTokens != 0 {
				return fmt.Errorf("excluded provider %q must contribute zero context", provider.Source)
			}
		default:
			return fmt.Errorf("bundle trace provider %q has invalid outcome %q", provider.Source, provider.Outcome)
		}
	}
	for source := range selectedSources {
		if !seen[source] {
			return fmt.Errorf("bundle trace omits selected provider %q", source)
		}
	}
	return nil
}

func verifyTree(dir string) (int, error) {
	files := 0
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("inspect bundle tree: %w", err)
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("resolve bundle entry %s: %w", path, err)
		}
		if rel != "." {
			if !safePortablePath(filepath.ToSlash(rel)) {
				return fmt.Errorf("bundle entry %s has a non-portable relative path", path)
			}
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect bundle entry %s: %w", path, err)
		}
		mode := info.Mode()
		switch {
		case mode&fs.ModeSymlink != 0:
			return fmt.Errorf("bundle entry %s is a symlink", path)
		case mode.IsDir():
			return nil
		case mode.IsRegular():
			files++
			return nil
		default:
			return fmt.Errorf("bundle entry %s is not a regular file or directory", path)
		}
	})
	if err != nil {
		return 0, err
	}
	return files, nil
}

func verifyManifest(dir string, manifest *Manifest) error {
	if manifest.Role == "" || manifest.RoleSkill == "" ||
		manifest.RoleSkillSource == "" || manifest.RoleSkillDigest == "" ||
		len(manifest.Personalities) == 0 {
		return fmt.Errorf("bundle manifest must name role skill provenance and personalities")
	}
	if manifest.RoleSkill != "role-"+manifest.Role {
		return fmt.Errorf("bundle manifest role skill %q does not match role %q", manifest.RoleSkill, manifest.Role)
	}
	if manifest.ModelTier == "" {
		manifest.ModelTier = schema.ModelTierFrontier
	} else if !schema.IsModelTier(manifest.ModelTier) {
		return fmt.Errorf("bundle manifest has unknown model tier %q", manifest.ModelTier)
	}
	seenPersonalities := map[string]bool{}
	for _, personality := range manifest.Personalities {
		if !safeSegment(personality) {
			return fmt.Errorf("bundle manifest contains unsafe personality %q", personality)
		}
		if seenPersonalities[personality] {
			return fmt.Errorf("bundle manifest repeats personality %q", personality)
		}
		seenPersonalities[personality] = true
	}
	if err := color.Legible(manifest.Color); err != nil {
		return fmt.Errorf("bundle manifest color: %w", err)
	}
	if len(manifest.Sources) == 0 {
		return fmt.Errorf("bundle manifest names no sources")
	}
	for _, source := range manifest.Sources {
		if source == "" {
			return fmt.Errorf("bundle manifest contains an empty source id")
		}
	}
	if len(manifest.Content) == 0 {
		return fmt.Errorf("bundle manifest names no logical content")
	}
	seenContent := map[string]bool{}
	for _, entry := range manifest.Content {
		if entry.ID == "" || seenContent[entry.ID] {
			return fmt.Errorf("bundle manifest has empty or duplicate logical content id %q", entry.ID)
		}
		if len(entry.Digest) != 71 || !strings.HasPrefix(entry.Digest, "sha256:") {
			return fmt.Errorf("bundle manifest logical content %q has invalid digest", entry.ID)
		}
		seenContent[entry.ID] = true
	}
	if err := requireRegularEntry(dir, "instructions", manifest.Delivery.Instructions); err != nil {
		return err
	}
	switch manifest.Delivery.Mode {
	case schema.DeliveryNativeSkills:
		if manifest.Delivery.CompiledContext != "" {
			return fmt.Errorf("native bundle unexpectedly names compiled_context")
		}
		return requireDirectoryEntry(dir, "skills_root", manifest.Delivery.SkillsRoot)
	case schema.DeliveryCompiled:
		if manifest.Delivery.SkillsRoot != "" {
			return fmt.Errorf("compiled bundle unexpectedly names skills_root")
		}
		return requireRegularEntry(dir, "compiled_context", manifest.Delivery.CompiledContext)
	default:
		return fmt.Errorf("bundle manifest has unknown delivery mode %q", manifest.Delivery.Mode)
	}
}

func selectedIdentities(trace *Trace) ([]Identity, error) {
	var identities []Identity
	for _, decision := range trace.Decisions {
		if decision.Kind != "skill" || decision.Outcome != resolver.OutcomeSelected {
			continue
		}
		skill, ok := strings.CutPrefix(decision.Subject, "skill:")
		if !ok || !safeSegment(skill) || !safeSourceID(decision.Source) {
			return nil, fmt.Errorf("bundle trace selects an unsafe identity path %q from %q",
				decision.Subject, decision.Source)
		}
		identities = append(identities, Identity{Source: decision.Source, Skill: skill})
	}
	if len(identities) == 0 {
		return nil, fmt.Errorf("bundle trace selects no skills")
	}
	return identities, nil
}

func verifyTraceProfiles(trace *Trace, manifest *Manifest) error {
	role := "role:" + manifest.Role
	found := map[string]bool{}
	selected := 0
	for _, decision := range trace.Decisions {
		if decision.Kind != "profile" || decision.Outcome != resolver.OutcomeSelected {
			continue
		}
		selected++
		found[decision.Subject] = true
	}
	if !found[role] {
		return fmt.Errorf("bundle trace does not select manifest profile %q", role)
	}
	for _, personality := range manifest.Personalities {
		subject := "personality:" + personality
		if !found[subject] {
			return fmt.Errorf("bundle trace does not select manifest profile %q", subject)
		}
	}
	for _, boundary := range manifest.Boundaries {
		subject := "boundary:" + boundary
		if !found[subject] {
			return fmt.Errorf("bundle trace does not select manifest profile %q", subject)
		}
	}
	expected := 1 + len(manifest.Personalities) + len(manifest.Boundaries)
	if selected != expected {
		return fmt.Errorf(
			"bundle trace selects %d profiles, expected role, %d personalities, and %d boundaries",
			selected, len(manifest.Personalities), len(manifest.Boundaries))
	}
	return nil
}

func verifyRepositoryDecisions(trace *Trace, manifest *Manifest) error {
	prior := ""
	for _, repository := range manifest.Repositories {
		if !safeRepositoryIdentity(repository.Identity) {
			return fmt.Errorf("bundle manifest contains unsafe repository identity %q", repository.Identity)
		}
		if repository.Identity <= prior {
			return fmt.Errorf("bundle manifest repositories must be strictly sorted and deduplicated")
		}
		prior = repository.Identity
		if repository.Source == "" || repository.Scope == "" || repository.Reason == "" {
			return fmt.Errorf("bundle manifest repository %q lacks decision provenance", repository.Identity)
		}
	}
	selected := map[string]resolver.Decision{}
	for _, decision := range trace.Decisions {
		if decision.Kind != "repository" || decision.Outcome != resolver.OutcomeSelected {
			continue
		}
		identity, ok := strings.CutPrefix(decision.Subject, "repository:")
		if !ok || !safeRepositoryIdentity(identity) || selected[identity].Subject != "" {
			return fmt.Errorf("bundle trace selects an unsafe or duplicate repository %q", decision.Subject)
		}
		selected[identity] = decision
	}
	if len(selected) != len(manifest.Repositories) {
		return fmt.Errorf("bundle trace selects %d repositories, manifest names %d", len(selected), len(manifest.Repositories))
	}
	for _, repository := range manifest.Repositories {
		decision, ok := selected[repository.Identity]
		if !ok {
			return fmt.Errorf("bundle trace does not select manifest repository %q", repository.Identity)
		}
		if decision.Source != repository.Source || decision.Reason != repository.Reason {
			return fmt.Errorf("bundle repository %q provenance disagrees with its decision trace", repository.Identity)
		}
	}
	return nil
}

func safeRepositoryIdentity(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && safeSegment(parts[0]) && safeSegment(parts[1])
}

func verifyIdentityTree(dir string, identities []Identity) error {
	root := filepath.Join(dir, "content", "skills")
	sources, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read canonical skills root: %w", err)
	}
	var found []string
	for _, sourceEntry := range sources {
		if !sourceEntry.IsDir() {
			return fmt.Errorf("canonical skills root contains non-directory %s", sourceEntry.Name())
		}
		skills, err := os.ReadDir(filepath.Join(root, sourceEntry.Name()))
		if err != nil {
			return fmt.Errorf("read identity source %s: %w", sourceEntry.Name(), err)
		}
		for _, skillEntry := range skills {
			if !skillEntry.IsDir() {
				return fmt.Errorf("identity source %s contains non-directory %s",
					sourceEntry.Name(), skillEntry.Name())
			}
			found = append(found, sourceEntry.Name()+"/"+skillEntry.Name())
		}
	}
	expected := make([]string, 0, len(identities))
	for _, identity := range identities {
		expected = append(expected, sourceSegment(identity.Source)+"/"+identity.Skill)
	}
	slices.Sort(found)
	slices.Sort(expected)
	if !slices.Equal(found, expected) {
		return fmt.Errorf("bundle identity tree %q does not match selected identities %q", found, expected)
	}
	for _, identity := range identities {
		name := identity.Source + "/" + identity.Skill
		skillDoc := filepath.Join(root, sourceSegment(identity.Source), identity.Skill, "SKILL.md")
		info, err := os.Stat(skillDoc)
		if err != nil {
			return fmt.Errorf("selected identity %q has no SKILL.md: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("selected identity %q SKILL.md is not a regular file", name)
		}
	}
	return nil
}

func requireRegularEntry(dir, name, rel string) error {
	path, err := bundleEntry(dir, name, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("bundle %s entry %q: %w", name, rel, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("bundle %s entry %q is not a regular file", name, rel)
	}
	return nil
}

func requireDirectoryEntry(dir, name, rel string) error {
	path, err := bundleEntry(dir, name, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("bundle %s entry %q: %w", name, rel, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundle %s entry %q is not a directory", name, rel)
	}
	return nil
}

func bundleEntry(dir, name, rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("bundle manifest names no %s entry point", name)
	}
	if !safePortablePath(rel) {
		return "", fmt.Errorf("bundle %s entry %q is not a safe relative path", name, rel)
	}
	return filepath.Join(dir, filepath.FromSlash(rel)), nil
}

func safeSegment(value string) bool {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return false
	}
	if strings.HasSuffix(value, ".") || strings.HasSuffix(value, " ") {
		return false
	}
	for _, character := range value {
		if character < 32 || strings.ContainsRune(`<>:"|?*`, character) {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}

func safeSourceID(value string) bool {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return false
	}
	for _, character := range value {
		if character < 32 {
			return false
		}
	}
	return true
}

// sourceSegment encodes logical source ids into portable filesystem segments.
// The trace and manifest retain the original id, including namespace colons.
func sourceSegment(value string) string {
	var encoded strings.Builder
	const hex = "0123456789ABCDEF"
	for _, octet := range []byte(value) {
		if octet >= 'a' && octet <= 'z' ||
			octet >= 'A' && octet <= 'Z' ||
			octet >= '0' && octet <= '9' ||
			octet == '-' || octet == '_' || octet == '.' {
			encoded.WriteByte(octet)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hex[octet>>4])
		encoded.WriteByte(hex[octet&0x0f])
	}
	return encoded.String()
}

func safePortablePath(value string) bool {
	if value == "" || value == "." || strings.Contains(value, `\`) || !fs.ValidPath(value) {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if !safeSegment(segment) {
			return false
		}
	}
	return true
}
