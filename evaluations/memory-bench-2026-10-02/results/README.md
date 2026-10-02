# Results, 2026-10-02

Run 09:11-09:52Z, in OpenCode 1.18.33 on `evaluation/ministral-3-14b`
(Amendment 2), with lore at `d251e22`. The case file has 29 cases, 20 L1 and
9 L2, sha256 `c3f527b49026fe1edfe5a20a143e5b1b455f1f1ef227ba19b24e717ec1148f57`.
`score.txt` is `just evalkit-memory-bench score FIXTURE <labels>`, and
`diagnose.txt` holds the post hoc measures. The rows hold ids, booleans and token
counts only.

## Preregistered measures (MEASURED, two runs each)

* **Strict recall, every `must_contain` string present.**
  * L1: C0 1/20 and 4/20, C1 3/20 and 5/20.
  * L2: both conditions 1/9 in both runs.
  * Floored in both. See the spec defect below. This number cannot rank anything.
* **Source fidelity.**
  * L1: C0 11 and 11, C1 11 and 14.
  * L2: C0 4 and 5, C1 7 and 7.
* **Tokens per question on lore.**
  * L1: C0 83,640 and 79,474, C1 68,526 and 66,450. That is about 17 percent fewer for C1, against a C0 run-to-run spread of 5 percent.
  * Sentinels: C1 costs more, 63,765 against 47,908.
* **Sentinels, old values.** C0 9/10, C1 10/10.
* **Update with no manual reindex.** C0 9/10 new and 0 stale, C1 10/10 new and 0 stale. Basic Memory's startup sync picked up the edit, and the later `reindex` found 0 files to index.
* **Restart.** C1 10/10. Every question starts a fresh MCP server.
* **Cross-harness, store layer.**
  * Claude Code lists the server as `✔ Connected`.
  * Codex lists it as configured, but its `mcp list` reports no health status.
  * A note one MCP stdio process wrote was read back by a fresh one (`token_found: true`, 21 tools).
  * Claude Code and Codex model recall was **not measured**.
* **Auditability.** C1 notes are plain Markdown on disk, and `git status` shows each write.

## The spec defect

The case writer was asked for one to three exact strings per case, and returned
3.28 on average. A 14B model paraphrases, so requiring every string floors both
conditions. Answers matched 0.27 to 0.50 of their strings. The defect is in my
case spec, not in either store. A rerun should use one exact string per case.

## Post hoc, labelled exploratory

* **Gold source reached** (stem in a tool input or result): L1 C0 0.90 and 0.90,
  C1 0.95 and 0.95. L2 C0 0.67 and 0.89, C1 1.00 and 1.00. C1's search results
  list note permalinks, which makes this easier for C1 to pass.
* **Gold strings present in the retrieved context:** L1 C0 0.52 and 0.57, C1 0.72
  and 0.80. L2 C0 0.56 and 0.64, C1 0.75 and 0.72. That is a gap of about 0.2,
  against a run-to-run spread under 0.08.
* **Answer string fraction:** L1 C0 0.27 and 0.40, C1 0.43 and 0.50. L2 C0 0.28
  and 0.28, C1 0.39 and 0.39.

## Against the prediction

* 1, L1 C0 >= C1: wrong direction, and both are below the predicted band.
* 2, L2 within 0.15: held, 1/9 each.
* 3, source fidelity: C0 below the 0.80 predicted.
* 4, 6 and 7: held.
* 5, C1 cheaper: held on lore at about 17 percent, but wrong on sentinels.
* 8, variance within 0.10: held on strict recall. C0 L1 moved by 0.15, which is outside it.

## Ranking, from Jev

`jev-1.13.0`, from the aggregate numbers only:
* **Ranks first:** Basic Memory, p=0.79 (current practice 0.21, Mem0 0, Graphiti 0), confidence 0.72.
* **Recommendation:** trial Basic Memory beside current practice and rerun a
  corrected case set before amending doctrine, p=0.60, against adopting now at
  p=0.38. Confidence is 0.46, so the decision is Kai's.
