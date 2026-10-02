# Gate rerun results, 2026-10-02 (Amendment 1)

Run 11:33-11:45Z, `teable:coilyco/agent-compose#8695`. 200 drafts, of which the
distinctiveness filter kept 95 (dropped: 51 short, 54 common). `cases.jsonl`
sha256 `9d344d7294b985548538aec5e308f59cbf98dfdd8cfa5bc8621a7cfe03368548`.
Every file that holds a case's answer is gold. Public corpus agentic-os
`5a07e64b`, Jev `jev-latest`, downstream on ministral-3-14b, jevroute `0be7810`.

## Measured, two reps

* **recall@10:** 85/95.
* **Selection:** Basic Memory rank 1 correct 60/95. Jev correct 69/95 in both
  reps, so +0.095.
* **Jev at 0.9:** pass 49 and 47, confident-wrong 6 and 6.
* **Gate precision** (pass / (pass + confident-wrong)): 0.891 and 0.887.
* **Narrowed** (Jev passed and narrowed to one note, 48 and 46 cases):
  * Recall: top 3 got 36 and 34, Jev's pick 37 and 35.
  * Harm (A right, B wrong) 2 and 2. Help 3 and 3.
  * Tokens 0.39x and 0.40x.
* **Whole run:** recall A 63 and 62, B 64 and 63. Tokens B 0.72x and 0.75x.
* **Rep agreement on the winner:** 93/95.

`summary.txt` was produced by `jevfilter.py summary` after one fix. The first
summary printed gate precision as 1.0, because jevroute's `pass` already implies
correct, so pass-and-correct over pass is always 1. The fixed formula is the one
above. No other number changed.

## Against the Amendment 1 prediction

* 1, >= 100 cases survive: failed, 95.
* 2, gate precision >= 0.85: held, 0.891 and 0.887.
* 3, pass rate 0.25-0.45: failed, 0.52 and 0.49. Jev is confident more often than
  predicted.
* 4, on narrowed cases:
  * Recall gain >= +0.10: **failed**, +1 of 48 and +1 of 46, about +0.02.
  * Harm <= 2: held, 2 and 2.
  * Tokens <= 0.5x: held, 0.39x and 0.40x.
* 5, whole run: held. Recall B >= A in both reps, tokens 0.72x and 0.75x.
* 6, Jev not a better ranker (within 0.05): **failed**. Jev out-picks rank 1 by
  0.095 once the labels are multi-gold.

## Verdict against the preregistered rule

**Not confirmed, and not refuted.** Confirmation needed 2, 4 and 5 in both reps,
and 4's +0.10 recall gain failed. Refutation needed harm above 2 or precision
below 0.75, and neither happened.

What the data supports:
* **The gate is safe and cheap.** Where Jev is confident, narrowing to its one
  note keeps answer quality level (+1, with 2 harms against 3 helps) at 0.4x the
  tokens. The whole run comes out at about 0.73x.
* **It is not a quality booster.** The first run's 5-6 to 8 gain, at n = 10-11,
  did not replicate at n = 46-48.
* **The first run's "not a better ranker" was a label artifact.** With multi-gold
  labels, Jev picks the answer-bearing note 9 more times out of 95 than Basic
  Memory's rank 1.

Jev is the subject, so no Jev verdict ranks this. The call is Kai's.
