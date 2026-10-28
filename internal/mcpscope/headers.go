package mcpscope

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// unreadableHeaders maps a header to its variables when every ${VAR} in the value
// is unset or blank with no default. A present header turns Claude's OAuth fallback off.
func unreadableHeaders(s Server, lookup func(string) (string, bool)) map[string][]string {
	dropped := map[string][]string{}
	for header, value := range s.Headers {
		var names []string
		for _, ref := range envRef.FindAllStringSubmatch(value, -1) {
			if current, _ := lookup(ref[1]); strings.TrimSpace(current) != "" || strings.TrimSpace(ref[2]) != "" {
				names = nil
				break
			}
			names = append(names, ref[1])
		}
		if names != nil {
			dropped[header] = names
		}
	}
	return dropped
}

func (sel Selection) headerLookup() func(string) (string, bool) {
	if sel.lookup == nil {
		return os.LookupEnv
	}
	return sel.lookup
}

// withReadableHeaders is s without the headers whose variables are unreadable.
// The remaining ${VAR} stay literal, so no secret reaches the config file.
func (sel Selection) withReadableHeaders(s Server) Server {
	dropped := unreadableHeaders(s, sel.headerLookup())
	if len(dropped) == 0 {
		return s
	}
	kept := make(map[string]string, len(s.Headers))
	for header, value := range s.Headers {
		if _, skip := dropped[header]; !skip {
			kept[header] = value
		}
	}
	s.Headers = kept
	return s
}

// HeaderOmissions is one line per header left off the config, so the launch
// names the cause. The value is never read into the line.
func (sel Selection) HeaderOmissions() []string {
	var lines []string
	for _, name := range sel.Selected {
		s := sel.servers[name]
		dropped := unreadableHeaders(s, sel.headerLookup())
		for _, header := range sortedKeys(s.Headers) {
			if variables, ok := dropped[header]; ok {
				lines = append(lines, fmt.Sprintf(
					"agent-compose: MCP %s: no %s header, %s is unset or empty. The server starts unauthenticated. If its SSM read failed because the AWS SSO session expired, run `aws sso login` and relaunch",
					name, header, strings.Join(variables, ", ")))
			}
		}
	}
	return lines
}
