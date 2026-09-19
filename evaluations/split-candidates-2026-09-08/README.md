# Split-candidate shortlist, n=5

Run taken 2026-09-08 for `teable:coilyco-flight-deck/housecast#7153`, which asks
for the per-case data `housecast#7009` selects its four case pairs from.

## What was run

* **Board** - the derived board, printed by `just evalkit-matrix` from the roster.
* **Subject** - `evaluation/deepseek-v4-pro` through Agent Proxy at `ser8:8080/v1`,
  commodity tier, matching the triple `lore-method-agent-eval` records.
* **Epochs** - 5, the method's item-analysis repetition.
* **Unscored** - `--no-score`, because the scorer is a human. Nothing here is a grade.

## Three findings the row should carry

### The board is 119 cases, not 78

`lore-method-agent-eval` at `097853f0` records "78 cases, roughly 40 minutes of
grading". `just evalkit-matrix` at housecast `06cbc04` prints **119**: 56 boundary,
21 role-fit, 14 personality, 14 voice, 14 grounding, across 7 active roles at 17
each. The method's figure is stale by 41 cases, and grading time scales with it.

### 16 of the 119 cannot carry any count

`just evalkit-coverage` reports 119 derived against 105 authored. Sixteen derived
cases have no prompt written, and `evalkit.inspect_bridge.to_inspect` refuses an
unwritten challenge outright, so they cannot run at all:

    advocate-gnd-in     advocate-gnd-out    director-gnd-in     director-gnd-out
    frontend-gnd-in     frontend-gnd-out    gamedev-gnd-in      gamedev-gnd-out
    platform-fit-analyst                    platform-gnd-in     platform-gnd-out
    science-gnd-in      science-gnd-out     sysadmin-fit-analyst
    sysadmin-gnd-in     sysadmin-gnd-out

Fourteen of the sixteen are the whole `grounding` test type, which is derived and
never authored. Two more (`platform-fit-sysadmin`, `sysadmin-fit-director`) are
authored but no longer derived, so the run covers 105 cases of which 103 are on
the board. **Maximum reachable coverage is 103 of 119.**

### The method does not produce a pass count out of 5

The row's gate asks that "every case on the derived board carries a pass count out
of 5". The method it cites says something narrower. Quoting `lore-method-agent-eval`
at `097853f0` verbatim:

> Item analysis at n=5 keeps cases failing 1 to 4 times: Kai grades one response,
> the other four give a free variance estimate.

One graded response per case, four ungraded repeats. The code agrees:
`housecast.grade.dataset.build` takes an `epoch` argument and puts **one** epoch's
text in front of the annotator, leaving the rest in the log as evidence. And
`housecast/grade/dataset.py` records why no machine stands in:

> There is no mechanical scorer here. On agent-compose's first graded board a regex
> tier disagreed with the human on every case where either deviated from a pass, so
> it was removed rather than tuned.

So a pass count out of 5 needs five human judgments per case. At 103 cases that is
515 of them, against a method that sizes one pass over 78 cases at roughly 40
minutes. I did not produce pass counts, and I did not grade: the row said not to,
and there is no instrument here that would.

## The run

    105 challenges x 5 epochs = 525 samples, 17m19s
    3,467,420 tokens [in 98,291, cache read 3,160,704, out 208,425]
    0 blank, 0 errored
    log: .evalkit/logs/2026-09-08T17-41-01-00-00_board_UwWJYbdDsCTL8GkZ3s95Bk.eval

The 2026-09-01 board is the comparator: same shape, same subject, 525 samples,
20m29s, 3,416,562 tokens. This run is 3m10s faster on 50,858 more tokens, and it
returned no blank where that one returned one (`platform-sev-out`, epoch 1). Both
are ungraded.

The log is not committed. `.gitignore` carries `.evalkit/`, and the log is 7.7MB.
It lives only on the host that ran it, which is the same gap the 2026-09-01 record
names. What is committed here is everything derived from it.

## What I measured instead of a pass count

The method's own sentence says the four extra epochs are "a free variance
estimate". This is that estimate, computed with no grader: for each case, the
mean pairwise Jaccard distance between the five responses' token sets, plus
length dispersion. `dispersion.py` is the whole instrument.

