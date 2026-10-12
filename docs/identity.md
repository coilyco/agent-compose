# Identity primitives and seats

## Personality identity primitives

The selected person source owns a renderer-neutral identity record for every
personality, so web, mobile, terminal, audio, and generated-art consumers
project the same semantics without agent-compose owning their presentation.
[The overlay](overlay.md) carries the rules a renderer respects.

### Emblem

```kdl
emblem { name "knot" "hitch" "lifeline"; emoji "🪢" }
```
Names run widest-reading first: the emoji's literal name, then what this roster
reads the mark as. `🎨` is an artist palette and the personality it marks is
playful, so either word alone is half the story. A renderer with room for one
takes the first, and every name is a unique lookup key.

### Motif

`motif` is one lowercase semantic token such as `wet-paint`, read as material or
texture rather than a CSS class or asset path. A motif is what a thing is made
of and an emblem is a thing you point at, so every motif is a material.

### Geometry

`geometry` is one lowercase semantic token such as `radial-facets`, the stable
shape language a renderer generates an avatar, sprite, or overlay figure from,
and the agent's own representation rather than a separate pet. `aterm` reads
both halves: mask, then ink.

### Body and stance

`body` is the creature in prose rather than tokens, parsed by its own validator
so tokens stay tokens. `stance` is its posture, on the **role** not a personality.
The animal and its colour come from the meld: one `species`, one `color_word`.

```kdl
body {
    archetype "sturdy compact body, simple rounded forms, thick tapering limbs"
    attachment "a thick rope knotted around its shoulder, running taut out of frame"
}
```
Every renderer uses one fixed expression vocabulary: `available`, `listening`,
`thinking`, `acting`, `waiting-for-human`, `blocked`, `completed`, `failed`,
`offline`. The owning runtime supplies the state, and agent-compose defines the
vocabulary without ever inferring it.

### Projection

The complete record and expression vocabulary ship in person snapshot schema v2
and palette schema v2, bundle role metadata stays compact and text-first, and
permissions, routing, model choice, and runtime authority remain outside the
identity contract.

A seat answers "who are you" in parts with their kinds, since Dragon-Butterfly,
Claude, and Mixpost are all names: *I am Dragon-Butterfly, an agent-compose
persona. My role is Developer Advocate. On this seat my legal name is
Mixpost.* An unauthored role has no separate personal name: its creature is
its whole identity, no pronoun attached, since a seat is not a person.
[The overlay](overlay.md) composes it.

## Naming the seat

A role's identity defaults to its own creature (species-pair, derived from its
personality meld - see [Personality identity primitives](#personality-identity-primitives)
above). A compose request may override it with a different name:

```kdl
compose {
    role "sysadmin"
    identity name="Echo"
    delivery "native-skills"
}
```

The `name` property is required and the node takes no arguments. Omitting the
node keeps the role's own seat.

A caller with its own identity would otherwise carry two, and a whole person
package makes every roster change something the copy chases.

### A role's own names

A role authors `identity` with a `name` and `alternates`, the names its second,
third and later live instances take. They keep the family word and vary the
first (`Turtle-Ox`, `Bison-Ox`, `Yak-Ox`), and no two roles share one. The caller
that knows the live instances sets `AGENT_COMPOSE_IDENTITY` to one of the role's
names for a launch or an overlay, and a name outside the pool is refused. A
launch clears it, so a child seat picks its own.

### What it does not do

It renames. Skills, methods, personalities, boundaries, and model tiers are
untouched. The personality meld is a different shape: it also drives the melded
favorite color, the nativeui theme tokens, and a two-per-role validator, so a
meld is several seams and a name is one.

### The dictatable short id

Terminal surfaces append the session's short id to the rendered name (`Angie
(Platform Engineer) uz86`), so a human can name one agent out loud among
several. Read from `AOS_NATIVE_SESSION`, never minted. See [whoami](whoami.md).

### The invariant this sits beside and what moves together

[The person contract](person-contract.md) says an overlay may not redefine
selected roles, personalities, definitions, or role personality sets. This is the
single exception, and the line it holds is between **who is speaking** and **what
the role is**: naming a seat is identity, and a caller wanting different role
content still brings its own package as one unit.

The override rewrites the role identity and every seat, because consumers read
different ones: the card and roster read the role, while the overlay,
statusline, native launcher, and bundle manifest read the seats. Rewriting one
alone yields a bundle whose card and statusline disagree. The bundle key moves
with it, since `cacheKey` hashes the rendered instructions and the card is in
them, so a renamed seat cannot reuse an unrenamed bundle.
