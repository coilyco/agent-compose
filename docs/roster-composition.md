# Roster composition

What a deployment may add to the roster, derive from it, or leave out of it.

## Roster overlays and derived roles

A role can be minted as data outside `seed/roster/data/` and declare only how it differs from its parent (teable:coilyco/agent-compose#8197).

### Roster overlay

A roster root holding `overlay.yaml` with `layers_over: core` adds to the next plain roster in the search order instead of replacing it. A root without the marker still replaces everything under it.

* An overlay entity directory, such as `data/role-<slug>/`, replaces the base directory of that name whole. Files never mix, so an overlay role cannot pick up its base namesake's `SKILL.md`. A top-level overlay file replaces the base's.
* Overlays do not stack, and an overlay with no roster under it is refused.

A private roster rides on the public seed this way. A host that must not hold it does not mount it, and runs the seed alone.

### Appended acts

An overlay root may also hold `acts.yaml`, which appends acts to shipped roles, personalities, and boundary sides instead of replacing them (Kai, 2026-09-01, teable:coilyco-flight-deck/agent-compose#1824). The shipped acts stay first, so a host holding both tools runs the portable one before the estate one.

```yaml
overlay: kai-estate
roles:
  sysadmin-senior:
    - {tool: signoz, text: "signoz the service error rate before claiming it moved"}
boundaries:
  modify-live-backend:
    own: [{tool: aosguard, text: "aosguard ops the before state, then the same read after"}]
```

* A boundary act names its side, `own`, `scoped`, or `defer`, and reaches only a seat holding that side.
* Appended acts follow no count guideline, since the guideline keeps the shipped roster readable by a stranger.
* Every name is checked. An unknown key, a misspelled role, a tool missing from its text, or a repeated act fails the load, since an act that reaches nobody looks exactly like an act nobody wrote.
* The estate-tool refusal still guards the shipped roster, which stays byte-identical for a host that mounts no overlay.

### Derived roles

`derives: <parent>` in a `role.yaml` merges the parent's fields under the child's. A mapping such as `voice` merges key by key. A scalar or list replaces the parent's whole, so dropping one boundary means restating the list. An explicit `null` removes the parent's key, as in `guardrail: null`. A child may keep its parent's guardrail, and no other role may name it. `role`, `order`, `skill`, and `archived` never inherit, `skill` defaults to `role-<slug>`, and `color_twin` is implied as the parent.

The loader refuses a derivation from itself, from an undefined role, or from a role that itself derives. It also refuses a derived role named as a boundary `owner`, because a derived role narrows a charter rather than owning one. A derived role may appear in `co_owners` only on a boundary its own parent owns, which narrows the parent's ownership instead of adding a new one.

A derived role ships its own `SKILL.md`. The parent's charter is not appended, because it states authority the child lacks: Senior Sysadmin's says it changes running systems, and Junior Sysadmin's exists to say the opposite.

`derives` reaches the snapshot and `catalog roles --json`, and `scripts/eval-prompts.sh` writes it to `derives.json` beside the prompts. The board then runs every parent case against the child too, re-keyed `<id>@<child>`, so a derived role never enters the board unmeasured.

### Co-owners

A boundary names one primary `owner` and may list `co_owners` beside it. A
co-owner receives the owner side exactly as the primary does, and the same rules
bind it: it may not also declare or scope the boundary, and it may not appear
twice. The identity card, the boundary matrix, and the evaluation board read
every owner as `OWNS`, and `catalog boundaries --json` carries `co_owners` next
to `owner`. A derived role may co-own only its parent's own boundary.

Sharing changes who holds the owner side. It does not say which systems that
side reaches, and a roster that needs such a limit states it in its deployment
layer rather than in `roster:core`. The shipped roster shares `modify-live-backend`
between its senior and access sysadmin seats.

## Boundary omission

When a deployment composes without a boundary, and what stops that meaning too much.

### The case for it

The defer side of a boundary is routing: hand this to the role that owns it. A
deployment where that role is not a seat has nowhere to route, so the rule
reads as a stop rather than a handoff, and the agent defers work nobody will
pick up. A single-agent deployment hits this on every boundary it defers.

A request states the absence directly:

```kdl
compose {
    role "scientist"
    boundary-omit "modify-live-backend" "seek-external-validation"
}
```

The omission removes the boundary from the composed set entirely: no body in
the bundle, no name on the identity card, no entry in the manifest. Naming a
boundary whose body is absent is worse than either, because the card then
describes doctrine the agent cannot read.

Three refusals keep the knob from meaning something it should not:

* An unknown boundary name fails rather than no-opping, matching the rest of
  the request parser.
* A boundary the role **owns** cannot be omitted. An owner losing its own
  boundary is a larger claim than a deferrer losing one, and it would leave the
  boundary with no side that holds it.
* A boundary the role does not activate fails too, so a stale request surfaces
  instead of quietly expressing nothing.

The decision trace records each omission as an excluded profile decision. A
bundle that quietly lacks a boundary is worse than one that never had it,
because the review surface stops telling the truth.
