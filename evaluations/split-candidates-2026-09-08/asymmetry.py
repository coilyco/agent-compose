"""Does the in/out gap survive reading the critiques, or is it a label dispute?

`validity.py` beside this asked whether dispersion predicted the grade and found
it did not. This asks the next question about the finding that replaced it: the
README reports roles failing at deferring about three and a half times as often
as at owning, and that number counts deductions without reading them.

`critique-coding.csv` codes all 14 deductions by what the critique disputes, and
carries the director seat's ruling on each disputed label.
`design` means it rejects the case's own in/out label or calls the case a bad
test, so the response may have been scored against a boundary its grader does
not hold. `model` means it faults what the response did. That coding is one
coder on free text, an interpretation rather than a measurement, and it is a
committed file precisely so a second reader can change a line and rerun this.

Permutation rather than Fisher because the deduction class is 19 of 105 and the
question here is a difference of proportions on 56 cases.
"""

from __future__ import annotations

import csv
import pathlib
import random

HERE = pathlib.Path(__file__).parent
HALF_N = 28
SEED = 20260910
ITERS = 200_000


def split(coding: list[dict], keep: set[str]) -> tuple[int, int]:
    rows = [r for r in coding if r["disputes"] in keep]
    return (sum(r["half"] == "in" for r in rows), sum(r["half"] == "out" for r in rows))


def permute(bad_in: int, bad_out: int) -> float:
    labels = [1] * bad_in + [0] * (HALF_N - bad_in) + [1] * bad_out + [0] * (HALF_N - bad_out)
    observed = abs(bad_out / HALF_N - bad_in / HALF_N)
    rng = random.Random(SEED)
    hits = 0
    for _ in range(ITERS):
        rng.shuffle(labels)
        a = sum(labels[:HALF_N]) / HALF_N
        b = sum(labels[HALF_N:]) / HALF_N
        hits += abs(b - a) >= observed
    return (hits + 1) / (ITERS + 1)


def main() -> int:
    coding = list(csv.DictReader((HERE / "critique-coding.csv").open()))
    print(f"coded deductions: {len(coding)}\n")
    for tag, keep in (("all deductions", {"design", "model"}),
                      ("model failures only", {"model"}),
                      ("label disputes only", {"design"})):
        bi, bo = split(coding, keep)
        report(tag, bi, bo)

    # The rulings are the director seat's, case by case, against the charter
    # text. See housecast#7307.
    print()
    for tag, keep in (("ruled: void excluded", {"keep"}),
                      ("ruled: pending counted", {"keep", "pending"})):
        rows = [r for r in coding if r["ruling"] in keep]
        report(tag, sum(r["half"] == "in" for r in rows), sum(r["half"] == "out" for r in rows))
    return 0


def report(tag: str, bad_in: int, bad_out: int) -> None:
    ri, ro = bad_in / HALF_N, bad_out / HALF_N
    ratio = f"{ro / ri:.2f}x" if ri else "undefined"
    print(f"{tag:24} in {bad_in:2}/{HALF_N} = {ri:.3f}   out {bad_out:2}/{HALF_N} = {ro:.3f}   "
          f"ratio {ratio:>9}   p = {permute(bad_in, bad_out):.4f}")


if __name__ == "__main__":
    raise SystemExit(main())
