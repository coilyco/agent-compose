package bundle

import (
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/resolver"
	"github.com/coilyco/agent-compose/v2/internal/schema"
)

func TestVerifyRepositoryDecisionsRejectsUnsortedManifest(t *testing.T) {
	manifest := &Manifest{Repositories: []schema.RepositorySelection{
		{Identity: "owner/two", Source: "policy", Scope: "role", Reason: "selected"},
		{Identity: "owner/one", Source: "policy", Scope: "role", Reason: "selected"},
	}}
	trace := &Trace{Decisions: []resolver.Decision{
		{Subject: "repository:owner/one", Kind: "repository", Source: "policy", Outcome: resolver.OutcomeSelected, Reason: "selected"},
		{Subject: "repository:owner/two", Kind: "repository", Source: "policy", Outcome: resolver.OutcomeSelected, Reason: "selected"},
	}}
	err := verifyRepositoryDecisions(trace, manifest)
	if err == nil || !strings.Contains(err.Error(), "strictly sorted") {
		t.Fatalf("unsorted repository error = %v", err)
	}
}
