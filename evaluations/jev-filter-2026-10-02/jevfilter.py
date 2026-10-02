"""Jev as a read-time relevance filter over Basic Memory search results.

  jevfilter.py retrieve FIXTURE PROJECT CASES OUT_DIR          top 10 per case, once (Basic Memory venv python)
  jevfilter.py jev OUT_DIR HOUSECAST REP                        one Jev choice per case, scored by jevroute
  jevfilter.py answer OUT_DIR CORPUS OPENCODE_CONFIG REP        downstream answer, top 3 against Jev's pick
  jevfilter.py summary OUT_DIR
  jevfilter.py selftest HOUSECAST

Public corpus only: Jev is hosted, so lore never enters this probe. Gold matches on
the full corpus-relative file path, because many notes share the stem SKILL.
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import urllib.request

ROUTE = os.environ.get("BENCH_ROUTE", "evaluation/ministral-3-14b")
INSTRUCTIONS = ("Which candidate note answers the question in the state? Choose the one note whose "
                "content holds the answer. Choose none if no candidate holds it.")
ENV_KEYS = ("HOME", "XDG_CONFIG_HOME", "BASIC_MEMORY_CONFIG_DIR", "BASIC_MEMORY_NO_PROMOS",
            "BASIC_MEMORY_FORCE_LOCAL", "BASIC_MEMORY_AUTO_UPDATE", "FASTEMBED_CACHE_PATH",
            "HF_HUB_OFFLINE", "FASTMCP_CHECK_FOR_UPDATES", "PATH")


def rows(path: str) -> list[dict]:
    return [json.loads(l) for l in open(path) if l.strip()]


def ok_set(cands: list[dict], gold: list[str]) -> list[str]:
    hits = [f"c{i + 1}" for i, c in enumerate(cands) if c["file_path"] in gold]
    return hits or ["none"]


def criteria(cands: list[dict]) -> dict[str, str]:
    out = {f"c{i + 1}": f"{c['title']} ({c['file_path']}): {c['excerpt']}" for i, c in enumerate(cands)}
    out["none"] = "None of these notes holds the answer."
    return out


def retrieve(fx: str, project: str, cases_path: str, out: str) -> None:
    from fastmcp import Client
    from fastmcp.client.transports import StdioTransport

    os.makedirs(out, exist_ok=True)
    env = {k: os.environ[k] for k in ENV_KEYS if k in os.environ}

    async def go() -> None:
        async with Client(StdioTransport(f"{fx}/bin/basic-memory", ["mcp", "--project", project], env=env)) as c:
            with open(f"{out}/candidates.jsonl", "w") as fh:
                for case in rows(cases_path):
                    r = await c.call_tool("search_notes", {"query": case["question"], "page_size": 10,
                                                           "output_format": "json"})
                    res = json.loads(" ".join(getattr(x, "text", "") for x in r.content))["results"]
                    cands = [{"file_path": x["file_path"], "title": x["title"],
                              "excerpt": " ".join(str(x.get("matched_chunk") or x.get("content") or "").split())[:400]}
                             for x in res[:10]]
                    fh.write(json.dumps({**case, "candidates": cands}) + "\n")

    asyncio.run(go())
    print(json.dumps({"retrieved": len(rows(f"{out}/candidates.jsonl"))}))


def jev(out: str, housecast: str, rep: str) -> None:
    sys.path.insert(0, housecast)
    from housecast.jevroute.bench import THRESHOLD, post_jev, score

    with open(f"{out}/jev-{rep}.jsonl", "w") as fh:
        for case in rows(f"{out}/candidates.jsonl"):
            body = {"model": "jev-latest", "state": {"question": case["question"]},
                    "questions": {"tool": {"type": "choice", "instructions": INSTRUCTIONS,
                                           "criteria": criteria(case["candidates"])}}}
            ok = ok_set(case["candidates"], case["sources"])
            row = score({"q": case["question"], "ok": ok}, post_jev(body), THRESHOLD)
            row.update({"id": case["id"], "rank1_correct": "c1" in ok, "gold_in_10": ok != ["none"]})
            fh.write(json.dumps(row) + "\n")
    print(json.dumps({"rep": rep, "rows": len(rows(f"{out}/jev-{rep}.jsonl"))}))


def chat(cfg: str, prompt: str) -> tuple[str, int]:
    prov = json.load(open(cfg))["provider"]["agent-proxy"]["options"]
    req = urllib.request.Request(f"{prov['baseURL'].rstrip('/')}/chat/completions", json.dumps({
        "model": ROUTE, "temperature": 0, "messages": [{"role": "user", "content": prompt}]}).encode(),
        {"Authorization": f"Bearer {prov['apiKey']}", "Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as resp:
        d = json.load(resp)
    return d["choices"][0]["message"]["content"] or "", d.get("usage", {}).get("prompt_tokens", 0)


def answer(out: str, corpus: str, cfg: str, rep: str) -> None:
    picks = {r["id"]: r for r in rows(f"{out}/jev-{rep}.jsonl")}
    read = lambda p: open(os.path.join(corpus, p), encoding="utf-8", errors="ignore").read()
    with open(f"{out}/answer-{rep}.jsonl", "w") as fh:
        for case in rows(f"{out}/candidates.jsonl"):
            top3 = [c["file_path"] for c in case["candidates"][:3]]
            j = picks[case["id"]]
            jev_ctx = ([case["candidates"][int(j["win"][1:]) - 1]["file_path"]]
                       if j.get("pass") and str(j.get("win", "")).startswith("c") else top3)
            row = {"id": case["id"], "jev_narrowed": jev_ctx != top3}
            for arm, ctx in (("a", top3), ("b", jev_ctx)):
                docs = "\n\n".join(f"--- {p} ---\n{read(p)}" for p in ctx)
                text, tok = chat(cfg, f"Answer the question from these notes in one sentence.\n\n{docs}\n\nQuestion: {case['question']}")
                row[f"{arm}_recall"] = case["must_contain"][0].lower() in text.lower()
                row[f"{arm}_tokens"] = tok
            fh.write(json.dumps(row) + "\n")
    print(json.dumps({"rep": rep, "rows": len(rows(f"{out}/answer-{rep}.jsonl"))}))


def summary(out: str) -> None:
    cands = rows(f"{out}/candidates.jsonl")
    n = len(cands)
    print(json.dumps({"n": n, "recall_at_10": sum(ok_set(c["candidates"], c["sources"]) != ["none"] for c in cands)}))
    for rep in ("r1", "r2"):
        if not os.path.exists(f"{out}/jev-{rep}.jsonl"):
            continue
        j = rows(f"{out}/jev-{rep}.jsonl")
        print(json.dumps({"rep": rep, "a_rank1_correct": sum(r["rank1_correct"] for r in j),
                          "b_correct": sum(r["correct"] for r in j), "b_pass": sum(r["pass"] for r in j),
                          "b_confident_wrong": sum(r["confident_wrong"] for r in j),
                          "errors": sum(1 for r in j if r.get("error"))}))
        if os.path.exists(f"{out}/answer-{rep}.jsonl"):
            a = rows(f"{out}/answer-{rep}.jsonl")
            print(json.dumps({"rep": rep, "a_recall": sum(r["a_recall"] for r in a), "b_recall": sum(r["b_recall"] for r in a),
                              "a_tokens_mean": round(sum(r["a_tokens"] for r in a) / len(a)),
                              "b_tokens_mean": round(sum(r["b_tokens"] for r in a) / len(a)),
                              "narrowed": sum(r["jev_narrowed"] for r in a)}))
    if all(os.path.exists(f"{out}/jev-{r}.jsonl") for r in ("r1", "r2")):
        w1 = {r["id"]: r.get("chosen") for r in rows(f"{out}/jev-r1.jsonl")}
        w2 = {r["id"]: r.get("chosen") for r in rows(f"{out}/jev-r2.jsonl")}
        print(json.dumps({"rep_winner_agreement": sum(w1[k] == w2.get(k) for k in w1), "of": len(w1)}))


def selftest(housecast: str) -> None:
    sys.path.insert(0, housecast)
    from housecast.jevroute.bench import score

    cands = [{"file_path": "a/SKILL.md", "title": "a", "excerpt": "x"},
             {"file_path": "b/SKILL.md", "title": "b", "excerpt": "y"}]
    assert ok_set(cands, ["b/SKILL.md"]) == ["c2"]           # full path, not the shared stem
    assert ok_set(cands, ["z/SKILL.md"]) == ["none"]
    assert set(criteria(cands)) == {"c1", "c2", "none"}
    ans = {"answers": {"tool": {"probabilities": {"c1": 0.95, "c2": 0.05}, "confidence": 0.95}}}
    r = score({"q": "q", "ok": ["c2"]}, ans, 0.9)
    assert r["confident_wrong"] and not r["correct"], r
    r = score({"q": "q", "ok": ["c1"]}, ans, 0.9)
    assert r["pass"] and not r["confident_wrong"], r
    print("selftest ok: 5 checks")


if __name__ == "__main__":
    cmd, args = sys.argv[1], sys.argv[2:]
    {"retrieve": retrieve, "jev": jev, "answer": answer, "summary": summary, "selftest": selftest}[cmd](*args)
