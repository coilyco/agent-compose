package person

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// The authored shape of a person package. Derived fields are filled in after
// the decode, so a package cannot assert them. See docs/person-packages.md.

type yamlAct struct {
	Side string `yaml:"side,omitempty"`
	Tool string `yaml:"tool,omitempty"`
	Text string `yaml:"text,omitempty"`
}

type yamlVoice struct {
	Summary string   `yaml:"summary,omitempty"`
	Person  string   `yaml:"person,omitempty"`
	Cadence string   `yaml:"cadence,omitempty"`
	Prefer  []string `yaml:"prefer,omitempty"`
	Avoid   []string `yaml:"avoid,omitempty"`
	Tell    string   `yaml:"tell,omitempty"`
}

type yamlOutro struct {
	Clean   string `yaml:"clean,omitempty"`
	Failure string `yaml:"failure,omitempty"`
}

type yamlScopedBoundary struct {
	Boundary string `yaml:"boundary"`
	Scope    string `yaml:"scope,omitempty"`
}

type yamlAdjacent struct {
	Role   string `yaml:"role"`
	Reason string `yaml:"reason,omitempty"`
}

type yamlIdentity struct {
	Name     string `yaml:"name,omitempty"`
	Pronouns string `yaml:"pronouns,omitempty"`
}

type yamlAgent struct {
	Harness   string `yaml:"harness,omitempty"`
	Name      string `yaml:"name,omitempty"`
	Pronouns  string `yaml:"pronouns,omitempty"`
	Channel   string `yaml:"channel,omitempty"`
	Tier      string `yaml:"tier,omitempty"`
	LegalName string `yaml:"legal_name,omitempty"`
	Key       string `yaml:"key,omitempty"`
}

type yamlForbid struct {
	Term   string `yaml:"term,omitempty"`
	Prefer string `yaml:"prefer,omitempty"`
}

type yamlCopyContract struct {
	Scope  string       `yaml:"scope,omitempty"`
	Forbid []yamlForbid `yaml:"forbid,omitempty"`
}

type yamlRoleEntity struct {
	Role             string               `yaml:"role"`
	Order            int                  `yaml:"order,omitempty"`
	DisplayName      string               `yaml:"display_name,omitempty"`
	Purpose          string               `yaml:"purpose,omitempty"`
	Briefing         string               `yaml:"briefing,omitempty"`
	Stance           string               `yaml:"stance,omitempty"`
	Element          string               `yaml:"element,omitempty"`
	Creature         string               `yaml:"creature,omitempty"`
	Skill            string               `yaml:"skill,omitempty"`
	Methods          []string             `yaml:"methods,omitempty"`
	ModelTier        []string             `yaml:"model_tier,omitempty"`
	Personalities    []string             `yaml:"personalities,omitempty"`
	Boundaries       []string             `yaml:"boundaries,omitempty"`
	ScopedBoundaries []yamlScopedBoundary `yaml:"scoped_boundaries,omitempty"`
	Adjacents        []yamlAdjacent       `yaml:"adjacents,omitempty"`
	Voice            *yamlVoice           `yaml:"voice,omitempty"`
	Outro            *yamlOutro           `yaml:"outro,omitempty"`
	Identity         *yamlIdentity        `yaml:"identity,omitempty"`
	Agents           []yamlAgent          `yaml:"agents,omitempty"`
	Seats            []yamlAgent          `yaml:"seats,omitempty"`
	Acts             []yamlAct            `yaml:"acts,omitempty"`
	CopyContract     *yamlCopyContract    `yaml:"copy_contract,omitempty"`
	Archived         bool                 `yaml:"archived,omitempty"`
}

type yamlEmblemEntity struct {
	Names []string `yaml:"names,omitempty"`
	Emoji string   `yaml:"emoji,omitempty"`
}

type yamlBodyEntity struct {
	Archetype  string `yaml:"archetype,omitempty"`
	Attachment string `yaml:"attachment,omitempty"`
}

type yamlSoundMarkEntity struct {
	Timbre  string `yaml:"timbre,omitempty"`
	Contour string `yaml:"contour,omitempty"`
	Pulse   string `yaml:"pulse,omitempty"`
}

type yamlPersonalityEntity struct {
	Personality string               `yaml:"personality"`
	Order       int                  `yaml:"order,omitempty"`
	Skill       string               `yaml:"skill,omitempty"`
	Color       string               `yaml:"color,omitempty"`
	Motif       string               `yaml:"motif,omitempty"`
	Geometry    string               `yaml:"geometry,omitempty"`
	Emblem      *yamlEmblemEntity    `yaml:"emblem,omitempty"`
	Body        *yamlBodyEntity      `yaml:"body,omitempty"`
	SoundMark   *yamlSoundMarkEntity `yaml:"sound_mark,omitempty"`
	Voice       *yamlVoice           `yaml:"voice,omitempty"`
	Verbs       []string             `yaml:"verbs,omitempty"`
	Aliases     []string             `yaml:"aliases,omitempty"`
	Acts        []yamlAct            `yaml:"acts,omitempty"`
}

type yamlBoundaryEntity struct {
	Boundary string    `yaml:"boundary"`
	Order    int       `yaml:"order,omitempty"`
	Skill    string    `yaml:"skill,omitempty"`
	Owner    string    `yaml:"owner,omitempty"`
	Summary  string    `yaml:"summary,omitempty"`
	Acts     []yamlAct `yaml:"acts,omitempty"`
}

