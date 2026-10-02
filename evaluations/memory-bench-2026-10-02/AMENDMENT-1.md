# Amendment 1: what the smoke test changed, before any case data

Written 2026-10-02 at about 08:50Z, after a one-question smoke test on a synthetic
sentinel and before any lore case existed or ran.

## Basic Memory cannot index a hidden directory

`basic_memory/index/filesystem.py:137` in 0.23.2 drops any path with a part
starting with `.`. That is in code, and no `.bmignore` or config setting removes
it. 129 of lore's 133 Markdown files are under `.agents/`, so the first index saw
6 files. MEASURED: `reindex` reported `6 observed, 6 indexed`.

* **For this bench:** C1's copy renames `.agents` to `agents` before ingest.
  After that, `140 observed, 134 indexed`, 0 errors, in 49.9 s wall time.
* **For adoption:** Basic Memory over lore needs lore restructured, or an ingest
  step that copies it. Either one is a cost the recommendation has to carry.

## State isolation

The first `reindex` wrote its database and config into the session home through
`XDG_CONFIG_HOME`, not the fixture. I moved it into the fixture, and every call
now sets `BASIC_MEMORY_CONFIG_DIR` and `XDG_CONFIG_HOME` inside the fixture. No
store outside the fixture holds lore.

## One neutral instruction for both conditions

In the first smoke run, C1 found the sentinel note in all 5 searches, but never
opened it. Search returns excerpts, so the answer missed. Both conditions now get
the same instruction: search, then open the source before answering, answer in
one or two sentences, and name the source path. The text is in `bench.py` `ASK`.

Smoke after the change, one synthetic question each (MEASURED):
* **C1:** correct, 8 tool calls, 162,772 input tokens.
* **C0:** correct, 2 tool calls, 23,607 input tokens.

## Source fidelity matches the file stem as a whole word

Basic Memory names a note by permalink without `.md`, so the check is the gold
file's stem as a whole word, not its basename. The grader selftest caught a
one-letter stem matching inside ordinary words, so the match is bounded by word
boundaries.

## Restart and re-ingest, made concrete

* **Restart.** OpenCode starts a fresh MCP server for every question, so every C1
  answer comes from a new process reading a store an earlier process wrote. The
  sentinel pass is the restart check.
* **Re-ingest.** After the sentinel values are edited, C1 is asked again with no
  manual step, which tests whether startup sync catches the edit. If it does not,
  `basic-memory reindex` runs and C1 is asked a third time. Both numbers are
  reported.

## Harness detail

`opencode run` with stdin open and no TTY hangs with no output. Every call closes
stdin.
