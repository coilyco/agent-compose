# Charter ambiguity as a split-test selection criterion: the pre-registration

Written 2026-09-10 for `teable:coilyco-flight-deck/housecast#7317`, at housecast
`64ecb7b`. Nothing here is a result about the criterion. The board it applies to
has not been scored and cannot be graded yet, and that ordering is the point of
the row.

## Why a pre-registration and not a run

Divergence went into production unfalsified. It ranked 105 cases, `housecast#7009`
selected its split-test pairs from that ranking, and only afterwards did anyone
ask whether it predicted anything. It did not: AUC 0.4608, permutation p = 0.85,
and 6 of 12 worst-half hits against a coin flip of 6.0. The replacement criterion
gets its falsification test written down first, with the numbers that decide it
fixed before any of them exist.

## The criterion

Set by the director seat. A case is a split-test candidate when the behaviour it
demands is not settled by a single unambiguous sentence in the governing charter.

* **0** - one verbatim sentence decides it
* **1** - a general rule decides it but has to be applied
* **2** - two sources bear on it and disagree, or no sentence reaches it

The hypothesis is that disagreement between graders rises with the score.

## The criterion covers 77 cases, not the whole board

The rubric asks whether one charter sentence decides the behaviour. That question
is answerable for boundary, role-fit and grounding, and it is not answerable for
personality or voice, so those two are outside the criterion. Ruled by the
director seat before any score existed, and the reason is worth stating rather
than recording as a trim.

A personality target reads, in full, `tenacious, composed alongside grounded`.
There is no sentence to look up. Whether a response is tenacious is a judgement
about register, and scoring it 0 because no sentence decides it, or 2 because no
sentence reaches it, are both category errors rather than readings.

**Voice is the one that would have broken the result.** A voice target carries a
literal word list, `Reach for: builds; lands; the contract is`. A word list is
verbatim, so the rubric read naively scores voice 0. But the demand underneath it
is `reaches for its own register and refuses the borrowed one`, which is pure
judgement and plausibly where two uncalibrated graders split hardest. That puts
the likeliest-to-disagree cases in the lowest bucket, which does not add noise. It
pulls the trend down. `validity.py` would then report a significant downward
trend, and the honest reading of that output is that the criterion failed, when
the criterion never applied to those rows. A pre-registration that bakes that in
is worse than none, because it launders the error through a correct procedure.

The arithmetic of what is left, from `scoring-sheet.csv`:

    test_type      derived  authored
    boundary            56        56
    role-fit            21        21
    personality         14        14
    voice               14        14
    grounding           14         0
    TOTAL              119       105

Excluding personality and voice leaves 91 derived and **77 authored**. 77 is the
number that matters. All 14 grounding cases are derived and none is authored, so
they carry no prompt, produce no response, and can never contribute a
disagreement. `in_criterion` in the sheet marks exactly those 77.

## The order of operations, which is the substance

1. The director seat fills `scoring-sheet.csv` for the 77 rows marked
   `in_criterion`, reading the charter, before any response has been graded and
   before any grader has been recruited.
2. The filled sheet is committed. That commit is the pre-registration.
3. The board runs and two graders grade it, writing `annotations.<grader>.yaml`.
4. `housecast grade disagreement --format json` produces the disagreement report.
5. `validity.py` joins the two and reports what it cannot influence.

A score assigned once the grades are visible is not a prediction, and that is
exactly the mistake that produced the row this one replaces.

## The gate's literal wording is not a falsification test

The gate says the criterion is retired if the 2s do not disagree more than the
0s. Taken literally that is a bare direction check, and `power.py` measures what
one is worth. Under the null, on the 77-case criterion set, it fires between
0.186 and 0.520 of the time depending on nothing but how many 2s the scoring
happened to produce:

    n2     p0   direction   fisher    trend
     4   0.05       0.186    0.011    0.060
     8   0.10       0.449    0.018    0.061
    15   0.10       0.480    0.021    0.054
    25   0.05       0.520    0.008    0.057

So a criterion that predicts nothing survives its own falsification test about
half the time, and the survival rate is a property of the scoring rather than of
the world.

The rule `validity.py` applies instead, fixed before any data:

* **Primary** - a one-sided Cochran-Armitage trend across 0, 1 and 2 at alpha
  0.05, `z >= 1.645`. It spends every scored case rather than the two end strata.
* **And** the direction has to hold, so a significant downward trend cannot be
  read as support.
* **Reported, never decisive** - the one-sided Fisher exact on the 0-vs-2 table,
  because that contrast is the gate's own wording, and the bare direction check,
  because a reader will want it.

## This board estimates the effect. It does not retire the criterion

At a 10% share of 2s and a base disagreement rate of 0.10, on 77 cases:

    lift    p2   direction   fisher    trend
     1.5  0.15       0.603    0.049    0.141
     2.0  0.20       0.725    0.100    0.243
     3.0  0.30       0.883    0.268    0.516
     4.0  0.40       0.959    0.472    0.753

At a doubling of the disagreement rate the primary test finds it 24% of the time.
The exclusion cost real power, and it was still correct: the same cell on the
full 105 read 0.287, so the honest test got harder rather than easier.

Raising the share of 2s does not rescue it. Going from 8 of 77 to 25 of 77 moves
power at a 3x lift from 0.516 to 0.604, because the 0 stratum shrinks as the 2
stratum grows. **Board size is the binding constraint, and the route to a real
test is more authored cases rather than a cleverer statistic.**

