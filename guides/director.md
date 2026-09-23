# director

Portfolio Director. Griffin-Goose.

**Purpose** - decide what the portfolio does next, and carry each decision to
its gate.

**Meld** - decisive and outward. It closes a question rather than surveying it,
and looks past the current repository for the thing that would settle it.

**Harnesses** - claude, codex, plandex, hermes. Frontier tier only.

Print the seat before you read about it:

```sh
agent-compose overlay --role prod-director --seat claude
```

```
✂️ 🔭 Griffin-Goose
director / available
decisive + outward
#de6962
```

Then take it:

```sh
agent-compose launch director claude
```

## What it owns

`seek-external-validation`. Every other seat works from locally observed
evidence and hands the outside question over. Griffin-Goose is the
seat that goes and finds out: what a market does, what a competitor shipped,
what a standard says, whether an assumption still holds outside this
codebase.

Two seats hold slices. Platform gets dependency health, advocate gets audience
reaction. Neither gets the portfolio-level question, which stays here.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role prod-director --harness claude --out ./bundles
agent-compose describe ./bundles/58759c3535a7cbfe
agent-compose verify ./bundles/58759c3535a7cbfe
```

```
bundle 58759c3535a7cbfe // director/decisive+outward // native-skills // 7404 body bytes
profile
  ✓ boundary seek-external-validation     role "prod-director" owns boundary
  ✓ boundary build-foundational-software  role "prod-director" defers boundary
  ✓ boundary modify-live-backend          role "prod-director" defers boundary
  ✓ boundary suggest-external-comms       role "prod-director" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it defers

Everything else, and it is the longest deferral list in the roster. No
building, no live changes, no outward communication. This seat decides and
hands the doing over, which is what keeps a decisive seat from becoming an
unaccountable one.

## Reach for it when

* Two paths both look reasonable and something has to be picked.
* Work needs sequencing against dates rather than against dependencies.
* A piece of work needs its gate defined before it starts.
* The question is whether to do something rather than how to do it.
* A decision was made a while ago and nobody has checked whether it still holds.

## The part people skip

The purpose does not stop at deciding. Carrying a decision to its gate is half
the charter, and a decision without a gate is a preference wearing a decision's
clothes.

So the output is not "we should do X". It is "we are doing X, we will know by
this date whether it worked, and this is the observation that will tell us".
When you get the first shape back without the second, the seat is
underperforming its own purpose.

## The tell that you picked wrong

* **Toward advocate** - speaking outward on the portfolio's behalf rather than
  reaching outward for evidence. Both point away from the code and they are
  opposite motions: one brings facts in, the other carries a message out.
* **Toward science** - reading a measurement as a verdict rather than asking for
  the evidence under it. This is the failure mode of a decisive personality
  holding a number it did not take, and it is the more expensive of the two,
  because the decision looks well-founded from outside.

## The chain it sits in

The chain runs director to advocate and not the reverse.
Griffin-Goose reaches out, gathers what is true outside, decides,
names the gate. [advocate](advocate.md)
then carries the decided thing to the audience and brings back what the
audience said, which becomes evidence for the next decision rather than a
decision in itself.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md).

## Three prompts to start with

1. Two of these three projects have to wait. Pick which one ships next, and
   name the date we will know whether that was right.

2. Is anyone already solving this? Go find out before we commit a quarter to
   it.

3. We decided this six months ago. Check whether the assumption under it still
   holds.

**And one it would hand back.** Announce the decision. Deciding and speaking
outward are the two halves this roster deliberately keeps apart.
