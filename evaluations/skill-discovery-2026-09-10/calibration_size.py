"""How many calibration items buy a usable interval on sensitivity and
specificity, for the judge AMENDMENT.md makes the critical path.

attenuation.py answers what an imperfect judge costs once its margins are
known. This answers how many labelled items it takes to know them at all. A
judge whose margins carry a wide interval hands the board a wide interval on
its own size, which is the same as not knowing.

Wilson score interval, which is the right one near 1.0 where the normal
approximation runs off the end of the scale. Four known-answer checks against
published values run first.
"""
import math

def wilson(k, n, z=1.96):
    if n == 0: return (0.0, 1.0)
    p = k / n
    d = 1 + z*z/n
    c = (p + z*z/(2*n)) / d
    h = z*math.sqrt(p*(1-p)/n + z*z/(4*n*n)) / d
    return (max(0.0, c-h), min(1.0, c+h))

def main():
    # Known answers, to 4dp and gated at 0.0005. An earlier draft of this table
    # carried 0.980 for the 95/100 upper bound, which is wrong by 0.0015, and a
    # 0.002 tolerance passed it. A check loose enough to accept a wrong expected
    # value is not gating anything.
    checks = [
        ("0/10   -> (0.0000, 0.2775)", wilson(0, 10),   (0.0000, 0.2775)),
        ("5/10   -> (0.2366, 0.7634)", wilson(5, 10),   (0.2366, 0.7634)),
        ("9/10   -> (0.5958, 0.9821)", wilson(9, 10),   (0.5958, 0.9821)),
        ("95/100 -> (0.8882, 0.9785)", wilson(95, 100), (0.8882, 0.9785)),
    ]
    ok = True
    for lab, got, exp in checks:
        good = abs(got[0]-exp[0]) < 0.0005 and abs(got[1]-exp[1]) < 0.0005
        ok &= good
        print(f"  {'ok  ' if good else 'FAIL'}  {lab}  got ({got[0]:.4f}, {got[1]:.4f})")
    if not ok:
        raise SystemExit("known-answer check failed, refusing to report")

    print()
    print("Interval half-width on ONE margin (se or sp), at true rate 0.90,")
    print("assuming the judge scores exactly 0.90 on that margin.")
    print(f"{'items on that margin':>21} {'k/n':>9} {'95% interval':>18} {'half-width':>11}")
    for n in (10, 15, 20, 25, 30, 40, 50, 75, 100):
        k = round(0.90 * n)
        lo, hi = wilson(k, n)
        print(f"{n:>21} {k:>4}/{n:<4} ({lo:.3f}, {hi:.3f})   {(hi-lo)/2:>9.3f}")

    print()
    print("What that does to the Youden index J = se + sp - 1, both margins at 0.90")
    print("(J drives board size as 1/J^2, per attenuation.py on the sibling row)")
    print(f"{'items PER margin':>17} {'J point':>8} {'J plausible range':>20} {'board multiplier range':>24}")
    for n in (10, 20, 30, 40, 50, 100):
        k = round(0.90 * n)
        lo, hi = wilson(k, n)
        j_lo = max(0.01, lo + lo - 1)
        j_hi = min(1.0, hi + hi - 1)
        j = 0.8
        print(f"{n:>17} {j:>8.2f} {f'{j_lo:.2f} to {j_hi:.2f}':>20} "
              f"{f'{1/j_hi**2:.1f}x to {1/j_lo**2:.1f}x':>24}")


if __name__ == "__main__":
    main()
