// Package overlay projects the canonical person model into a compact,
// caller-stateful identity surface.
package overlay

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/coilyco/agent-compose/v2/internal/person"
)

const (
	Format        = "agent-compose.overlay.v1"
	SchemaVersion = 1
)

type Personality struct {
	Name      string        `json:"name"`
	Species   string        `json:"species"`
	ColorWord string        `json:"color_word"`
	Color     string        `json:"color"`
	Motif     string        `json:"motif"`
	Geometry  string        `json:"geometry"`
	Emblem    person.Emblem `json:"emblem"`
	Body      person.Body   `json:"body"`
}

type Document struct {
	Format          string `json:"format"`
	SchemaVersion   int    `json:"schema_version"`
	Person          string `json:"person"`
	Role            string `json:"role"`
	RoleDisplayName string `json:"role_display_name"`
	Purpose         string `json:"purpose"`
	// Stance is the role's posture, not a personality's. See docs/identity.md.
	Stance string      `json:"stance,omitempty"`
	Seat   person.Seat `json:"seat"`
	// Annotation is the composed identity string a renderer shows verbatim,
	// `Moss-Toad (Engineer)`.
	Annotation string `json:"annotation"`
	// Identity is the four-part sentence a seat answers "who are you" with.
	// Seat-resolved, so it lives here rather than on the card (#396).
	Identity string `json:"identity,omitempty"`

	// A field rather than only prose, because a consumer parsed Identity for
	// it (#7362). Derived from the meld, with no element segment (#7485).
	Creature string `json:"creature,omitempty"`

	// A renderer that draws a role with no roster entry gets the carried
	// object from here, so it is projected rather than left to a roster.
	Carried *person.Carried `json:"carried,omitempty"`
	// Outro carries voice at the close, so a hold banner is not invented.
	Outro         *person.Outro `json:"outro,omitempty"`
	Expression    string        `json:"expression"`
	FavoriteColor string        `json:"favorite_color"`
	// Background is solved across the whole roster, so a renderer that tints
	// its own cannot tell two roles apart. See internal/palette/role-palette.txt.
	Background    string        `json:"background"`
	Personalities []Personality `json:"personalities"`
}

func Build(p *person.Person, roleName, harness, expression string) (*Document, error) {
	if p == nil {
		return nil, fmt.Errorf("build overlay: person is nil")
	}
	role, ok := p.Roles[roleName]
	if !ok {
		return nil, fmt.Errorf("build overlay: role %q is not defined", roleName)
	}
	var seat person.Seat
	for _, candidate := range role.Seats {
		if candidate.Selector() == harness {
			seat = candidate
			break
		}
	}
	if seat.Selector() == "" {
		return nil, fmt.Errorf("build overlay: role %q has no %q seat", roleName, harness)
	}
	if !validExpression(expression) {
		return nil, fmt.Errorf("build overlay: expression %q is not defined", expression)
	}

	displayName := p.RoleDisplayName(roleName)
	doc := &Document{
		Format:          Format,
		SchemaVersion:   SchemaVersion,
		Person:          p.Name,
		Role:            roleName,
		RoleDisplayName: displayName,
		Purpose:         role.Purpose,
		Stance:          role.Stance,
		Seat:            seat,
		Annotation:      person.SeatAnnotation(seat.Name, displayName),
		Outro:           role.Outro,
		Identity: person.IdentitySentence(
			seat.Name, displayName, seat.LegalName, p.RoleCreature(roleName),
		),
		Creature:   p.RoleCreature(roleName),
		Carried:    role.Carried,
		Expression: expression,
	}
	colors := make([]string, 0, len(role.Personalities))
	for _, name := range role.Personalities {
		binding, exists := p.Personalities[name]
		if !exists {
			return nil, fmt.Errorf("build overlay: role %q names missing personality %q", roleName, name)
		}
		colors = append(colors, binding.Color)
		doc.Personalities = append(doc.Personalities, Personality{
			Name:      name,
			Species:   binding.Species,
			ColorWord: binding.ColorWord,
			Color:     binding.Color,
			Motif:     binding.Motif,
			Geometry:  binding.Geometry,
			Emblem:    binding.Emblem,
			Body:      binding.Body,
		})
	}
	doc.FavoriteColor = role.FavoriteColor
	doc.Background = role.Background
	return doc, nil
}

func Marshal(doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("marshal overlay: document is nil")
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal overlay: %w", err)
	}
	return append(raw, '\n'), nil
}

func RenderText(doc *Document, width int) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("render overlay: document is nil")
	}
	if width < 20 {
		return "", fmt.Errorf("render overlay: width must be at least 20")
	}
	marks := make([]string, 0, len(doc.Personalities))
	names := make([]string, 0, len(doc.Personalities))
	for _, personality := range doc.Personalities {
		marks = append(marks, personality.Emblem.Emoji)
		names = append(names, personality.Name)
	}
	role := strings.ReplaceAll(doc.Role, "-", " ")
	// The card prints the role on its own, so the seat carries just its name.
	seat := person.SeatLabel(doc.Seat.Name)
	wide := fmt.Sprintf(
		"%s  %s  //  %s / %s  //  %s  //  %s",
		strings.Join(marks, " "),
		seat,
		role,
		doc.Expression,
		strings.Join(names, " + "),
		doc.FavoriteColor,
	)
	if utf8.RuneCountInString(wide) <= width {
		return wide + "\n", nil
	}
	var lines []string
	for _, text := range []string{
		strings.Join(marks, " ") + "  " + seat,
		role + " / " + doc.Expression,
		strings.Join(names, " + "),
		doc.FavoriteColor,
	} {
		lines = append(lines, wrapText(text, width)...)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		candidate := lines[last] + " " + word
		if utf8.RuneCountInString(candidate) <= width {
			lines[last] = candidate
			continue
		}
		lines = append(lines, word)
	}
	return lines
}

func validExpression(value string) bool {
	for _, expression := range person.ExpressionVocabulary() {
		if value == expression {
			return true
		}
	}
	return false
}
