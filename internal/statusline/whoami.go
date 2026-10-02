package statusline

import (
	"encoding/json"

	"github.com/coilyco/agent-compose/v2/internal/agentid"
	"github.com/coilyco/agent-compose/v2/internal/bundle"
	"github.com/coilyco/agent-compose/v2/internal/person"
)

// Whoami returns what this session calls itself: `Angie uz86`, or "" with
// no composition rather than inventing one. See docs/whoami.md.
func Whoami(opts Options) (string, error) {
	projection := resolveProjection(opts.Target)
	if projection.Bundle == "" {
		return "", nil
	}
	manifest, err := bundle.ReadManifest(projection.Bundle)
	if err != nil {
		return "", err
	}
	return person.WithShortID(
		seatDisplayName(manifest, projection.Layout),
		agentid.FromEnv(),
	), nil
}

// Record is Whoami plus the bundle fingerprint, so a transcript joins to an
// exact composition. Metadata only. See docs/manifest-schema.md.
type Record struct {
	Format string `json:"format"`
	Seat   string `json:"seat"`
	Role   string `json:"role"`
	Bundle string `json:"bundle"`
}

// WhoamiJSON returns "" with no composition, matching Whoami rather than
// synthesising one.
func WhoamiJSON(opts Options) (string, error) {
	projection := resolveProjection(opts.Target)
	if projection.Bundle == "" {
		return "", nil
	}
	manifest, err := bundle.ReadManifest(projection.Bundle)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(Record{
		Format: "agent-compose.whoami.v1",
		Seat:   person.WithShortID(seatDisplayName(manifest, projection.Layout), agentid.FromEnv()),
		Role:   manifest.Role,
		Bundle: bundle.Fingerprint(*manifest),
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}
