"""Did the charter-ambiguity score predict grader disagreement?

The instrument `housecast#7317` pre-registers. It reads the scores the director
seat committed before grading began and the disagreement report `housecast grade
disagreement` produced after, and applies the pre-registered rule to the join.

It does not compute disagreement itself. That definition belongs to
`housecast.grade.agreement`, which decides what a missing grade means, and a
second copy here would be a second answer to the same question.

Two things it refuses to do, both deliberate:

* **It refuses to score a personality or voice case.** They are outside the
  criterion, and a voice target is a verbatim word list that would score 0 while
  plausibly disagreeing most. Quietly readmitting them produces a clean downward
  trend and an honest-looking report that the criterion was falsified, which is
  the one failure documentation cannot prevent. So this is an assertion.
* **It refuses to retire the criterion on a board without the power to.** Below
  the pre-registered floor it reports an effect estimate with an interval and
  says so, because a null at 24% power is a wide estimate rather than evidence
  of absence.

Run `--shuffle` for the negative control. Read the number against the 0.032 to
0.092 band in `calibration.txt`, not against 0.05: one run conditions on one
realization of the disagreement vector.
"""

from __future__ import annotations

import argparse
import csv
import json
import pathlib
import random
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))

from stats import (
    SCORES,
    Z_ONE_SIDED,
    cochran_armitage,
    fisher_one_sided,
    newcombe,
    normal_sf,
    simulate_power,
)

ALPHA = 0.05
# Outside the criterion by the director seat's ruling, recorded before any score
# existed. `scoring_sheet.py` carries the reasoning.
EXCLUDED_TEST_TYPES = frozenset({"personality", "voice"})
# The effect the design is sized against, and the power it must reach before a
# verdict is allowed. Fixed here rather than chosen once the numbers are in.
TARGET_LIFT = 2.0
POWER_FLOOR = 0.80
POWER_TRIALS = 4000


class CriterionSetError(Exception):
    """A scored row that the criterion never covered."""


def read_scores(path: pathlib.Path) -> tuple[dict[str, int], dict[str, str]]:
    """Case id to 0/1/2, and case id to test type. Unscored rows are dropped."""
    scored: dict[str, int] = {}
    types: dict[str, str] = {}
    with path.open(newline="") as handle:
        for row in csv.DictReader(handle):
            raw = (row.get("score") or "").strip()
            if raw == "":
                continue
            if raw not in {"0", "1", "2"}:
                raise ValueError(f"{row['case']}: score {raw!r} is not 0, 1 or 2")
            scored[row["case"]] = int(raw)
            types[row["case"]] = (row.get("test_type") or "").strip()
    return scored, types


def enforce_criterion_set(types: dict[str, str]) -> None:
    """Refuse outright rather than report a number off the wrong rows."""
    offenders = sorted(case for case, kind in types.items() if kind in EXCLUDED_TEST_TYPES)
    if offenders:
        shown = ", ".join(offenders[:5])
        more = f", and {len(offenders) - 5} more" if len(offenders) > 5 else ""
        raise CriterionSetError(
            f"{len(offenders)} scored cases are outside the criterion: {shown}{more}. "
            "Personality and voice were excluded before any score existed. Scoring them "
            "puts the cases likeliest to disagree into the 0 bucket, which inverts the "
            "trend. Clear those scores rather than running this."
        )
    unknown = sorted(case for case, kind in types.items() if not kind)
    if unknown:
        raise CriterionSetError(
            f"{len(unknown)} scored cases carry no test_type, so the criterion set "
            f"cannot be checked: {', '.join(unknown[:5])}. Regenerate the sheet."
        )


def read_disagreement(path: pathlib.Path) -> dict[str, bool]:
    """Case id to disagreed, over the cases every grader reached."""
    report = json.loads(path.read_text())
    return {case["id"]: not case["agreed"] for case in report["per_case"] if "agreed" in case}


def tabulate(scores: dict[str, int], disagreed: dict[str, bool]) -> list[tuple[int, int]]:
    """(disagreed, total) per score level, over cases carrying both."""
    counts = {s: [0, 0] for s in SCORES}
    for case, score in scores.items():
        if case not in disagreed:
            continue
        counts[score][1] += 1
        counts[score][0] += int(disagreed[case])
    return [(counts[s][0], counts[s][1]) for s in SCORES]


