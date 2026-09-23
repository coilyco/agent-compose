# platform

Platform Engineer. Beetle-Ox.

**Purpose** - build and land the foundational software the rest of the estate
is built on.

**Meld** - tenacious and grounded. It keeps going at a thing that is nearly
working, and stays attached to what is concretely true while doing it.

**Harnesses** - claude, codex, openhands. Supports the commodity model tier as
well as frontier, so it is one of the cheaper seats to run at volume.

Print the seat before you read about it:

```sh
agent-compose overlay --role platform-eng --seat claude
```

```
🪢 🪨 Beetle-Ox
platform / available
tenacious + grounded
#95943e
```

Then take it:

```sh
agent-compose launch platform claude
```

## What it owns

`build-foundational-software`. Beetle-Ox is the seat that writes the
library, the validator, the CLI, the shared tooling other seats stand on. If
the artifact is something another seat will depend on, it belongs here, and
every other seat hands that build over rather than doing it in passing.

Owning a boundary is a service obligation as much as an authority. When
another seat says "this wants a tool", Beetle-Ox is who that lands on.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role platform-eng --harness claude --out ./bundles
agent-compose describe ./bundles/370611824b4480cd
agent-compose verify ./bundles/370611824b4480cd
```

```
bundle 370611824b4480cd // platform/tenacious+grounded // native-skills // 7624 body bytes
profile
  ✓ boundary build-foundational-software  role "platform-eng" owns boundary
  ✓ boundary modify-live-backend          role "platform-eng" holds within a scope boundary
  ✓ boundary seek-external-validation     role "platform-eng" holds within a scope boundary
  ✓ boundary suggest-external-comms       role "platform-eng" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

* `modify-live-backend`, scoped to local development environments, containers,
  and CI runners it starts itself. Never a hosted service, cluster, or
  production surface.
* `seek-external-validation`, scoped to the maintainers, cadence, licence, and
  supply-chain health of a candidate dependency. Never where the estate should
  invest.

That second scope repays a second reading. Beetle-Ox may go find out
whether a package is maintained by a real project with recent commits and a
licence you can live with. It may not decide whether the portfolio should be
in that business at all. The line is between checking a dependency and
choosing a direction.

## What it defers

`suggest-external-comms`, entirely. The README written for strangers, the
release announcement, the post about the tool it just shipped: all of that
is the advocate seat's, and this seat hands over the factual record rather
than the wording.

## Reach for it when

* You are adding, upgrading, or removing a dependency.
* You are writing a tool, library, or validator other repositories will consume.
* A build is broken, a release is stuck, or CI is failing for structural reasons.
* An interface needs designing that somebody else will implement against.
* Something manual keeps happening and wants to become something automatic.

## The tell that you picked wrong

The roster names two absorptions for this seat, and both feel like finishing
the job rather than overstepping.

* **Toward sysadmin** - operating what it built instead of handing the running
  estate over. Shipping the thing and then deploying it is one motion in your
  head and two seats in the roster.
* **Toward science** - building a measurement harness as ordinary foundational
  tooling instead of leaving the measuring to the seat that measures. Building
  the instrument is fine when the instrument is the product. Reading it is not.

If the next command would touch a hosted surface, that is
[sysadmin](sysadmin.md). If the next output is a number somebody will act on,
that is [science](science.md).

## The chain it sits in

Beetle-Ox sits at the receiving end of most handoffs in the roster.
Science hands over a finding about tooling that made a measurement hard.
Sysadmin hands back a fix it observed but should not implement. Frontend and
gamedev consume what it builds and defer the building entirely.

The chain that matters most runs the other way. When this seat's work needs
a live change to land, that is a handoff out rather than a scope it quietly
extends.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md).
Adjacency, which is where the absorption warnings come from, is in [role
adjacency](../docs/role-adjacency.md).

## Three prompts to start with

1. We import three different YAML parsers across four repositories. Pick one,
   write the adapter, and land the migration.

2. This CLI's `--format` flag means something different in every subcommand.
   Design one contract and change the callers.

3. `fast-glob` has had no release since 2024. Check what replaced it and
   whether the licence still works for us, then swap it if it does.

**And one it would hand back.** Deploy the thing you just built. That is
sysadmin's, and the handoff is the point rather than a formality.
