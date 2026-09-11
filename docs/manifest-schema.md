# Bundle manifest schema

`manifest.json` names what was composed and where the entry points are.

`manifest.json` names what was composed and where the entry points are:

```json
{
  "format": "agent-compose.bundle",
  "role": "platform",
  "role_skill": "role-platform",
  "role_skill_source": "roster:core:role:platform",
  "role_skill_digest": "sha256:<digest>",
  "model_tier": "frontier",
  "personalities": ["tenacious", "grounded"],
  "color": "#90a66a",
  "sources": ["roster:core", "aos-public"],
  "content": [
    {
      "id": "roster:core:role:platform:identity",
      "digest": "sha256:<digest>"
    }
  ],
  "delivery": {
    "mode": "native-skills",
    "instructions": "content/instructions.md",
    "skills_root": "content/skills"
  }
}
```

A compiled bundle replaces `skills_root` with `compiled_context` pointing at
`delivery/compiled.md` while the canonical skill trees stay in the tree for
inspection. Every named entry point must exist inside the bundle. `format` is a
plain marker, not a trust or cryptographic boundary. The built-in verifier
checks structural integrity: safe relative entry points, regular files and
directories only, complete delivery data, unique logical content IDs with
SHA-256 digests, and one identity tree per trace-selected skill. A consumer
that needs content authentication still hashes or signs the tree itself.
`sources` records which places content came from, by stable id, so the trace
and a human reader can refer to them. Locators and absolute paths never appear.
`personalities` preserves the role's declaration order. `color` is their melded
favorite, derived from every component color. `identity.display_name` carries the
role title, so a consumer labels a seat without loading a roster the delivered
bundle no longer ships beside.
[person-contract.md](person-contract.md) owns the legibility and blend rules.
`model_tier` records the caller's `frontier`, `commodity`, or `oss`
compatibility lane. It does not identify or route a runtime model and never
changes selected context. A manifest from before the tier field is read as
`frontier`, matching the earlier implicit default. Retired `model_class` fields
in older JSON are ignored. `role_skill`, `role_skill_source`, and
`role_skill_digest` bind the role identity to its canonical doctrine. `content`
records the effective logical role skill and methods, invariant, personality
definitions, evaluation assets, copy contract, and compact role identity
metadata. `diff` compares these stable IDs and digests without reopening the
authoring roots. Local filesystem paths never appear.

## The voice profile

`delivery.voice_profile` names `content/voice-profile.json`, a linter profile
for the `writing-voice-guide-linter` engine holding the rules this seat lints
its own prose against. It merges two kinds of rule.

* **Carried** - a selected skill shipping a root `profile.json` contributes its
  hand-written rules. Discovery reads the document rather than matching a skill
  name, so a source ships a house style without agent-compose knowing what that
  source called it. A carried rule keeps the keys `Rule` declares and loses any
  other, so `blocking` is declared there: dropping it ships a gate as a warning.
* **Generated** - one rule per term on the seat's melded `voice.avoid` bank,
  role first then personalities, deduped so the role bank keeps a shared term.

A carried rule wins a collision on id and sorts first, because a hand-written
pattern says something a generated one cannot.

A generated pattern anchors `\b` on whichever ends of the term are word
characters and matches internal whitespace as `\s+`. Both halves earn their
place: unanchored, `just` matches inside `adjust` and `easy` inside `greasy`,
which is the over-flag that teaches a seat to skip the linter, and without the
whitespace class a phrase that wraps a line stops matching.

A generated hint names the bank and stops. What the linter says to a writer
about their own prose belongs to the seat that owns the words.

A seat with no avoid bank and no carried profile writes no file and the field
is absent. The engine refuses an empty rule list rather than reading it as zero
rules, so writing one would fail to load at the point of use.

Before this the only consumer of `avoid` was the identity card's `**Refuse**`
line, so a seat was told to refuse a word no checker had heard of. Measured at
housecast `evaluations/voice-catchphrase-2026-09-08`.

## Bundle fingerprint

`bundle.Fingerprint` names the composition a manifest describes, so a session
transcript can point at an exact bundle without copying any of it. It covers
`role`, the three `role_skill*` fields, `model_tier`, `personalities`,
`boundaries`, `sources`, and every `content[]` id and digest in manifest order,
under a versioned prefix so a later change to the covered set cannot collide
with a fingerprint minted under the old rule.

`agent-compose whoami --json` emits it beside the seat label:

```json
{"format":"agent-compose.whoami.v1","seat":"Angie [she] uz86","role":"platform","bundle":"sha256:..."}
```

Metadata only. No skill bodies, no host paths, and no projection means no
record rather than a synthesised one, matching the silence rule in
[whoami](whoami.md).

**It names a composition and does not attest to one.** `verify` checks
`content[].digest` for shape and never recomputes it against bytes, so a
producer writing well-formed but untrue digests mints a well-formed but untrue
fingerprint. Joining transcripts on it is sound. Trusting it as provenance is
not, and closing that gap is a change to `verify` rather than to this value.

## See also

- [kdl-contracts.md](kdl-contracts.md) - the requests this schema records.
