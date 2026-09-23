package main

import (
	"slices"
	"testing"

	"github.com/coilyco-flight-deck/agent-compose/v2/internal/launch"
)

func TestRoleAttributionEnvNamesTheRole(t *testing.T) {
	env := roleAttributionEnv("prod-director")
	want := launch.AttributionRoleEnv + "=prod-director"
	if !slices.Contains(env, want) {
		t.Fatalf("roleAttributionEnv(prod-director) = %v, want it to contain %q", env, want)
	}
}

func TestAnUnknownRoleAttributesNothing(t *testing.T) {
	// Stamping a wrong role is worse than stamping none, because a reader
	// cannot tell an absent trailer from an incorrect one.
	for _, role := range []string{"", "   ", "\t"} {
		if env := roleAttributionEnv(role); len(env) != 0 {
			t.Errorf("roleAttributionEnv(%q) = %v, want nothing", role, env)
		}
	}
}

func TestEveryDeployedRoleRoundTrips(t *testing.T) {
	for _, role := range []string{
		"platform-eng", "sysadmin", "scientist", "frontend-eng", "game-dev", "prod-director", "dev-advocate",
	} {
		env := roleAttributionEnv(role)
		if len(env) != 1 || env[0] != launch.AttributionRoleEnv+"="+role {
			t.Errorf("roleAttributionEnv(%q) = %v", role, env)
		}
	}
}
