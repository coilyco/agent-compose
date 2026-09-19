# Written before `power.py` ran

Stamped `2026-09-10T07:00:01Z`, `date -u`, at housecast `7b9f8cc`. This file
exists so the power numbers below cannot be read back as the ones I expected.

The gate on `teable:coilyco-flight-deck/housecast#7317` says the criterion is
"retired if the 2s do not disagree more than the 0s". That is a direction check,
not a significance test. Three numbers decide whether it is worth running.

1. **False-keep rate of the bare direction check under the null.** With no real
   effect, `p2_hat > p0_hat` is close to a coin flip, so I expect the criterion
   to survive its own falsification test between **0.35 and 0.50** of the time.
   Ties at small n push it below 0.5 rather than above.
2. **Power of a one-sided Fisher exact 0-vs-2 contrast**, at n=105, a 10% share
   of 2s (n2 about 10), base disagreement 0.10, and a 3x lift to 0.30. I expect
   **under 0.35**.
3. **Power of a Cochran-Armitage trend test** across all three levels at the same
   cell, which spends every case rather than two strata. I expect it higher than
   the Fisher contrast, **0.40 to 0.55**.

If 1 lands where I expect, the gate as written retires or keeps by coin flip and
needs replacing before the board is scored, not after.

# Written before the n=77 rerun

Stamped `2026-09-10T07:16:23Z`, `date -u`, after the director seat ruled personality and voice
out of the criterion and before `power.py` ran again on what is left.

The criterion set drops from 105 to 77. Grounding contributes nothing either
way, because all 14 of its cases are derived and none is authored. At a 10%
share of 2s that is `n2` about 8 against 10, and `n0` about 46 against 60.

* Trend power at `p0 = 0.10` and a 3x lift: **0.48 to 0.55**, down from 0.603.
* Trend power at the same base and a 2x lift: **0.22 to 0.26**, down from 0.287.

If the second lands where I expect, a 2x effect is invisible on this board about
three times in four, which is the number that turns the run into a pilot.
