# Cross-harness memory bench, lore against off-the-shelf stores

For `teable:coilyco/agentic-os#8632`. Kai asked whether an off-the-shelf memory
store, configured and not built, beats current practice: lore read through its
skills and grep.

* [`PREREGISTER.md`](PREREGISTER.md) - claim, conditions, cases and correctness, fixed before any run.
* [`PREDICTION.txt`](PREDICTION.txt) - the numbers expected, timestamped before any run.
* [`AMENDMENT-1.md`](AMENDMENT-1.md) - what the smoke test changed, before any case data.
* [`AMENDMENT-2.md`](AMENDMENT-2.md) - the model moves to ministral-3-14b, with a game on the GPU.
* [`AMENDMENT-3.md`](AMENDMENT-3.md) - the rerun: one exact answer string per lore entry, with its prediction.
* [`casegen.py`](casegen.py) - writes lore cases with the local model, grounding each answer string in its source.
* [`bench.py`](bench.py) - runs and grades both conditions in OpenCode. `just evalkit-memory-bench selftest`.
* [`results/rerun/`](results/rerun/) - the Amendment 3 rerun, one string per entry, with Jev's verdict.
* [`results/`](results/) - the 2026-10-02 run: scores, post hoc measures, and Jev's ranking.
* [`audits/`](audits/) - supply-chain audits of Basic Memory, Mem0 and Graphiti, read-only, with Jev verdicts.

Lore is private, so no lore text, case or raw answer lands here. Results carry
case ids, counts, scores and the sha256 of the private case file.
