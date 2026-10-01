# Cascade

The cascade turns doctrine sources into each harness's global context when
`~/.agent-compose/agent-compose.yaml` exists. Missing config is a no-op.

Bare `acompose` summarizes its roster, outputs, load points, repository plan, skill
links, and repaired drift. `--reapply` recreates outputs and links, and `--verbose`
emits each source, override, manifest, and link as `source => destination`.

`person_policy: external-only` requires `person_source`. A bad package aborts
before roster or cascade projection can restore the embedded default.

All state lives under `~/.agent-compose`: config, outputs, repository plan, roster,
and cache. A legacy `~/.config/agent-compose` migrates on first use and leaves
a compatibility symlink through the cutover tracked in agentic-os#618.

Explicit `sources` compose first in listed order. Each `roots` entry then adds
sorted `AGENTS.COMPOSE.md` files. That filename marks always-global doctrine
that harness context does not also load.

## Selection and rewrites

A machine may declare `scopes`, and a source declares its own in YAML
frontmatter and composes only when the two intersect. Omitting the machine
key disables filtering entirely. Under active filtering an untagged source
never leaks in. Frontmatter `harnesses` restricts a source to named
harnesses. Composed bodies are rewritten for their new home: frontmatter
stripped, `## See also` navigation dropped, and relative markdown links
absolutized against the source's own directory.

`source_delivery: import` emits an `@path` pointer, not a body, so a doubly-delivered source loads once.
It reaches named `sources`, which must resolve; a `roots` entry inlines and keeps its override.

A sibling `AGENTS.<harness>.md` beside a source patches it for one harness:
sections replace by verbatim heading, new headings append, and an ambiguous
heading fails the compose loudly. When harness slices diverge - by selection
or by override - output splits into `COMPOSED.<harness>.md` files. Identical
slices share one `COMPOSED.md`, and obsolete banner-carrying outputs are
removed on convergence.

## Appendix

`appendix` composes configured blocks after every source, so they carry the
tail of the composed context. Each entry holds exactly one of `text` (inline
markdown) or `path` (a file), and an entry that carries both or neither fails
the config load.

```yaml
appendix:
  - text: |
      ## Checkin dashboard
      Open these before answering a checkin.
  - path: ~/.config/agent-compose/appendix/deploy.md
    roles: [platform, sysadmin]
```

A native role bundle carries the appendix apart from the sources, because a
heading-keyed rewrite of the source body would swallow it. It lands at the tail
of the bundle's instructions. A `path` entry is rewritten the way a source is,
and inline `text` composes verbatim.

An entry with no `roles` is global and reaches every composed output. An entry
naming `roles` composes only for those roles, which by construction leaves it
out of the role-less harness load point: a session-home launch renders its own
operating base per role and is the only reader. A repo-scope launch reads the
host file, so it sees global blocks alone.

Cascade never loads a person, so `roles` is checked for slug shape and nothing
more. A native launch does resolve one, and warns there for any configured slug
the roster does not define, because a block that composes for no one otherwise
passes silently. A missing `path` warns and skips on convergence, exactly as a
missing source does, and fails under `--check`.

## Host config defaults

An absent key resolves from the host, and a set key always wins. Only
`operating_context` is required.

* `sources` - each `operating_context` repository's `AGENTS.md`, imported. A
  repository without one is skipped, and naming `sources` turns this off.
* `roots` - `sources` beside the config. `roots: []` opts out.
* `skill_catalog_manifest` - `~/.config/aos/catalogues.json` when present. It
  projects AOS-verified roots, under the [trust contract](skill-catalogues.md).
* `load_points`, `skill_load_points` - the layout table. `opencode: true` opts a
  non-cascade harness in at its table path, and an unknown name fails.

## Outputs

Each configured load point (claude and codex by default, others via
`load_points`, `null` to opt out) is symlinked at its harness's composed
file, backing up any pre-existing regular file to `.bak`. The strict
[`repository-plan.yaml`](repository-plan.md) is emitted beside the composed
output. It compiles operating context, global policy, role policy, provider
uses, and resident-only pins from trusted KDL with sealed input provenance.
See [Repository plan](repository-plan.md).

`--dry-run` previews only real changes. `--check` verifies every output
against a fresh compose and fails with a diff on drift. Writes happen only
on change, so a converged host recomposes silently.
`agent-compose config validate <path>` checks staged host configuration and a
linked strict provider document without writes.

## Native skill roots

Bare compose can also link authored skill catalogs into harness-native skill
directories through [`skill_load_points`](skill-selectors.md). Native skill
linking uses the compiled residency set from `repository-plan.yaml`.
Repositories contribute `.agents/skills`. The compiled set precedes verified
local catalogues. Existing unowned entries win. Missing entries warn and skip,
while other inspection failures remain fatal. Agent-compose records links in
`~/.agent-compose/skill-mounts.json` and removes only stale links that still
match that ownership record. Fleet pointer aggregation, conditional category
gating, and per-repo capability pulls remain rollout policy outside this
substrate operation.

## See also

* [integration.md](integration.md) - how roster and cascade fit together.
* [repository-policy.md](repository-plan.md) - strict repository grammar and projections.
* [projection.md](projection.md) - repo and home load-point projection.
