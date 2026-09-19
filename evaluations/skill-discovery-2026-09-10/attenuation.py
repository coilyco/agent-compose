"""What an imperfect judge costs, in sample size.

The drafts forced the primary outcome from a regex pair to a judge, because a
model can avoid two strings and still write the third opener. A judge
misclassifies. Under non-differential misclassification the observed rate is

    p_obs = p * se + (1 - p) * (1 - sp)

so a contrast shrinks by exactly the Youden index J = se + sp - 1, and n scales
as 1 / J^2. Both arms are graded by the same judge, which is what makes the
attenuation non-differential and the correction this clean.

    just run evaluations/skill-discovery-2026-09-10/attenuation.py
"""

from __future__ import annotations

from math import ceil, sqrt

from power import n_normal

ALPHA = 0.05


def observed(p: float, se: float, sp: float) -> float:
    return p * se + (1 - p) * (1 - sp)


def youden(se: float, sp: float) -> float:
    return se + sp - 1


def n_with_judge(p0: float, p1: float, se: float, sp: float) -> int | None:
    """n per arm at 80% power on what the judge actually reports."""
    q0, q1 = observed(p0, se, sp), observed(p1, se, sp)
    if q0 - q1 < 1e-9:
        return None
    return n_normal(q1, q0)


def check() -> None:
    # A perfect judge attenuates nothing.
    assert abs(observed(0.5, 1.0, 1.0) - 0.5) < 1e-12, "perfect judge"
    assert youden(1.0, 1.0) == 1.0, "perfect Youden"
    # A coin-flip judge destroys the contrast entirely.
    assert abs(observed(0.9, 0.5, 0.5) - 0.5) < 1e-12, "coin flip judge"
    assert youden(0.5, 0.5) == 0.0, "zero Youden"
    # Attenuation equals the Youden index, at the reference contrast.
    lhs = observed(0.50, 0.9, 0.9) - observed(0.25, 0.9, 0.9)
    assert abs(lhs - 0.25 * youden(0.9, 0.9)) < 1e-12, "attenuation identity"
    print("check: 5 known answers pass\n")


if __name__ == "__main__":
    check()

    print("ATTENUATION, true contrast C0 = 0.50 against C1 = 0.25")
    print("  se    sp    J      observed contrast   n/arm at 80%")
    for se, sp in ((1.00, 1.00), (0.95, 0.95), (0.90, 0.90), (0.85, 0.85),
                   (0.80, 0.80), (0.75, 0.75), (0.70, 0.70), (0.60, 0.60)):
        j = youden(se, sp)
        obs = observed(0.50, se, sp) - observed(0.25, se, sp)
        n = n_with_judge(0.50, 0.25, se, sp)
        print(f"  {se:.2f}  {sp:.2f}  {j:.2f}   {obs:+.3f}              "
              f"{'unmeasurable' if n is None else n}")

    print("\nCOST RELATIVE TO A PERFECT GRADER, the 1/J^2 law")
    base = n_with_judge(0.50, 0.25, 1.0, 1.0)
    for se in (0.95, 0.90, 0.85, 0.80, 0.75, 0.70):
        n = n_with_judge(0.50, 0.25, se, se)
        j = youden(se, se)
        print(f"  se=sp={se:.2f}  J={j:.2f}  n={n:4d}  "
              f"ratio={n / base:.2f}x  1/J^2={1 / j**2:.2f}x")

    print("\nWHAT THE APPROVED BOARD OF 120/ARM CAN STILL SEE")
    print("  se=sp   J      minimum true contrast detectable at 80%")
    for se in (1.00, 0.95, 0.90, 0.85, 0.80, 0.75):
        j = youden(se, se)
        found = None
        for step in range(1, 50):
            p1 = 0.50 - step / 100
            if p1 <= 0.01:
                break
            n = n_with_judge(0.50, p1, se, se)
            if n is not None and n <= 120:
                found = 0.50 - p1
                break
        print(f"  {se:.2f}   {j:.2f}   "
              f"{'nothing at any size' if found is None else f'{found:.2f}'}")
