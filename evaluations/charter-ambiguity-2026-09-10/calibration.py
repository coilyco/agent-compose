"""Does the retirement rule in `validity.py` fire at the alpha it claims?

A rule that decides whether a criterion survives has to be checked before it
decides anything, and the first `--shuffle` run came back at 0.0730 against a
nominal 0.05, which is 4 binomial SE high. Two candidate causes: the normal
approximation inside Cochran-Armitage going anti-conservative at eight cases in
the top stratum, or one realization's conditional permutation null sitting high.

Both are measurable, so this measures both instead of picking one.
"""

from __future__ import annotations

import pathlib
import random
import statistics
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))

import validity

HERE = pathlib.Path(__file__).parent
# 46/23/8 over the 77 cases in the criterion, a 10% share of 2s, and a
# disagreement rate no one has measured yet. The shape rather than the truth.
STRATA = (46, 23, 8)
RATE = 0.15
SEED = 4242


def unconditional(trials: int, rng: random.Random) -> dict[str, float]:
    """Draw scores and disagreement independently. No effect exists to find."""
    fired = trend = direction = 0
    for _ in range(trials):
        counts = [(sum(1 for _ in range(n) if rng.random() < RATE), n) for n in STRATA]
        result = validity.verdict(counts)
        fired += bool(result["retained"])
        trend += bool(result["trend_fires"])
        direction += bool(result["direction"])
    return {
        "retained": fired / trials,
        "trend": trend / trials,
        "direction": direction / trials,
    }


def conditional(realizations: int, perms: int, rng: random.Random) -> list[float]:
    """The CLI's own `--shuffle` control, repeated over many realizations."""
    labels = [0] * STRATA[0] + [1] * STRATA[1] + [2] * STRATA[2]
    rates = []
    for _ in range(realizations):
        disagreed = [rng.random() < RATE for _ in labels]
        pool = list(labels)
        fired = 0
        for _ in range(perms):
            rng.shuffle(pool)
            counts = [[0, 0], [0, 0], [0, 0]]
            for score, flag in zip(pool, disagreed, strict=True):
                counts[score][1] += 1
                counts[score][0] += flag
            if validity.verdict([tuple(c) for c in counts])["retained"]:
                fired += 1
        rates.append(fired / perms)
    return sorted(rates)


def main() -> int:
    rng = random.Random(SEED)
    lines = [f"strata {STRATA}, disagreement {RATE}, seed {SEED}, alpha {validity.ALPHA}", ""]

    trials = 20000
    out = unconditional(trials, rng)
    lines.append(f"unconditional, {trials} independent draws, no true effect")
    lines.append(f"  retained (trend AND direction)  {out['retained']:.4f}")
    lines.append(f"  trend alone                     {out['trend']:.4f}")
    lines.append(f"  direction alone                 {out['direction']:.4f}")
    lines.append(f"  binomial se at 0.05             {(0.05 * 0.95 / trials) ** 0.5:.4f}")
    lines.append("")

    realizations, perms = 200, 500
    rates = conditional(realizations, perms, rng)
    lines.append(f"conditional, {realizations} realizations x {perms} permutations")
    lines.append(f"  mean    {statistics.fmean(rates):.4f}")
    lines.append(f"  median  {statistics.median(rates):.4f}")
    lines.append(f"  5th     {rates[realizations // 20]:.4f}")
    lines.append(f"  95th    {rates[realizations - realizations // 20 - 1]:.4f}")
    lines.append(f"  min     {rates[0]:.4f}")
    lines.append(f"  max     {rates[-1]:.4f}")

    text = "\n".join(lines)
    (HERE / "calibration.txt").write_text(text + "\n")
    print(text)
    return 0


if __name__ == "__main__":
    sys.exit(main())
