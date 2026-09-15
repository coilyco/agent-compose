package catalogmanifest

import (
	"fmt"
	"net/url"
	"strings"
)

// Source is one catalogue's origin, qualified by the forge serving it.
// Why a bare owner/repo will not do: docs/skill-catalogues.md.
type Source struct {
	Forge string
	Owner string
	Repo  string
	Path  string
	Ref   string
}

// String is the recorded form: host-qualified, no scheme, ref retained.
func (s Source) String() string {
	recorded := strings.Join([]string{s.Forge, s.Owner, s.Repo, s.Path}, "/")
	if s.Ref != "" {
		recorded += "@" + s.Ref
	}
	return recorded
}

// SkillAddress names one skill the way a request reaches it, dropping the
// catalogue's own path.
func (s Source) SkillAddress(name string) string {
	return strings.Join([]string{s.Forge, s.Owner, s.Repo, name}, "/")
}

// parseSource qualifies a bare owner/repo against fallback. A source carrying
// its own scheme ignores fallback.
func parseSource(raw, fallback string) (Source, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Source{}, fmt.Errorf("names no source")
	}

	body, ref := raw, ""
	// Rightmost @, because a ref cannot contain one and a host can.
	if at := strings.LastIndex(raw, "@"); at >= 0 {
		body, ref = raw[:at], raw[at+1:]
		if ref == "" {
			return Source{}, fmt.Errorf("source %q ends in @ with no ref", raw)
		}
	}

	forge := ""
	if scheme := strings.Index(body, "://"); scheme >= 0 {
		parsed, err := url.Parse(body)
		if err != nil {
			return Source{}, fmt.Errorf("source %q is not a usable URL: %w", raw, err)
		}
		if parsed.Host == "" {
			return Source{}, fmt.Errorf("source %q carries a scheme but no host", raw)
		}
		forge = parsed.Host
		body = strings.TrimPrefix(parsed.Path, "/")
	} else {
		if fallback == "" {
			return Source{}, fmt.Errorf(
				"source %q names no forge and no default forge is declared: "+
					"give the manifest a top-level \"forge\", or write the source "+
					"as a full URL",
				raw,
			)
		}
		forge = fallback
	}

	segments := strings.Split(strings.Trim(body, "/"), "/")
	if len(segments) < 3 {
		return Source{}, fmt.Errorf(
			"source %q must name owner, repo and a catalogue path",
			raw,
		)
	}
	for _, segment := range segments {
		if segment == "" {
			return Source{}, fmt.Errorf("source %q has an empty path segment", raw)
		}
	}
	return Source{
		Forge: forge,
		Owner: segments[0],
		Repo:  segments[1],
		Path:  strings.Join(segments[2:], "/"),
		Ref:   ref,
	}, nil
}

// parseForge reduces a declared forge to the host a record names, accepting
// both the URL and bare-host forms.
func parseForge(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("forge %q is not a usable URL: %w", raw, err)
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("forge %q carries a scheme but no host", raw)
		}
		return parsed.Host, nil
	}
	if strings.ContainsAny(raw, "/@") {
		return "", fmt.Errorf("forge %q must be a host or a URL, not a path", raw)
	}
	return raw, nil
}
