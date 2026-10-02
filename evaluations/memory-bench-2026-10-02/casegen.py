"""Write lore cases with a local model, so the science seat never reads lore.

Usage: casegen.py LORE_DIR OUT.jsonl OPENCODE_CONFIG [--l1 20] [--l2 10] [--seed 8632]

Every model call goes to Agent Proxy on BENCH_ROUTE, default
`evaluation/ministral-3-14b`, which resolves to Ollama on kai-tower-3026. Prints counts, rejects and the sha256 of the output,
never case text. A case survives only if each `must_contain` string appears
verbatim in its own source file and not in the question.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import random
import re
import sys
import urllib.request

ROUTE = os.environ.get("BENCH_ROUTE", "evaluation/ministral-3-14b")
LINK = re.compile(r"\]\(([^)#\s]+\.md)(?:#[^)]*)?\)")

L1_PROMPT = """You write one quiz question about the document below.
Rules:
- The answer must be a specific fact stated in the document: a name, number, term, rule or setting.
- must_contain is 1 to 3 short strings (3 to 40 characters each) copied EXACTLY from the document, which a correct answer must include.
- The question must not contain any must_contain string.
Reply with JSON only: {"question": "...", "must_contain": ["..."]}

DOCUMENT (path: %s):
%s"""

L2_PROMPT = """You write one quiz question that needs BOTH documents below to answer. Document A links to document B.
Rules:
- A correct answer must combine a fact from A with a fact from B.
- must_contain_a is 1 or 2 short strings (3 to 40 characters) copied EXACTLY from A. must_contain_b is the same for B.
- The question must not contain any must_contain string.
Reply with JSON only: {"question": "...", "must_contain_a": ["..."], "must_contain_b": ["..."]}

DOCUMENT A (path: %s):
%s

DOCUMENT B (path: %s):
%s"""


def chat(base: str, key: str, prompt: str) -> dict | None:
    body = json.dumps({"model": ROUTE, "temperature": 0, "messages": [{"role": "user", "content": prompt}]})
    req = urllib.request.Request(f"{base}/chat/completions", body.encode(), {
        "Authorization": f"Bearer {key}", "Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as resp:
        text = json.load(resp)["choices"][0]["message"]["content"] or ""
    match = re.search(r"\{.*\}", text, re.S)
    try:
        return json.loads(match.group(0)) if match else None
    except json.JSONDecodeError:
        return None


def grounded(strings: object, doc: str, question: str) -> list[str] | None:
    if not isinstance(strings, list) or not 1 <= len(strings) <= 3:
        return None
    doc_l, q_l = doc.lower(), question.lower()
    for s in strings:
        if not isinstance(s, str) or not 3 <= len(s.strip()) <= 40:
            return None
        if s.strip().lower() not in doc_l or s.strip().lower() in q_l:
            return None
    return [s.strip() for s in strings]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("lore")
    ap.add_argument("out")
    ap.add_argument("opencode_config")
    ap.add_argument("--l1", type=int, default=20)
    ap.add_argument("--l2", type=int, default=10)
    ap.add_argument("--seed", type=int, default=8632)
    a = ap.parse_args()
    prov = json.load(open(a.opencode_config))["provider"]["agent-proxy"]["options"]
    base, key = prov["baseURL"].rstrip("/"), prov["apiKey"]

    files = sorted(
        os.path.relpath(os.path.join(d, f), a.lore)
        for d, _, fs in os.walk(a.lore) for f in fs
        if f.endswith(".md") and not os.path.relpath(os.path.join(d, f), a.lore).startswith("bench/"))
    read = lambda p: open(os.path.join(a.lore, p), encoding="utf-8", errors="ignore").read()
    pairs = sorted({(f, os.path.normpath(os.path.join(os.path.dirname(f), m)))
                    for f in files for m in LINK.findall(read(f))
                    if os.path.normpath(os.path.join(os.path.dirname(f), m)) in files
                    and os.path.normpath(os.path.join(os.path.dirname(f), m)) != f})
    rng = random.Random(a.seed)
    rng.shuffle(files)
    rng.shuffle(pairs)
    cases, rejects = [], {"l1": 0, "l2": 0}

    for path in files:
        if sum(c["kind"] == "L1" for c in cases) >= a.l1:
            break
        doc = read(path)
        if len(doc) < 400:
            continue
        got = chat(base, key, L1_PROMPT % (path, doc))
        must = grounded((got or {}).get("must_contain"), doc, str((got or {}).get("question", "")))
        if not got or not must or not got.get("question"):
            rejects["l1"] += 1
            print(json.dumps({"l1_reject": rejects["l1"]}), file=sys.stderr, flush=True)
            continue
        cases.append({"id": f"L1-{len(cases) + 1:02d}", "kind": "L1", "question": got["question"],
                      "must_contain": must, "sources": [path]})
        print(json.dumps({"l1_kept": len(cases)}), file=sys.stderr, flush=True)

    used_sources: set[str] = set()
    for src, tgt in pairs:
        if sum(c["kind"] == "L2" for c in cases) >= a.l2:
            break
        if src in used_sources:
            continue
        doc_a, doc_b = read(src), read(tgt)
        got = chat(base, key, L2_PROMPT % (src, doc_a, tgt, doc_b))
        q = str((got or {}).get("question", ""))
        must_a = grounded((got or {}).get("must_contain_a"), doc_a, q)
        must_b = grounded((got or {}).get("must_contain_b"), doc_b, q)
        if not q or not must_a or not must_b:
            rejects["l2"] += 1
            print(json.dumps({"l2_reject": rejects["l2"]}), file=sys.stderr, flush=True)
            continue
        used_sources.add(src)
        n = sum(c["kind"] == "L2" for c in cases) + 1
        cases.append({"id": f"L2-{n:02d}", "kind": "L2", "question": q,
                      "must_contain": must_a + must_b, "sources": [src, tgt]})
        print(json.dumps({"l2_kept": n}), file=sys.stderr, flush=True)

    with open(a.out, "w") as fh:
        for c in cases:
            fh.write(json.dumps(c) + "\n")
    digest = hashlib.sha256(open(a.out, "rb").read()).hexdigest()
    print(json.dumps({"l1": sum(c["kind"] == "L1" for c in cases), "l2": sum(c["kind"] == "L2" for c in cases),
                      "rejects": rejects, "link_pairs": len(pairs), "sha256": digest}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
