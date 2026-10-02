# Why seats ask Kai to merge green PRs

For `teable:coilyco/agentic-os#8656`. This is read-only and makes no merges.
It covers 48 hours of Claude Code session transcripts under
`~/.claude/projects`, measured 2026-10-02 at 11:52Z. Codex had no transcripts in
the window. `mergeasks.py extract` keeps event kinds, timestamps, tool names and
PR coordinates only, never message text, and `classify.txt` is its output.
`just evalkit-merge-asks selftest` runs 13 checks.

## Measured

* **Merge attempts:** 174 in 42 sessions. 153 succeeded, 18 errored, and 3 were
  denied by the auto-mode classifier. The stated reasons were Modify Shared
  Resources, Real-World Transactions, and none given.
* **All 18 errors are the same Forgejo refusal:** `405: not allowed to merge
  [reason: The head branch is behind the base branch]`. By repo: website 7,
  infrastructure 5, deploy 2, eco-app 2, agentic-os 2.
* **Merge asks to the human: 33.**
  * After a classifier denial: **0**.
  * After the 405 behind-base refusal: **17**.
  * With no merge attempt at all: 9.
  * After an earlier merge in the same session had succeeded: 7.
* **Human "merge" messages:** 21 of the 48 arrived after a PR event in that
  session. 8 followed a seat's ask, 6 followed a PR the seat opened and never tried
  to merge, 5 followed a successful merge, 1 followed a denial, and 1 followed a
  failed attempt. The other 27 had no prior PR event in the session, mostly
  long dispatch prompts that mention merging.

## What it is

The record's hypothesis, classifier denials, accounts for at most 1 of the 33
asks and 3 of the 174 attempts. The dominant cause is a third one: **branch
protection requires an up-to-date head, and seats have no update step**. The
forgejo MCP exposes no update verb, although Forgejo `16.0.2+gitea-1.22.0` serves
`POST /repos/{owner}/{repo}/pulls/{index}/update` (merges base into head). Doctrine
names no recovery for the 405 either, so a correct seat stops and asks. The
remaining 16 asks, plus the 6 stops after opening a PR, are doctrine misses
with no runtime obstacle.

Jev (`jev-1.13.0`, aggregates only):
* **Dominant cause:** behind-base 405 at p=0.93, confidence 0.90.
* **Primary fix:** the update-branch recovery at p=0.65, confidence 0.54.
  Against: a harness permission rule 0.16, a repo setting 0.15, AGENTS text
  alone 0.04.

## The fix, at the right layers

1. **Tooling (eng-platform):** add an `update_pull-request` verb to the forgejo
   MCP, calling `POST /repos/{o}/{r}/pulls/{n}/update?style=merge`.
2. **Doctrine (eng-platform, in agentic-os `agentic_os/generators/generate_git_workflow.py`):**
   one line in the generated block. On a 405 "head branch is behind the base
   branch", update the branch (that verb, or merge `origin/main` into it and push,
   never force), wait for green, and merge without asking.
3. **No harness rule is needed for this.** The 3 denials are rare. A rule
   letting a seat merge its own green PR is worth a spec only if denials recur
   after 1 and 2 land.

## Caveats

* The ask detector is a regex, and its precision is unmeasured. Some of the 9
  "no attempt" asks may not be merge asks at all.
* The window holds Claude Code only. Codex seats are unmeasured.
* The 17 behind-base asks were classified from event order, not content.
