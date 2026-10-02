# Amendment 1: a lighter rerun after the low-memory stop

Written 2026-10-02T13:40:13Z, before the rerun. Kai chose this option.

The first run was stopped at about 12:59Z when Claude Code reaped it for low
memory on the Mac, after 1 of 6 models. Every text file in the corpus was
chunked, and `bge-small-en-v1.5` embedded 10.3 chunks/s on CPU. Its result on
that corpus, kept as a record and not compared below (MEASURED): recall 54/95
at 1, 76/95 at 5, 78/95 at 10, 384 dimensions, 65.6 MB of model files.

## What changes

* **Corpus:** Markdown files only.
* **Cases:** only those with at least one Markdown gold file. Gold is narrowed
  to its Markdown files, and n is reported.
* **Models:** `BAAI/bge-small-en-v1.5` (baseline, rerun on this corpus),
  `nomic-ai/nomic-embed-text-v1.5`, `snowflake/snowflake-arctic-embed-m`,
  `mixedbread-ai/mxbai-embed-large-v1`, and `Qwen/Qwen3-Embedding-0.6B-Q`
  (the quantized build). Dropped for memory: `google/embeddinggemma-300m`
  (1.24 GB) and full-size `Qwen3-Embedding-0.6B` (2.38 GB).
* **Memory:** one model per process, run one after another, with batch size 16.

The measures and the decision rule are unchanged. The predictions stand as
written, with the ones about dropped models void.
