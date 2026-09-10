package person

import (
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
)

// OverrideRoleIdentity renames a role's seat and changes nothing else about the
// role. Seats move with it, and why is docs/identity.md.
func (p *Person) OverrideRoleIdentity(roleName, name, pronouns string) error {
	name = strings.TrimSpace(name)
	pronouns = strings.TrimSpace(pronouns)
	if name == "" || pronouns == "" {
		return fmt.Errorf("override role identity: %q needs both a name and pronouns", roleName)
	}
	role, ok := p.Roles[roleName]
	if !ok {
		return fmt.Errorf("override role identity: role %q is not defined", roleName)
	}
	role.Identity = &AgentIdentity{Name: name, Pronouns: pronouns}
	for index := range role.Seats {
		role.Seats[index].Name = name
		role.Seats[index].Pronouns = pronouns
	}
	// Roles is a map of structs, so the local copy has to be stored back.
	p.Roles[roleName] = role
	return nil
}

// VoiceBank is one contributor to a melded voice, kept with its label so a
// consumer can say which side of the meld a term came from.
type VoiceBank struct {
	Label string
	Voice *Voice
}

// VoiceBanks melds in role-then-personality order. The banks concatenate rather
// than override, because a wider bank is the point.
func (p *Person) VoiceBanks(roleName string, role Role) []VoiceBank {
	banks := []VoiceBank{}
	if role.Voice != nil {
		banks = append(banks, VoiceBank{p.RoleDisplayName(roleName), role.Voice})
	}
	for _, name := range role.Personalities {
		if binding, ok := p.Personalities[name]; ok && binding.Voice != nil {
			banks = append(banks, VoiceBank{displaySlug(name), binding.Voice})
		}
	}
	return banks
}

func (p *Person) renderVoice(roleName string, role Role) string {
	type source struct {
		label string
		voice *Voice
	}
	sources := []source{}
	for _, bank := range p.VoiceBanks(roleName, role) {
		sources = append(sources, source{bank.Label, bank.Voice})
	}
	if len(sources) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("## Voice\n\n")
	prefer, avoid := []string{}, []string{}
	for _, s := range sources {
		fmt.Fprintf(&out, "* **%s** - %s\n", s.label, s.voice.Summary)
		if s.voice.Cadence != "" {
			fmt.Fprintf(&out, "  * cadence - %s\n", s.voice.Cadence)
		}
		if s.voice.Tell != "" {
			fmt.Fprintf(&out, "  * tell - %s\n", s.voice.Tell)
		}
		for _, word := range s.voice.Prefer {
			if !slices.Contains(prefer, word) {
				prefer = append(prefer, word)
			}
		}
		for _, word := range s.voice.Avoid {
			if !slices.Contains(avoid, word) {
				avoid = append(avoid, word)
			}
		}
	}
	if role.Voice != nil && role.Voice.Person != "" {
		fmt.Fprintf(&out, "\n**Person // %s**\n", role.Voice.Person)
	}
	if len(prefer) > 0 {
		fmt.Fprintf(&out, "\n**Reach for** - `%s`\n", strings.Join(prefer, "` `"))
	}
	if len(avoid) > 0 {
		fmt.Fprintf(&out, "\n**Refuse** - `%s`\n", strings.Join(avoid, "` `"))
	}
	out.WriteString("\n")
	return out.String()
}

// renderActs melds in role-then-personality order, as renderVoice does.
// Boundary acts render beside the side the seat holds, not here.
func (p *Person) renderActs(roleName string, role Role) string {
	type source struct {
		label string
		acts  []Act
	}
	sources := []source{}
	if len(role.Acts) > 0 {
		sources = append(sources, source{p.RoleDisplayName(roleName), role.Acts})
	}
	for _, name := range role.Personalities {
		if binding, ok := p.Personalities[name]; ok && len(binding.Acts) > 0 {
			sources = append(sources, source{displaySlug(name), binding.Acts})
		}
	}
	if len(sources) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("## Run\n\n")
	out.WriteString(
		"These are the minimum, not the illustration. An attribute you cannot name an act" +
			" for is one that did not fire.\n\n",
	)
	for _, s := range sources {
		fmt.Fprintf(&out, "* **%s**\n", s.label)
		for _, act := range s.acts {
			fmt.Fprintf(&out, "  * %s\n", act.Text)
		}
	}
	out.WriteString("\n")
	return out.String()
}

