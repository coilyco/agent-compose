# How big the calibration set has to be

Stamped `2026-09-10T19:51:49Z` at housecast `a46d2b3`. `AMENDMENT.md` made the
judge calibration the critical path and did not say how many labelled items it
takes. This answers that and changes nothing else on the row. The stamped files
beside it are left as they were.

`attenuation.py` answers what an imperfect judge costs once its margins are
known. This answers how many items it takes to know them at all.

## The measurement

`calibration_size.py`, Wilson score interval, four known-answer checks gated at
0.0005 before it prints.

The first draft of that check carried 0.980 as the upper bound for 95 of 100,
which is wrong by 0.0015, and a 0.002 tolerance passed it. A check loose enough
to accept a wrong expected value gates nothing, so the expected values are now
exact to four places and the tolerance is tighter than the error that slipped
through. Recorded rather than quietly fixed, because it is the same shape as
the defect this row exists to catch. Assuming the judge scores 0.90 on a margin,
the 95% interval on that margin and what it does to the Youden index that
drives board size as `1/J^2`:

    items per margin   J point   J plausible range   board multiplier
              10        0.80        0.19 to 0.96      1.1x to 27.2x
              20        0.80        0.40 to 0.94       1.1x to 6.3x
              30        0.80        0.49 to 0.93       1.2x to 4.2x
              40        0.80        0.54 to 0.92       1.2x to 3.4x
              50        0.80        0.57 to 0.91       1.2x to 3.0x
             100        0.80        0.65 to 0.89       1.3x to 2.4x

**Ten per margin is not a calibration.** It leaves the required board anywhere
between 1x and 27x, which is indistinguishable from not having measured. The
curve flattens around 40.

So 40 positives and 40 negatives is the target, 30 and 30 is the floor worth
reporting from, and below 20 a judge with an unknown margin is worse than no
judge, because it produces a number with an interval nobody can defend.

Three real positives already exist, from `AMENDMENT.md`. One is contested and
stays labelled contested.

## The requirement that decides whether the set is worth building

Hard negatives. The judge's failure mode is firing on any contrastive prose, so
a set whose negatives are obviously clean measures nothing. The negatives have
to sit next to the move without being it: ordinary factual comparison, direct
praise with no absent third party, a rarity claim that is true and load-bearing,
and a comparison to a named present party.

The contamination rule that killed the pre-registered regex primary applies
here unchanged. No calibration item may reuse a phrasing the rail itself names,
or the set measures string matching and reports it as judgement.

Each item carries its provenance: real or constructed, who wrote it, whether it
was sent. The advocate seat authors both the rail's failures and the items, and
that is a fact the grader needs and cannot otherwise see.

## What this does not do

It does not make E0 more likely to clear its gate. A well-calibrated judge
measuring a contrast of zero still measures zero, and that outcome retires the
question rather than answering it.

Whether to build the judge at all is `teable:coilyco-flight-deck/housecast#7368`,
which is the Portfolio Director's call rather than this row's.
