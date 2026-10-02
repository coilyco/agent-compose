# Pre-registration: which local embedding model Agent Proxy's /v1/embeddings serves

Seat: science (Frog-Ox). Written 2026-10-02T12:09:14Z, before any embedding run.
`teable:coilyco/agent-proxy#8650`, after eng-platform merged the endpoint
(agent-proxy PR 173). The model is pulled on kai-tower-3026 under Ollama,
beside a game on the same GPU.

## Question

Which local embedding model gives the best retrieval on a corpus like ours, at a
size small enough to keep loaded beside the game?

## Setup, frozen

* **Corpus and cases.** The public agentic-os repo at `5a07e64b`, and the 95
  multi-gold cases from `../jev-filter-2026-10-02/results/gate-rerun/cases.jsonl`.
  Each case has a distinctive answer, and every file containing it is gold.
* **Candidates, run in-process with FastEmbed 0.8.1 on this Mac:**
  `BAAI/bge-small-en-v1.5` (Basic Memory's default, the baseline),
  `nomic-ai/nomic-embed-text-v1.5`, `mixedbread-ai/mxbai-embed-large-v1`,
  `snowflake/snowflake-arctic-embed-m`, `google/embeddinggemma-300m`, and
  `Qwen/Qwen3-Embedding-0.6B`. Each uses FastEmbed's own query and passage
  embedding, so model-specific prefixes are applied the same way for all.
* **Retrieval.** Text files are split into 1,200-character chunks. A file scores
  its best chunk, by cosine against the question.
* **Measures:** recall@1, @5 and @10 against the gold set. Also the model file
  size on disk, as a proxy for GPU memory (EXPECTED to track Ollama's footprint
  within about 2x), passage embedding throughput on this CPU, and dimension.

## Not measured here

Whether each model has an Ollama tag, and its real GPU memory beside the game.
Both get confirmed by sysadmin at pull time and recorded on #8650. Ollama's
quantization can shift quality a little from the FastEmbed weights.

## Decision

Jev picks from the shortlist, given these aggregates and the size limit. Jev is
not the subject here.
