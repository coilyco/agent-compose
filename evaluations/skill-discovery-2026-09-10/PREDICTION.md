# Written before `power.py` ran

Stamped `2026-09-10T08:13:57Z`, `date -u`, at housecast `3c3a2e4`. This file
exists so the power numbers below cannot be read back as the ones I expected.

The design is `README.md`. Three numbers decide whether E0 is worth running.

1. **n per arm for 80% power at a 25-point lift**, C0 = 0.50 to C1 = 0.75,
   one-sided alpha 0.05, two independent proportions. I expect **50 to 60**.
2. **n per arm for 80% power at a 15-point lift**, 0.50 to 0.65, same alpha. I
   expect **120 to 160**.
3. **Minimum detectable effect at n = 30 per arm**, 80% power, base 0.50. I
   expect **30 to 35 points**.

If 3 lands where I expect, a 30-per-arm pilot can only see an effect large
enough that nobody needed a measurement to believe it, and the pilot is worth
running for the base rate alone rather than for the contrast.

# Written before E0 runs

Stamped `2026-09-10T08:13:57Z`, same reading, before any generation exists.

4. **C0, the held-out violation rate with no rail text available.** Gem broke
   the rail on 2 of 3 recruiter drafts, n=3 and self-reported. Flattery by
   comparison is a natural move in recruiter-reply register, so I expect the
   held-out violation rate to be **0.35 to 0.65**, and I expect it to be
   clearly above zero. If it comes in under 0.10, the rail is not a live
   failure mode under this prompt and the whole line of work is answering a
   question the data does not pose.
5. **C1, the held-out violation rate with the rail text in the system prompt.**
   The shown half of the rail names two phrasings and the graded half names two
   others, so C1 measures generalisation rather than string avoidance. I expect
   **0.10 to 0.30**: better than C0, and well short of zero, because a model
   told not to say `which is rare` will reach for `unlike most` unless it has
   understood the move rather than the string.
6. **C1 minus C0**, the ceiling on what any discoverability or composition fix
   can buy. I expect **+0.20 to +0.45**.

If 6 comes in under +0.10, then loading the skill barely changes the artifact,
and H1 against H2 is a choice between two cosmetic fixes. That result would
retire the question rather than answer it, and it is the outcome I am least
confident about and most want to be wrong about cheaply.

# Written before the attenuation grid ran

Stamped `2026-09-10T08:22:56Z`, `date -u`, at housecast `919dba6`. The
reporting seat returned the three real openers and one of them defeats the
grader I pre-registered, so the primary outcome changes from a regex pair to a
judge. A judge misclassifies, and misclassification attenuates a contrast
toward zero. This is stamped before the new grid runs.

7. **The attenuation factor.** With sensitivity `se` and specificity `sp`, I
   expect the observed contrast to shrink by the Youden index `se + sp - 1`,
   and I expect n to scale as its inverse square. At `se = sp = 0.90` that is a
   factor of 0.80 and a 1.56x cost in n.
8. **n per arm at 80% power, 25-point true effect, judge at se = sp = 0.90.** I
   expect **180 to 200**, against the 120 the perfect-grader board assumed.
9. **The floor.** Below some Youden index the design cannot be rescued by
   sample size at any affordable board. I expect that floor to sit near
   **J = 0.5**, meaning a judge agreeing with a human much worse than 0.75 on
   each margin makes this unmeasurable rather than expensive.

If 8 lands where I expect, the board Kai approved at 120 per arm is
underpowered for the design the drafts forced, and the honest move is to say so
before spending it rather than after.
