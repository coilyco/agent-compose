# Role adjacency

Adjacency names the two roles whose work a role is most likely to absorb, and
why. It is the axis the evaluation board reads to author role-fit cases.

## Directed, not symmetric

Absorption risk runs one way. The Systems Administrator sequencing follow-up
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
role "sysadmin" {
    skill "role-sysadmin"
    boundary "suggest-external-comms" "seek-external-validation"
    boundary-scoped "build-foundational-software" scope="executable configuration only your own estate consumes"
    adjacent "platform" reason="implementing the fix instead of handing it back with observed evidence"
    adjacent "director" reason="sequencing follow-up work after an incident instead of surfacing it as findings"
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
open. Ten of the twenty below point at a seat that owns a boundary. An edge whose
reason restates that boundary is not wrong, but it tests compliance rather than
something new, and `teable:coilyco-flight-deck/agent-compose#7486` re-derives
which of the ten do.

## Core Roster graph

```text
platform -> analyst, science
sysadmin -> platform, analyst
science  -> platform, gamedev
frontend -> advocate, gamedev
gamedev  -> frontend, sysadmin
director -> advocate, science
advocate -> frontend, director
analyst  -> sysadmin, director
psych    -> analyst, director
reporter -> advocate, director
```

In-degree is not even. Measured across the ten `role.yaml` files, director
receives four edges, advocate and analyst receive three each, five seats receive
two, and psych and reporter receive none. Only out-degree is enforced, in
`internal/person/person.go`, so in-degree is an authoring observation rather than
a rule. Both zero-in-degree seats landed without any existing edge being
re-pointed toward them, which is what a late addition looks like rather than a
property of the seat.

## See also

* [Role boundaries](role-boundaries.md) - shared behavior allocated to one owner.
* [Boundary owners](ownership.md) - the two-sided relationship.
* [Role skills](role-briefings.md) - charter and progressive-disclosure model.
* [Evaluation](evaluation.md) - deterministic packs and review policy.
