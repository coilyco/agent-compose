"""Retrieval quality, size and speed for local embedding candidates, on a public corpus.

  embedpick.py run CORPUS CASES CACHE_DIR OUT.jsonl MODEL...
  embedpick.py selftest

Files split into 1,200-character chunks, and a file scores its best chunk.
recall@k is whether any gold file ranks in the top k. Runs in FastEmbed
in-process, so it measures the model rather than Ollama's serving of it.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import time

CHUNK = 1200


def load_corpus(root: str, md_only: bool = False) -> dict[str, list[str]]:
    out = {}
    for d, _, fs in os.walk(root):
        for f in fs:
            if md_only and not f.endswith(".md"):
                continue
            p = os.path.relpath(os.path.join(d, f), root)
            try:
                t = open(os.path.join(root, p), encoding="utf-8").read()
            except (UnicodeDecodeError, OSError):
                continue
            if t.strip():
                out[p] = [t[i:i + CHUNK] for i in range(0, len(t), CHUNK)]
    return out


def recall_at(ranked: list[str], gold: list[str], k: int) -> bool:
    return any(p in gold for p in ranked[:k])


def rank_files(qv, chunk_vecs, owners: list[str]) -> list[str]:
    sims = chunk_vecs @ qv
    best: dict[str, float] = {}
    for s, o in zip(sims.tolist(), owners):
        if s > best.get(o, -9):
            best[o] = s
    return sorted(best, key=lambda p: -best[p])


def run(corpus: str, cases_path: str, cache: str, out: str, *models: str) -> None:
    import numpy as np
    from fastembed import TextEmbedding

    md_only = os.environ.get("EMBED_MD_ONLY") == "1"
    docs = load_corpus(corpus, md_only)
    owners = [p for p, cs in docs.items() for _ in cs]
    chunks = [c for cs in docs.values() for c in cs]
    cases = [json.loads(l) for l in open(cases_path) if l.strip()]
    if md_only:
        cases = [{**c, "gold": [g for g in c["gold"] if g.endswith(".md")]} for c in cases]
        cases = [c for c in cases if c["gold"]]
    print(json.dumps({"files": len(docs), "chunks": len(chunks), "cases": len(cases)}), flush=True)
    with open(out, "a") as fh:
        for name in models:
            mdir = os.path.join(cache, name.replace("/", "__"))
            emb = TextEmbedding(name, cache_dir=mdir)
            t0 = time.monotonic()
            cv = np.array(list(emb.passage_embed(chunks, batch_size=int(os.environ.get('EMBED_BATCH', '64')))), dtype=np.float32)
            secs = time.monotonic() - t0
            cv /= np.linalg.norm(cv, axis=1, keepdims=True)
            qv = np.array(list(emb.query_embed([c["question"] for c in cases])), dtype=np.float32)
            qv /= np.linalg.norm(qv, axis=1, keepdims=True)
            hits = {1: 0, 5: 0, 10: 0}
            for q, c in zip(qv, cases):
                ranked = rank_files(q, cv, owners)
                for k in hits:
                    hits[k] += recall_at(ranked, c["gold"], k)
            size = subprocess.run(["du", "-sk", mdir], capture_output=True, text=True).stdout.split()[0]
            row = {"model": name, "dim": int(cv.shape[1]), "recall_at_1": hits[1], "recall_at_5": hits[5],
                   "recall_at_10": hits[10], "n": len(cases), "passage_chunks_per_s": round(len(chunks) / secs, 1),
                   "model_files_kb": int(size)}
            fh.write(json.dumps(row) + "\n")
            print(json.dumps(row), flush=True)


def selftest() -> None:
    import numpy as np

    assert recall_at(["a", "b", "c"], ["c"], 3) and not recall_at(["a", "b", "c"], ["c"], 2)
    cv = np.array([[1, 0], [0, 1], [0.9, 0.1]], dtype=np.float32)
    assert rank_files(np.array([1, 0], dtype=np.float32), cv, ["x", "y", "x"]) == ["x", "y"]
    assert rank_files(np.array([0, 1], dtype=np.float32), cv, ["x", "y", "x"])[0] == "y"
    print("selftest ok: 3 checks")


if __name__ == "__main__":
    cmd, args = sys.argv[1], sys.argv[2:]
    {"run": run, "selftest": selftest}[cmd](*args)
