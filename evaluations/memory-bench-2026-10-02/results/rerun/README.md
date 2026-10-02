# Rerun results, 2026-10-02 (Amendment 3)

Run 10:07-10:49Z, `teable:coilyco/agent-compose#8673`. Same protocol as the first
run except the case spec: one exact string per lore entry. The case file has 30
cases, 20 L1 and 10 L2, at 1.33 strings per case, sha256
`840a218048eeaee609ed85491255f363559932943f94ab3d4f01e9aa55b58f2e`. casegen
rejected 2 L1 drafts and 38 L2 drafts. `score2.txt` is the score output, and
the rows hold ids, booleans and token counts only.

## Measured, two runs each

* **L1 strict recall:** C0 9/20 and 11/20, mean 0.50. C1 14/20 and 13/20, mean
  0.675. The gap is 0.175, against a run-to-run spread of 0.10 for C0 and 0.05
  for C1.
* **L2 strict recall:** 2/10 in all four runs. Cross-link cases still floor for
  both on a 14B model, so this measures nothing between them.
* **Source cited:** L1 C0 10 and 14, C1 15 and 14. L2 C0 5 and 7, C1 8 and 7.
* **Tokens per question:** L1 C0 74,927 and 75,828, C1 62,555 and 64,420, so
  C1 is about 16 percent lower. On L2, C1 is lower as well.
* **Sentinels:** C1 10/10 old and 10/10 new with no manual reindex, 0 stale. C0
  10/10 old and 9/10 new, 0 stale.

## Against the Amendment 3 prediction

1. Both L1 above 0.40: held, C0 0.45 and 0.55, C1 0.70 and 0.65.
2. C1 >= C0 by 0.1 to 0.2: held on L1 at 0.175. Not shown on L2, which is floored.
3. C1 tokens 10 to 25 percent lower: held, at about 16 percent.

## Jev

`jev-1.13.0`, from these aggregates:
* **Is the L1 gap real:** yes at p=0.60, unclear 0.28, noise 0.12. Confidence 0.40.
* **Recommendation:** adopt after the host trial (`teable:coilyco/agentic-os-kai#8677`)
  confirms cross-harness wiring, then amend doctrine, p=0.45. Adopt now 0.28,
  more measurement 0.14, keep current practice 0.13. Confidence is 0.27, so the
  decision is Kai's.
