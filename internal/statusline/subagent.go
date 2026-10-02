package statusline

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/coilyco/agent-compose/v2/internal/bundle"
	"github.com/coilyco/agent-compose/v2/internal/project"
)

// SubagentRequest is the tick payload Claude Code writes to stdin. The command
// runs once per tick with every live row, not once per row.
type SubagentRequest struct {
	Columns int            `json:"columns"`
	Tasks   []SubagentTask `json:"tasks"`
}

// SubagentTask is one agent-panel row. Only the fields this renderer reads are
// declared; the harness sends more, and unknown keys stay ignored.
type SubagentTask struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Label  string `json:"label"`
	CWD    string `json:"cwd"`
}

// SubagentRow is one decoration. Claude Code validates {id, content} strictly
// and drops any line that fails, so both fields stay non-empty.
type SubagentRow struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// RenderSubagents decorates each row with the identity of the projection at
// that row's own working directory. See docs/statusline.md.
func RenderSubagents(input io.Reader, opts Options) (string, error) {
	var request SubagentRequest
	if err := json.NewDecoder(input).Decode(&request); err != nil {
		return "", fmt.Errorf("read subagent status line request: %w", err)
	}
	cache := map[string]string{}
	var out strings.Builder
	for _, task := range request.Tasks {
		if strings.TrimSpace(task.ID) == "" {
			continue
		}
		target := strings.TrimSpace(task.CWD)
		if target == "" {
			target = strings.TrimSpace(opts.Target)
		}
		if target == "" {
			target = "."
		}
		content, ok := cache[target]
		if !ok {
			content = subagentIdentity(target, opts)
			cache[target] = content
		}
		if content == "" {
			continue
		}
		encoded, err := json.Marshal(SubagentRow{ID: task.ID, Content: content})
		if err != nil {
			return "", fmt.Errorf("encode subagent row %q: %w", task.ID, err)
		}
		out.Write(encoded)
		out.WriteByte('\n')
	}
	return out.String(), nil
}

// subagentIdentity is the per-row row text: the personality emblems and the
// painted seat, with the role and harness the projection actually chose.
func subagentIdentity(target string, opts Options) string {
	projection, _ := project.FindProjection(target)
	if projection.Bundle == "" {
		return ""
	}
	manifest, err := bundle.ReadManifest(projection.Bundle)
	if err != nil {
		return "⚠ composition unreadable"
	}
	marks := personalityMarks(manifest, opts)
	if marks == "" {
		marks = "🧩"
	}
	return fmt.Sprintf(
		"%s %s // %s@%s",
		marks,
		paint(manifest.Color, seatDisplayName(manifest, projection.Layout), opts),
		manifest.Role,
		projection.Layout,
	)
}
