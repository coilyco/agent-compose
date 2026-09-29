# Evaluation

How role and personality behavior is measured here, and the two tools that do it. The board derives from
the roster, runs against one subject, and is graded by a human.

## Two tools, one seam

**`evalkit`** is the Python runner, in this repository. It runs the subject against `challenges.yaml`
and writes the dataset. It left housecast under housecast#7961.

**`housecast.grade`** is the grading half, shipped from `coilyco-flight-deck/housecast` under that
package's `eval` extra so the pairing rule has one home. It holds no runner and no model client, so
grading never spends a token and never touches a deployed system. Run `housecast grade help` for the
exhaustive reference. a deployed lane grades through it too, against a live harness rather than a composed
prompt, so the pairing rule has one implementation across both.

One home per contract keeps the seam honest, and it now names the eval pack rather than the roster:
**the pack schema, the coverage rules, and the record writer have exactly one implementation**, because
two parsers is the failure the split avoids. Composition lives here, and housecast holds only the grader (housecast#8041).

## The triple

Three parties, none holding two seats. The **generator** authors candidate cases, an agent working
with a human. The **subject** produces responses, a model reached through whatever transport the
deployment supplies. The **grader** scores them, a second human, by hand.

A role's `model-tier` is a deployment compatibility claim rather than what the board tests, and tier
does not change selected context, so one subject measures every role's composed text. See [model
tiers](harness-vendoring.md).

## The pipeline

```text
generator                 ->  challenges.yaml
inspect eval              ->  .eval log      (five epochs each, unscored)
evalkit.filter            ->  dataset.yaml   (epoch 1 attached to every challenge)
housecast grade annotate  ->  annotations.yaml
housecast grade taxonomy  ->  failure modes, ranked
```

Go turns annotations into the canonical record, so totals and verdicts come from the pack rule rather
than from the annotator.

## The dataset is derived

`evalkit.matrix` reads the roster and prints the cases it implies. Boundaries and their owners produce
the pairs, adjacency produces the role-fit targets, and each role's meld produces the personality cases,
so adding a boundary, flipping an adjacency edge, or swapping a personality moves the challenge list on its
own and the dataset cannot drift from the roster. Adjacency reasons become the role-fit descriptors
directly, which is the text a generator needs to build the right confusion. Execution is one case per
session, because batching a role's cases would save machine time and no human time while manufacturing
the reflexive deferral the in-out pair exists to catch.

## The pair is the scoring unit

A boundary case comes in halves: one inside where the role must own the work, one outside where it must
defer. A role passing one and failing the other is a boundary failure rather than fifty percent, so a
degenerate always-defer policy scores zero, and `housecast grade` reports pair results, not halves.

## There is no mechanical scorer

Challenges once carried a `discriminator`, a regex list for the failing behavior, and item analysis used
the match count to pick which challenges reached the annotator. Across nine pass-or-fail ones the patterns
and the grader agreed on nothing that mattered: one false positive, two false negatives, six agreements
where nothing happened. It measured something, but not what the grading measures, so it was **deleted
rather than tuned**. The model-graded predecessor went the same way in `agent-compose#262`, its gating
lane unable to fail and its producer and reviewer sharing a model. One written challenge each now, all
annotated, and the retired records stay under `evaluations/retired-*`. Epochs stay at five, the
annotator sees epoch 1, and the rest stay in the log as a failure-spread estimate at no grading cost.

## Annotation

Boundary and role fit are pass or fail with a 50-word cap. Personality is fit, undecided, or does not fit,
with a 100-word cap. Notes are recorded only on a deduction. The caps were measured: below roughly 25 words
a deferring half drops the factual handoff its boundary requires, and 50 words fit one slide at large type.
`undecided` is a signal rather than a hedge, so a cluster of it is item analysis for the one tier with no
mechanical filter.

Challenges are ordered **role-major**, so an annotator loads one charter and holds it across that role's
challenges. `--roster` prints purpose, boundaries, adjacency reasons, and personalities per group,
`--role` annotates a subset, and grading saves after every decision. A deduction records a critique and,
where one exists, a verbatim span from the output, verified before it is accepted. `housecast grade
taxonomy` is the axial step: it groups deductions by structural axis, then by shared critique terms, and
ranks by frequency, producing a list of failure modes rather than a score.

## Export is one way

`housecast grade export` projects a committed run into a display payload and nothing returns, so the
surface reading it is a rebuildable projection rather than a second home for evidence. Pairs travel as
their own structure, so a renderer gets `complete` and `passed` rather than a half-graded pair, and
`critique` and `evidence` stay out unless `--include-private` asks for them. Export **refuses rather
than scrubs** when a record looks like it carries a secret, because a scrubber that misses a pattern
ships the secret, and withheld text is not scanned since text that never leaves cannot leak. Recognized:
AWS key ids, bearer and API tokens, JWTs, private key blocks, SSM parameter paths, chat-platform ids,
tailnet hosts, and email addresses.

`format: agent-compose.eval-export.v1`. Nothing is authored in the projection, so nothing has to come
back. A review UI that writes decisions would need a return path, and a second writer against the
record is the failure the one-way projection exists to prevent, so adopting one stays undecided.

## Commands

The `evalkit-*` verbs and `grade-serve` live in this justfile, and the other `grade-*` verbs are
housecast's. The one that needs saying: `just check` runs ruff, format, mypy strict, and pytest from `scripts/check.sh` rather than pre-commit, because
that config is managed by agentic-os and a hand-added hook is lost on the next sync.

## The rest of the stack

* [Eval references](eval-references.md) - six external reference points, with notes on [the
  papers](eval-ref-papers.md) and [the platforms](eval-ref-platforms.md). Inspect is adopted for the run
  leg, nothing was adopted for grading, and those pages say why.
* [housecast grading](https://forgejo.coilysiren.me/coilyco-flight-deck/housecast/src/branch/main/docs/grading.md)
  - the shared grading layer, its profile contract, and the probe layer under it.
* [Deleting the Mechanical Scorer](https://coilysiren.me/posts/deleting-the-mechanical-scorer/) - the
  measurement that retired the discriminator tier, and why a rule written twice was the one extracted.
