# Pre-registration: does an off-the-shelf memory store beat lore and grep

Seat: science (Frog-Ox). Written 2026-10-02T08:19:19Z, before any candidate is
installed or any case exists. `teable:coilyco/agentic-os#8632`, brief on its
description comment, Kai's decisions on comment `rec3ITzOOmH8hbCtslu` and on the
record's later decision. agent-compose at `0f8f9e4`, lore at `d251e22`.

## Claim

An off-the-shelf memory store, configured and not built, run fully local and
read over MCP, answers questions about the lore corpus at least as well as
current practice. It returns the updated value after a lore edit, survives a
restart, and is reachable from Claude Code, Codex and OpenCode.

## What is excluded before the run, and why

* **Mem0.** Its OpenMemory MCP server was retired upstream, and the server that
  replaced it has no MCP and accepts only hosted LLM and embedder providers.
  Getting an MCP surface means building a wrapper. See `audits/mem0.md`.
* **Graphiti.** Its embedder only calls an HTTP `/embeddings` endpoint, and
  Agent Proxy serves `/v1/chat/completions` and `/v1/models` only. No local
  route exists, so it cannot run under the lore rule. See `audits/graphiti.md`.
* **Claude Code and Codex model behavior.** Agent Proxy serves neither the
  Messages API nor the Responses API, and Kai declined both a vendor exception
  and new routes. Those two are tested at the config and store layer only, and
  their recall behavior is **not measured**.

## Setup, frozen

* **Conditions.** Both run in OpenCode 1.18.33 with `--pure`, from a throwaway
  config under the fixture. `model` and `small_model` are both
  `agent-proxy/evaluation/ornith-35b`. At deploy `ca7d09df` that route resolves
  to Ollama `ornith:35b` on kai-tower-3026 with no fallbacks. Web, bash, edit,
  task and skill tools are denied, and sharing and auto-update are off.
  * **C0, control.** A copy of lore at `d251e22` is the working directory, with
    read, grep, glob and list allowed and no MCP.
  * **C1, Basic Memory 0.23.2.** An empty working directory, filesystem tools
    denied, and the `basic-memory` MCP server over stdio. Its project root is a
    second copy of lore. Auto-update and promos are off, `FORCE_LOCAL` is on,
    and the embedding model is fetched once, after which `HF_HUB_OFFLINE=1`.
* **Lore stays local.** No lore text enters the science seat's context, a
  third-party model, or this repository. Cases, gold answers and raw responses
  stay in the fixture. This repository gets case ids, counts, scores, and the
  sha256 of the private case file.

## Cases

* **L1, single entry, n=20, and L2, cross-link, n=10.** Written by `ornith-35b`
  through the local route from lore. Each case has a question, a list of short
  `must_contain` strings, and the source path or paths. L2 cases must need two
  entries joined by a link. Kai chose this so the seat never reads lore, and the
  cost is that nobody off her machines checks the cases. Kai should spot-check
  them.
* **S, synthetic sentinels, n=10.** Fictional facts the seat writes, in a new
  file inside each lore copy, so the update and restart checks can be read.

## Correctness, defined now

* **Recall.** Every `must_contain` string appears in the answer, case-folded
  and with whitespace collapsed.
* **Source fidelity.** The answer names the basename of a gold source path.
* **Update.** After a sentinel value is edited and C1 re-syncs, the answer holds
  the new value and not the old one.
* **Restart.** After the MCP server is killed and started again, sentinels are
  still recalled.
* **Tokens per question.** Input plus output tokens from OpenCode's JSON events.
* **Cross-harness, store layer.** `claude mcp list` and `codex mcp list` under
  throwaway config dirs show the server connected, and a note written by one MCP
  process is read back by a new one. No model call.
* **Setup friction and auditability.** Descriptive. Steps and files counted,
  and whether `git diff` shows what the store holds.

Each condition runs twice with the same settings, and a move is reported only
beyond that run-to-run spread. Grading is programmatic and runs locally. The
ranked recommendation comes from Jev on the aggregate numbers only, never on
lore text. Predictions are in `PREDICTION.txt`.