// RenderRoleIdentityCard keeps identity texture visible while long-form role
// and personality doctrine remains lazy-loaded from ordinary skills.
func (p *Person) RenderRoleIdentityCard(roleName, meldedColor string, boundaries []string) (string, error) {
	role, ok := p.Roles[roleName]
	if !ok {
		return "", fmt.Errorf("render role identity card: role %q is not defined", roleName)
	}
	if meldedColor == "" {
		return "", fmt.Errorf("render role identity card: role %q has no melded color", roleName)
	}
	var out strings.Builder
	roleSkill := p.RoleSkillID(roleName)
	fmt.Fprintf(&out, "# %s\n\n%s\n\n", p.RoleDisplayName(roleName), role.Purpose)
	fmt.Fprintf(&out, "**Role skill // `%s`**\n", roleSkill)
	if len(role.Methods) > 0 {
		fmt.Fprintf(&out, "**Role methods // `%s`**\n", strings.Join(role.Methods, "` // `"))
	}
	boundarySkills := p.boundarySkillIDs(boundaries)
	if len(boundarySkills) > 0 {
		fmt.Fprintf(&out, "**Boundaries // `%s`**\n", strings.Join(boundarySkills, "` // `"))
	}
	if role.Guardrail != "" {
		fmt.Fprintf(&out, "**Guardrail // `%s`**\n", p.Guardrails[role.Guardrail].Skill)
	}
	fmt.Fprintf(&out, "**Favorite color // `%s`**\n", meldedColor)
	if role.Identity != nil {
		fmt.Fprintf(
			&out,
			"**Agent // %s (%s)**\n",
			role.Identity.Name,
			role.Identity.Pronouns,
		)
	}
	if len(role.Seats) > 0 {
		out.WriteString("**Seats")
		for _, seat := range role.Seats {
			if role.Identity != nil {
				fmt.Fprintf(&out, " // %s", seat.Selector())
				continue
			}
			fmt.Fprintf(&out, " // %s: %s (%s)", seat.Selector(), seat.Name, seat.Pronouns)
		}
		out.WriteString("**\n")
	}
	// Ahead of the meld on purpose: a correction arriving after five kilobytes of
	// register guidance competes with it rather than bounding it.
	if role.Guardrail != "" {
		rail := p.Guardrails[role.Guardrail]
		fmt.Fprintf(&out, "\n## Guardrail // %s\n\n", displaySlug(role.Guardrail))
		fmt.Fprintf(&out, "%s\n\n", strings.TrimSpace(rail.Card))
		fmt.Fprintf(&out, "The procedure is `%s`. Load it before you rely on it.\n", rail.Skill)
	}
	out.WriteString("\n## Personality meld\n\n")
	for _, name := range role.Personalities {
		binding, exists := p.Personalities[name]
		if !exists {
			return "", fmt.Errorf("render role identity card: personality %q is not defined", name)
		}
		fmt.Fprintf(&out, "### %s %s\n\n", binding.Emblem.Emoji, displaySlug(name))
		fmt.Fprintf(
			&out,
			"**%s // %s // %s**\n\n",
			binding.Color,
			strings.Join(binding.Emblem.Names, " / "),
			binding.Motif,
		)
		description, err := p.personalityDescription(binding)
		if err != nil {
			return "", fmt.Errorf("render role identity card: personality %q: %w", name, err)
		}
		fmt.Fprintf(&out, "%s\n\n", description)
	}
	if section := p.renderVoice(roleName, role); section != "" {
		out.WriteString(section)
	}
	if section := p.renderActs(roleName, role); section != "" {
		out.WriteString(section)
	}
	if len(boundaries) > 0 {
		out.WriteString("## Boundaries\n\n")
		for _, name := range boundaries {
			binding, exists := p.Boundaries[name]
			if !exists {
				return "", fmt.Errorf("render role identity card: boundary %q is not defined", name)
			}
			side := "you defer this"
			sideKey := "defer"
			if binding.Owner == roleName {
				side = "you own this"
				sideKey = "own"
			}
			scopeText := ""
			for _, scoped := range role.ScopedBoundaries {
				if scoped.Name == name {
					side = "you hold this within a scope"
					sideKey = "scoped"
					scopeText = ". Your scope: " + scoped.Scope
					break
				}
			}
			fmt.Fprintf(&out, "* `%s` - %s. %s%s\n", binding.Skill, side, binding.Summary, scopeText)
			for _, act := range binding.ActsForSide(sideKey) {
				fmt.Fprintf(&out, "  * %s\n", act.Text)
			}
		}
		out.WriteString("\n")
	}
	named := append([]string{roleSkill}, boundarySkills...)
	for _, name := range role.Personalities {
		named = append(named, p.Personalities[name].Skill)
	}
	if role.Guardrail != "" {
		named = append(named, p.Guardrails[role.Guardrail].Skill)
	}
	out.WriteString("## Active doctrine\n\n")
	sizes, total := p.skillBodySizes(roleName, named)
	// The summaries above read as the whole of it, so the card states what they
	// compress. See docs/identity.md and the measurement in issue #303.
	if total > 0 {
		fmt.Fprintf(
			&out,
			"Everything above summarizes %s bytes of doctrine across these %d skills, and a"+
				" summary is not the operative text. Before acting, load each one:\n\n",
			thousands(total), len(sizes),
		)
	} else {
		out.WriteString("Before acting, load:\n\n")
	}
	for _, skill := range named {
		if size := sizes[skill]; size > 0 {
			fmt.Fprintf(&out, "* `%s` - %s bytes\n", skill, thousands(size))
			continue
		}
		fmt.Fprintf(&out, "* `%s`\n", skill)
	}
	return out.String(), nil
}

