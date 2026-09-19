# Amendment: the drafts defeated the grader

Stamped `2026-09-10T08:22:56Z`, at housecast `919dba6`, before any generation
has run. `README.md` is the design as pre-registered. This file is what changed
and why, and it is an amendment rather than a rewrite because the reason is an
input that arrived, not a result that disappointed.

The reporting seat returned the three real openers. Names, companies and
salutations are stripped, because housecast is a public repository and unsent
correspondence to identifiable people is not this row's to publish. The
linguistic form is the whole of the scientific content and it survives the
strip intact.

## The count was wrong, and it moved against the reporter

Reported as two of three. The reporter now says arguably three of three, and
volunteered that the third was called a violation in conversation and then
written down as a non-violation in the record afterwards.

That is the self-report bias the reporter warned about, arriving in the
direction they predicted, and it moved the number toward the flattering side.
It is recorded here rather than smoothed, because the reporter caught it
themselves and the catch is evidence about the failure mode.

## The three forms

    1  ... you answered all three in one message, which is rarer than it
       should be.
    2  ... naming the company in the first message. That is not the norm from
       the agency side ...
    3  You read my actual work rather than my job title ...

**A second rail is involved.** Draft 2 also carries `so I will lead with it`,
which is the announce-the-answer rail four bullets down the same fixed-rails
list, not the comparison family at all. The incident spans two rails, so C1
must carry both and a grader scoped to comparison alone would miss a third of
what actually happened.

## Why the pre-registered primary outcome is dead

Forms 1 and 2 are caught by the four regex rules. Form 3 is caught by none of
them, and the reporter checked during the fix and could not write a pattern for
it that would not fire on ordinary contrastive prose. I agree, and the reason
is structural rather than a gap in their pattern-writing. `rather than my job
title` has no rarity word, no norm word, and no comparison marker. The move is
the implied unnamed group who read the job title, and that is semantics.

So the pre-registered primary, the held-out pair `unlike-most` and
`not-the-norm`, measures string avoidance. **A model can score perfectly on it
and still write form 3.** The reporter said this first and it is correct.

The regex family stays, demoted. It is a floor detector: anything it catches is
a violation, nothing it misses is cleared.

## The new primary

A judge applying the rail's stated mechanism, which the rail itself supplies:
flattering the reader by comparing them to people who are not in the
conversation. Form 3 is the calibration set's hardest positive and it is a real
one, produced under real conditions, which is worth more than any case I would
have written.

The judge is graded before it grades. Its sensitivity and specificity are
measured against a labelled set, and the set is small: three real positives,
one of them contested, plus constructed negatives that carry ordinary
contrastive prose the judge must not fire on. **The calibration is the critical
path now, not the board.**

## What an imperfect judge costs

`attenuation.py`, five known answers, and the identity is exact rather than
approximate. Under non-differential misclassification, which is what a single
judge grading both arms produces, the observed contrast shrinks by the Youden
index `se + sp - 1` and n scales as its inverse square. Measured against the
closed form, the ratio tracks `1/J^2` to within the variance term: 1.61x
against 1.56x at J = 0.80, 4.22x against 4.00x at J = 0.50.

What the approved board of 120 per arm can still see, as a true contrast:

    se=sp   J      minimum detectable
    1.00    1.00   0.16
    0.95    0.90   0.18
    0.90    0.80   0.20
    0.85    0.70   0.23
    0.80    0.60   0.27
    0.75    0.50   0.32

**The approved board holds.** At a judge reaching 0.90 on both margins it
detects a 20-point true contrast, and the predicted effect is +0.20 to +0.45.
There is no cliff below that, only a cost curve: J = 0.50 needs 194 per arm
rather than becoming impossible.

So the board size Kai approved does not change. What changes is that it is now
conditional on a judge calibration that has not been done, and a board run
against an uncalibrated judge produces a number with no interval anyone can
defend.

## Where I was wrong

Three predictions stamped in `PREDICTION.md` before `attenuation.py` ran, and
two missed.

The 1/J^2 law landed exactly as predicted.