So the deliverable is an estimate, not a verdict. `validity.py` computes power at
a pre-registered 2x target from the strata actually scored, and prints a verdict
only when it clears a pre-registered floor of 0.80. Below that it reports the
difference in disagreement rates with a Newcombe 95% interval, says no verdict is
authorised, and records what the rule would have said so a later powered board
has something to compare against. **A null at 24% power is a wide estimate rather
than evidence of absence**, and making that mechanical means nobody has to
remember it as a caveat.

## The instrument refuses two things

**It refuses to score a personality or voice case.** Not documented, asserted.
`enforce_criterion_set` raises and the CLI exits 2 without reporting a number.
The failure this prevents is not this board. It is someone in six months running
the instrument on a sheet that quietly readmitted voice, getting a clean downward
trend, and reporting the criterion as falsified. Documentation cannot stop that
and an assertion can.

**It refuses to retire the criterion on a board without the power to.** The floor
above, applied to the realized strata.

Both refusals are tested. On a sheet with the 14 voice rows scored, the CLI exits
2 and names them.

## The instrument is checked, and it is calibrated

`test_validity.py` asserts every statistic against a value computed off the
machine: Fisher against the tea-tasting table's 17/70, the trend z against a
hand-computed 10 / sqrt(40/9), the Newcombe interval against the published
worked example for 56/70 against 48/80, which reproduces to `(0.0524, 0.3339)`.
28 tests, all passing, run before the instrument saw a real score. It is not in
`testpaths`, which stops at the two packages, so run it with
`just test evaluations/charter-ambiguity-2026-09-10/test_validity.py`.

`calibration.txt` carries what the rule does when nothing is there to find, at
the 46/23/8 strata the criterion set implies:

    unconditional, 20000 independent draws, no true effect
      retained (trend AND direction)  0.0593
      trend alone                     0.0594
      direction alone                 0.4455

    conditional, 200 realizations x 500 permutations
      mean 0.0598   median 0.0580   5th 0.0320   95th 0.0920

**0.0593 against a nominal 0.05 is mildly anti-conservative**, and it got worse
with the smaller board: the same measurement at 60/37/8 read 0.0542. The normal
approximation inside Cochran-Armitage is doing that, at eight cases in the top
stratum. It is reported rather than corrected because no verdict fires on this
board anyway, and a rule that cannot retire anything cannot over-retire. A board
that clears the power floor should have this remeasured at its own strata first.

Read a single `--shuffle` run against the 0.032 to 0.092 band and not against
0.05, because one run conditions on one realization of the disagreement vector.

The `direction alone` line, 0.4455 under a null with no effect in it, is the
power table's finding arriving by a second route.

## What the power gate caught on synthetic data

An end-to-end run on the 77 rows with scores and disagreement drawn
independently, so no effect exists by construction:

    estimate  2s minus 0s = +0.1179  95% CI [-0.0592, +0.5163]
    trend     z = +1.7254  p = 0.0422
    direction 2s above 0s: 0.1667 vs 0.0488
    power at a 2x lift, from these strata: 0.170
    NO VERDICT.

The trend cleared alpha, the direction held, and the bare rule would have said
RETAINED on pure noise. The gate withheld it and the interval straddles zero.
That is one draw rather than a rate, and the rate is the 0.0593 above, but it is
what the failure looks like when it happens.

## Where I was wrong

`PREDICTION.md` was written before each run and missed four times.

Before the first `power.py`: Fisher power at the reference cell came in at 0.357
against a predicted ceiling of 0.35, and the trend test at 0.603 against a
predicted 0.40 to 0.55. The direction check's false-keep rate landed inside the
predicted 0.35 to 0.50 at the reference cell and outside it at the edges, which
the prediction did not anticipate varying at all.

Then I predicted the 0.0730 negative control meant an anti-conservative
approximation at 0.06 to 0.09. At 105 the measurement said 0.0542, so that was
wrong. It is worth noting the smaller board later produced 0.0593, which is the
effect I claimed to see in the wrong place.

Before the n=77 rerun I predicted trend power of 0.48 to 0.55 at a 3x lift and
0.22 to 0.26 at 2x. Measured 0.516 and 0.243. Both inside.

## What is blocked, and what this row does not need

Disagreement needs two graders and this board has one, so no verdict can be
produced here regardless of the power floor. `housecast#7160` holds the
recruitment behind Kai naming a community, and `housecast#7158` is blocked at the
same gate.

None of that blocks the scoring, which needs a charter and a reader. Kai has
ruled that it goes to a fresh session rather than a tired one, because 77 charter
readings have no useful checkpoint: a partial sheet is not a pre-registration.
The labour is deferred and the sequence is not. **The scoring belongs to the
director seat.**

## Files

* `PREDICTION.md` - written before each run, kept so the numbers cannot be read
  back as the ones I expected.
* `stats.py` - Fisher, Cochran-Armitage, Wilson, Newcombe, and the power
  simulator. One copy, because `power.py` and `validity.py` both need them and
  two copies are two answers.
* `power.py` - the grid over stratum sizes and rates, at both board sizes.
  Writes `power.csv`, 120 cells, 20000 trials each.
* `power.csv` and `power.txt` - the envelope this design fits inside, and the run.
* `scoring_sheet.py` - derives the sheet through `evalkit.matrix`, so a roster
  change moves it. Regenerate against the roster the next run actually uses.
* `scoring-sheet.csv` - 119 derived, 105 authored, 77 `in_criterion`, `score` and
  `why` empty.
* `validity.py` - the analysis, the two refusals, and the retirement rule. Reads
  the filled sheet and the disagreement report, and computes no disagreement of
  its own, because `housecast.grade.agreement` owns that definition.
* `test_validity.py` - 28 known-answer checks on it.
* `calibration.py` and `calibration.txt` - what the rule does when nothing is
  there to find.