// skillBodySizes reports each named skill's own bytes. An external person
// package may ship no readable source, so a missing size is omitted.
func (p *Person) skillBodySizes(roleName string, named []string) (map[string]int, int) {
	sizes := make(map[string]int, len(named))
	total := 0
	// The role skill is not under definitions/skills, so it comes from the raw
	// bytes ingest kept.
	roleSkill := p.RoleSkillID(roleName)
	if raw := p.roleSkills[roleName]; len(raw) > 0 {
		sizes[roleSkill] = len(raw)
		total += len(raw)
	}
	if p.source == nil {
		return sizes, total
	}
	for _, skill := range named {
		if _, done := sizes[skill]; done {
			continue
		}
		raw, err := fs.ReadFile(p.source, "definitions/skills/"+skill+"/SKILL.md")
		if err != nil {
			continue
		}
		sizes[skill] = len(raw)
		total += len(raw)
	}
	return sizes, total
}

func thousands(value int) string {
	digits := strconv.Itoa(value)
	var out strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(digit)
	}
	return out.String()
}

func displaySlug(value string) string {
	parts := strings.Split(value, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[index] = string(runes)
	}
	return strings.Join(parts, " ")
}

func (p *Person) personalityDescription(binding Personality) (string, error) {
	if p.source == nil {
		return "Load the personality skill for its complete behavioral definition.", nil
	}
	raw, err := fs.ReadFile(p.source, "definitions/skills/"+binding.Skill+"/SKILL.md")
	if err != nil {
		return "", err
	}
	return skillDescription(raw)
}

func skillDescription(raw []byte) (string, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", fmt.Errorf("missing YAML frontmatter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return "", fmt.Errorf("unterminated YAML frontmatter")
	}
	for _, line := range strings.Split(text[4:4+end], "\n") {
		if value, found := strings.CutPrefix(line, "description: "); found {
			if sentence, _, found := strings.Cut(value, ". "); found {
				return sentence + ".", nil
			}
			return strings.TrimSpace(value), nil
		}
	}
	return "", fmt.Errorf("missing description")
}

type RoleTranscriptOptions struct {
	Color     bool
	TrueColor bool
	// Expanded adds what the role and personality skills already carry: the
	// briefing, the credits, the expressions. Texture is not gated. See #323.
	Expanded bool
}

