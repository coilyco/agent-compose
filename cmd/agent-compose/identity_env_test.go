package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/person"
)

func loadShipped(t *testing.T) func() (*person.Person, error) {
	t.Helper()
	return person.Load
}

func TestLaunchIdentityIsEmptyWithoutTheVariableAndLoadsNothing(t *testing.T) {
	t.Setenv(identityEnv, "")
	name, err := launchIdentity(func() (*person.Person, error) {
		t.Fatal("the roster was loaded with no identity to check")
		return nil, nil
	}, "eng-platform")
	if err != nil || name != "" {
		t.Fatalf("got %q, %v", name, err)
	}
}

func TestLaunchIdentityAcceptsAnAlternateAndRenamesThePerson(t *testing.T) {
	p, err := person.Load()
	if err != nil {
		t.Fatal(err)
	}
	alternate := p.RoleIdentityNames("sysadmin-senior")[1]
	t.Setenv(identityEnv, alternate)
	name, err := launchIdentity(func() (*person.Person, error) { return p, nil }, "sysadmin-senior")
	if err != nil || name != alternate {
		t.Fatalf("got %q, %v, want %q", name, err, alternate)
	}
	if got := p.RoleIdentityName("sysadmin-senior"); got != alternate {
		t.Errorf("the person answers to %q, want %q", got, alternate)
	}
}

func TestLaunchIdentityRefusesAnotherRolesName(t *testing.T) {
	t.Setenv(identityEnv, "Griffin-Goose")
	_, err := launchIdentity(loadShipped(t), "sysadmin-senior")
	if err == nil || !strings.Contains(err.Error(), identityEnv) || !strings.Contains(err.Error(), "not one of sysadmin-senior's names") {
		t.Fatalf("got %v", err)
	}
}

func TestLaunchIdentityNamesTheVariableWhenTheRosterWillNotLoad(t *testing.T) {
	t.Setenv(identityEnv, "Bison-Ox")
	_, err := launchIdentity(func() (*person.Person, error) { return nil, errors.New("no roster") }, "sysadmin-senior")
	if err == nil || !strings.Contains(err.Error(), identityEnv) || !strings.Contains(err.Error(), "no roster") {
		t.Fatalf("got %v", err)
	}
}

// A child seat must pick its own name, so a launch clears the variable.
func TestALaunchClearsTheIdentityVariable(t *testing.T) {
	found := false
	for _, name := range nativeLaunchSelectorEnv {
		found = found || name == identityEnv
	}
	if !found {
		t.Errorf("%s is not cleared before exec, so a nested launch would inherit it", identityEnv)
	}
}
