# Jev as a read-time relevance filter over Basic Memory

For `teable:coilyco/agentic-os#8668`, on a public corpus because Jev is hosted and lore stays local.

* [`PREREGISTER.md`](PREREGISTER.md) - claim, arms, and correctness, fixed before any run.
* [`PREDICTION.txt`](PREDICTION.txt) - expected numbers, timestamped before any run.
* [`AMENDMENT-1.md`](AMENDMENT-1.md) - confirm the confidence gate, with multi-gold labels and larger n.
* [`jevfilter.py`](jevfilter.py) - retrieval, the Jev choice scored by housecast jevroute, and the downstream answer. `just evalkit-jev-filter selftest HOUSECAST`.
* [`results/gate-rerun/`](results/gate-rerun/) - the Amendment 1 rerun: the gate is safe and cheap, but the quality gain did not replicate.
* [`results/`](results/) - the 2026-10-02 run, the label correction, and the paired downstream check.
