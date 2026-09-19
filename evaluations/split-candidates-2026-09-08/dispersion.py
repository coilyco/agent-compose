"""Per-case response dispersion across epochs, computed without a grader.

The board runs unscored because the scorer is a human, so a pass count out of 5
does not exist until someone annotates. This measures the thing a pass count was
being used as a proxy for: whether the subject answers the same case the same
way twice. It is a variability statistic, not a grade, and nothing here decides
whether a response was correct.
"""

from __future__ import annotations

import argparse
import csv
import glob
import re
import statistics
from collections import defaultdict
from itertools import combinations
from pathlib import Path

from inspect_ai.log import read_eval_log

WORD = re.compile(r"[a-z0-9']+")


def tokens(text: str) -> set[str]:
    return set(WORD.findall(text.lower()))


def jaccard_distance(left: set[str], right: set[str]) -> float:
    if not left and not right:
        return 0.0
    return 1.0 - len(left & right) / len(left | right)


def completion(sample) -> str:
    output = getattr(sample, "output", None)
    return (getattr(output, "completion", "") or "").strip() if output else ""


def mean_divergence(token_sets: list[set[str]]) -> float:
    pairs = [jaccard_distance(a, b) for a, b in combinations(token_sets, 2)]
    return statistics.fmean(pairs) if pairs else 0.0


def jackknife_se(token_sets: list[set[str]]) -> float:
    """Leave one response out.

    The 10 pairwise distances among 5 responses share responses, so treating
    them as independent halves the standard error and makes small in/out gaps
    look separable when they are not.
    """
    n = len(token_sets)
    if n < 3:
        return 0.0
    reps = [mean_divergence(token_sets[:i] + token_sets[i + 1:]) for i in range(n)]
    mean = statistics.fmean(reps)
    return ((n - 1) / n * sum((r - mean) ** 2 for r in reps)) ** 0.5


def summarise(texts: list[str]) -> dict[str, float]:
    lengths = [len(t.split()) for t in texts]
    token_sets = [tokens(t) for t in texts]
    pairs = [jaccard_distance(a, b) for a, b in combinations(token_sets, 2)]
    mean_len = statistics.fmean(lengths) if lengths else 0.0
    return {
        "epochs": len(texts),
        "blank": sum(1 for t in texts if not t),
        "mean_words": round(mean_len, 1),
        # Dispersion of length, scale-free so a terse case and a long one compare.
        "length_cv": round(statistics.pstdev(lengths) / mean_len, 4) if mean_len else 0.0,
        "divergence": round(mean_divergence(token_sets), 4),
        "divergence_se": round(jackknife_se(token_sets), 4),
        "divergence_max": round(max(pairs), 4) if pairs else 0.0,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--log", default="", help="path to the .eval log; default is the newest")
    parser.add_argument("--out", default="dispersion.csv")
    args = parser.parse_args()

    path = args.log or sorted(glob.glob(".evalkit/logs/*.eval"))[-1]
    log = read_eval_log(path)
    print(f"log: {path}")
    print(f"status: {log.status}")

    by_case: dict[str, list[str]] = defaultdict(list)
    errors: dict[str, int] = defaultdict(int)
    meta: dict[str, dict] = {}
    for sample in log.samples or []:
        case = str(sample.id)
        by_case[case].append(completion(sample))
        if getattr(sample, "error", None):
            errors[case] += 1
        meta.setdefault(case, dict(sample.metadata or {}))

    rows = []
    for case, texts in sorted(by_case.items()):
        row = {"case": case}
        keys = ("entity", "test_type", "attribute", "half", "pair_id")
        row.update({k: meta[case].get(k, "") for k in keys})
        row.update(summarise(texts))
        row["errors"] = errors[case]
        rows.append(row)

    out = Path(args.out)
    with out.open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)

    print(f"cases: {len(rows)}  samples: {sum(r['epochs'] for r in rows)}")
    print(f"wrote {out}")
    ranked = sorted(rows, key=lambda r: r["divergence"], reverse=True)
    print("\nmost divergent 15 cases:")
    for row in ranked[:15]:
        print(f"  {row['divergence']:.4f}  {row['case']:<30} {row['test_type']:<12} n={row['epochs']}")
    print("\nleast divergent 10 cases:")
    for row in ranked[-10:]:
        print(f"  {row['divergence']:.4f}  {row['case']:<30} {row['test_type']:<12} n={row['epochs']}")

    # The method scores a boundary as a pair, never a half, so rank pairs too.
    pairs: dict[str, list[dict]] = defaultdict(list)
    for row in rows:
        if row["pair_id"]:
            pairs[str(row["pair_id"])].append(row)
    pair_rows = []
    for pair_id, halves in pairs.items():
        by_half = {str(h["half"]): h for h in halves}
        if set(by_half) != {"in", "out"}:
            continue
        owns, defers = by_half["in"], by_half["out"]
        gap = owns["divergence"] - defers["divergence"]
        # Errors add in quadrature, so the gap is noisier than either half.
        gap_se = (owns["divergence_se"] ** 2 + defers["divergence_se"] ** 2) ** 0.5
        pair_rows.append(
            {
                "pair_id": pair_id,
                "entity": owns["entity"],
                "attribute": owns["attribute"],
                "divergence": round((owns["divergence"] + defers["divergence"]) / 2, 4),
                "divergence_se": round(gap_se / 2, 4),
                "in": owns["divergence"],
                "out": defers["divergence"],
                "gap": round(gap, 4),
                "gap_se": round(gap_se, 4),
                "gap_sigma": round(gap / gap_se, 1) if gap_se else 0.0,
                "worst_half": "in" if gap > 0 else "out",
                "worst_half_separable": abs(gap) > 2 * gap_se,
            }
        )
    pair_out = out.with_name(out.stem + "-pairs.csv")
    with pair_out.open("w", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(pair_rows[0].keys()))
        writer.writeheader()
        writer.writerows(sorted(pair_rows, key=lambda r: r["divergence"], reverse=True))
    print(f"\npairs: {len(pair_rows)}  wrote {pair_out}")
    print("\nall pairs, ranked:")
    for i, row in enumerate(sorted(pair_rows, key=lambda r: r["divergence"], reverse=True), 1):
        star = "*" if row["worst_half_separable"] else " "
        print(
            f"  {i:>2}. {row['divergence']:.4f} +/-{row['divergence_se']:.4f}  "
            f"{row['pair_id']:<14} in={row['in']:.4f} out={row['out']:.4f}  "
            f"worst={row['worst_half']:<3}{star} ({row['gap_sigma']:+.1f} SE)"
        )
    print("\n  * = the in/out gap clears 2 SE. Every other worst-half label is noise.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
