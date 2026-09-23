package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/person"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/resolver"
	"github.com/coilyco-flight-deck/agent-compose/v2/internal/schema"
)

// A defer-side boundary routes work to the seat that owns it. A deployment
// with no such seat composes without it. See docs/ownership.md.

func TestBoundaryOmitLeavesNoTraceInTheBundle(t *testing.T) {
	out := t.TempDir()
	result, err := Run(fixture(t, "boundary-omit.kdl"), out)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	// Every seat owns or scopes at least one boundary, so the manifest keeps
	// the scoped grant and loses only the deferrals the request named.
	m := readManifest(t, result.Bundle.Dir)
	if len(m.Boundaries) != 1 || m.Boundaries[0] != "build-foundational-software" {
		t.Fatalf("manifest boundaries = %v, want the scoped grant alone", m.Boundaries)
	}
	instructions, err := os.ReadFile(filepath.Join(result.Bundle.Dir, "content", "instructions.md"))
	if err != nil {
		t.Fatal(err)
	}
	// Naming a boundary whose body is absent is the failure this replaces, so
	// the card has to lose the name and not merely the prose.
	for _, absent := range []string{
		"boundary-modify-live-backend",
		"boundary-suggest-external-comms",
		"boundary-seek-external-validation",
	} {
		if strings.Contains(string(instructions), absent) {
			t.Errorf("identity card still carries %q", absent)
		}
	}
	if !strings.Contains(string(instructions), "you hold this within a scope") {
		t.Error("identity card lost the scoped grant the request never omitted")
	}
	skills := filepath.Join(result.Bundle.Dir, "content", "skills")
	matches, err := filepath.Glob(filepath.Join(skills, "*", "boundary-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || !strings.HasSuffix(matches[0], "boundary-build-foundational-software") {
		t.Fatalf("bundle boundary skills = %v, want the scoped grant alone", matches)
	}
}

// A bundle that quietly lacks a boundary is worse than one that never had it,
// because the review surface stops telling the truth.
func TestBoundaryOmitIsRecordedInTheDecisionTrace(t *testing.T) {
	out := t.TempDir()
	result, err := Run(fixture(t, "boundary-omit.kdl"), out)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	excluded := map[string]bool{}
	for _, d := range result.Resolution.Decisions {
		if d.Kind == "profile" && d.Outcome == resolver.OutcomeExcluded {
			excluded[d.Subject] = true
		}
	}
	for _, want := range []string{
		"boundary:modify-live-backend",
		"boundary:suggest-external-comms",
		"boundary:seek-external-validation",
	} {
		if !excluded[want] {
			t.Errorf("decision trace does not record %s as excluded", want)
		}
	}
}

func TestBoundaryOmitRefusesWhatItCannotMean(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		role, omit, want string
	}{
		"unknown boundary": {"platform-eng", "no-such-boundary", "unknown boundary"},
		// An owner losing its own boundary is a larger claim than a deferrer
		// losing one, and this knob is not allowed to make it.
		"owned by the role": {"sysadmin-senior", "modify-live-backend", "is owned by role"},
		// A grant is a permission, so omitting it would widen the seat. Every
		// seat now touches every boundary, so this replaces the inactive case.
		"held within a scope": {"platform-eng", "modify-live-backend", "is held within a scope"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := &schema.Request{
				Role:              tc.role,
				Delivery:          "native-skills",
				BoundaryOmissions: []string{tc.omit},
			}
			_, err := resolver.Resolve(req, p, nil, nil)
			if err == nil {
				t.Fatalf("omitting %q for role %q was accepted", tc.omit, tc.role)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
