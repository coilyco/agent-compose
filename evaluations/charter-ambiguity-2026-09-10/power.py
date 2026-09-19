"""Can the board detect the charter-ambiguity effect at all?

`housecast#7317` pre-registers a criterion and a retirement rule: score every
case 0/1/2 for charter ambiguity before grading, then retire the criterion if
the 2s do not disagree more than the 0s. That rule is a direction check. This
asks what a direction check is worth on a board this size, and what a test with
a stated alpha would cost instead.

Two board sizes, because the director seat ruled personality and voice out of
the criterion after the first run. 77 is the criterion set and the one that
decides anything. 105 is the whole authored board, kept so the cost of the
exclusion is legible rather than asserted.

Nothing here reads the board. It is a simulation over stratum sizes and rates
nobody has measured yet, so read it as the operating envelope the design has to
fit inside, not as a result about roles.
"""

from __future__ import annotations

import csv
import pathlib
import random
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))

from stats import simulate_power

HERE = pathlib.Path(__file__).parent

# 77 authored boundary and role-fit cases carry the criterion. 105 is every
# authored case, before the exclusion.
BOARDS = (77, 105)
TRIALS = 20000
ALPHA = 0.05
SEED = 20260910

# Share of the board scoring 2, then 1. The rest score 0. The director seat
# scored seven cases while forming the hypothesis, far too few to estimate a
# share from, so the grid spans plausible ones.
SHARE_2 = (0.05, 0.10, 0.20, 0.33)
SHARE_1 = 0.30
BASE = (0.05, 0.10, 0.20)  # disagreement rate among the 0s
LIFT = (1.0, 1.5, 2.0, 3.0, 4.0)  # multiplier on the 2s; 1.0 is the null


def main() -> int:
    rng = random.Random(SEED)
    rows = []
    for board in BOARDS:
        for share2 in SHARE_2:
            n2 = round(board * share2)
            n1 = round(board * SHARE_1)
            n0 = board - n1 - n2
            for p0 in BASE:
                for lift in LIFT:
                    out = simulate_power((n0, n1, n2), p0, lift, TRIALS, rng, ALPHA)
                    rows.append(
                        {
                            "board": board,
                            "share_2": share2,
                            "n0": n0,
                            "n1": n1,
                            "n2": n2,
                            "p0": p0,
                            "p2": round(min(1.0, p0 * lift), 4),
                            "lift": lift,
                            "null": lift == 1.0,
                            "direction": round(out["direction"], 4),
                            "fisher": round(out["fisher"], 4),
                            "trend": round(out["trend"], 4),
                        }
                    )

    out_path = HERE / "power.csv"
    with out_path.open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]), lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)

    print(f"boards {BOARDS}, {TRIALS} trials per cell, alpha {ALPHA}, seed {SEED}")
    print(f"wrote {out_path.name}, {len(rows)} cells\n")

    for index, board in enumerate(BOARDS):
        if index:
            print()
        print(f"--- board {board} ---")
        print("under the null (lift 1.0):")
        print(f"{'n2':>4} {'p0':>6} {'direction':>10} {'fisher':>8} {'trend':>8}")
        for r in rows:
            if r["board"] == board and r["null"]:
                print(
                    f"{r['n2']:>4} {r['p0']:>6} {r['direction']:>10.3f} "
                    f"{r['fisher']:>8.3f} {r['trend']:>8.3f}"
                )
        print(f"\npower at p0 = 0.10, board {board}:")
        print(f"{'n2':>4} {'lift':>5} {'p2':>6} {'direction':>10} {'fisher':>8} {'trend':>8}")
        for r in rows:
            if r["board"] == board and r["p0"] == 0.10 and not r["null"]:
                print(
                    f"{r['n2']:>4} {r['lift']:>5} {r['p2']:>6} {r['direction']:>10.3f} "
                    f"{r['fisher']:>8.3f} {r['trend']:>8.3f}"
                )
    return 0


if __name__ == "__main__":
    sys.exit(main())
