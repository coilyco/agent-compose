// Package nativeui projects a selected person into the Claude Code terminal
// surfaces a composed identity can drive: a custom theme and the settings
// fragment that selects it and carries the role's spinner voice.
package nativeui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/color"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
)

// Unknown tokens are dropped silently by the harness, so emission uses a
// fixed slot map. See docs/claude-native-ui-plan.md.
const baseTheme = "dark"

// The subagent palette is eight fixed tokens. A role claims the nearest slot
// rather than adding one.
var subagentSlots = map[string]string{
	"red_FOR_SUBAGENTS_ONLY":    "#dc2626",
	"blue_FOR_SUBAGENTS_ONLY":   "#6a9bcc",
	"green_FOR_SUBAGENTS_ONLY":  "#16a34a",
	"yellow_FOR_SUBAGENTS_ONLY": "#ca8a04",
	"purple_FOR_SUBAGENTS_ONLY": "#827dbd",
	"orange_FOR_SUBAGENTS_ONLY": "#d97757",
	"pink_FOR_SUBAGENTS_ONLY":   "#c46686",
	"cyan_FOR_SUBAGENTS_ONLY":   "#0891b2",
}

// Theme is the document Claude Code reads from ~/.claude/themes/<slug>.json.
type Theme struct {
	Name      string            `json:"name"`
	Base      string            `json:"base"`
	Overrides map[string]string `json:"overrides"`
}

// SpinnerVerbs replaces or extends the harness verb vocabulary.
type SpinnerVerbs struct {
	Mode  string   `json:"mode"`
	Verbs []string `json:"verbs"`
}

// SpinnerTips adds role doctrine to the waiting surface. ExcludeDefault stays
// false so the harness keeps teaching its own features.
type SpinnerTips struct {
	ExcludeDefault bool     `json:"excludeDefault"`
	Tips           []string `json:"tips"`
}

// StatusLineCommand is a pull-based row renderer the harness invokes. Nothing
// is installed for it, so it survives --safe-mode.
type StatusLineCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// Settings is the fragment merged into ~/.claude/settings.json.
type Settings struct {
	Theme              string            `json:"theme"`
	SpinnerVerbs       SpinnerVerbs      `json:"spinnerVerbs"`
	SpinnerTipsEnabled bool              `json:"spinnerTipsEnabled"`
	SpinnerTips        SpinnerTips       `json:"spinnerTipsOverride"`
	SubagentStatusLine StatusLineCommand `json:"subagentStatusLine"`
}

// SubagentStatusLineCommand renders the composed identity per agent-panel row.
const SubagentStatusLineCommand = "acompose statusline --subagent --color"

// Bundle is everything one role contributes to the native surfaces.
type Bundle struct {
	Role     string   `json:"role"`
	Slug     string   `json:"slug"`
	Theme    Theme    `json:"theme"`
	Settings Settings `json:"settings"`
}

// Options tunes emission for callers that do not want the default voice.
type Options struct {
	// SpinnerMode is "replace" or "append". Replace makes the role
	// unmistakable and discards the harness vocabulary.
	SpinnerMode string
	// SlugPrefix namespaces theme slugs so several people can coexist.
	SlugPrefix string
}

func (o Options) spinnerMode() string {
	if o.SpinnerMode == "" {
		return "replace"
	}
	return o.SpinnerMode
}

func (o Options) slugPrefix() string {
	if o.SlugPrefix == "" {
		return "aos-"
	}
	return o.SlugPrefix
}

