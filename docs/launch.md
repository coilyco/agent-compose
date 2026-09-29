# Launch-time refresh

A `--` on the compose verb freshens context, then hands the process to the
real command:

```
agent-compose compose <request.kdl> --layout <name> --target <dir> -- <command> [args...]
agent-compose compose -- <command> [args...]
```

The first form refreshes a bundle and its projection before exec. The bare
form converges the host (roster plus cascade) before exec. Refresh is
compose plus project. Both halves are already idempotent - the
bundle cache reuses identical inputs and projection replaces only its own
files - so a warm launch is a no-op that validates and execs. The warm path
runs in single-digit milliseconds on the reference host, well inside the
250 ms budget the test suite enforces.

The assigned-role shorthand is separate:

```
acompose <role> <harness> [harness arguments...]
```

It resolves eligible host providers, selects the complete role bundle, and
projects through the harness layout before exec. Unlike generic request
refresh, an assigned-role launch does not fall back to a prior projection.
Starting with the wrong stale role would violate the caller assignment.
Launch consumers may pass their model-tier decision through
`AGENT_COMPOSE_MODEL_TIER`. Agent Compose defaults it to `frontier` and clears
the launch-only variable before handing control to the harness. The retired
`AGENT_COMPOSE_MODEL_CLASS` variable is ignored and stripped during migration.
`AGENT_COMPOSE_RUNTIME_HOME` similarly selects a prepared session home. Agent
Compose switches `HOME`, `CODEX_HOME`, `XDG_CONFIG_HOME`, and Claude's config
directory only after composition, then clears the control variable.

## Recursion guard

Launch sets `AGENT_COMPOSE_LAUNCH` in the child environment before exec. A
nested launch sees the sentinel, skips refresh, and execs straight through,
so a shadowed binary that wraps a harness can never recurse into itself.

The assigned-role verb reads the same sentinel and starts a second seat rather
than execing through, because starting a seat is a deliberate act rather than a
wrapper re-entering itself. It is bounded by `AGENT_COMPOSE_LAUNCH_DEPTH` to one
hop. See [native-role-launch.md](native-role-launch.md).

## Failure behavior

**A generic refresh never blocks a launch that has context to run with.** When
compose or project fails and the target holds a validated last-known-good
projection, launch warns on stderr and proceeds with it. Without one it
aborts. An effective `external-only` person policy disables this fallback,
because the prior projection may have used the embedded package.

**An assigned-role launch classes each startup step as warn or refuse.** A
step that shapes behavior warns and launches, and a step that bounds reach
refuses, since a plain Claude or Codex seat loads the whole user-level MCP set.
`startup_policy.go` holds the classes, and
a drift test holds this list to it:

* `launch-depth` - refuse - the one-hop nested-launch bound.
* `host-converge` - warn - skill catalogs and the composed base.
* `state-directory` - refuse - `~/.agent-compose`, where the MCP config lands.
* `person` - warn - the host person policy. Failure falls back to the embedded person.
* `operating-base` - warn - a session home's operating base and appendix.
* `projection-guard` - refuse - a nested launch projecting over its own load points.
* `role-composition` - warn - identity, personality, voice, skills, doctrine and UI settings, in a staged session home.
* `role-composition-repo-scope` - refuse - the same without a session home, where the working directory may hold another role's projection. It also refuses a working directory at or above a repository, such as the projects root or an org directory, since every session beneath would inherit the role.
* `card` - warn - the identity card.
* `launch-pause` - warn - the Press Enter gate.
* `selector-environment` - warn - clearing the parent's selectors.
* `runtime-home` - warn - pointing the harness at the session home.
* `telemetry` - warn - Claude metrics export.
* `mcp-scope` - refuse - the role MCP set, for every harness: Claude's `--strict-mcp-config`, Codex's disabled servers, goose's `--no-profile` set, or OpenCode's inline config.

**The MCP scope fails closed.** No readable roster or no
`~/.mcporter/mcporter.json` refuses, where both used to launch unscoped, and an
empty server set is not a fallback either. A caller's own `--mcp-config` is a
scope and is used as given. Each harness's form is in
[the MCP role scope](claude-launch-identity.md#mcp-role-scope).

**A degraded launch stays visible**, since the harness repaints over stderr.
Each skipped step prints `<step> did not load`. On a terminal, stdout then
carries `ESC ] 7750 ; agent-compose ; degraded=<steps> BEL`, which aterm shows
as the session's `degraded` field in `aterm agents` and the dashboard. Claude
gets the list through `--append-system-prompt` unless the caller set one, and
Codex through `-c developer_instructions=`. `--spec-out` fails instead, having
no composition to write.

## Concurrency

Concurrent identical launches converge on one cache entry: the materializer
stages beside the target and the rename loser reuses the winner. Concurrent
launches for different requests or targets stay isolated by construction -
distinct cache keys, distinct target directories, and a per-target lock file
(`.agent-compose/lock`) serializing projection writes.

## Wrapper installation requirements

Binary shadowing rollout belongs to the infrastructure repo. A wrapper that
fronts a harness must exec `agent-compose compose` with its fixed request,
layout, and target, forward the original argv after `--`, and resolve the
real harness binary through normal PATH lookup - the sentinel, not PATH
surgery, is what prevents recursion. No rollout code lives here.

## See also

* [projection.md](projection.md) - the load-point layer launch drives.
* [native-role-launch.md](native-role-launch.md) - assigned native sessions.
* [bundle-protocol.md](bundle-protocol.md) - cache identity and atomicity.
* [architecture.md](architecture.md) - composition inputs and ownership.
