---
ward:
  workflow: pull-request-and-merge
---
# Agent instructions

## Scope

Agent-compose delivers and launches agent context for native or isolated
harness consumers. It materializes bundles, projects them at harness load
points, and runs the launch path. **It owns composition**: the Go engine here
(`cmd/agent-compose`, `internal/person`, `internal/compose`, and neighbors) is
the canonical roster language and composer. It briefly moved to
coilyco-flight-deck/housecast under #337, and moved back once the Go engine
caught up to what that repository's Python port had grown - housecast#8041.
housecast is now only the role-agnostic grader this repository's `evalkit` hands
its board to (housecast#7961). It is public source and embeds Kai's
public-safe portfolio roster, personalities, and composition defaults. It also
accepts one complete external person package that replaces that default for an
independent deployment. Keep private identity detail, machines, credentials,
and deployment values out of the repo.

## Project shape

The repository ships a Go engine under `cmd/agent-compose` and `internal/`,
with contract fixtures in `testdata/contracts`. The shipped inventory is
[`docs/FEATURES.md`](docs/FEATURES.md). The KDL schema, bundle format, and
package layout land together so the code never precedes its public boundary.

## Repo boundaries

* `agent-compose` owns the compiler, schema, resolver, cache, bundle format,
  harness adapters, diagnostics, and Kai's public-safe default person package,
  including role methods determined by that package's cross-role policy.
* Each selected person package owns its operating purpose, personality
  catalog bindings, compatibility, and selection policy.
* `agent-compose` owns the personality invariant and canonical personality
  definitions alongside the person configuration that binds them.
* `agentic-os` owns reusable knowledge sources, general skills, capability
  providers, and editorial validators. It consumes person-owned role methods
  without carrying source copies.
* An external person package stays outside this public repo and fully replaces
  the embedded default. Private overlays may add scoped instructions to the
  selected package without redefining its content.
* Launch consumers own execution permissions, role authority, runtime-fact
  resolution, and their handoff schemas. Shared role slugs do not transfer
  permission ownership into agent-compose.
* `infrastructure` owns installation, binary shadowing rollout, host paths, and
  fleet convergence.
* Product repos are not an agent-compose concept. A repo may host capability
  files that a source locator references, and it owns whatever local skills
  only that repo needs.

## Commands

Route development through just using the [justfile](justfile). The
current command surface is:

* `just smoke` / `smoke-verbose` - test isolated host convergence.
* `just test` - run all repository validation (Go tests plus hooks).
* `just build` / `fmt` / `lint` / `install` / `tidy` - Go engine verbs.
* `just pre-commit` - explicit spelling of the hook sweep alone.

Do not invoke a language tool that has no verb here. Add new build, test,
lint, and install verbs to the justfile before using them.

## Validation

Run `just test` before every commit. The agentic-os hook catalog enforces
the documentation trifecta, flat docs, cross-links, public-safe prose, comment
discipline, and secret scanning. Add focused engine tests with each executable
slice once implementation begins.

## Safety

Context is not a credential transport. Bundles may contain instructions,
skills, and declarative routing metadata, but never tokens, auth stores, opaque
host identifiers, or mutable harness state. Materialized bundles are immutable
and mounted read-only. A failed refresh must not partially replace a known-good
bundle.

## Cross-repo contracts

The input contract names the requested role, delivery, optional person package,
and capability sources without importing launcher policy. The role activates
its complete ordered personality set. The output contract is a manifest plus a
filesystem tree that consumers treat as opaque. Agent-compose combines exactly
one person package with capability providers and scoped overlays. Native
harness wrappers and staged-home adapters consume the output contract. Changes
to either contract need compatibility tests against both paths before release.

## Release

Canonical development lives on Forgejo and releases run on GitHub Actions. Issues live in the Teable
tracker, cited as `teable:<owner>/<repo>#<n>`. Every push to
`main` queues the single-stage release workflow, validates the commit, bumps the
minor version, and publishes version-stamped cross-platform binaries plus
Homebrew and Scoop metadata. Manual dispatch may select patch or major instead.
Protocol compatibility is deliberately thin: consumers check the manifest
`format` marker and nothing else.

## Agent rules

<!-- BEGIN managed by agentic-os/scripts/apply-git-workflow.py -->
### Git workflow

**This repo runs the `pull-request-and-merge` lane**, declared as `ward.workflow` in this file's frontmatter. The agent commits to a task branch, pushes it, opens a Forgejo pull request, and **merges that pull request itself** once it is green. The author of the code is the one who merges it. Opening the pull request is a step, never the stopping point.

The fleet runs one lane, and it authorizes the agent end to end. Pushing straight to `main` is over: `merge-remote-main` is retired, so no repo can declare its way back to one.

* `pull-request-and-merge` - the agent commits to a task branch, pushes it, opens a pull request, and merges that pull request itself once it is green.

**Every lane slug names what the AGENT does, never what someone else does.** `pull-request-and-merge` carries the merge because the agent that authored the code merges its own pull request. `pull-request` drops `-and-merge` because the author stops at the pull request and the director merge lane takes over. Reading `pull-request-and-merge` as "someone else merges it later" inverts the two and leaves finished work sitting unmerged.

**These actions are pre-authorized on every lane, and the agent MUST take them without asking first.** Committing, creating a branch, pushing a branch, pushing the lane's own destination, and opening a pull request are ordinary reversible work, not the destructive wall that earns a question. Stopping to ask is how a turn ends with the work stranded in a dirty worktree.

* **ALWAYS commit** in-scope work and **ALWAYS push** it to the canonical remote before pausing, reporting a checkpoint, handing off, or ending a turn. A local-only commit is not a checkpoint.
* **ALWAYS open the pull request** in the same turn as the branch's first push, on every lane except `remote-branch-only`. A pushed branch with no pull request is litter nobody reviews.
* **NEVER `--no-verify`** and **NEVER force-push**. Those two are the real walls, and they stay closed.
* **ALWAYS merge your own pull request on `pull-request-and-merge`**, in the same turn, as soon as it is green. Reporting it as open and awaiting someone is the failure this lane exists to prevent.
* **NEVER merge on `pull-request` or `remote-branch-only`.** Those two stop where they stop, and the director merge lane carries a `pull-request` from there.
<!-- END managed by agentic-os/scripts/apply-git-workflow.py -->

Keep one issue per independently verifiable vertical slice. Kai-specific,
public-safe role and personality policy is first-class product
content. Do not generalize it prematurely, copy ordinary AOS skills into this
repo, or allow personality to alter truthfulness, authority, safety, rollback,
or completion.
Generated bundles and rendered references stay uncommitted. Graded
evaluation evidence is committed here under `evaluations/`, and `evalkit` runs the board.
Update [`docs/FEATURES.md`](docs/FEATURES.md) only when a significant
capability actually ships.

## Checkout residency

This repo is not in Agent Compose's `repository-plan.yaml`, so it has no
resident checkout under `~/projects/<owner>/`. That is intentional. Work it
from a task-scoped temporary clone, and remove that clone once the work lands.

A temporary root can be purged at any time, so commit and push before pausing,
switching tasks, or ending a session. The remote is the only durable artifact.

## See also

* [README.md](README.md) - human-facing product boundary and status.
* [docs/FEATURES.md](docs/FEATURES.md) - current shipped inventory.
* [justfile](justfile) - development recipes.
* [Catalog trifecta convention](https://forgejo.coilysiren.me/coilyco-flight-deck/agentic-os/src/branch/main/docs/features-release-tooling.md) - shared entry-point structure.