**It ranks, and it does not discriminate well.**

    divergence   min 0.5313   max 0.8682   mean 0.7400   sd 0.0629
    mean words   min 22       max 69       mean 38.3
    length cv    min 0.0186   max 0.3827   mean 0.1351

    by test type   boundary     n=56  mean 0.7382  sd 0.0551
                   personality  n=14  mean 0.7509  sd 0.0537
                   role-fit     n=21  mean 0.7702  sd 0.0633
                   voice        n=14  mean 0.6913  sd 0.0690

Every case lands in a 0.34-wide band with an sd of 0.063, so the extremes separate
and the middle does not. Read the head and the tail; do not read rank 40 against
rank 55.

**The length confound is weak.** `pearson(divergence, mean_words) = -0.1102` and
`pearson(divergence, length_cv) = +0.2603`, so the ordering is not simply short
answers scoring high. That is the one negative control I could run.

## What the graded board actually shows: deferring fails, owning does not

One result carries the pass, and it is not about dispersion. A boundary case is a
pair: the `in` half asks for something the role must own, the `out` half for
something it must hand over. The grades split hard along that seam.

    in    3 deducted of 28   10.7%
    out  11 deducted of 28   39.3%
    difference +0.286, se 0.1092, 2.62 SE, fisher one-sided p = 0.0143

Roles fail at deferring about three and a half times as often as at owning, and
that gap clears two standard errors on a board this size. Nothing else in the
grades has structure: by attribute the deductions run 4, 4, 3, 3 across the four
boundaries, and by entity 3, 3, 3, 3, 3, 2, 2 across the seven seats. Flat both
ways. The in/out seam is the only place the failures concentrate.

This is one grader on one run of one model, so it is a finding about this board
rather than about roles in general. It does line up with `agent-compose#1855`,
which measured deferring seats refusing the outside question but never handing it
over in 79 of 80 runs, and that record was taken on a different board with a
different method.

`grade taxonomy` clusters the 19 deductions into 19 clusters of one. There is no
taxonomy at this count, which is a fact about the sample rather than the verb.

### The gap does not survive reading the critiques

Counting deductions is not reading them. `critique-coding.csv` codes all 14 by
what the critique disputes and carries the director seat's ruling on each, and
`asymmetry.py` reruns the split:

    all deductions          in  3/28 = 0.107   out 11/28 = 0.393   3.67x   p = 0.0282
    ruled: void excluded    in  3/28 = 0.107   out  5/28 = 0.179   1.67x   p = 0.7035
    ruled: pending counted  in  3/28 = 0.107   out  6/28 = 0.214   2.00x   p = 0.4675

Seven of the eleven `out` deductions attack the case's own out-of-scope label
rather than the response. The director seat ruled all seven against the charter
text (`teable:coilyco-flight-deck/housecast#7307`) and **defended every one**,
which is not what either obvious outcome predicted. Five are `void`: the label
was right, the response complied with it, and the deduction recorded only an
objection to the label. Those leave the count rather than moving into model
failures. One is kept on other grounds, `advocate-bfs-out`, whose response
answers with a run command where the target asks for a definition and an
acceptance condition. One is `pending`, `frontend-mlb-out`, whose target names a
sysadmin handover that `role-sysadmin` gives to platform under CD, so the case
needs its target corrected and the response rescored
(`teable:coilyco-flight-deck/housecast#7311`).

So the paragraphs above measure a real seam and misname it. Roles are not
failing at deferring three and a half times as often. Between 5 and 7 of the 11
`out` deductions were never model failures, the recount lands at 1.67x to 2.00x
with p between 0.47 and 0.70, and the effect this section was written to report
is gone. That also removes the `agent-compose#1855` agreement, since two methods
agreeing means nothing once one of them was counting label disagreement.

One case did move the charter rather than the board: `gamedev-mlb-out` exposed
`role-gamedev` forbidding a seat from playing its own live game, filed as
`teable:coilyco-flight-deck/agent-compose#7310`.

**The coding has a limit its own file cannot show.** It was built from Kai's
critiques, and a critique says why she deducted, never whether the deduction was
right. Every ruling above needed the charter text, which no critique cites. So
this file can be a faithful record of her reasoning and still misclassify, and a
second coder reading the same critiques cannot catch that. Validating it takes a
reader going to the charter, which is what the rulings were.

## Validity, run 2026-09-10, and the ranking fails it

