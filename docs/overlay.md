# Identity overlay

`agent-compose overlay` projects one selected person-package member into a
compact surface.
It is deliberately noninteractive. Terminal panes can use the plain card, and
other renderers can consume the same versioned JSON document.

## Text

Supply a role, harness seat, and renderer state:

```sh
agent-compose overlay \
  --person-source ./person \
  --role platform-eng \
  --seat codex \
  --expression acting
```

Omit `--person-source` to use the embedded `roster:core` default. The default
width is 40 columns, and `--width 200` collapses the fields onto one line.
Output carries no control sequences, so pipes and CI receive stable plain text.

## JSON

Add `--json` for `agent-compose.overlay.v1`. It contains:

* person, role, `role_display_name`, purpose, and selected seat
* `annotation`, the composed identity string a renderer shows verbatim
* `outro`, the `clean` and `failure` lines a session closes with
* `identity`, the four-part sentence the seat answers "who are you" with, from
  [identity.md](identity.md). Its legal name is authored per seat and never
  derived, so a non-model seat answers with its own product name, an unauthored
  seat drops that clause, and naming it never asserts a model selection
* the caller-supplied expression
* the role's derived favorite color, and its derived `background`
* every personality's color and identity primitives, with its `geometry` token
  and prose `body`, plus the role's `stance`, which no personality carries

## Rules a generating renderer respects

Each was paid for by a specific failure while the first creatures were drawn.

* **Anatomy leads, the object follows.** `archetype` describes a creature before
  `attachment` names the object. Reversing it produced a fairground ride.
* **Stance derives from the signature, never the bond.** Reading a bond for
  posture collapsed a hauling creature into a settled quadruped. Stance living
  on the role forbids it.
* **A bond tints, it never adds a second object.** Two named objects make a
  renderer drop one, and it drops the signature. A bond tints via `motif`.
* **The creature is painted from `color`, and the prose never names it.** A
  color word in `archetype` or `attachment` restates a hex that can move
  without it. The prose may name a material's intrinsic color, which `motif`
  carries: brass is brass whatever the accent does. The test is whether changing
  the hex would make the sentence false.
* **A meld whose two geometries match cannot be told apart.** `playful` and
  `imaginative` were both `radial` and melded, so the bond converted the
  signature rather than dressing it. Every arrangement is now distinct.

Clause ordering, style, proportion locks, and detail register stay with the
renderer, which names a personality by slug and reads these fields back. The
JSON projects the selected person model, not a second policy source.

## Why the background is derived here

`favorite_color` is an accent, solved in the terminal-legible band. A renderer
tinting that accent into a near-black of its own lands seven backgrounds inside
the side-by-side JND of each other: the closest pair measured 0.0109 in OKLab
against a solved 0.0386.

Separation is a property of the set, so only the roster sees every role at once.
`background` holds each role's hue, spaces the set equally at one low lightness
and chroma, and picks the rotation turning every role the least. Roster loading
asserts a floor on the closest pair, so an eighth role that runs out of hue
circle fails at load rather than shipping two windows nobody can tell apart. The
derived set is committed in
[`internal/palette/role-palette.txt`](../internal/palette/role-palette.txt), so
a roster edit arrives as a diff.

## Outro

`outro` is what a session says as it closes, `clean` and `failure`, so the tone
differs between finishing and falling over: **Measured. The numbers are in the
log.** against **It stopped before the last reading.**

Authored per **role**, not melded. Voice melds because it is read once as
doctrine, while an outro is read in half a second by somebody already leaving,
and role plus two personalities runs to three sentences, which is a paragraph
rather than a banner. It carries voice rather than telemetry for the same
reason: whoever reads a closing banner is leaving rather than debugging.

## Annotation

`annotation` is the one identity string every terminal surface shows for a
session, so a window title, a status row, and a launch flag never drift apart:

```text
Angie (Platform Engineer)
```

Agent Compose owns the shape, so a renderer with the document shows the field
rather than reassembling it from seat name and role. `annotation` and
`outro` are both additive to `agent-compose.overlay.v1`, so a consumer built
before them keeps parsing unchanged. The plain text card stops at `Angie`,
since it already prints the role on its own line.

## State boundary

The caller supplies one expression from the fixed person vocabulary. Unknown
roles, seats, and expressions fail closed, and agent-compose never inspects a
process, trace, log, queue, runtime, or launcher state to infer one. That keeps
the overlay suitable for terminals, browser shells, streams, and future mobile
surfaces without turning identity data into observability.

## See also

* [identity-primitives.md](identity.md) - renderer semantics.
* [person-packages.md](person-packages.md) - external package selection.
* [person-snapshot.md](person-contract.md) - complete person export.