I predicted 180 to 200 per arm at a judge of 0.90 and a 25-point effect.
Measured 74. I anchored the multiplier on 120, which is the n for the predicted
effect range, rather than on 46, which is the n for the 25-point reference cell
the question actually named. Mixing two reference points inside one arithmetic
step is the whole of the error, and it made me predict a crisis where the
measurement found none.

I predicted a floor near J = 0.5 below which no board rescues the design.
There is no floor. It degrades smoothly through 194 per arm at J = 0.50 and 305
at J = 0.40, and calling a cost curve a cliff was the wrong shape.

Both misses ran the same direction, toward believing the drafts had broken more
than they broke.

## What may be published from this well, going forward

The strip above was checked by the advocate seat rather than accepted on my
word, at `2026-09-10T08:27Z`, against the merged record and the pull request
body. It holds, and the reasoning that decides it is one I had not stated
explicitly: **nothing any of the three recruiters wrote appears anywhere in this
row.** Every published fragment is Kai's own drafted prose. Third-party identity
outranks Kai's own private facts, and it is not implicated here because no
counterparty is quoted. What is exposed is our own error in our own words.

That reasoning generalises, and this well will be drawn from again, so the rule
it implies is written here rather than left in a transcript:

* Only Kai's own drafted clauses travel to a public repository.
* A counterparty's words never do, and neither do the surrounding thread facts.
* If a future item needs a counterparty's sentence to be the scientific
  content, that item goes to a private repository and is referenced from here.
  The self-contained record is the thing that gets given up, not the boundary.

Ruled by the advocate seat, which owns the disclosure judgement. Recorded here
because it binds this evaluation's data handling, which is mine, and because a
boundary that lives only in a conversation gets re-decided under time pressure
by whoever arrives next.

**The general rule is not this file's.** It landed in `coilyco-bridge/lore`,
entry `lore-rule-disclosure-gradient`, at `2026-09-10T08:31:13Z`, and that entry
owns it for every public surface: a repository, a published artifact, a post, a
talk. What is above is the same rule as it binds this row, kept here so the
record stays readable on its own. If the two ever disagree, lore is right and
this is stale.

## A third instance, and it is only an anecdote

While landing that entry the advocate seat noted that this was the second time
in one day it had reasoned to a correct answer while failing to reach the
written rule that already said it. The rule that time was its own, about durable
work product, and it broke it inside the sentence announcing that it was writing
something down so nobody would decide it under pressure.

**This is an anecdote and it is not evidence.** One seat, self-reported, a
different rule family from the one under test, and no instrument touched it.

It is recorded because of what it does to the design rather than to the result.
The shape is identical to the incident: a written rule exists, the agent does
not reach it, and the agent reaches a defensible answer anyway. It recurred in a
seat actively primed on this exact failure mode, which is the condition all
three hypotheses predict should be most protective. If the shape crosses rule
families and survives priming, then a formatting defect in one skill family
explains less of it than the framing implies.

**It partly rehabilitates a request I set aside.** The reporting seat asked
first for a base rate: given a task whose class clearly maps to a skill, how
often does the skill load. I deprioritised it in favour of bracketing the
ceiling, on the grounds that the payoff has to exist before the mechanism
matters. That ordering still holds for the spend. But this observation is a
reason to think the base rate is the more informative number of the two, and it
should be measured across families rather than derived from the writing family
alone, which was the original request and was better than my compression of it.

Nothing here changes E0. It changes what E1 should be if E0 clears, and
`teable:coilyco-flight-deck/housecast#7328` carries it.

## One calibration item has a reader, and the rest do not

Form 3 is the only one of the three that was sent. It has a real recipient, and
as of this writing no reply.

So exactly one item in the calibration set has a behavioural outcome attached,
and it is unobserved. This bounds what the set can establish. The judge is being
calibrated on whether a sentence performs the move the rail names, which is a
question about the text. Whether the move actually lands badly on a reader is a
different question, this row does not measure it, and one pending reply would
not settle it either.

Worth stating because the rail's stated justification is about the reader, and a
grader that never sees a reader is measuring the rule rather than the harm it
exists to prevent. That gap is real and is not a defect in the design, as long
as no result from this row is reported as evidence about readers.

## Files added

* `attenuation.py` - the misclassification correction, five known answers, and
  what the approved board can still see.
* `attenuation.txt` - the run.
