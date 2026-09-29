# The staged home and the task runner

The staged home a launch builds, and the just task runner around it.

## Staged-home handoff

A composition adapter can turn one immutable bundle into a launcher-neutral
home tree without teaching agent-compose the launcher's manifest, container,
permission, or lifecycle model.

The adapter starts with an empty private directory and runs:

```text
agent-compose verify <bundle>
agent-compose project <bundle> \
  --layout <agent> \
  --scope home \
  --target <empty-home>
```

The selected agent controls the only accepted load points:

* `claude` - `.claude/CLAUDE.md` and optional `.claude/skills/`
* `codex` - `.codex/AGENTS.md` and optional `.agents/skills/`
* `goose` - `.config/goose/.goosehints` and optional `.agents/skills/`
* `opencode` - `.config/opencode/AGENTS.md` and optional `.agents/skills/`

Agent-compose also writes `.agent-compose/` projection state. That directory
can contain `projection.json` and a platform lock file. It is not agent
context. After `project` returns and no other process can use the private
staging directory, an adapter removes `.agent-compose/`, validates that only
the selected load points remain, and wraps those files in its own handoff
schema.

The adapter owns that new schema. Agent-compose never parses or emits it.
Agent-compose also never starts a container, selects runtime authority, mounts
a tool, or invokes a launch consumer.

The role in the source bundle selects context. A matching role slug in another
system does not transfer permissions or merge authority into the projected
home.
When the bundle was composed from
[role-scoped providers](role-selection.md), home projection preserves
the same selected skill inventory as native projection. Projection does not
re-resolve providers or mutate the immutable input bundle.

The generic projection remains useful on its own through `agent-compose` and
the `acompose` host entrypoint. No composition root is required for native use.

## just, the task runner

Every development verb is a recipe in the repo-root [justfile](../justfile).
`just` alone lists them.

Per-repo `ward exec` is retired, under the principle that ward is out-of-band
flight control, so a repo should mention it in passing rather than route its
whole build through it. The pattern is
now the fleet norm.

### What changed

All 30 verbs moved with **identical names and identical command lines**. The
`commands:` block is gone, and so is the `.ward/ward.yaml` that held it.

Arguments pass straight through, so the `--` separator is retired:

```
just evalkit-export evaluations/pilot/ops-board-2026-08-12-regraded
just test
```

### And then `.ward/ward.yaml` went too

It outlived the verbs by carrying a `catalog:` block, because
`check_catalog_block` pinned that exact path and `catalog-trifecta` wanted
README, AGENTS, and FEATURES to link it. Both are retired and the cross-repo
graph that read the block is gone, so the file is deleted fleet-wide.

### One line of comment per recipe

`just` reads only the **last** comment line above a recipe, so a wrapped
description silently truncates to its tail. agentic-os#1048 found this the
hard way. Keep descriptions on one line.

### What was lost

**The clean-tree gate.** `ward exec` refused a repo verb while the working tree
was dirty, so an audit row could be reconstructed from git history. `just` has
no equivalent and this repo now has none.

Nothing here depended on it: no non-doc reference to `--audit-override-dirty`
or the gate exists, which matches what agentic-os#1048 found in its own
corpus. Recorded because it is a real reduction, not because it broke
anything.

### Related

* [Release](release.md) - the release verbs, now recipes.