// RenderRoleMetadata returns complete selected identity facts for each bundle.
// Appearance catalogue entries remain in the person snapshot and documentation.
func (p *Person) RenderRoleMetadata(roleName, meldedColor string) (string, error) {
	role, ok := p.Roles[roleName]
	if !ok {
		return "", fmt.Errorf("render role metadata: role %q is not defined", roleName)
	}
	if meldedColor == "" {
		return "", fmt.Errorf("render role metadata: role %q has no melded color", roleName)
	}

	var out strings.Builder
	out.WriteString("## Active role metadata\n\n")
	out.WriteString("Agent-compose selected these public-safe facts for the agent.\n\n")
	fmt.Fprintf(&out, "* Provider: `%s`\n", p.ProviderID())
	fmt.Fprintf(&out, "* Role: `%s`\n", roleName)
	fmt.Fprintf(&out, "* Purpose: %s\n", role.Purpose)
	if role.Identity != nil {
		fmt.Fprintf(
			&out,
			"* Agent identity: `%s` (pronouns: `%s`)\n",
			role.Identity.Name,
			role.Identity.Pronouns,
		)
	}
	if len(role.Methods) > 0 {
		fmt.Fprintf(&out, "* Role methods: `%s`\n", strings.Join(role.Methods, "`, `"))
	}
	out.WriteString("* Component personalities:\n")
	for _, name := range role.Personalities {
		binding, exists := p.Personalities[name]
		if !exists {
			return "", fmt.Errorf("render role metadata: personality %q is not defined", name)
		}
		fmt.Fprintf(
			&out,
			"  * `%s`: skill `%s`, favorite color `%s`, emblem `%s` `%s`, motif `%s`, geometry `%s`, sound mark `%s` / `%s` / `%s`\n",
			name,
			binding.Skill,
			binding.Color,
			binding.Emblem.Emoji,
			strings.Join(binding.Emblem.Names, " / "),
			binding.Motif,
			binding.Geometry,
			binding.SoundMark.Timbre,
			binding.SoundMark.Contour,
			binding.SoundMark.Pulse,
		)
	}
	fmt.Fprintf(&out, "* Melded favorite color: `%s`\n", meldedColor)

	if len(role.Seats) == 0 {
		out.WriteString("* Known agent seats: none\n")
	} else {
		out.WriteString("* Known agent seats:\n")
		for _, seat := range role.Seats {
			if role.Identity != nil {
				fmt.Fprintf(&out, "  * `%s`%s\n", seat.Selector(), seatRoutingSuffix(seat))
				continue
			}
			fmt.Fprintf(&out, "  * `%s`: `%s` (pronouns: `%s`)\n",
				seat.Selector(), seat.Name, seat.Pronouns)
		}
	}
	fmt.Fprintf(&out, "* Renderer expressions: `%s`\n", strings.Join(ExpressionVocabulary(), "`, `"))
	return out.String(), nil
}

