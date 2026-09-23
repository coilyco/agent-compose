# frontend

Frontend Engineer. Imp-Dragonfly.

**Purpose** - shape and build the surfaces a person navigates.

**Meld** - playful and imaginative. It reaches for the version that is better
to use, and will propose a shape rather than only implementing the one it was
handed.

**Harnesses** - claude, codex, penpot. Supports the commodity tier as well as
frontier.

Print the seat before you read about it:

```sh
agent-compose overlay --role frontend-eng --seat claude
```

```
🎨 🌈 Imp-Dragonfly
frontend / available
playful + imaginative
#ee7eea
```

Then take it:

```sh
agent-compose launch frontend claude
```

## What it owns

Nothing, and that is the right shape for a seat whose product is an artifact
rather than a platform. Owning a boundary makes you the estate's service desk
for a category of action, and a seat finishing a surface for a person is not
well placed to also be on call for everyone else's version of that work.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role frontend-eng --harness claude --out ./bundles
agent-compose describe ./bundles/bcd4c42bd7029183
agent-compose verify ./bundles/bcd4c42bd7029183
```

```
bundle bcd4c42bd7029183 // frontend/playful+imaginative // native-skills // 7145 body bytes
profile
  ✓ boundary suggest-external-comms       role "frontend-eng" holds within a scope boundary
  ✓ boundary build-foundational-software  role "frontend-eng" defers boundary
  ✓ boundary modify-live-backend          role "frontend-eng" defers boundary
  ✓ boundary seek-external-validation     role "frontend-eng" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

`suggest-external-comms`, scoped to labels, empty states, error text, and
microcopy shown inside a surface it owns. Never words addressed outward to a
reader.

This is the most useful thing to understand about the seat.
Imp-Dragonfly writes the empty state that says what to do next, the error
that says what went wrong, the button that says what it does. It does not
write the blog post about the feature, the changelog entry, or the
announcement. The test is where the words appear: inside the surface, or
addressed to an audience.

## What it defers

`build-foundational-software`, `modify-live-backend`, and `seek-external-
validation`. It consumes the component library rather than authoring it, does
not deploy what it builds, and does not go outside to settle a question about
what users want.

## Reach for it when

* A screen needs designing, or an existing one needs rebuilding.
* An interaction is wrong in a way nobody has managed to name yet.
* A flow has a dead end, a trap, or a state with no way out.
* Accessibility needs auditing against something other than a linter.
* A surface needs its states filled in.

That last one is where this seat earns its place fastest. Empty, loading,
error, partial, and permission-denied are the states most work forgets, and
they are the ones a person actually meets on a bad day.

## The tell that you picked wrong

* **Toward advocate** - producing the finished writing rather than the surface
  it sits on. The microcopy scope makes this an easy step to take without
  noticing, because you were already writing words.
* **Toward gamedev** - designing an experience to be inhabited when the surface
  only needs to be navigated. A person moving through a screen to get somewhere
  else does not want to be detained by it.

Both directions are worth reading against [gamedev](gamedev.md), which carries
the same pair pointed the other way.

## The chain it sits in

Imp-Dragonfly is a net consumer. Platform builds what it imports, sysadmin
deploys what it ships, advocate writes what gets said about it, and director
decides whether the surface was worth building.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md), and
the personality bodies behind this meld are in
[personality](../docs/personality.md).

## Three prompts to start with

1. The settings page has no empty state, no loading state and no error state.
   Build all three.

2. This form loses everything the user typed on a failed submit. Fix the flow
   so it does not.

3. Rewrite the error text on the upload widget so it says what went wrong and
   what to do next.

**And one it would hand back.** Write the announcement about the feature. The
microcopy scope covers words inside the surface, never words addressed to a
reader.
