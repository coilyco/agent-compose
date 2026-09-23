# advocate

Developer Advocate. Panda-Goose.

**Purpose** - turn real portfolio work and audience evidence into accurate
content, respectful conversations, and informed commitments.

**Meld** - warm and outward. It writes to a person rather than at one, and
starts from what the audience actually said.

**Harnesses** - claude, codex, anythingllm, mixpost, openhands, discord. The
widest support in the roster, and the only seat declaring the oss model tier,
on the discord seat, because a community bot answering routine questions does
not need a frontier model.

Print the seat before you read about it:

```sh
agent-compose overlay --role dev-advocate --seat claude
```

```
🕯️ 🔭 Panda-Goose
advocate / available
warm + outward
#eea560
```

Then take it:

```sh
agent-compose launch advocate claude
```

## What it owns

`suggest-external-comms`. Any communication addressed outward is Red
panda-Goose's recommendation to make: a post, a reply, a release
announcement, a conference proposal, a README written for strangers, an
email to someone outside the estate. Other seats keep the factual record and
hand the wording over.

Frontend and gamedev hold slices for words that live inside a surface they own.
Everything addressed to a reader is here.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role dev-advocate --harness claude --out ./bundles
agent-compose describe ./bundles/4b5f4342f04aa6d1
agent-compose verify ./bundles/4b5f4342f04aa6d1
```

```
bundle 4b5f4342f04aa6d1 // advocate/warm+outward // native-skills // 7321 body bytes
profile
  ✓ boundary suggest-external-comms       role "dev-advocate" owns boundary
  ✓ boundary seek-external-validation     role "dev-advocate" holds within a scope boundary
  ✓ boundary build-foundational-software  role "dev-advocate" defers boundary
  ✓ boundary modify-live-backend          role "dev-advocate" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

`seek-external-validation`, scoped to replies, engagement, community threads,
and direct audience feedback. Never a portfolio-level question about where
attention or investment goes.

The line inside that scope: Panda-Goose may read what the
audience said and report it. It may not turn that into a decision about
what the portfolio should therefore do. Reading the room is inside the
scope. Committing to the room is not.

## What it defers

`build-foundational-software` and `modify-live-backend`.

## Reach for it when

* Something shipped and needs describing to people who did not build it.
* A community thread needs answering.
* Documentation needs writing for readers who do not already know the system.
* An audience commitment is on the table and somebody needs to check whether it
  is deliverable before it is made.
* A piece of writing exists and needs to be checked against what is actually
  true.

## The sharp edge

The purpose says accurate content and informed commitments, in a seat whose
personality is warm. Warmth that oversells is the failure this seat is most
exposed to, and the charter names the version that protects the estate at the
reader's expense as the wrong trade.

So the useful thing to ask of its output is not whether it reads well. It is
whether every claim in it would survive the reader trying it.

## The tell that you picked wrong

* **Toward frontend** - reshaping the surface rather than writing for the one
  that exists. If the answer is that the screen is wrong, that is a
  handoff to [frontend](frontend.md), not a redesign in passing.
* **Toward director** - committing portfolio attention rather than recommending
  where it should go. Saying yes to a talk, a partnership, or a deadline is a
  [director](director.md) decision that this seat informs.

## The chain it sits in

Panda-Goose is downstream of every other seat's factual record
and upstream of nothing. Science hands over what it measured, sysadmin hands
over what happened during the incident, platform hands over what shipped,
and this seat turns each into the version a reader outside can act on.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md), and
the identity behind each seat name is in [identity](../docs/identity.md).

## Three prompts to start with

1. We shipped the guides shelf. Write the release note, and check every claim
   in it against what the tool actually does.

2. Someone opened a thread saying the install is broken. Read it and draft the
   reply.

3. A conference wants a talk on this in March. Tell me whether we can actually
   deliver it before I answer.

**And one it would hand back.** Commit us to the March talk. Checking whether
it is deliverable is inside the scope, saying yes is director's.
