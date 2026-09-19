"""Does response dispersion predict how a case was graded?

`dispersion.py` ranked all 105 cases by how far the five epochs diverged from
each other, and its record said plainly what it could not do: "There is no
human-graded case anywhere in this repository, so I cannot show that lexical
instability predicts grader disagreement. Treat the ranking as a hypothesis
about where to look."

The board is graded now, by one grader. That is not the gate's correlation,
which is dispersion against *disagreement between* graders and needs a second
one. It is the weaker claim the single pass can carry: dispersion against the
grade itself. Read it as a check on whether the ranking pointed anywhere at all.
"""

from __future__ import annotations

import csv
import math
import pathlib
import sys

import yaml

HERE = pathlib.Path(__file__).parent
DEDUCTIONS = {"fail", "does-not-fit", "undecided"}


def pearson(xs: list[float], ys: list[float]) -> float:
    n = len(xs)
    mx, my = sum(xs) / n, sum(ys) / n
    num = sum((x - mx) * (y - my) for x, y in zip(xs, ys))
    dx = math.sqrt(sum((x - mx) ** 2 for x in xs))
    dy = math.sqrt(sum((y - my) ** 2 for y in ys))
    return num / (dx * dy) if dx and dy else float("nan")


def main(annotations_path: pathlib.Path) -> int:
    graded = {
        row["id"]: row["label"]
        for row in yaml.safe_load(annotations_path.read_text())["annotations"]
    }
    cases = {row["case"]: row for row in csv.DictReader((HERE / "dispersion.csv").open())}
    pairs = list(csv.DictReader((HERE / "dispersion-pairs.csv").open()))

    print(f"graded {len(graded)} of {len(cases)} cases\n")

    for label, keep in (("binary (boundary, role-fit)", {"boundary", "role-fit"}),
                        ("fit (personality, voice)", {"personality", "voice"})):
        ids = [c for c, r in cases.items() if r["test_type"] in keep and c in graded]
        div = [float(cases[c]["divergence"]) for c in ids]
        bad = [1.0 if graded[c] in DEDUCTIONS else 0.0 for c in ids]
        n_bad = int(sum(bad))
        print(f"{label}: n={len(ids)}, deductions={n_bad}")
        if 0 < n_bad < len(ids):
            hi = [d for d, b in zip(div, bad) if b]
            lo = [d for d, b in zip(div, bad) if not b]
            print(f"  pearson(divergence, deduction) = {pearson(div, bad):+.4f}")
            print(f"  mean divergence, deducted = {sum(hi)/len(hi):.4f}")
            print(f"  mean divergence, kept     = {sum(lo)/len(lo):.4f}")
            print(f"  difference                = {sum(hi)/len(hi) - sum(lo)/len(lo):+.4f}")
        print()

    # The sharpest available test: dispersion named a worst half per pair before
    # anyone graded. Where exactly one half was deducted, was it that one?
    hits = misses = 0
    separable_hits = separable_total = 0
    for pair in pairs:
        halves = {h: graded.get(f"{pair['pair_id']}-{h}") for h in ("in", "out")}
        if None in halves.values():
            continue
        deducted = [h for h, label in halves.items() if label in DEDUCTIONS]
        if len(deducted) != 1:
            continue
        correct = deducted[0] == pair["worst_half"]
        hits += correct
        misses += not correct
        if pair["worst_half_separable"] == "True":
            separable_total += 1
            separable_hits += correct

    decided = hits + misses
    print(f"worst-half prediction, pairs where exactly one half was deducted: n={decided}")
    if decided:
        print(f"  dispersion named the deducted half: {hits} of {decided} = {hits/decided:.1%}")
        print(f"  coin flip would be {decided/2:.1f}")
    if separable_total:
        print(f"  restricted to the 2 pairs whose gap cleared 2 SE: "
              f"{separable_hits} of {separable_total}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(pathlib.Path(sys.argv[1])))
