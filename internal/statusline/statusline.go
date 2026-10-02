// Package statusline renders compact identity and composition health from the
// immutable bundle selected by the nearest Agent Compose projection.
package statusline

import (
	"fmt"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/agentid"
	"github.com/coilyco/agent-compose/v2/internal/bundle"
	"github.com/coilyco/agent-compose/v2/internal/color"
	"github.com/coilyco/agent-compose/v2/internal/person"
	"github.com/coilyco/agent-compose/v2/internal/project"
	"github.com/coilyco/agent-compose/v2/internal/resolver"
	"github.com/coilyco/agent-compose/v2/internal/schema"
)

type Options struct {
	Target    string
	Color     bool
	TrueColor bool
}

// Render returns an empty string when no composition applies. It resolves the
// same way whoami does, so the row and the hook never name one session twice.
func Render(opts Options) (string, error) {
	projection := resolveProjection(opts.Target)
	if projection.Bundle == "" {
		return "", nil
	}
	manifest, err := bundle.ReadManifest(projection.Bundle)
	if err != nil {
		return warning("composition unreadable", opts), nil
	}
	trace, err := bundle.ReadTrace(projection.Bundle)
	if err != nil {
		return warning("composition trace unreadable", opts), nil
	}
	return render(projection, manifest, trace, opts), nil
}

func render(
	projection project.Projection,
	manifest *bundle.Manifest,
	trace *bundle.Trace,
	opts Options,
) string {
	modelTier := manifest.ModelTier
	if modelTier == "" {
		modelTier = schema.ModelTierFrontier
	}
	seatName := sessionSeatDisplayName(manifest, projection.Layout)

	marks := personalityMarks(manifest, opts)
	if marks == "" {
		marks = "🧩"
	}
	seatText := paint(manifest.Color, seatName, opts)

	skills, tokens, warnings := providerTotals(trace)
	health := paint("#5fa87a", "✓ composed", opts)
	if warnings > 0 {
		label := "source"
		if warnings != 1 {
			label = "sources"
		}
		health = paint(
			"#d98e48",
			fmt.Sprintf("⚠ %d %s skipped", warnings, label),
			opts,
		)
	}

	return fmt.Sprintf(
		"  %s  %s // %s@%s // %s // %d skills / ~%s catalog // %s",
		marks,
		seatText,
		manifest.Role,
		projection.Layout,
		modelTier,
		skills,
		compact(tokens),
		health,
	)
}

// seatDisplayName renders the seat's own name, stopping short of the role the
// next field already names. It degrades rather than inventing a seat.
func seatDisplayName(manifest *bundle.Manifest, layout string) string {
	if seat := selectedSeat(manifest.Identity.Seats, layout); seat.Name != "" {
		return person.SeatLabel(seat.Name)
	}
	return manifest.Role
}

// sessionSeatDisplayName adds the short id, which the subagent rows omit: it
// names the session, so every row would repeat it. See docs/statusline.md.
func sessionSeatDisplayName(manifest *bundle.Manifest, layout string) string {
	return person.WithShortID(seatDisplayName(manifest, layout), agentid.FromEnv())
}

func selectedSeat(seats []person.Seat, layout string) person.Seat {
	for _, seat := range seats {
		if seat.Selector() == layout {
			return seat
		}
	}
	return person.Seat{}
}

func personalityMarks(manifest *bundle.Manifest, opts Options) string {
	marks := make([]string, 0, len(manifest.Identity.Personalities))
	for _, personality := range manifest.Identity.Personalities {
		mark := personality.Emblem.Emoji
		if mark != "" {
			marks = append(marks, paint(personality.Color, mark, opts))
		}
	}
	return strings.Join(marks, " ")
}

func providerTotals(trace *bundle.Trace) (int, int64, int) {
	var skills int
	var tokens int64
	var warnings int
	for _, provider := range trace.Providers {
		if provider.Outcome == resolver.OutcomeSelected {
			skills += provider.Skills
			tokens += provider.ApproximateTokens
		}
		if provider.Warning {
			warnings++
		}
	}
	return skills, tokens, warnings
}

func compact(value int64) string {
	switch {
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.0fk", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func warning(message string, opts Options) string {
	return "  🧩  " + paint("#d98e48", "⚠ "+message, opts)
}

func paint(hex, text string, opts Options) string {
	if !opts.Color || hex == "" {
		return text
	}
	return color.ANSI(hex, text, opts.TrueColor)
}
