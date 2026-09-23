// Package roleslug maps a retired role slug to the slug that replaced it, so a
// launch or a provider roles.kdl still naming the old slug keeps resolving while
// the fleet migrates. Temporary by design: teable:coilyco-flight-deck/agent-compose#8086.
package roleslug

import "strings"

var retired = map[string]string{
	"platform":        "platform-eng",
	"science":         "scientist",
	"frontend":        "frontend-eng",
	"gamedev":         "game-dev",
	"director":        "prod-director",
	"manager":         "prod-manager",
	"advocate":        "dev-advocate",
	"senior-sysadmin": "sysadmin-senior",
	"junior-sysadmin": "sysadmin-junior",
}

// Canonical trims role and returns its current slug, or role unchanged when it
// is not a retired one.
func Canonical(role string) string {
	role = strings.TrimSpace(role)
	if current, ok := retired[role]; ok {
		return current
	}
	return role
}