// RenderRoleTranscript returns the complete selected slice of the person
// snapshot as deterministic, flush-left terminal text.
func (p *Person) RenderRoleTranscript(
	roleName, meldedColor string,
	opts RoleTranscriptOptions,
) (string, error) {
	role, ok := p.Roles[roleName]
	if !ok {
		return "", fmt.Errorf("render role transcript: role %q is not defined", roleName)
	}
	if meldedColor == "" {
		return "", fmt.Errorf("render role transcript: role %q has no melded color", roleName)
	}
	var out strings.Builder
	var roleBlock strings.Builder
	roleBlock.WriteString("role metadata\n")
	fmt.Fprintf(&roleBlock, "%s: %s // provided by: %s\n", p.providerKind(), p.Name, p.ProviderID())
	fmt.Fprintf(&roleBlock, "role: %s\n", roleName)
	fmt.Fprintf(&roleBlock, "purpose: %s\n", role.Purpose)
	if role.Identity != nil {
		fmt.Fprintf(
			&roleBlock,
			"agent identity: %s // pronouns: %s\n",
			role.Identity.Name,
			role.Identity.Pronouns,
		)
	}
	if len(role.Methods) > 0 {
		fmt.Fprintf(&roleBlock, "methods: %s\n", strings.Join(role.Methods, " // "))
	}
	fmt.Fprintf(&roleBlock, "personalities: %s\n", strings.Join(role.Personalities, " // "))
	fmt.Fprintf(&roleBlock, "melded color: %s\n", meldedColor)
	// The briefing restates the role skill the agent loads anyway.
	if opts.Expanded {
		writeTranscriptParagraphs(&roleBlock, "briefing", role.Briefing)
	}
	roleBlock.WriteString("seats:\n")
	for _, seat := range role.Seats {
		if role.Identity != nil {
			fmt.Fprintf(&roleBlock, "seat %s%s\n", seat.Selector(), seatRoutingSuffix(seat))
			continue
		}
		fmt.Fprintf(&roleBlock, "seat %s: %s // pronouns: %s\n", seat.Selector(), seat.Name, seat.Pronouns)
	}

	writeTranscriptSection(&out, meldedColor, "personality metadata\n", opts)
	width := 0
	for _, name := range role.Personalities {
		if len(name) > width {
			width = len(name)
		}
	}
	for index, name := range role.Personalities {
		binding, exists := p.Personalities[name]
		if !exists {
			return "", fmt.Errorf("render role transcript: personality %q is not defined", name)
		}
		if index > 0 {
			out.WriteByte('\n')
		}
		block, err := p.personalityTexture(name, binding, width, opts.Expanded)
		if err != nil {
			return "", err
		}
		writeTranscriptSection(&out, binding.Color, block, opts)
	}

	if opts.Expanded {
		expressions := fmt.Sprintf(
			"renderer expressions: %s\n",
			strings.Join(ExpressionVocabulary(), " // "),
		)
		out.WriteByte('\n')
		writeTranscriptSection(&out, meldedColor, expressions, opts)
	}
	out.WriteByte('\n')
	writeTranscriptSection(&out, meldedColor, roleBlock.String(), opts)
	return out.String(), nil
}

// personalityTexture repeats the key on the left of every line, so one
// personality greps out of the block.
func (p *Person) personalityTexture(
	name string,
	binding Personality,
	width int,
	expanded bool,
) (string, error) {
	var out strings.Builder
	key := fmt.Sprintf("personality: %-*s", width, name)
	fmt.Fprintf(&out, "%s // %s %s // %s // %s\n",
		key, binding.Emblem.Emoji, strings.Join(binding.Emblem.Names, " / "),
		binding.Motif, binding.Color)
	fmt.Fprintf(&out, "%s // geometry: %s\n", key, binding.Geometry)
	fmt.Fprintf(&out, "%s // body: %s\n", key, binding.Body.Archetype)
	fmt.Fprintf(&out, "%s // emblem sits: %s\n", key, binding.Body.Attachment)
	fmt.Fprintf(&out, "%s // sound: %s, %s, %s\n",
		key, binding.SoundMark.Timbre, binding.SoundMark.Contour, binding.SoundMark.Pulse)
	fmt.Fprintf(&out, "%s // skill: %s\n", key, binding.Skill)
	return out.String(), nil
}

func seatRoutingSuffix(seat Seat) string {
	parts := make([]string, 0, 2)
	if seat.Channel != "" {
		parts = append(parts, "channel: "+seat.Channel)
	}
	if seat.Tier != "" {
		parts = append(parts, "tier: "+seat.Tier)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, " // ") + ")"
}

func writeTranscriptSection(
	out *strings.Builder,
	hex, text string,
	opts RoleTranscriptOptions,
) {
	if opts.Color {
		out.WriteString(color.ANSI(hex, text, opts.TrueColor))
		return
	}
	out.WriteString(text)
}

func writeTranscriptParagraphs(out *strings.Builder, label, value string) {
	fmt.Fprintf(out, "%s:\n", label)
	wrote := false
	for _, raw := range strings.Split(strings.TrimSpace(value), "\n\n") {
		paragraph := strings.Join(strings.Fields(raw), " ")
		if paragraph == "" {
			continue
		}
		if wrote {
			out.WriteByte('\n')
		}
		out.WriteString(paragraph)
		out.WriteByte('\n')
		wrote = true
	}
	if !wrote {
		out.WriteString("(none)\n")
	}
}

// boundarySkillIDs names the skills for one already-composed boundary set.
func (p *Person) boundarySkillIDs(boundaries []string) []string {
	ids := make([]string, 0, len(boundaries))
	for _, name := range boundaries {
		if binding, exists := p.Boundaries[name]; exists {
			ids = append(ids, binding.Skill)
		}
	}
	return ids
}
