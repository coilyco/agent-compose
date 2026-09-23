# science

Applied Scientist. Frog-Ox.

**Purpose** - measure how agents, models, and inference actually behave on real
hardware.

**Meld** - empirical and grounded. It produces the reading rather than
reasoning toward the answer, and refuses an abstraction that outruns the
evidence under it.

**Harnesses** - claude, codex, openhands. Frontier tier only.

Print the seat before you read about it:

```sh
agent-compose overlay --role scientist --seat claude
```

```
🧪 🪨 Frog-Ox
science / available
empirical + grounded
#3ed7a9
```

Then take it:

```sh
agent-compose launch science claude
```

## What it owns

Nothing. Alone in the roster.

That is the design rather than a gap in it. Look at the science column in
`agent-compose catalog boundaries` and you get three deferrals and one scope. A
seat whose output is evidence should not also hold the authority to act on it,
because a measurement taken by the party who will act on it is worth less than
one taken by a party who will not.

The tool will tell you this itself, which is worth preferring over the
paragraph above:

```sh
agent-compose bundle materialize --role scientist --harness claude --out ./bundles
agent-compose describe ./bundles/2f34a234d7775894
agent-compose verify ./bundles/2f34a234d7775894
```

```
bundle 2f34a234d7775894 // science/empirical+grounded // native-skills // 7174 body bytes
profile
  ✓ boundary build-foundational-software  role "scientist" holds within a scope boundary
  ✓ boundary modify-live-backend          role "scientist" defers boundary
  ✓ boundary seek-external-validation     role "scientist" defers boundary
  ✓ boundary suggest-external-comms       role "scientist" defers boundary

bundle verified: 9 skills // 13 files
```

The role has to be declared in your `.agents/roles.kdl` first, or materialize
reports the roles that are.

## What it holds a slice of

`build-foundational-software`, scoped to its own runners, probes, graders, and
aggregation. It builds the instrument. It does not build the thing under test.

The seam is where a scoped build quietly becomes a shared one. A grader that
only this seat runs is inside the scope. The same grader promoted into shared
tooling other seats depend on has crossed into [platform](platform.md).

## What it defers

`modify-live-backend`, `suggest-external-comms`, and `seek-external-
validation`.

Reads stay open throughout, and this matters more than the deferrals do.
Fetching logs, describing a cluster, listing pods, and pasting the raw output
are all in reach. Only the write is not. A deferral here produces a handoff
carrying the exact command that was not run and its expected result, so the
owning seat runs one line rather than re-investigating from scratch.

## Reach for it when

* You want to know whether a change actually moved a number.
* Two models, two prompts, or two configurations need comparing on your workload
  rather than on someone's benchmark.
* A regression might be real or might be noise and nobody has separated them.
* Something you believe about your system has never been checked against it.
* A claim is about to enter a durable artifact and wants a source under it.

## How it works

The habits are the point, and they are what you should check the output
against.

* The expected number is written down before the command runs, so the prediction
  is timestamped ahead of its own result.
* Anything about to be called many, several, or a lot gets counted first.
* A delta is reported only after the same thing ran twice and the two outputs
  were diffed, so a move is separable from noise.
* An after-number arrives with its before-number, in the same sentence where
  possible.

If your science seat is reporting improvements without a baseline, you have a
broken instance of the seat rather than a subtle judgement call.

## The tell that you picked wrong

* **Toward platform** - building foundational software outside its own
  instrument scope, because the file was already open.
* **Toward gamedev** - reporting what a session felt like rather than what it
  measured. Qualitative is not forbidden, but it is labelled, and it never
  arrives dressed as a reading.

## The chain it sits in

The intended chain is three links and nothing in it needs a seat to hold an
authority it should not have. Frog-Ox measures and hands over a
finding. Turtle-Ox runs the live command against the system.
Beetle-Ox fixes the tool that made the finding necessary. Each step is a
different bundle.

Boundary mechanics are in [role boundaries](../docs/role-boundaries.md), and
the context budget this seat works under is in [science context
budget](../docs/science-context-budget.md).

## Three prompts to start with

1. Did the new prompt actually cut token use, or is that noise? Run it twice at
   the same seed and show me both numbers.

2. We think the cache hit rate is above 90 percent. Measure it and tell me what
   it actually is.

3. Two models are candidates for this workload. Benchmark both against our real
   traffic shape rather than a public benchmark.

**And one it would hand back.** Apply the fix the measurement implies. Reads
stay open here, so it will hand you the exact command it did not run.
