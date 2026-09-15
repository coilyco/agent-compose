# agent-compose features

Inventory of what ships today.

## The roster is mounted, not embedded

The binary carries no roster. It resolves one from `AGENT_COMPOSE_ROSTER`, then
`~/.agent-compose/roster`, then the seed a package manager installed beside it,
and names every path it tried when none resolves. brew and scoop install that
seed, so a roster change needs no rebuild and an upgrade never overwrites an
edited roster in the state directory (#336). The release also ships the composed
bundle set, addressed `<role>-<tier>-<delivery>`, because housecast composes at
build time and never runs on a user's machine (#412).

## Composition moved to housecast

The roster language, the composition engine, and the eval board runner live in
`coilyco-flight-deck/housecast` as of #337. This repository consumes housecast
from Forgejo through `[tool.uv.sources]`, pinned by tag, and keeps `checks/`
to prove the Go engine still composes identical bundles until #339 deletes it.

Where composition lives is a public boundary, which is why it is recorded here
rather than only in the issue.

## Composition engine

* `agent-compose compose` turns a KDL request into an immutable bundle.
* `roster:core` describes 10 seats and 11 personalities, three [archived](role-selection.md)
  on 2026-09-15: refused by the resolver, keeping their charter, art and edges. Each seat
  melds two traits, and every boundary reaches all ten, exactly one as owner.
  [Role briefings](role-briefings.md) own each charter.
* [External person packages](person-packages.md) and
  [local personality libraries](personality.md) replace the default. Their
  manifests and fragments are authored as YAML or KDL, and a package may hold
  both while it converts (#335).
* `personality-library "roster:core"` lets an external package bind the
  [embedded core personalities](personality.md) by slug instead of vendoring
  their bodies. Roles, seats, and identity stay package-exclusive.
* `.agents/roles.kdl` owns [skill-provider repos](role-selection.md), skills, and [repository policy](repository-plan.md).
* [Three model tiers](harness-vendoring.md) declare Core role deployment
  compatibility without changing selected context.
* Materialization promotes admitted `COMPOSED.md` to native `SKILL.md`.
* Resolver traces provider and content outcomes with budgets.
* `boundary-omit` drops a defer-side boundary whose owning seat the deployment does not have, and refuses a scoped grant. See [boundary omission](boundary-omission.md).
* Atomic materialization verifies staged and reused bundles.
* Canonical skills use identity cards and compiled fallback. Role bodies cap at
  1200 words, [role boundaries](role-boundaries.md) at 200 words per side.
* [Role adjacency](role-boundaries.md) names each role's two likeliest absorptions.
* [Attribute acts](kdl-contracts.md) give every role, personality, and
  boundary side three named things to run. The identity card renders them, and
  a boundary's acts follow the side the seat holds rather than the owner's.

## Load-point projection

* `agent-compose project` places verified bundles transactionally at repo or
  container-home load points for four harnesses.
* Sidecar ownership protects foreign files and restores prior owned state.

## Launch-time refresh

* [Launch](native-role-launch.md) adds color, an Enter gate, and a Codex intro.
* A Claude launch passes [identity flags](claude-launch-identity.md), not files.
* Refresh uses validated fallback unless `external-only` forbids it.
* `launch` starts a [second seat from inside a session](native-role-launch.md),
  one hop deep and never over the caller's own load points.

## Inspection

* `config validate` strictly checks staged host configuration without writes.
* `agent-compose describe` renders a collapsible decision tree. `--why`
  follows one item from consideration to outcome.
* `agent-compose diff` reports semantic changes. `verify` checks entry points,
  delivery, traces, and selected identities.
* [Catalogues and export](skill-catalogues.md) provide inspection,
  reproducible archives, and logical content diff.
* `compose` renders the role metadata and the identity texture. `--explain`
  adds the briefing, the credits, the expressions, and the decisions.
* [Evaluation](evaluation.md) derives the board from the roster, runs it with
  housecast's `evalkit`, and grades it by hand with `housecast grade`. No mechanical
  scorer anywhere in the loop.
* [V2 migration](release.md) maps v1 roles without aliases.
* TTY colors use canonical identity and pass an OKLab legibility gate.
  Redirects and `NO_COLOR` stay plain.

## Identity surfaces

* [Identity primitives](identity.md) give every personality a `species`, a
  `color_word`, an emblem name list, a `motif`, a `geometry`, and a prose
  `body`. A seat's creature is derived from its meld rather than
  authored, so two seats sharing a personality share an animal.
* [Identity renderers](statusline.md) cover the palette, overlays, the
  `acompose statusline` row, `--subagent` rows, and the
  [short id](whoami.md). [`whoami`](whoami.md) prints it. The
  [overlay](overlay.md) carries a separation-solved window `background` beside
  the accent, so a renderer no longer tints its own.
* [`native-ui`](claude-native-ui-surfaces.md) emits per-role Claude Code themes.

## Roster artifact and cascade

* `agent-compose roster --out <dir>` renders lazy-loaded role and personality
  skills plus [native adaptation](native-adaptation.md).
* Bare convergence emits deterministic [`person.json`](person-contract.md).
* `cascade` emits harness doctrine and the role/residency
  `repository-plan.yaml` with sealed provenance.
* `bundle materialize` returns a verified role/harness bundle with provenance.
* Bare `acompose` converges hosts. `--reapply` forces the layout, `--verbose`
  traces `source => destination`, `-- <command>` refreshes then execs, and Ward
  smoke proves idempotence.
* [`appendix`](cascade.md) composes inline or file tail blocks, optionally scoped to named roles.
* [`source_delivery`](cascade.md) emits a source as an `@path` import rather than inlining it, so a source the repository cascade also delivers is loaded once; a missing target fails convergence rather than shipping a pointer to nothing.
* [Local skill catalogues](skill-catalogues.md) consume AOS roots, each addressed by a forge-qualified source that travels with the root, so a bare owner/repo real on two forges fails rather than picking one, and one name meaning two different skills stops the converge naming both. An [org grant](skill-selectors.md) admits a whole owner from the compiled set, open to repositories added later and bounded only by its selectors, and `skill_requests` reaches one skill by address, failing the converge by name when it resolves to nothing.
* [Release](release.md) publishes unreleased deltas under a hold, with roll-forward recovery.
* A bundle carries a [voice profile](manifest-schema.md) merging a skill's rules with the seat's avoid banks, so a Refuse reaches a checker.

## See also

* [../README.md](../README.md) - product boundary and current status.
* [../AGENTS.md](../AGENTS.md) - repo-specific operating rules.
* [../justfile](../justfile) - development recipes.
* [Catalog trifecta](https://forgejo.coilysiren.me/coilyco-flight-deck/agentic-os/src/branch/main/docs/features-release-tooling.md).