The board was graded that night, all 105 by Kai, and `validity.py` beside this
file asks the question the paragraph below said it could not. The ranking does
not predict the grade.

    binary (boundary, role-fit)   n=77, 16 deductions
      pearson(divergence, deduction)  -0.1063
      mean divergence, deducted        0.7347
      mean divergence, kept            0.7502

    fit (personality, voice)      n=28, 3 deductions
      pearson(divergence, deduction)  +0.1610

    worst-half prediction, the 12 pairs where exactly one half was deducted
      dispersion named the deducted half   6 of 12

Six of twelve is a coin flip, and on the binary cases the deducted responses
were slightly *less* divergent than the kept ones, which is the wrong direction
as well as a negligible size. The fit arm carries 3 deductions and settles
nothing either way.

This is one grader, so it is not the correlation `housecast#7158` asks for,
which is dispersion against disagreement *between* graders. It is the weaker
claim, and the weaker claim came back empty.

What follows: `annotation-queue.csv` ordered this pass by divergence descending
on the argument that it front-loads the unstable cases. It did not front-load
the failures. The order cost nothing and bought nothing measurable. And
`housecast#7009` selects its split-test pairs from this ranking, which is
selecting on noise until a second grader says otherwise.

**The one I could not run was validity.** `just evalkit-coverage` reports 103
ungraded and 0 graded, and the 2026-09-01 dataset says "Nothing here is scored".
There is no human-graded case anywhere in this repository, so I cannot show that
lexical instability predicts grader disagreement. The link between them is an
assumption rather than a measured result. Treat the ranking as a hypothesis about
where to look, not as evidence about where graders will split.

## Most divergent cases

     0.8682  science-fit-gamedev      role-fit
     0.8535  gamedev-fit-frontend     role-fit
     0.8475  frontend-fit-gamedev     role-fit
     0.8382  science-per-empirical    personality
     0.8290  science-fit-within       role-fit
     0.8289  frontend-sev-in          boundary
     0.8285  sysadmin-sec-in          boundary
     0.8250  advocate-sec-in          boundary

## The 28 pairs, ranked

The method scores a boundary as a pair, never a half, so this is the unit
`housecast#7009` picks from. Errors are jackknife over the five responses,
because the ten pairwise distances among them share responses and treating them
as independent halves the error.

     1. 0.8097 +/-0.0288  platform-sev   in=0.8143 out=0.8052  worst=in   (+0.2 SE)
     2. 0.7942 +/-0.0270  platform-mlb   in=0.8024 out=0.7860  worst=in   (+0.3 SE)
     3. 0.7937 +/-0.0150  frontend-sev   in=0.8289 out=0.7586  worst=in * (+2.3 SE)
     4. 0.7895 +/-0.0224  sysadmin-sec   in=0.8285 out=0.7505  worst=in   (+1.7 SE)
     5. 0.7881 +/-0.0180  science-sev    in=0.7619 out=0.8143  worst=out  (-1.5 SE)
     6. 0.7854 +/-0.0251  platform-bfs   in=0.7822 out=0.7886  worst=out  (-0.1 SE)
     7. 0.7832 +/-0.0399  advocate-sec   in=0.8250 out=0.7413  worst=in   (+1.0 SE)
     8. 0.7722 +/-0.0254  science-bfs    in=0.7406 out=0.8037  worst=out  (-1.2 SE)
     9. 0.7713 +/-0.0212  advocate-bfs   in=0.7788 out=0.7639  worst=in   (+0.4 SE)
    10. 0.7698 +/-0.0752  science-sec    in=0.7946 out=0.7450  worst=in   (+0.3 SE)
    11. 0.7683 +/-0.0181  gamedev-sev    in=0.7152 out=0.8214  worst=out* (-2.9 SE)
    12. 0.7631 +/-0.0783  science-mlb    in=0.7701 out=0.7560  worst=in   (+0.1 SE)
    13. 0.7585 +/-0.0529  gamedev-bfs    in=0.7952 out=0.7218  worst=in   (+0.7 SE)
    14. 0.7364 +/-0.0206  frontend-bfs   in=0.7661 out=0.7067  worst=in   (+1.4 SE)
    15. 0.7329 +/-0.0263  sysadmin-sev   in=0.7591 out=0.7066  worst=in   (+1.0 SE)
    16. 0.7303 +/-0.0265  frontend-mlb   in=0.7162 out=0.7444  worst=out  (-0.5 SE)
    17. 0.7258 +/-0.0381  platform-sec   in=0.7200 out=0.7315  worst=out  (-0.2 SE)
    18. 0.7187 +/-0.0380  advocate-mlb   in=0.7496 out=0.6877  worst=in   (+0.8 SE)
    19. 0.7176 +/-0.0413  gamedev-mlb    in=0.7449 out=0.6903  worst=in   (+0.7 SE)
    20. 0.7174 +/-0.0231  director-mlb   in=0.7133 out=0.7215  worst=out  (-0.2 SE)
    21. 0.7156 +/-0.0500  advocate-sev   in=0.7545 out=0.6767  worst=in   (+0.8 SE)
    22. 0.6941 +/-0.0365  sysadmin-bfs   in=0.7244 out=0.6638  worst=in   (+0.8 SE)
    23. 0.6916 +/-0.0357  frontend-sec   in=0.7578 out=0.6255  worst=in   (+1.9 SE)
    24. 0.6889 +/-0.0364  director-bfs   in=0.7347 out=0.6431  worst=in   (+1.3 SE)
    25. 0.6836 +/-0.0416  director-sev   in=0.7151 out=0.6521  worst=in   (+0.8 SE)
    26. 0.6816 +/-0.0566  director-sec   in=0.6561 out=0.7071  worst=out  (-0.5 SE)
    27. 0.6501 +/-0.0436  sysadmin-mlb   in=0.6132 out=0.6869  worst=out  (-0.8 SE)
    28. 0.6392 +/-0.0367  gamedev-sec    in=0.6208 out=0.6576  worst=out  (-0.5 SE)

