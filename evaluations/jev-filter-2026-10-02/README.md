# Jev as a read-time relevance filter over Basic Memory

For `teable:coilyco/agentic-os#8668`, on a public corpus because Jev is hosted and lore stays local.

* [`PREREGISTER.md`](PREREGISTER.md) - claim, arms, and correctness, fixed before any run.
* [`PREDICTION.txt`](PREDICTION.txt) - expected numbers, timestamped before any run.
* [`jevfilter.py`](jevfilter.py) - retrieval, the Jev choice scored by housecast jevroute, and the downstream answer. `just evalkit-jev-filter selftest HOUSECAST`.
* [`results/`](results/) - the 2026-10-02 run, the label correction, and the paired downstream check.
