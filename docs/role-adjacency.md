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
open. Two of the eighteen below name a confusion the owner's boundary already
blocks, which is not wrong but tests compliance rather than something new.

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
```

In-degree is not even. Measured across the nine `role.yaml` files, analyst and
director receive three edges each, five seats receive two, and psych receives
none. Only out-degree is enforced, in `internal/person/person.go`, so in-degree
is an authoring observation rather than a rule. The ninth seat landed without
re-pointing an edge, and its in-degree of zero is the orthogonality signal
below rather than an omission.

## Measuring orthogonality

A seat is orthogonal when the axis it works on is one every other seat crosses
and none owns. Psych is the roster's example, and three properties separate it
from the eight seats around it. All three are read from the roster files.

* **Owns a boundary nobody scopes.** `hold-emotional-weight` is the only one of
  the five with zero scoped holders, against two, three, three, and two for the
  rest. The behavior is all-or-nothing, because a bounded grant to do a little
  of it names the harm rather than a safe subset.
* **Receives no adjacency edge.** Nothing drifts toward it, where every other
  seat receives two or three.
* **Holds no scoped grant.** It defers the other four boundaries outright.
  Director does too and fails the first two, so the three are read together.

A fourth property is not structural and carries most of the value. The guardrail
inverts a resting behavior of the base model rather than holding a trained
practice to its own standard. `withheld-comfort` exists because an untrained
assistant agrees, where `provable-results` and `reversible-steps` hold a craft
to a standard it already recognises. A seat whose failure mode is the model's
default does work no ordinary charter produces.

### Testing a candidate seat

Propose the behavior, then try to write its scoped grant. A candidate that
scopes cleanly is an ordinary seat and joins the roster on the usual terms.
Retiring a shipped surface scopes cleanly, since retiring your own module is
sensible, so it fails this test despite naming a real gap that no charter
covers. Characterizing a person who is not in the conversation does not scope,
because "inside your own artifact" is the failure the boundary would exist to
prevent.

## See also

* [Role boundaries](role-boundaries.md) - shared behavior allocated to one owner.
* [Boundary owners](ownership.md) - the two-sided relationship.
* [Role skills](role-briefings.md) - charter and progressive-disclosure model.
* [Evaluation](evaluation.md) - deterministic packs and review policy.
