# Claude Code native UI surfaces

Which parts of the Claude Code terminal UI a composed identity can drive.
Verified against the installed Claude Code binary at version 2.1.221 by reading
the shipped settings schema and theme loader, not from documentation. The build
order lives in [claude-native-ui-plan.md](claude-native-ui-plan.md).

## What is available

* **Custom themes** - `~/.claude/themes/<slug>.json`, selected as
  `"theme": "custom:<slug>"`. Carries the role color into the prompt border, the
  skill and permission accents, and the Clawd mascot body.
* **Status line** - `statusLine` runs a command that receives session JSON on
  stdin. Already implemented as `agent-compose statusline`.
* **Subagent status line** - `subagentStatusLine.command` runs once per tick with
  every agent-panel row on stdin and returns `{id, content}` per line. Shipped as
  `agent-compose statusline --subagent`.
* **Spinner verbs** - `spinnerVerbs` with `mode` of `append` or `replace`, plus
  a `verbs` array. The single cheapest per-role voice signal.
* **Spinner tips** - `spinnerTipsOverride` with `excludeDefault` and `tips`,
  gated by `spinnerTipsEnabled`.
* **Session display name** - the `--name` launch flag, shown in the prompt box,
  the `/resume` picker, and the terminal title. Pure launch argument, no host
  file is touched. Shipped, with `--settings`, as
  [launch identity](claude-launch-identity.md).
* **Output styles and syntax highlighting** - both are plugin components, so a
  generated plugin can carry them alongside a theme.

## What is not available

* The ASCII logo and the welcome banner. No override string exists in the
  binary. The mascot is recolorable through the `clawd_body` and `claude`
  tokens, never redrawn.
* Borders, box drawing, and component layout. Themes carry color tokens only.
* Free per-subagent color. The subagent palette is a fixed set of eight tokens
  suffixed `_FOR_SUBAGENTS_ONLY`. A role can recolor a slot but cannot add one.

## Theme file contract

The loader accepts three keys and silently drops anything else.

* `name` - display string, falls back to the file slug.
* `base` - one of `dark`, `light`, `light-daltonized`, `dark-daltonized`,
  `light-ansi`, `dark-ansi`. Anything else falls back to `dark`.
* `overrides` - a flat map. A key is kept only when the chosen base theme
  already defines that token, and a value is kept only when it matches
  `#rgb`, `#rrggbb`, `rgb(r,g,b)`, `ansi256(n)`, or `ansi:<name>`.

Silent dropping is the trap worth designing around. A misspelled token does not
error, it just does nothing, so a generator needs to validate token names
against the base theme rather than trusting its own output.

Files over 256KB are skipped with a warning. The whole directory is watched, so
a rewritten theme file is picked up without a restart.

## Cluster CLI deny

The Claude settings fragment a native launch passes as `--settings` refuses bare
`kubectl` and `helm`, so seats change the cluster through the operator CLI's
kubectl verbs. The owner of the `modify-live-backend` boundary keeps bare `kubectl
exec` alone, until exec moves onto that operator CLI.

### Two layers

* `permissions.deny` for `kubectl` and `helm`, by name and by path, on every
  role but the owner. A deny in any tier beats every allow, so the host's
  `Bash(*)` cannot reopen it, and for the same reason it cannot carve out exec.
* A `PreToolUse` Bash hook, `agent-compose hook bash-guard`, on every role, with
  `--allow-kubectl exec` for the owner. It parses the command with
  `mvdan.cc/sh`, so a flag before the verb, `sudo`, `env`, `xargs`, `bash -c`,
  `eval`, a heredoc, a pipe into a shell, and `ssh` or `python -c` text are all
  judged by the verb they would run. A computed command on a line naming a
  cluster CLI is refused. Exit 2 blocks and shows the reason to the model.

The owner comes from the roster's boundary table, and a roster naming none
denies every role. The launch resolves `agent-compose` on `PATH` into the hook,
because a hook that cannot start does not block. Observed on Claude Code
2.1.283: exec ran, while `kubectl apply`, `helm upgrade`, and the `bash -c` form
were refused.

### What it does not cover

A `just` verb or a script file runs its own kubectl unseen. Those call sites move
to the operator CLI instead. Anything that writes a file and runs it later, or
builds the name at runtime from pieces, gets past a guard on ordinary use. A
caller-supplied `--settings` replaces the fragment whole, per
[caller precedence](claude-launch-identity.md#caller-precedence).

## Safe mode caveat

`--safe-mode` disables custom themes, keybindings, output styles, and plugins
together. Every surface above except the status line command and the session
name vanishes in a troubleshooting session. Identity that must survive safe mode
belongs in the composed context, never in the theme.

## Generated output

`agent-compose native-ui` emits a theme and settings fragment per role.
[`examples/claude-native-ui/`](../examples/claude-native-ui/README.md) is the
checked-in result for `roster:core`, and its README explains how a boundary becomes
a theme.