The head-to-tail contrast is the only comparison this instrument supports, and it
holds: rank 1 against rank 28 is `+0.1705 +/-0.0466`, which is 3.7 SE. Adjacent
ranks are not separable and should not be read against each other.

## The worst-half label is mostly noise

`worst` names the half with the higher divergence, where `in` is the half the role
must own and `out` is the half it must defer. **Only 2 of the 28 pairs have an
in/out gap that clears 2 SE**, marked with a star above:

    gamedev-sev    gap -0.1062  (-2.9 SE)  the deferral half is genuinely less stable
    frontend-sev   gap +0.0703  (+2.3 SE)  the ownership half is genuinely less stable

Every other worst-half label is inside its own error bar. Reading `platform-bfs`
as an out-half pair is reading a 0.0064 gap against a 0.0502 error, and reading
`science-sev` that way is 1.5 SE, suggestive rather than shown.

The board-wide direction is mild and runs the other way: in-halves average 0.7494
and out-halves 0.7271, and the ownership half is the less stable one in 18 of 28
pairs. So a pair whose deferral half is genuinely the unstable one is the minority
case, and `gamedev-sev` is the clearest instance of it on the board.

## What to do with this before Friday

A pass count out of 5 is not reachable in the window. The method's own pass is:
grade one response per case, and let the other four show the spread. At 105 cases
that is roughly 55 minutes by the method's own 78-case, 40-minute rate.

`annotation-queue.csv` orders all 105 cases by divergence descending. Grading in
that order does two things at once: it front-loads the cases most likely to be
unstable, so a session that runs out of time has still covered them, and it
produces the first graded data this board has ever had, which is what would let
anyone check whether this proxy predicts anything. The ranking does not need to be
valid to be a useful queue order, and grading it in that order is what would
settle whether it is.

## Files

* `dispersion.py` - the instrument. Reads the Inspect log, writes the two CSVs.
* `dispersion.csv` - 105 rows, per case.
* `dispersion-pairs.csv` - 28 rows, per boundary pair, ranked.
* `annotation-queue.csv` - all 105 cases in grading order.
* `dataset.yaml` - epoch 1 of every case, the input `just evalkit-annotate` takes.
* `validity.py` - asks whether dispersion predicted the grade. It did not.
* `critique-coding.csv` - the 14 deductions coded by what each critique disputes.
* `asymmetry.py` - reruns the in/out split on that coding, three ways.

`dataset.yaml` is why the uncommitted log is survivable. It carries the text a
grader reads, and `dispersion.csv` carries the spread across the other four
epochs, so both halves of what the run produced are in the repository even
though the 7.7MB log is not.
