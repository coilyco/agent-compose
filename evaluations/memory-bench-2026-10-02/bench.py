"""Run and grade the memory bench in OpenCode, keeping lore text out of the output.

Usage:
  bench.py run FIXTURE COND RUN_LABEL CASES.jsonl   one pass of every case under c0 or c1
  bench.py sentinels FIXTURE VERSION                rewrite bench/sentinels.md in both copies to old|new
  bench.py score FIXTURE RUN_LABEL...               aggregate public rows into a table
  bench.py selftest

Raw answers stay under FIXTURE/raw. `run` writes FIXTURE/public/<label>.jsonl, which
holds case ids, booleans and token counts only, and is the file a results commit
copies. The fixture layout is set up by hand and recorded in AMENDMENT-1.md.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time

MODEL = "agent-proxy/" + os.environ.get("BENCH_ROUTE", "evaluation/ministral-3-14b")
HOW = {"c0": "Use your file search and read tools on the files in the working directory.",
       "c1": "Use your memory tools."}
ASK = ("{how} Search for the answer, then open the source document before you answer, because search "
       "results are only excerpts. Answer in one or two sentences and name the source file path you "
       "used. Question: {q}")


def norm(s: str) -> str:
    return " ".join(s.lower().split())


def grade(text: str, must: list[str], sources: list[str], stale: list[str] | None = None) -> dict:
    t = norm(text)
    stems = {os.path.splitext(os.path.basename(p))[0].lower() for p in sources}
    row = {"recall": all(norm(m) in t for m in must),
           "source": any(re.search(rf"(?<![\w-]){re.escape(s)}(?![\w-])", t) for s in stems) if stems else None}
    if stale is not None:
        row["stale"] = any(norm(m) in t for m in stale)
    return row


def parse(path: str) -> tuple[str, list[str], int, int]:
    text, tools, tin, tout = "", [], 0, 0
    for line in open(path, encoding="utf-8", errors="ignore"):
        if not line.strip().startswith("{"):
            continue
        e = json.loads(line)
        p = e.get("part", {})
        if e.get("type") == "text":
            text += p.get("text", "")
        elif e.get("type") == "tool_use":
            tools.append(p.get("tool"))
        elif e.get("type") == "step_finish":
            tok = p.get("tokens", {})
            tin += tok.get("input", 0)
            tout += tok.get("output", 0)
    return text, tools, tin, tout


def ask(fx: str, cond: str, question: str, raw: str) -> float:
    cwd = os.path.join(fx, "c0" if cond == "c0" else "empty")
    env = {**os.environ, "OPENCODE_CONFIG": os.path.join(fx, f"oc-{cond}.json"),
           "XDG_CONFIG_HOME": os.path.join(fx, "oc", "config"),
           "XDG_DATA_HOME": os.path.join(fx, "oc", "data"),
           "XDG_STATE_HOME": os.path.join(fx, "oc", "state")}
    start = time.monotonic()
    with open(raw, "w") as out, open(raw + ".err", "w") as err:
        subprocess.run(["/opt/homebrew/bin/opencode", "run", "--pure", "--format", "json", "-m", MODEL,
                        "--dir", cwd, ASK.format(how=HOW[cond], q=question)],
                       cwd=cwd, env=env, stdin=subprocess.DEVNULL, stdout=out, stderr=err, timeout=900)
    return time.monotonic() - start


def run(fx: str, cond: str, label: str, cases_path: str) -> None:
    cases = [json.loads(l) for l in open(cases_path) if l.strip()]
    os.makedirs(os.path.join(fx, "raw", label), exist_ok=True)
    os.makedirs(os.path.join(fx, "public"), exist_ok=True)
    with open(os.path.join(fx, "public", f"{label}.jsonl"), "w") as pub:
        for c in cases:
            raw = os.path.join(fx, "raw", label, f"{c['id']}.jsonl")
            try:
                secs = ask(fx, cond, c["question"], raw)
            except subprocess.TimeoutExpired:
                secs = None
            text, tools, tin, tout = parse(raw)
            row = {"id": c["id"], "kind": c["kind"], "cond": cond, "label": label,
                   **grade(text, c["must_contain"], c.get("sources", []), c.get("stale")),
                   "answered": bool(text.strip()), "tool_calls": len(tools),
                   "tokens_in": tin, "tokens_out": tout, "secs": round(secs, 1) if secs else None}
            pub.write(json.dumps(row) + "\n")
            pub.flush()
            print(json.dumps(row), flush=True)


def sentinels(fx: str, version: str) -> None:
    facts = json.load(open(os.path.join(fx, "sentinels.v1.json")))
    body = "# Bench sentinels\n\nFictional facts for the memory bench. None of this is real.\n\n" + "".join(
        "- " + f["tmpl"].format(f[version]) + "\n" for f in facts)
    for cond in ("c0", "c1"):
        open(os.path.join(fx, cond, "bench", "sentinels.md"), "w").write(body)
    cases = [{"id": f"{f['id']}-{version}", "kind": f"S-{version}", "question": f["q"],
              "must_contain": [f[version]], "sources": ["bench/sentinels.md"],
              "stale": [f["old"]] if version == "new" else None} for f in facts]
    with open(os.path.join(fx, f"sentinel-cases.{version}.jsonl"), "w") as fh:
        for c in cases:
            fh.write(json.dumps(c) + "\n")
    print(json.dumps({"version": version, "facts": len(facts)}))


def score(fx: str, labels: list[str]) -> None:
    for label in labels:
        rows = [json.loads(l) for l in open(os.path.join(fx, "public", f"{label}.jsonl"))]
        for kind in sorted({r["kind"] for r in rows}):
            k = [r for r in rows if r["kind"] == kind]
            out = {"label": label, "kind": kind, "n": len(k),
                   "recall": sum(r["recall"] for r in k),
                   "source": sum(bool(r["source"]) for r in k),
                   "answered": sum(r["answered"] for r in k),
                   "tokens_per_q": round(sum(r["tokens_in"] + r["tokens_out"] for r in k) / len(k)),
                   "tool_calls_per_q": round(sum(r["tool_calls"] for r in k) / len(k), 1)}
            if any("stale" in r for r in k):
                out["stale"] = sum(bool(r.get("stale")) for r in k)
            print(json.dumps(out))


def selftest() -> None:
    g = grade("The door is painted  Saffron, per bench/sentinels.md.", ["saffron"], ["bench/sentinels.md"])
    assert g == {"recall": True, "source": True}, g
    g = grade("It is saffron (see note bench/sentinels).", ["saffron"], ["bench/sentinels.md"], ["cobalt"])
    assert g == {"recall": True, "source": True, "stale": False}, g
    g = grade("Cobalt, formerly saffron.", ["cobalt"], ["x/a.md"], ["saffron"])
    assert g == {"recall": True, "source": False, "stale": True}, g
    g = grade("No idea.", ["port 7391", "Kestrel"], ["a.md", "b.md"])
    assert g["recall"] is False, g
    g = grade("Port 7391, owned by kestrel.", ["port 7391", "Kestrel"], [])
    assert g == {"recall": True, "source": None}, g
    print("selftest ok: 5 grade cases")


if __name__ == "__main__":
    cmd, args = sys.argv[1], sys.argv[2:]
    if cmd == "run":
        run(*args)
    elif cmd == "sentinels":
        sentinels(*args)
    elif cmd == "score":
        score(args[0], args[1:])
    elif cmd == "selftest":
        selftest()
    else:
        sys.exit(__doc__)
