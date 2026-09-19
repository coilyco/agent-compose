"""Power for E0, two independent proportions, one-sided.

The outcome is binary per generation: does the draft trip a held-out
comparison-family rule. C0 has no rail text, C1 has half of it. The contrast is
C1 minus C0, and the test is a one-sided Fisher exact, which is what two small
proportions honestly support.

Stdlib only, so the number does not depend on which venv ran it. Fisher is
exact rather than approximate, so the decision table is precomputed once per
(n1, n2) and the simulation is a lookup.

    just run evaluations/skill-discovery-2026-09-10/power.py
"""

from __future__ import annotations

import random
from math import comb, sqrt

ALPHA = 0.05
TRIALS = 20000
SEED = 20260910


def fisher_one_sided(a: int, b: int, c: int, d: int) -> float:
    """P(X >= a) on the 2x2 [[a,b],[c,d]], hypergeometric. Row/col totals fixed."""
    r1, r2, c1 = a + b, c + d, a + c
    n = r1 + r2
    denom = comb(n, c1)
    lo, hi = max(0, c1 - r2), min(r1, c1)
    return sum(comb(r1, x) * comb(r2, c1 - x) for x in range(a, hi + 1)) / denom


def decision_table(n0: int, n1: int) -> list[list[bool]]:
    """reject[x0][x1] for the one-sided test that C1 violates LESS than C0."""
    return [
        [fisher_one_sided(x0, n0 - x0, x1, n1 - x1) <= ALPHA for x1 in range(n1 + 1)]
        for x0 in range(n0 + 1)
    ]


def power(p0: float, p1: float, n0: int, n1: int, trials: int = TRIALS) -> float:
    rng = random.Random(SEED)
    table = decision_table(n0, n1)
    hits = 0
    for _ in range(trials):
        x0 = sum(rng.random() < p0 for _ in range(n0))
        x1 = sum(rng.random() < p1 for _ in range(n1))
        hits += table[x0][x1]
    return hits / trials


def n_normal(p0: float, p1: float, alpha: float = ALPHA, beta: float = 0.20) -> int:
    """Closed-form two-proportion n per arm. The approximation, for comparison."""
    za, zb = 1.6448536269514722, 0.8416212335729143
    pbar = (p0 + p1) / 2
    num = (
        za * sqrt(2 * pbar * (1 - pbar))
        + zb * sqrt(p0 * (1 - p0) + p1 * (1 - p1))
    ) ** 2
    from math import ceil

    return ceil(num / (p1 - p0) ** 2)


def check() -> None:
    """Known answers, computed off the machine, before any real number is quoted."""
    # Fisher's tea tasting, 4/4 correct on the 8-cup design: 1/70.
    assert abs(fisher_one_sided(4, 0, 0, 4) - 1 / 70) < 1e-12, "tea tasting"
    # The 3/1 vs 1/3 table: (16 + 1)/70 = 17/70, the published one-sided value.
    assert abs(fisher_one_sided(3, 1, 1, 3) - 17 / 70) < 1e-12, "17/70"
    # Closed form at the reference cell, hand-computed above: 46 per arm.
    assert n_normal(0.50, 0.75) == 46, f"reference cell {n_normal(0.50, 0.75)}"
    # A null contrast cannot beat alpha more than alpha of the time.
    assert power(0.50, 0.50, 60, 60) <= 0.06, "null calibration"
    print("check: 4 known answers pass\n")


if __name__ == "__main__":
    check()

    print("CLOSED FORM, n per arm for 80% power, one-sided alpha 0.05")
    print("  base   lift    C1     n/arm")
    for p0 in (0.40, 0.50, 0.60):
        for lift in (0.10, 0.15, 0.25, 0.35):
            p1 = p0 - lift  # C1 violates less
            if p1 <= 0.02:
                continue
            print(f"  {p0:.2f}  -{lift:.2f}  {p1:.2f}  {n_normal(p1, p0):6d}")

    print("\nEXACT (Fisher, simulated), power at the sizes a run might buy")
    print("  n/arm   C0    C1    power")
    for n in (20, 30, 40, 60, 80, 120):
        for p0, p1 in ((0.50, 0.25), (0.50, 0.35), (0.50, 0.40)):
            print(f"  {n:5d}  {p0:.2f}  {p1:.2f}   {power(p0, p1, n, n):.3f}")

    print("\nMINIMUM DETECTABLE EFFECT at 80% power, base C0 = 0.50")
    for n in (20, 30, 40, 60, 80, 120):
        mde = None
        for step in range(1, 50):
            p1 = 0.50 - step / 100
            if p1 <= 0.01:
                break
            if power(0.50, p1, n, n, trials=4000) >= 0.80:
                mde = 0.50 - p1
                break
        print(f"  n={n:3d}  MDE = {mde if mde is None else f'{mde:.2f}'} points")

    print("\nNULL CALIBRATION, no true effect")
    for n in (30, 60, 120):
        print(f"  n={n:3d}  false-positive rate = {power(0.50, 0.50, n, n):.4f}")
