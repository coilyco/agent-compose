# Amendment 1: confirm the confidence gate, with multi-gold labels and larger n

Written 2026-10-02T11:32:38Z, before any rerun case exists. `teable:coilyco/agent-compose#8695`.

The first run's single-source gold labels were wrong, because answers repeat
across notes. Its label-free downstream signal was that Jev helps as a
confidence gate, with n = 10 to 11. This rerun confirms or refutes that claim.

## What changes

* **Cases.** `casegen.py --single --l1 200 --l2 0` on the same public corpus,
  agentic-os at `5a07e64b`, seed 8632. 200 is the draft ceiling.
* **Deterministic multi-gold labels, with no model judgment.** A case survives
  only if its answer string is at least 6 characters and appears, case-folded, in
  at most 3 corpus files. Its gold set is every corpus file that contains it. That
  makes containment close to answering, and `ok` covers every answer-bearing
  note. Jev is the subject, so no model labels the gold.
* **Unchanged:** retrieval (Basic Memory top 10), the Jev choice and jevroute
  `score()` at 0.9, the downstream arms (top 3, against Jev's pick when it
  passes), ministral-3-14b, and two reps.

## The claim, stated as the measures that confirm it

Jev as a gate: where it clears 0.9 and narrows to one note, the answer is no
worse and much cheaper.

Prediction, EXPECTED, written now:
1. At least 100 cases survive the distinctiveness filter.
2. Gate precision, pass and correct over pass, >= 0.85.
3. Pass rate between 0.25 and 0.45.
4. On narrowed cases: B recall - A recall >= +0.10, harm (A right, B wrong)
   <= 2 per rep, and B tokens <= 0.5x A.
5. Whole run: B recall >= A recall, and B tokens <= 0.85x.
6. Selection overall: |B correct - A rank 1 correct| <= 0.05, so Jev is not a
   better ranker.

The claim is **confirmed** if 2, 4 and 5 hold in both reps, and **refuted** if
harm exceeds 2 in either rep or gate precision falls below 0.75.
