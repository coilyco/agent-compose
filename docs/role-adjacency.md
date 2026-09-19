# Role adjacency

Adjacency names the two roles whose work a role is most likely to absorb, and
why. It is the axis the evaluation board reads to author role-fit cases.

## Directed, not symmetric

Absorption risk runs one way. The Senior Sysadmin sequencing follow-up
work after an incident is a live confusion, while the Portfolio Director
rarely reaches for a runbook. Declaring that pair symmetrically would buy a case
nobody fails.

So a role names who it drifts toward, and the reverse edge is a separate
decision. Do not add a symmetry check.

## Out-degree is fixed at two

Two forces the roster to pick the sharpest confusions rather than list every
plausible neighbour. Every role declares exactly two, or the roster declares
none at all. The all-or-nothing rule keeps external person packages authored
before this axis loading unchanged.

## The reason is generator input

Each edge carries a `reason`. It is not commentary. An adjacency case has to
construct one specific confusion, and a generator handed a bare pair will
invent the wrong one on exactly the edges that matter. An edge nobody would
guess is either a mistake or load-bearing, and the reason is how a reader tells
which.

## Package layout

Declare one node per edge, since each edge carries its own reason:

```kdl
role "senior-sysadmin" {
    skill "role-senior-sysadmin"
    boundary "suggest-external-comms" "seek-external-validation"
    boundary-scoped "build-foundational-software" scope="executable configuration only your own estate consumes"
    adjacent "platform" reason="implementing the fix instead of handing it back with observed evidence"
    adjacent "analyst" reason="attesting that her own change closed the risk instead of handing the assurance over to the seat that did not make it"
    personality "protective" "grounded"
}
```

Loading fails on a self-edge, a repeated target, a missing reason, an unknown
target role, or an out-degree other than two.

## How adjacency differs from a boundary

A boundary removes one behavior from several roles and allocates it to an
owner, so it already tests each member against that owner. Adjacency covers the
confusions no boundary allocates.

Spend adjacency slots accordingly. An edge earns its slot when it points where
no boundary reaches: at a seat that owns none, or at a gap the allocation leaves
open. Fourteen of the twenty-four below point at a seat that owns a boundary
(eleven of twenty-two as of `junior-sysadmin`, added 2026-09-16, plus
`admin-assist -> director` and `admin-assist -> advocate`, added 2026-09-19).
An edge whose reason restates that boundary is not wrong, but it tests
compliance rather than something new.

## Core Roster graph

```text
platform        -> analyst, science
senior-sysadmin -> platform, analyst
junior-sysadmin -> platform, analyst
science         -> platform, gamedev
frontend        -> advocate, gamedev
gamedev         -> frontend, senior-sysadmin
director        -> advocate, science
advocate        -> frontend, director
analyst         -> senior-sysadmin, director
psych           -> analyst, director
reporter        -> advocate, director
admin-assist    -> director, advocate
```

In-degree is not even. Measured across the twelve `role.yaml` files (ten
original, plus `junior-sysadmin` added 2026-09-16 as a hands-off counterpart to
the renamed `senior-sysadmin`, plus `admin-assist` added 2026-09-19 as a
hands-off counterpart to `director`), director receives five edges, advocate
and analyst receive four, platform receives three, and psych, reporter,
junior-sysadmin and admin-assist receive none. Only out-degree is enforced, in
`internal/person/person.go`, so in-degree is an authoring observation rather
than a rule.

Analyst, psych and reporter were archived on 2026-09-15 and the graph above is
otherwise unchanged, because archiving retires a seat from selection and keeps
everything else. So platform, senior-sysadmin and junior-sysadmin all still
declare an edge to analyst, and out-degree still validates at two for all
twelve. What did change is downstream: `evalkit` derives role-fit cases from
the live seats only, so the edges pointing at analyst still derive a case for
platform, senior-sysadmin and junior-sysadmin, while the three archived seats
derive none of their own.

`sysadmin` itself was renamed to `senior-sysadmin` on 2026-09-16, paired with
the new `junior-sysadmin`. Every edge that pointed at the old slug (`platform`,
`gamedev`, and `analyst`, the three shown above) was repointed in the same
pass, along with the boundary owner and the guardrail it carries.

## See also

* [Role boundaries](role-boundaries.md) - shared behavior allocated to one owner.
* [Boundary owners](ownership.md) - the two-sided relationship.
* [Role skills](role-briefings.md) - charter and progressive-disclosure model.
* [Evaluation](evaluation.md) - deterministic packs and review policy.