// decodeEntity rejects unknown keys, so a typo fails rather than dropping the
// field it was meant to set.
func decodeEntity(raw []byte, into any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(into); err != nil {
		return err
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("needs exactly one document")
	}
	return nil
}

func (v *yamlVoice) model() *Voice {
	if v == nil {
		return nil
	}
	return &Voice{
		Summary: v.Summary,
		Person:  v.Person,
		Cadence: v.Cadence,
		Prefer:  v.Prefer,
		Avoid:   v.Avoid,
		Tell:    v.Tell,
	}
}

func actModels(entries []yamlAct) []Act {
	if len(entries) == 0 {
		return nil
	}
	acts := make([]Act, 0, len(entries))
	for _, entry := range entries {
		acts = append(acts, Act{Tool: entry.Tool, Text: entry.Text, Side: entry.Side})
	}
	return acts
}

// An agent is keyed by its harness. A seat carries its own key and no harness,
// which is the distinction the KDL agent and seat nodes drew.
func seatModels(agents, seats []yamlAgent) []Seat {
	out := make([]Seat, 0, len(agents)+len(seats))
	for _, entry := range agents {
		out = append(out, Seat{
			Key: entry.Harness, Harness: entry.Harness,
			Name: entry.Name, Pronouns: entry.Pronouns,
			Channel: entry.Channel, Tier: entry.Tier, LegalName: entry.LegalName,
		})
	}
	for _, entry := range seats {
		out = append(out, Seat{
			Key: entry.Key, Harness: entry.Harness,
			Name: entry.Name, Pronouns: entry.Pronouns,
			Channel: entry.Channel, Tier: entry.Tier, LegalName: entry.LegalName,
		})
	}
	return out
}

func (r *yamlRoleEntity) model() Role {
	role := Role{
		DisplayName:         r.DisplayName,
		Purpose:             r.Purpose,
		Briefing:            r.Briefing,
		Stance:              r.Stance,
		Element:             r.Element,
		Creature:            r.Creature,
		Skill:               r.Skill,
		Archived:            r.Archived,
		Methods:             r.Methods,
		SupportedModelTiers: r.ModelTier,
		Personalities:       r.Personalities,
		Boundaries:          r.Boundaries,
		Voice:               r.Voice.model(),
		Acts:                actModels(r.Acts),
	}
	if r.Outro != nil {
		role.Outro = &Outro{Clean: r.Outro.Clean, Failure: r.Outro.Failure}
	}
	if r.Identity != nil {
		role.Identity = &AgentIdentity{Name: r.Identity.Name, Pronouns: r.Identity.Pronouns}
	}
	for _, scoped := range r.ScopedBoundaries {
		role.ScopedBoundaries = append(role.ScopedBoundaries,
			ScopedBoundary{Name: scoped.Boundary, Scope: scoped.Scope})
	}
	for _, adjacent := range r.Adjacents {
		role.Adjacents = append(role.Adjacents,
			Adjacent{Role: adjacent.Role, Reason: adjacent.Reason})
	}
	role.Seats = seatModels(r.Agents, r.Seats)
	if r.CopyContract != nil {
		contract := &CopyContract{Scope: r.CopyContract.Scope}
		for _, forbid := range r.CopyContract.Forbid {
			contract.Rules = append(contract.Rules,
				CopyRule{Forbid: forbid.Term, Prefer: forbid.Prefer})
		}
		role.CopyContract = contract
	}
	return role
}

func (p *yamlPersonalityEntity) model() Personality {
	personality := Personality{
		Skill:    p.Skill,
		Color:    p.Color,
		Motif:    p.Motif,
		Geometry: p.Geometry,
		Voice:    p.Voice.model(),
		Verbs:    p.Verbs,
		Aliases:  p.Aliases,
		Acts:     actModels(p.Acts),
	}
	if p.Emblem != nil {
		personality.Emblem = Emblem{Names: p.Emblem.Names, Emoji: p.Emblem.Emoji}
	}
	if p.Body != nil {
		personality.Body = Body{Archetype: p.Body.Archetype, Attachment: p.Body.Attachment}
	}
	if p.SoundMark != nil {
		personality.SoundMark = SoundMark{
			Timbre:  p.SoundMark.Timbre,
			Contour: p.SoundMark.Contour,
			Pulse:   p.SoundMark.Pulse,
		}
	}
	return personality
}

func (b *yamlBoundaryEntity) model() Boundary {
	return Boundary{
		Skill:   b.Skill,
		Summary: b.Summary,
		Owner:   b.Owner,
		Acts:    actModels(b.Acts),
	}
}

const yamlFragmentExt = ".yaml"

type yamlManifest struct {
	Person  string `yaml:"person"`
	Roster  string `yaml:"roster"`
	Library string `yaml:"library"`
}

// yamlManifestName returns the declared node and name for a YAML manifest.
func yamlManifestName(raw []byte) (string, string, error) {
	var manifest yamlManifest
	if err := decodeEntity(raw, &manifest); err != nil {
		return "", "", err
	}
	declared := 0
	var node, name string
	for _, candidate := range []struct{ node, value string }{
		{"person", manifest.Person},
		{"roster", manifest.Roster},
		{"library", manifest.Library},
	} {
		if candidate.value != "" {
			declared++
			node, name = candidate.node, candidate.value
		}
	}
	if declared != 1 {
		return "", "", fmt.Errorf("manifest needs exactly one of person, roster, or library")
	}
	return node, name, nil
}
