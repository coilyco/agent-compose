# Results, 2026-10-02

Run 11:23-11:27Z. Public corpus agentic-os `5a07e64b`, 40 L1 cases (`cases.jsonl`,
sha256 `57ae84cefd9963783a82222c0d7f0a331d4a07d2dc9a5215f8af649c2e0192b4`).
Basic Memory 0.23.2, Jev `jev-latest` through Agent Proxy, downstream on
`evaluation/ministral-3-14b`. Scored with housecast jevroute `0be7810`. Every
row is committed, because the corpus is public.

## Preregistered measures (MEASURED, two reps)

* **recall@10:** 37/40.
* **A, Basic Memory rank 1, correct:** 18/40.
* **B, Jev correct:** 25/40 in both reps. B pass (correct at >= 0.9): 11 and 12.
  B confident-wrong: 5 and 4. Errors: 0.
* **Downstream recall:** A 27/40 in both reps, B 30 and 29.
* **Downstream prompt tokens, mean:** A 3,531, B 3,011 and 2,982, which is 0.85x.
* **Jev rep agreement on the winner:** 38/40.

## The gold labels were wrong, and that reverses selection

`confident-wrong-check.txt`: all 8 confident-wrong picks that named a note named
one that **also contains the answer string**. The gold label assumed one source
per answer, but answers repeat across notes in this corpus. The only genuine
confident-wrong is L1-21 in each rep, where Jev picked `none` while an
answer-bearing note was in the list.

`posthoc-answer-bearing.txt` rescored selection with `ok` = any candidate whose
file contains the answer. This is post hoc and labelled so:
* 39/40 cases have an answer-bearing note in the 10, at 2.74 per case on average.
* A, rank 1: 32/40. B, Jev: 31/40. B pass: 14. B confident-wrong: 2. Same in both reps.

That scoring is lenient: a short answer such as `60%` can sit in a note that does
not answer the question, which flatters rank 1. The truth sits between the two
scorings. Neither one shows Jev picking better than Basic Memory's own ranking.

## Downstream is the label-free signal

`downstream-paired.txt`, scored on the answer itself:
* **Where Jev passed (10 and 11 cases) and narrowed to its one note:** B recall
  8 against A 5 and 6, at 0.39 to 0.40x the tokens.
* **Harm:** narrowing never turned a right answer wrong, 0 cases in either rep.
* **Elsewhere:** B equals A, because the fallback is the same top 3.

The gain is real in direction and small in reach. Jev clears 0.9 on about 30
percent of cases, so the whole-run saving is 0.85x and not the 0.5x predicted.

## Against the prediction

* 1, recall@10 >= 0.80: held, 0.925.
* 2, B beats A by >= 0.10: held on the preregistered labels at +0.175, but that
  is an artifact of the labels. Post hoc it is -0.025.
* 3, confident-wrong <= 2: failed on the preregistered labels (5 and 4). Post hoc
  it is 2 and 2.
* 4, pass 0.40-0.75: failed, 0.275 and 0.30.
* 5, B tokens <= 0.5x and B recall >= A - 0.05: tokens failed at 0.85x, recall held.
* 6, rep agreement >= 90 percent: held, 95 percent.

## What this supports

Jev does not beat Basic Memory's own ranking at choosing the note. Used as a
gate, where you narrow only when Jev is confident and otherwise keep the top 3,
it improved downstream answers on the cases it fired and cut their context by
about 60 percent, with no observed harm, n = 10 to 11. That is worth one
corrected rerun before anyone builds it: multi-gold labels and a larger n. Jev
is the subject here, so no Jev verdict ranks this. The call is Kai's.
