# Release and the v2 migration

How agent-compose releases, and what the v2 migration changed.

## Release

Agent Compose releases are Forgejo-canonical. Every push to `main` enters a
no-cancel queue and validates the exact commit, and the owning
`scripts/release-impact.sh` classifier decides whether publication runs.
Automatic publication occurs when the unreleased diff from the latest reachable
`v*` release tag changes shipped product inputs:

* the Go command or internal engine, and the Core Roster shipped in `seed/`
* Go module dependencies
* release binary construction
* Homebrew or Scoop rendering, and the `scripts/ci` steps that publish

Documentation, results, examples, tests, and development workflow changes still
validate but do not create a product version. The event base is the fallback
before the first release tag, and this roll-forward window keeps a failed
release eligible when a later main push carries only recovery evidence. The
classifier fails closed to publication when its base revision is unavailable.

An automatic release publishes a new minor version once its binaries build,
cross-compiling macOS, Linux and Windows, creating the Forgejo release, uploading
checksums and package files, and updating Homebrew and Scoop when tokens allow.
Those two bumps run **before** the GitHub mirror check, whose poll ceiling stays
well under the job's `timeout-minutes`. Neither channel reads the mirror, and a
step killed mid-loop takes every later step down with it, so a red release may
still have shipped: check the tap (`teable:coilyco-flight-deck/agent-compose#6994`).

### Major release hold

A tracked `.release-major` pauses automatic publication while a breaking stack
lands. Main validation continues. Manual workflow dispatch ignores the hold and
remains the only major-version and explicit-tag path.

For v2.0, keep the hold until the Core Roster stack, complete frontier and OSS
evidence, independent QA verdict, generated scorecard, and clean-main smoke are
all present. Then an operator dispatches the exact landed revision with a major
bump. Remove the hold in a follow-up commit only after v2.0.0 and both package
channels are verified. Removing the hold alone does not publish another
version.

## Roster recall pass

Four seats retitle and three seat names change, on the finding that a seat name
earns recall by carrying its own job, in sound or in meaning. Slugs, boundaries,
personalities, tiers, and colors all hold, so only displayed text moves:
`platform` to Agentic Platform Engineer, `eval` to Evaluation Engineer seated by
`Evie`, `frontend` to Design Engineer, `tpm` to Portfolio Director seated by
`Portia`, and the `sysadmin` seat from `Olaf` (he) to `Vera` (she). `gamedev`
and `devrel` are untouched.

## Agent Compose v3 roster migration

v3 takes the Core Roster from nine seats to seven and renames every slug. There
are no compatibility aliases either way, so the whole set moves in one break
rather than drifting through two: `ai` and `engineer` named a technology and a
generic category while every other slug named a practice.

### Role destinations

* `engineer` becomes `platform`, displayed as Developer Platform Engineer.
* `ops` becomes `sysadmin`, displayed as Systems Administrator.
* `ai` becomes `eval`, displayed as Agent Evaluation Engineer.
* `design` becomes `frontend`, displayed as Frontend Design Engineer.
* `creator` becomes `devrel`, displayed as Developer Advocate.
* `gamedev` is unchanged.
* `director` and `exec` merge into `tpm`, displayed as Technical Program
  Manager, on the finding that they split on seniority rather than practice.
* `qa` is cut. Its done-condition folds into `eval`, and code review folds into
  `tpm` as a gate decision.

Three boundaries rename with it: `build-software` to
`build-foundational-software`, `modify-live-system` to `modify-live-backend`,
and `suggest-human-comms` to `suggest-external-comms`. `modify-live-backend` is
re-ownered to sysadmin and `seek-external-validation` to tpm. Every boundary now
reaches all seven seats as one owner, two scoped, and four deferring.

Melds drop from three traits to two, one signature plus one bond shared with a
sibling seat, and the trait vocabulary falls from 19 to 10. The nine retired
bodies are preserved verbatim in git at `12c7198^`, under
`internal/person/data/personality-{bold,candid,curious,diplomatic,editorial,meticulous,nurturing,reflective,skeptical}`.

Update launch commands, Ward role selections, composed-skill bindings,
`AGENT_GIT_ATTRIBUTION_ROLE`, per-role Claude themes, and evaluation inputs to
the new identifiers. Scored records from earlier rosters remain historical
evidence and retain their original role identities.

## Agent Compose v2 roster migration

Agent Compose v2 renames the baked provider from `person:kai` to `roster:core`
and emits the Core Roster slugs, including AI Engineer as `ai`. It is an
intentional major-version break with no compatibility aliases for old role
identifiers. The roster returned to eight roles there, after audience-facing
work proved more coherent as one Content Creator loop, and Game Developer
joined later as the ninth.

### Role destinations

* `engineer`, `director`, and `qa` keep their slugs.
* `ops` remains `ops`, displayed as DevOps.
* `designer` becomes `design`, displayed as Designer.
* `advisor`, `ceo`, and `pm` become `exec`, displayed as Executive Strategist.
* `technical-writer`, `social`, `content`, `community`, `outreach`, `sales`,
  and `customer-success` become `creator`, displayed as Content Creator.

v2 moved `remote_skill_sources`, `remote_skill_cache_ttl`, and `mcp_inventory`
to AOS, which hydrates and verifies remote catalogues, projects native MCP and
Codex approval policy, then passes `skill_catalog_manifest` here. Those removed
keys fail strict config loading instead of being ignored.

### External packages

External `person "<name>"` packages remain supported through the validated
package contract and keep `person:<name>` provenance. A package may adopt a
`roster "<name>"` manifest when it intends to emit `roster:<name>` provenance.
The package selector remains exclusive and never inherits Core Roster roles or
definitions.
