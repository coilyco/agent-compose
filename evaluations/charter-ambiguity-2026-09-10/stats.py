"""The statistics `power.py` and `validity.py` both need, in one place.

They were duplicated across the two files, which is two answers to one question
and the thing that goes wrong quietly when only one copy gets fixed. No scipy in
this directory, so the closed forms are written out.
"""

from __future__ import annotations

import math
import random

SCORES = (0, 1, 2)
Z_95 = 1.959963985  # two-sided 95%, for the interval
Z_ONE_SIDED = 1.6448536  # one-sided 5%, for the trend decision


def fisher_one_sided(a: int, b: int, c: int, d: int) -> float:
    """P(as extreme or more) for [[a,b],[c,d]], testing a/(a+b) > c/(c+d)."""
    row1, row2 = a + b, c + d
    col1 = a + c
    total = row1 + row2
    if row1 == 0 or row2 == 0 or col1 == 0 or col1 == total:
        return 1.0
    hi = min(row1, col1)
    return sum(
        math.comb(row1, k) * math.comb(row2, col1 - k) / math.comb(total, col1)
        for k in range(a, hi + 1)
    )


def cochran_armitage(counts: list[tuple[int, int]]) -> float:
    """One-sided z for a linear trend in proportion across 0, 1, 2."""
    total = sum(n for _, n in counts)
    events = sum(d for d, _ in counts)
    if total == 0 or events in (0, total):
        return 0.0
    p = events / total
    mean = sum(s * n for s, (_, n) in zip(SCORES, counts, strict=True)) / total
    num = sum(s * (d - n * p) for s, (d, n) in zip(SCORES, counts, strict=True))
    var = p * (1 - p) * sum(n * (s - mean) ** 2 for s, (_, n) in zip(SCORES, counts, strict=True))
    return num / math.sqrt(var) if var > 0 else 0.0


def normal_sf(z: float) -> float:
    return 0.5 * math.erfc(z / math.sqrt(2))


def wilson(x: int, n: int, z: float = Z_95) -> tuple[float, float]:
    """Score interval for one proportion. Behaves at 0 and n where Wald does not."""
    if n == 0:
        return (float("nan"), float("nan"))
    centre = (x + z * z / 2) / (n + z * z)
    half = z * math.sqrt(x * (n - x) / n + z * z / 4) / (n + z * z)
    return (max(0.0, centre - half), min(1.0, centre + half))


def newcombe(x1: int, n1: int, x2: int, n2: int, z: float = Z_95) -> tuple[float, float]:
    """Hybrid-score interval for p1 - p2, Newcombe 1998 method 10.

    An interval rather than a p-value is what an underpowered board can honestly
    produce, so this is the estimator the pilot reports.
    """
    if n1 == 0 or n2 == 0:
        return (float("nan"), float("nan"))
    p1, p2 = x1 / n1, x2 / n2
    l1, u1 = wilson(x1, n1, z)
    l2, u2 = wilson(x2, n2, z)
    delta = p1 - p2
    lower = delta - math.sqrt((p1 - l1) ** 2 + (u2 - p2) ** 2)
    upper = delta + math.sqrt((u1 - p1) ** 2 + (p2 - l2) ** 2)
    return (max(-1.0, lower), min(1.0, upper))


def simulate_power(
    sizes: tuple[int, int, int],
    p0: float,
    lift: float,
    trials: int,
    rng: random.Random,
    alpha: float = 0.05,
) -> dict[str, float]:
    """How often each rule fires at these strata and rates. lift 1.0 is the null."""
    n0, n1, n2 = sizes
    p2 = min(1.0, p0 * lift)
    p1 = (p0 + p2) / 2  # the 1s sit halfway, which is the trend test's own assumption
    direction = fisher = trend = 0
    for _ in range(trials):
        d0 = sum(1 for _ in range(n0) if rng.random() < p0)
        d1 = sum(1 for _ in range(n1) if rng.random() < p1)
        d2 = sum(1 for _ in range(n2) if rng.random() < p2)
        if n0 and n2 and d2 / n2 > d0 / n0:
            direction += 1
        if fisher_one_sided(d2, n2 - d2, d0, n0 - d0) <= alpha:
            fisher += 1
        if cochran_armitage([(d0, n0), (d1, n1), (d2, n2)]) >= Z_ONE_SIDED:
            trend += 1
    return {
        "direction": direction / trials,
        "fisher": fisher / trials,
        "trend": trend / trials,
    }
