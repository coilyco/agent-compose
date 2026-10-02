# Pre-registration: Jev as a read-time relevance filter over Basic Memory

Seat: science (Frog-Ox). Written 2026-10-02T11:16:27Z, before the corpus is built or any case
exists. `teable:coilyco/agentic-os#8668`, scoped to read-time filtering by
the #8632 results. agent-compose at `220b8dc`, housecast at `0be7810` (jevroute).

## The lore boundary, and why this is a public slice

Jev is hosted. agent-proxy `docs/systemone-shim.md` routes `/v1/systemone` to
`https://api.typesafe.ai/v1/systemone`, regime `hosted`. A relevance filter
sends candidate text to Jev, so this cannot run on lore under Kai's rule. Per
that rule, it runs on a public-safe slice and says so. **The corpus is the public
agentic-os repository at `5a07e64b`** (288 Markdown files). A result here says
whether a Jev filter helps Basic Memory. It does not license sending lore to Jev.

## Claim

Given Basic Memory's top 10 search results for a question, a Jev `choice` picks
the note that answers it more often than Basic Memory's own rank 1, rarely picks
wrong at high confidence, and lets a local answerer use one note instead of three
without losing recall.

## Setup, frozen

* **Corpus.** `git archive` of agentic-os at `5a07e64b`, with `.agents` renamed
  to `agents` (Basic Memory skips hidden paths). Basic Memory 0.23.2 gets its own
  project under the isolated config dir, `FASTMCP_CHECK_FOR_UPDATES=off`.
* **Cases.** `casegen.py --single --l1 40 --l2 0`, seed 8632, on
  `evaluation/ministral-3-14b`. That gives one exact string per case, grounded in
  its one source. The corpus is public, so the seat may read the cases.
* **Candidates.** `search_notes(query=question, page_size=10)` over MCP stdio.
  Each candidate is a permalink, a title and the returned excerpt.
* **Arm A, no filter.** Basic Memory's rank 1.
* **Arm B, Jev.** One `choice` per case through Agent Proxy on `jev-latest`.
  The criteria are the 10 candidates plus `none`, and the callsite instruction
  asks which note answers the question. It is scored with housecast jevroute's
  `score()`: threshold 0.9, with `confident_wrong` kept apart. `ok` is any
  candidate whose permalink stem equals the gold stem, or `none` when the gold
  note is not among the 10.
* **Downstream.** ministral answers in one chat completion, with no tools, from:
  * **A:** the full text of the top 3 notes.
  * **B:** the full text of Jev's pick when it passes, otherwise the same top 3.
  * It is scored on strict recall of the one string and on prompt tokens.
* **Reps.** Jev and downstream run twice. Retrieval is deterministic and runs once.

## Correctness, defined now

* **recall@10:** the gold stem is among the 10 candidates.
* **A correct:** rank 1 is gold.
* **B correct, pass, confident_wrong:** as jevroute's `score()` defines them.
* **Downstream recall:** the string is present, case-folded. Tokens are
  `usage.prompt_tokens`.

Jev is the subject here, so no Jev verdict ranks this result. The numbers stand
alone, and the recommendation is Kai's.
