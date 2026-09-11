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
so tokens stay tokens. `stance` is its posture and lives on the **role** beside
`purpose`, never on a personality.

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

### Sound mark

Each `sound-mark` declares `timbre`, `contour`, and `pulse` tokens, a short
semantic seed for notifications or conversation entry. A renderer may synthesize
it or map it to an asset, and agent-compose ships no audio files, playback,
volume policy, or event routing.

### Projection

The complete record and expression vocabulary ship in person snapshot schema v2
and palette schema v2, bundle role metadata stays compact and text-first, and
permissions, routing, model choice, and runtime authority remain outside the
identity contract.

A seat answers "who are you" in four parts with their kinds, since Gem, Claude,
and Dragon-Butterfly are all names: *I am Gem, an agent-compose persona. My role
is Developer Advocate. On this seat my legal name is Mixpost, and my creature is
the Dragon-Butterfly.* [The overlay](overlay.md) composes it.

## Naming the seat

A compose request may rename the seat it composes:

```kdl
compose {
    role "sysadmin"
    identity name="Echo" pronouns="it"
    delivery "native-skills"
}
```

Both properties are required and the node takes no arguments. Omitting it keeps
the role's own seat, which is what every existing request does.

### Why it exists

A caller that already has an identity would otherwise carry two. Sirens Echo
composes `sysadmin` for its operator doctrine and answers as Sirens Echo, so
without this its prompt introduced Vera as well, a name from a different context,
in a lane whose policy forbids describing itself at all. The alternative was a
whole person package, which replaces the roster as one unit: copying nine roles
to rename one seat makes every future roster change something the copy chases.

### What it does not do

It renames. Skills, methods, personalities, boundaries, and model tiers are
untouched, and no equivalent exists for any of them. The personality meld is the
one people ask for next and is a different shape: `role.Personalities` also
drives the melded favorite color, the nativeui theme tokens, and a validator
requiring exactly two on a core role. A meld is several seams, a name is one.

### The dictatable short id

Terminal surfaces append the session's short id to the rendered name (`Angie
[she] (Platform Engineer) uz86`), so a human can name one agent out loud among
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