// Build projects every role in catalogue order.
func Build(p *person.Person, opts Options) ([]Bundle, error) {
	bundles := make([]Bundle, 0, len(p.RoleOrder))
	for _, name := range p.RoleOrder {
		bundle, err := BuildRole(p, name, opts)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

// BuildRole projects one role into its theme and settings fragment.
func BuildRole(p *person.Person, roleName string, opts Options) (Bundle, error) {
	role, ok := p.Roles[roleName]
	if !ok {
		return Bundle{}, fmt.Errorf("unknown role %q", roleName)
	}

	colors := make([]string, 0, len(role.Personalities))
	for _, name := range role.Personalities {
		personality, ok := p.Personalities[name]
		if !ok {
			return Bundle{}, fmt.Errorf("role %q names missing personality %q", roleName, name)
		}
		colors = append(colors, personality.Color)
	}
	roleColor := role.FavoriteColor

	overrides, err := themeOverrides(roleColor, colors)
	if err != nil {
		return Bundle{}, fmt.Errorf("role %q theme: %w", roleName, err)
	}

	slug := opts.slugPrefix() + roleName
	displayName := role.DisplayName
	if role.Identity != nil && role.Identity.Name != "" {
		displayName = fmt.Sprintf("%s (%s)", role.DisplayName, role.Identity.Name)
	}

	return Bundle{
		Role: roleName,
		Slug: slug,
		Theme: Theme{
			Name:      displayName,
			Base:      baseTheme,
			Overrides: overrides,
		},
		Settings: Settings{
			Theme: "custom:" + slug,
			SpinnerVerbs: SpinnerVerbs{
				Mode:  opts.spinnerMode(),
				Verbs: verbsFor(p, role),
			},
			SpinnerTipsEnabled: true,
			SpinnerTips:        SpinnerTips{Tips: tipsFor(role)},
			SubagentStatusLine: StatusLineCommand{
				Type:    "command",
				Command: SubagentStatusLineCommand,
			},
		},
	}, nil
}

// tipsFor states the charter, the lock on it, and the boundary. A tip lands while
// the reader is waiting, so it carries doctrine rather than voice.
func tipsFor(role person.Role) []string {
	tips := []string{role.Purpose}
	if role.Identity != nil && role.Identity.Name != "" {
		tips = append(tips, fmt.Sprintf(
			"%s holds the %s charter. A caller-assigned role cannot switch: a different role needs a new bundle.",
			role.Identity.Name, role.DisplayName,
		))
	}
	if len(role.Personalities) > 0 {
		tips = append(tips, "Boundary: "+strings.Join(role.Personalities, ", ")+".")
	}
	return tips
}

// themeOverrides assigns the role color to the frame and each personality to
// one interaction. Readability tokens stay at base.
func themeOverrides(roleColor string, melded []string) (map[string]string, error) {
	overrides := map[string]string{
		"claude":           roleColor,
		"clawd_body":       roleColor,
		"briefLabelClaude": roleColor,
		"promptBorder":     roleColor,
		"skill":            melded[0],
		"autoAccept":       melded[0],
		"permission":       melded[len(melded)/2],
		"bashBorder":       melded[len(melded)/2],
		"suggestion":       melded[len(melded)-1],
		"remember":         melded[len(melded)-1],
	}

	for token, paired := range map[string]string{
		"claudeShimmer":       "claude",
		"promptBorderShimmer": "promptBorder",
		"permissionShimmer":   "permission",
	} {
		shimmer, err := color.Shimmer(overrides[paired])
		if err != nil {
			return nil, err
		}
		overrides[token] = shimmer
	}

	slotNames := make([]string, 0, len(subagentSlots))
	slotColors := make([]string, 0, len(subagentSlots))
	for name := range subagentSlots {
		slotNames = append(slotNames, name)
	}
	sort.Strings(slotNames)
	for _, name := range slotNames {
		slotColors = append(slotColors, subagentSlots[name])
	}
	nearest, err := color.Nearest(roleColor, slotColors)
	if err != nil {
		return nil, err
	}
	for _, name := range slotNames {
		if subagentSlots[name] == nearest {
			overrides[name] = roleColor
			break
		}
	}

	return overrides, nil
}

// verbsFor concatenates the boundary in role order so the spinner reads as the
// whole personality set rather than the dominant one.
func verbsFor(p *person.Person, role person.Role) []string {
	verbs := []string{}
	for _, name := range role.Personalities {
		verbs = append(verbs, p.Personalities[name].Verbs...)
	}
	return verbs
}
