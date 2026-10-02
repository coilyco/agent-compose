// Package launch is the refresh-then-exec path: compose, project, and hand
// off to the real harness with a recursion guard and a last-known-good net.
package launch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/compose"
	"github.com/coilyco/agent-compose/v2/internal/personpolicy"
	"github.com/coilyco/agent-compose/v2/internal/project"
)

// EnvSentinel marks a process launched by agent-compose. Both call sites read
// it to skip work a parent already did, and neither refuses on it alone.
const EnvSentinel = "AGENT_COMPOSE_LAUNCH"

// EnvDepth counts the agent-compose launches a process sits inside, where the
// sentinel only says that it is inside one. Absent means none.
const EnvDepth = "AGENT_COMPOSE_LAUNCH_DEPTH"

// MaxNestedDepth bounds deliberate agent-to-agent launches. One hop lets a
// session start a second seat while a runaway loop still terminates.
const MaxNestedDepth = 1

// DepthEnv stamps the depth a launched process runs at.
func DepthEnv(depth int) string {
	return EnvDepth + "=" + strconv.Itoa(depth)
}

// NestedDepth resolves the depth a launched child runs at, refusing anything
// past MaxNestedDepth. The bound is the guard rather than an opt-in. #403
func NestedDepth(sentinel, depth string) (int, error) {
	current := 0
	if trimmed := strings.TrimSpace(depth); trimmed != "" {
		parsed, err := strconv.Atoi(trimmed)
		if err != nil || parsed < 0 {
			return 0, fmt.Errorf("%s must be a non-negative count, got %q", EnvDepth, depth)
		}
		current = parsed
	}
	if strings.TrimSpace(sentinel) == "" {
		return 0, nil
	}
	next := current + 1
	if next > MaxNestedDepth {
		return 0, fmt.Errorf(
			"nested launch depth %d exceeds the %d-hop bound; a launched seat cannot launch a further one",
			next,
			MaxNestedDepth,
		)
	}
	return next, nil
}

// AttributionRoleEnv carries the composed role to the git attribution shim.
// Per-session: the host projection is global. See coilysiren/inbox#362.
const AttributionRoleEnv = "AGENT_GIT_ATTRIBUTION_ROLE"

// SessionBundleEnv and SessionLayoutEnv bind a session to the bundle it was
// launched with, which a path walk cannot do. See docs/whoami.md.
const (
	SessionBundleEnv = "AGENT_COMPOSE_SESSION_BUNDLE"
	SessionLayoutEnv = "AGENT_COMPOSE_SESSION_LAYOUT"
)

type Options struct {
	RequestPath  string
	Layout       string
	TargetDir    string
	OutDir       string
	PersonPolicy string
	PersonSource string
}

type Result struct {
	BundleDir    string
	BundleReused bool
	Projected    int
	Fallback     bool
	Warning      string
	Warnings     []string
}

// Refresh composes and projects, falling back to a validated last-known-good
// projection on failure with the cause surfaced as a warning.
func Refresh(opts Options) (*Result, error) {
	composed, err := compose.RunWithOptions(opts.RequestPath, opts.OutDir, compose.Options{
		PersonPolicy: opts.PersonPolicy,
		PersonSource: opts.PersonSource,
	})
	if err != nil {
		return fallbackOr(opts, compose.IsExternalOnlyError(err), fmt.Errorf("compose: %w", err))
	}
	projected, err := project.Project(composed.Bundle.Dir, opts.Layout, opts.TargetDir)
	if err != nil {
		return fallbackOr(opts, composed.ExternalOnly, fmt.Errorf("project: %w", err))
	}
	return &Result{
		BundleDir:    composed.Bundle.Dir,
		BundleReused: composed.Bundle.Reused,
		Projected:    len(projected.Files),
		Warnings:     append([]string(nil), composed.Resolution.Warnings...),
	}, nil
}

func fallbackOr(opts Options, externalOnly bool, cause error) (*Result, error) {
	if externalOnly || opts.PersonPolicy == personpolicy.ExternalOnly {
		return nil, fmt.Errorf("external-only person policy prohibits last-known-good fallback: %w", cause)
	}
	if err := project.Validate(opts.TargetDir); err != nil {
		return nil, fmt.Errorf("refresh failed and no last-known-good projection is usable (%v): %w", err, cause)
	}
	return &Result{Fallback: true, Warning: cause.Error()}, nil
}
