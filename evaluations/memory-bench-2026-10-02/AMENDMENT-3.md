# Amendment 3: the rerun, one exact answer string per lore entry

Written 2026-10-02T10:05:51Z, before any rerun case exists. `teable:coilyco/agent-compose#8673`,
after Kai chose a trial and then a rerun on `teable:coilyco/agentic-os#8632`.

The 2026-10-02 run floored strict recall in both conditions, because cases
carried 3.28 exact strings on average. This rerun changes the case spec and
nothing else.

* **L1:** exactly one `must_contain` string, the fact the question asks for.
* **L2:** exactly one string from each of the two linked documents, so two.
* **Unchanged:** the verbatim-in-source and not-in-question checks, seed 8632,
  `evaluation/ministral-3-14b`, both conditions, the instruction, the grader,
  two runs each, and the sentinels. `casegen.py --single` selects this spec.

Prediction, EXPECTED, written now:
1. Strict L1 recall lifts off the floor in both conditions, each above 0.40.
2. C1 >= C0 on L1 and on L2, by about the 0.1 to 0.2 gap the post hoc
   context measure showed. A gap inside the run-to-run spread counts as no
   difference.
3. C1 tokens per lore question stay 10 to 25 percent below C0.
