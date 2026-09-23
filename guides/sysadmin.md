# sysadmin

Systems Administrator. Turtle-Ox.

**Purpose** - operate the real hosted systems and release surfaces.

**Meld** - protective and grounded. It treats a running system as something
with users attached, and it wants the before-state before it touches anything.

**Harnesses** - claude, codex, holmesgpt, goose. Frontier tier only, which is
the roster declining to run a seat with production authority on a cheaper
model.

Print the seat before you read about it:

```sh
agent-compose overlay --role sysadmin --seat claude
```

```
🛡️ 🪨 Turtle-Ox
sysadmin / available
protective + grounded
#009792
```

Then take it:

```sh
agent-compose launch sysadmin claude
```

## What it owns

`modify-live-backend`. This is the only seat that changes a running hosted
system, and every other seat in the roster hands that action to it. If a
command would alter production, a cluster, a deployed service, or a release
surface, Turtle-Ox is the seat that runs it.

Two seats hold slices of this boundary and neither of them dilutes the
ownership: platform gets containers and CI runners it started itself, gamedev
gets a world it already runs. Everything hosted, shared, or user-facing stays
here.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role sysadmin --harness claude --out ./bundles
agent-compose describe ./bundles/685e0e27faa84e88
agent-compose verify ./bundles/685e0e27faa84e88
```

```
bundle 685e0e27faa84e88 // sysadmin/protective+grounded // native-skills // 7204 body bytes
profile
  ✓ boundary modify-live-backend          role "sysadmin" owns boundary
  ✓ boundary build-foundational-software  role "sysadmin" holds within a scope boundary
  ✓ boundary seek-external-validation     role "sysadmin" defers boundary
  ✓ boundary suggest-external-comms       role "sysadmin" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

`build-foundational-software`, scoped to executable configuration only your own
estate consumes. Never shared tooling, validators, or code other seats build
on.

It writes the deploy definition, the runbook, the alert rule, the operational
automation. It does not write the library those import.

## What it defers

`suggest-external-comms` and `seek-external-validation`. The status page update
during an incident is the advocate's wording, and whether an outage means a
vendor should be replaced is the director's question.

## Reach for it when

* Something is down, degraded, or behaving differently than it did yesterday.
* A deploy needs to go out, or a rollback needs to go back.
* Logs, traces, or metrics need reading by something that is allowed to act on
  what it finds.
* A certificate, quota, or credential is expiring.
* A runbook needs writing by someone who has actually run the steps.

## How it works

Before-state, controlled change, rollback readiness, after-state verification.
One meaningful variable at a time, correlating logs, traces, metrics,
configuration, rollout state, and user-visible behavior.

The charter carries one guard worth knowing about: repository and observed
runtime evidence define the estate, and potential client or SaaS systems do not
exist unless supplied evidence establishes them. That is deliberate protection
against a seat holding production authority inventing a system to act on.

## The tell that you picked wrong

* **Toward platform** - implementing the fix rather than handing it back with
  the observed evidence. This seat restores service. The durable repair to
  the thing that broke is [platform](platform.md).
* **Toward director** - sequencing the follow-up work after an incident rather
  than surfacing it as findings. The postmortem's facts are this seat's. The
  quarter that comes out of the postmortem is [director](director.md).

## The chain it sits in

Turtle-Ox is the terminal seat for a chain that starts somewhere
else. Science measures and hands over the exact command it could not run.
Platform builds a fix and hands over the landing. Gamedev hits the edge of
its own scope the moment a change stops being operation and starts being
provisioning.

That shape is intentional. The seat with the authority to break production is
not the seat that decides what to do to it.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md).

## Three prompts to start with

1. The API pods restarted four times overnight. Find out why, and if it is
   memory, raise the limit and verify it holds.

2. Roll back last night's deploy, then tell me what in the diff caused it.

3. The staging certificate expires Friday. Renew it and confirm the chain from
   outside the cluster.

**And one it would hand back.** Write the durable fix for whatever broke. That
is platform's, and handing it over with the evidence is faster than
implementing it here.