def verdict(counts: list[tuple[int, int]]) -> dict[str, object]:
    (d0, n0), _, (d2, n2) = counts
    z = cochran_armitage(counts)
    rate0 = d0 / n0 if n0 else float("nan")
    rate2 = d2 / n2 if n2 else float("nan")
    lo, hi = newcombe(d2, n2, d0, n0)
    return {
        "z": z,
        "trend_p": normal_sf(z),
        "contrast_p": fisher_one_sided(d2, n2 - d2, d0, n0 - d0) if n0 and n2 else 1.0,
        "rate0": rate0,
        "rate2": rate2,
        "difference": rate2 - rate0 if n0 and n2 else float("nan"),
        "ci_low": lo,
        "ci_high": hi,
        "direction": bool(n0 and n2 and rate2 > rate0),
        "trend_fires": z >= Z_ONE_SIDED,
        "contrast_fires": (fisher_one_sided(d2, n2 - d2, d0, n0 - d0) <= ALPHA)
        if n0 and n2
        else False,
        # The decision the rule would reach. Whether it is allowed to stand is
        # `powered` below, which is a fact about the board rather than the data.
        "retained": bool(z >= Z_ONE_SIDED and n0 and n2 and rate2 > rate0),
    }


def achieved_power(counts: list[tuple[int, int]], seed: int) -> float:
    """Power at the pre-registered target lift, from the strata actually scored."""
    (d0, n0), (_, n1), (_, n2) = counts
    if not (n0 and n2):
        return 0.0
    base = d0 / n0
    if base == 0:
        return 0.0
    return simulate_power(
        (n0, n1, n2), base, TARGET_LIFT, POWER_TRIALS, random.Random(seed), ALPHA
    )["trend"]


def render(counts: list[tuple[int, int]], result: dict[str, object], power: float) -> str:
    lines = ["score  disagreed  compared  rate"]
    for score, (d, n) in zip(SCORES, counts, strict=True):
        rate = f"{d / n:.4f}" if n else "n/a"
        lines.append(f"{score:>5}  {d:>9}  {n:>8}  {rate:>6}")
    lines.append(f"total  {sum(d for d, _ in counts):>9}  {sum(n for _, n in counts):>8}")
    lines.append("")
    lines.append(
        f"estimate  2s minus 0s = {result['difference']:+.4f}  "
        f"95% CI [{result['ci_low']:+.4f}, {result['ci_high']:+.4f}]"
    )
    lines.append(f"trend     z = {result['z']:+.4f}  p = {result['trend_p']:.4f}")
    lines.append(f"contrast  0 vs 2 fisher one-sided p = {result['contrast_p']:.4f}")
    lines.append(
        f"direction 2s {'above' if result['direction'] else 'not above'} 0s: "
        f"{result['rate2']:.4f} vs {result['rate0']:.4f}"
    )
    lines.append("")
    lines.append(f"power at a {TARGET_LIFT:g}x lift, from these strata: {power:.3f}")
    if power >= POWER_FLOOR:
        lines.append(
            "VERDICT: criterion RETAINED" if result["retained"] else "VERDICT: criterion RETIRED"
        )
    else:
        lines.append(
            f"NO VERDICT. Power {power:.3f} is below the pre-registered floor of "
            f"{POWER_FLOOR:.2f}, so this board estimates the effect and cannot retire "
            "the criterion. Read the interval above. A null here is a wide estimate, "
            "not evidence of absence."
        )
        lines.append(
            "The rule would have said "
            + ("RETAINED" if result["retained"] else "RETIRED")
            + ", recorded so a later powered board can be compared against it."
        )
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Charter-ambiguity score against disagreement.")
    parser.add_argument("--scores", type=pathlib.Path, required=True, help="filled scoring sheet")
    parser.add_argument(
        "--disagreement",
        type=pathlib.Path,
        required=True,
        help="`housecast grade disagreement --format json` output",
    )
    parser.add_argument(
        "--shuffle",
        type=int,
        metavar="N",
        help="negative control: permute scores N times and report how often the rule fires",
    )
    parser.add_argument("--seed", type=int, default=20260910)
    args = parser.parse_args(argv)

    scores, types = read_scores(args.scores)
    try:
        enforce_criterion_set(types)
    except CriterionSetError as exc:
        print(f"refusing to report: {exc}", file=sys.stderr)
        return 2

    disagreed = read_disagreement(args.disagreement)
    joined = [case for case in scores if case in disagreed]
    if not joined:
        print("no case carries both a score and a grade from every grader", file=sys.stderr)
        return 1

    counts = tabulate(scores, disagreed)
    result = verdict(counts)
    power = achieved_power(counts, args.seed)
    print(f"scored {len(scores)}, compared {len(disagreed)}, joined {len(joined)}\n")
    print(render(counts, result, power))

    if args.shuffle:
        rng = random.Random(args.seed)
        values = [scores[case] for case in joined]
        fired = 0
        for _ in range(args.shuffle):
            rng.shuffle(values)
            permuted = dict(zip(joined, values, strict=True))
            if verdict(tabulate(permuted, disagreed))["retained"]:
                fired += 1
        print(
            f"\nnegative control: {fired}/{args.shuffle} permutations would have retained "
            f"({fired / args.shuffle:.4f}). Read against the 0.032 to 0.092 band in "
            "calibration.txt, not against 0.05."
        )
    return 0


if __name__ == "__main__":
    sys.exit(main())
