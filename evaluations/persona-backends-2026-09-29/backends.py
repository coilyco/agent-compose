"""Per-persona backend check for teable:coilyco/housecast#8487 (PREREGISTER.md).

routes                                   mapped models against Agent Proxy /v1/models
run <bundles dir> <out> <config> <label> [reps]   rounds for one config, one record per call
jev <out> <label>                        one Jev stance score per round
score <out> <label> [label ...]          lost subjects, latency, divergence
grid <out>                               24 persona-by-route assignments, recombined from single-route runs
compare <out> <label>                    a direct run against the pooled baseline runs A and B
selftest
"""
import json, os, random, re, sys, threading, time, urllib.error, urllib.request
from concurrent.futures import ThreadPoolExecutor

PROXY = os.environ.get("BACKENDS_PROXY", "http://ser8:8080")
CALLLOG = os.environ.get("CALLLOG")  # one line per proxy call, so a day's count is a wc -l and not a guess
HERE = os.path.dirname(os.path.abspath(__file__))
PROMPTS = os.path.join(HERE, "..", "bundle-divergence-2026-09-24", "prompts.json")
ROLES = {"evie": "scientist", "delphi": "frontend-eng", "sprite": "game-dev", "gem": "dev-advocate"}
PRO = "evaluation/deepseek-v4-pro"
MAPPED = {"gem": "mistral", "delphi": "kimi", "sprite": "qwen", "evie": "deepseek-v4-1"}
CONFIGS = {
    "current": dict.fromkeys(ROLES, PRO),
    "flash": dict.fromkeys(ROLES, "evaluation/deepseek-v4-flash"),
    "gemini": dict.fromkeys(ROLES, "chat/gemini-3-5-flash"),
    "minimax": dict.fromkeys(ROLES, "chat/minimax-m3"),
    "glm": dict.fromkeys(ROLES, "chat/glm-5-3"),
    "chatdefault": dict.fromkeys(ROLES, "chat/default"),
    # The assignment Kai chose for the 2026-09-30 room, measured as run C1.
    "room": {"delphi": PRO, "evie": "chat/gemini-3-5-flash", "gem": "chat/minimax-m3", "sprite": "chat/glm-5-3"},
    "available-mix": {"evie": "evaluation/deepseek-v4-flash", "gem": "chat/gemini-3-5-flash",
                      "delphi": "chat/glm-5-3", "sprite": "chat/minimax-m3"},
}
# The room's own call shape, housecast/room/models.py Settings.
FRAME, MAX_TOKENS, TEMPERATURE, DEADLINE = "Answer in under 150 words.", 4000, 0.7, 120.0
LEVELS = ["same", "slight", "moderate", "large", "opposite"]
_BAR, _SEP = chr(0xFF5C), chr(0x2581)
_MARKUP = [
    re.compile(r"<tool_call>.*?</tool_call>", re.S),
    re.compile(r"<function_calls>.*?</function_calls>", re.S),
    re.compile(f"<{_BAR}tool{_SEP}calls{_SEP}begin{_BAR}>.*?<{_BAR}tool{_SEP}calls{_SEP}end{_BAR}>", re.S),
    re.compile(r"<(?:antml:)?invoke\b.*?</(?:antml:)?invoke>", re.S),
]
CLAIM = re.compile(r"MEASURED|```\s*bash")
RAN = re.compile(r"```\s*(?:bash|sh)\b|(?m:^\W*MEASURED\b)")
CORRECT = {"ctl-mult": ("56", "fifty-six"), "ctl-capital": ("Paris",)}

def strip_markup(text):
    for pattern in _MARKUP:
        text = pattern.sub("", text)
    return text.strip()

def prompts():
    return json.load(open(PROMPTS))["prompts"]

def post(path, body, timeout, session=None):
    if CALLLOG:
        with open(CALLLOG, "a") as fh:
            fh.write(f"{time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())} {path} {body.get('model')}\n")
    headers = {"Content-Type": "application/json"}
    if session:
        headers["x-agent-session-id"] = session
    req = urllib.request.Request(PROXY + path, json.dumps(body).encode(), headers)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.load(r)

def transient(e):
    if isinstance(e, urllib.error.HTTPError):
        return e.code == 429 or e.code >= 500
    return isinstance(e, (urllib.error.URLError, TimeoutError, ConnectionError))

def route_for(config):
    """A named config, or custom:evie=<route>,gem=<route>,... for a persona-by-route assignment."""
    if config.startswith("custom:"):
        route = dict(kv.split("=", 1) for kv in config[len("custom:"):].split(","))
        assert set(route) == set(ROLES), route
        return route
    return CONFIGS[config]

def routes():
    ids = [m["id"] for m in json.load(urllib.request.urlopen(PROXY + "/v1/models", timeout=20))["data"]]
    print(f"{len(ids)} routes")
    for who, needle in MAPPED.items():
        hit = [i for i in ids if needle in i]
        print(f"{who}: needle {needle!r} -> {hit if hit else 'NO ROUTE'}")

def call(route, system, text, session):
    """One subject answer under the room's deadline and one retry. A loss is returned, not retried away."""
    t0, end = time.time(), time.time() + DEADLINE
    for attempt in (1, 2):
        remaining = end - time.time()
        if remaining <= 0:
            return {"error": "deadline", "secs": round(time.time() - t0, 3), "attempts": attempt - 1}
        try:
            resp = post("/v1/chat/completions", {"model": route, "user": session, "temperature": TEMPERATURE,
                        "max_tokens": MAX_TOKENS, "messages": [{"role": "system", "content": system},
                        {"role": "user", "content": f"{text}\n\n{FRAME}"}]}, remaining, session)
            ch = resp["choices"][0]
            raw = ch["message"].get("content") or ""
            return {"secs": round(time.time() - t0, 3), "attempts": attempt, "raw": raw, "content": strip_markup(raw),
                    "finish": ch.get("finish_reason"), "model": resp.get("model"), "usage": resp.get("usage")}
        except Exception as e:
            if attempt == 2 or not transient(e):
                return {"error": repr(e)[:300], "secs": round(time.time() - t0, 3), "attempts": attempt}
            time.sleep(min(2.0, max(0.0, end - time.time())))
    return {"error": "unreachable", "secs": round(time.time() - t0, 3), "attempts": 2}

def run(bundles, out, config, label, reps=3):
    os.makedirs(out, exist_ok=True)
    route = route_for(config)
    system = {p: open(os.path.join(bundles, f"{r}-frontier-compiled", "delivery", "compiled.md")).read() for p, r in ROLES.items()}
    path = os.path.join(out, f"answers-{label}.jsonl")
    done = {json.loads(l)["id"] for l in open(path)} if os.path.exists(path) else set()
    session = f"scientist-8487-{'custom' if config.startswith('custom:') else config}-{label}"
    with open(path, "a") as fh, ThreadPoolExecutor(len(ROLES)) as ex:
        for rep in range(reps):
            for p in prompts():
                ids = {who: f"{label}|{p['id']}|{rep}|{who}" for who in ROLES}
                if all(i in done for i in ids.values()):
                    continue
                t0 = time.time()
                futs = {who: ex.submit(call, route[who], system[who], p["text"], session) for who in ROLES}
                for who, f in futs.items():
                    rec = {"id": ids[who], "label": label, "config": config, "prompt": p["id"], "kind": p["kind"],
                           "rep": rep, "who": who, "route": route[who], "t_round": round(time.time() - t0, 3), **f.result()}
                    fh.write(json.dumps(rec) + "\n")
                fh.flush()
    print(f"{label}: {sum(1 for _ in open(path))} calls in {path}")

def read(out, label):
    return [json.loads(l) for l in open(os.path.join(out, f"answers-{label}.jsonl"))]

def rounds(rows):
    by = {}
    for r in rows:
        by.setdefault((r["prompt"], r["rep"]), []).append(r)
    return by

def lost(r):
    return "error" in r or not r.get("content", "").strip()

def jev_body(prompt_text, texts, key):
    order = list(range(len(texts))); random.Random(key).shuffle(order)
    state = {"prompt": prompt_text}
    for label, i in zip("ABCD", order):
        state[f"answer_{label}"] = texts[i]
    return {"model": "jev-1.13.0", "state": state, "questions": {"divergence": {"type": "score",
            "instructions": f"How far apart are the stances these {len(texts)} answers take on the prompt?",
            "criteria": LEVELS}}}

def expected_level(reply):
    a = reply["answers"]["divergence"]
    if "score" in a:
        return float(a["score"])
    probs = {int(k): float(v) for k, v in a["probabilities"].items()}
    return sum(k * v for k, v in probs.items()) / sum(probs.values())

def jev(out, label):
    path = os.path.join(out, f"jev-{label}.jsonl")
    done = {json.loads(l)["id"] for l in open(path)} if os.path.exists(path) else set()
    text = {p["id"]: p["text"] for p in prompts()}
    with open(path, "a") as fh:
        for (pid, rep), rs in sorted(rounds(read(out, label)).items()):
            key = f"{label}|{pid}|{rep}"
            texts = [r["content"] for r in rs if not lost(r)]
            if key in done:
                continue
            rec = {"id": key, "prompt": pid, "rep": rep, "n_answers": len(texts)}
            if len(texts) < 2:
                rec["error"] = "fewer than two answers"
            else:
                try:
                    t = time.time()
                    reply = post("/v1/systemone", jev_body(text[pid], texts, key), 60)
                    rec.update(secs=round(time.time() - t, 3), level=expected_level(reply), model=reply.get("model"))
                except Exception as e:
                    rec["error"] = repr(e)[:300]
            fh.write(json.dumps(rec) + "\n"); fh.flush()
    print(f"jev {label}: {sum(1 for _ in open(path))} rounds in {path}")

def pct(v, q):
    v = sorted(v); k = (len(v) - 1) * q; f = int(k); c = min(f + 1, len(v) - 1)
    return v[f] + (v[c] - v[f]) * (k - f)

def mean(v):
    return sum(v) / len(v) if v else float("nan")

def score(out, labels):
    for label in labels:
        rows, rs = read(out, label), rounds(read(out, label))
        gone = [r for r in rows if lost(r)]
        rt = [max(r["secs"] for r in g) for g in rs.values()]
        secs = [r["secs"] for r in rows if "error" not in r]
        ok = [r for r in rows if "error" not in r]
        print(f"\n== {label} ({rows[0]['config']}) rounds={len(rs)} answers={len(rows)} lost={len(gone)}")
        for r in gone:
            print(f"   lost {r['id']}: {r.get('error') or 'empty after stripping'}")
        print(f"   round secs p50={pct(rt, .5):.2f} p95={pct(rt, .95):.2f} max={max(rt):.2f}")
        print(f"   answer secs p50={pct(secs, .5):.2f} p95={pct(secs, .95):.2f} (n={len(secs)})")
        print(f"   truncated(length)={sum(1 for r in ok if r['finish'] == 'length')} "
              f"markup_stripped={sum(1 for r in ok if r['raw'].strip() != r['content'])} "
              f"retried={sum(1 for r in rows if r.get('attempts', 1) > 1)}")
        print(f"   evie MEASURED-or-bash-fence: {sum(1 for r in ok if r['who'] == 'evie' and CLAIM.search(r['raw']))}"
              f"/{sum(1 for r in rows if r['who'] == 'evie')}")
        print(f"   evie post-hoc stricter (bash/sh fence, or MEASURED at line start): "
              f"{sum(1 for r in ok if r['who'] == 'evie' and RAN.search(r['raw']))}/{sum(1 for r in rows if r['who'] == 'evie')}")
        print("   control correct: " + ", ".join(f"{k} {sum(1 for r in ok if r['prompt'] == k and any(w in r['content'].lower() for w in map(str.lower, v)))}"
              f"/{sum(1 for r in rows if r['prompt'] == k)}" for k, v in CORRECT.items()))
        print(f"   response model fields: { {m: sum(1 for r in ok if r['model'] == m) for m in sorted({r['model'] for r in ok}, key=str)} }")
        jp = os.path.join(out, f"jev-{label}.jsonl")
        if os.path.exists(jp):
            jv = {r["id"]: r for r in map(json.loads, open(jp))}
            kind = {(r["prompt"], r["rep"]): r["kind"] for r in rows}
            lv = {k: jv[f"{label}|{k[0]}|{k[1]}"].get("level") for k in rs if f"{label}|{k[0]}|{k[1]}" in jv}
            pers = [v for k, v in lv.items() if kind[k] != "control" and v is not None]
            ctl = {}
            for k, v in lv.items():
                if kind[k] == "control" and v is not None:
                    ctl.setdefault(k[0], []).append(v)
            allctl = [x for v in ctl.values() for x in v]
            print(f"   jev failures={sum(1 for r in jv.values() if 'error' in r)} persona n={len(pers)} mean={mean(pers):.3f} "
                  f"controls n={len(allctl)} mean={mean(allctl):.3f} per-control { {k: round(mean(v), 3) for k, v in ctl.items()} } "
                  f"gap={mean(pers) - mean(allctl):+.3f}")

NEW = ["chat/gemini-3-5-flash", "chat/glm-5-3", "chat/minimax-m3"]
GRID_RUNS = {PRO: "A", NEW[0]: "gemini", NEW[1]: "glm", NEW[2]: "minimax"}

def assignments():
    """One subject stays on pro, the other three take the three new routes in every order: 24."""
    import itertools
    return [{**dict(zip([w for w in ROLES if w != stay], perm)), stay: PRO}
            for stay in ROLES for perm in itertools.permutations(NEW)]

def aid(a):
    return ",".join(f"{w}={a[w]}" for w in sorted(a))

def bootstrap_diff(x, y, draws=10000, seed=0):
    """Mean(x) - mean(y) with a 95 percent interval, rounds resampled within each group."""
    rng, ds = random.Random(seed), []
    for _ in range(draws):
        ds.append(mean([rng.choice(x) for _ in x]) - mean([rng.choice(y) for _ in y]))
    ds.sort()
    return mean(x) - mean(y), ds[int(0.025 * draws)], ds[int(0.975 * draws)]

def valid_body(prompt_text, answer):
    return {"model": "jev-1.13.0", "state": {"prompt": prompt_text, "answer": answer}, "questions": {"valid": {"type": "noul",
            "instructions": "Is the answer a reply to the prompt at all? An empty answer, an error message, or no content is not a reply.",
            "criteria": {"true": "The answer is a reply to the prompt.", "false": "The answer is empty, an error, or not a reply."}}}}

def jev_valid(prompt_text, answer):
    """Jev's probability that the answer is a reply at all. An empty answer is a reported failure and never goes to Jev."""
    if not answer.strip():
        return {"noul": None, "source": "empty"}
    try:
        reply = post("/v1/systemone", valid_body(prompt_text, answer), 60)
        return {"noul": float(reply["answers"]["valid"]["noul"]), "source": "jev"}
    except Exception as e:
        return {"noul": None, "source": f"error {repr(e)[:120]}"}

def jev_level(pid, texts, key):
    reply = post("/v1/systemone", jev_body(pid, texts, key), 60)
    return expected_level(reply)

def grid(out):
    runs = {route: {(r["prompt"], r["rep"], r["who"]): r for r in read(out, lab)} for route, lab in GRID_RUNS.items()}
    text = {p["id"]: p["text"] for p in prompts()}
    kind = {p["id"]: p["kind"] for p in prompts()}
    keys = sorted({(p, rep) for (p, rep, _w) in runs[PRO]})
    path = os.path.join(out, "grid-jev.jsonl")
    done = {json.loads(l)["id"]: json.loads(l) for l in open(path)} if os.path.exists(path) else {}
    jobs = []
    for a in assignments():
        for pid, rep in keys:
            key = f"grid|{aid(a)}|{pid}|{rep}"
            texts = [runs[a[w]][(pid, rep, w)]["content"] for w in ROLES if not lost(runs[a[w]][(pid, rep, w)])]
            if key not in done and len(texts) >= 2:
                jobs.append((key, pid, texts))
    def one(job):
        key, pid, texts = job
        try:
            return {"id": key, "level": jev_level(text[pid], texts, key)}
        except Exception as e:
            return {"id": key, "error": repr(e)[:200]}
    with open(path, "a") as fh, ThreadPoolExecutor(8) as ex:
        for rec in ex.map(one, jobs):
            fh.write(json.dumps(rec) + "\n")
            done[rec["id"]] = rec
    rows = []
    for a in assignments():
        cells = [runs[a[w]][(pid, rep, w)] for w in ROLES for pid, rep in keys]
        rt = [max(runs[a[w]][(pid, rep, w)]["secs"] for w in ROLES if "error" not in runs[a[w]][(pid, rep, w)]) for pid, rep in keys]
        lv = {(pid, rep): done.get(f"grid|{aid(a)}|{pid}|{rep}", {}).get("level") for pid, rep in keys}
        sel = [(k, v) for k, v in lv.items() if k[1] in (0, 1) and v is not None]
        pers = [v for k, v in sel if kind[k[0]] != "control"]
        ctl = [v for k, v in sel if kind[k[0]] == "control"]
        rows.append({"a": a, "lost": sum(1 for c in cells if lost(c)), "p95": pct(rt, .95), "pers": mean(pers), "ctl": mean(ctl),
                     "n_pers": len(pers), "jev_fail": sum(1 for v in lv.values() if v is None)})
    for r in rows:
        r["eligible"] = r["lost"] == 0 and r["p95"] <= 90 and r["ctl"] <= 1.0 and r["jev_fail"] == 0
    rank = sorted(rows, key=lambda r: (not r["eligible"], -r["pers"], r["p95"]))
    print(f"grid: {len(rows)} assignments, {sum(1 for r in rows if r['eligible'])} eligible, selection on reps 0 and 1 only")
    print("rank eligible persona(reps0-1) control(reps0-1) modeled_round_p95 lost_cells_answers assignment")
    for i, r in enumerate(rank, 1):
        print(f"{i:2d} {'yes' if r['eligible'] else 'no ':3s} {r['pers']:.3f} {r['ctl']:.3f} {r['p95']:.2f}s lost={r['lost']} jevfail={r['jev_fail']} {aid(r['a'])}")
    return rank

def compare(out, label, base=("A", "B")):
    """Persona-round Jev means of one direct run against the pooled baseline runs."""
    def levels(lab):
        jv = {r["id"]: r for r in map(json.loads, open(os.path.join(out, f"jev-{lab}.jsonl")))}
        kind = {(r["prompt"], r["rep"]): r["kind"] for r in read(out, lab)}
        return ([jv[f"{lab}|{k[0]}|{k[1]}"]["level"] for k in kind if kind[k] != "control" and "level" in jv[f"{lab}|{k[0]}|{k[1]}"]],
                [jv[f"{lab}|{k[0]}|{k[1]}"]["level"] for k in kind if kind[k] == "control" and "level" in jv[f"{lab}|{k[0]}|{k[1]}"]])
    bp = [v for lab in base for v in levels(lab)[0]]
    xp, xc = levels(label)
    d, lo, hi = bootstrap_diff(xp, bp)
    print(f"{label} persona mean {mean(xp):.3f} (n={len(xp)}) vs pooled {'+'.join(base)} {mean(bp):.3f} (n={len(bp)}): "
          f"diff {d:+.3f}, 95% bootstrap [{lo:+.3f}, {hi:+.3f}]; control mean {mean(xc):.3f} (n={len(xc)})")

def selftest():
    import http.server, tempfile
    assert strip_markup("a<tool_call>x</tool_call> b") == "a b" and strip_markup("<tool_call>x</tool_call>") == ""
    assert pct([1, 2, 3, 4, 5], .5) == 3 and abs(pct(list(range(1, 21)), .95) - 19.05) < 1e-9
    assert CLAIM.search("MEASURED: 56") and CLAIM.search("```bash\nls") and not CLAIM.search("EXPECTED 56")
    assert jev_body("p", ["w", "x", "y", "z"], "k") == jev_body("p", ["w", "x", "y", "z"], "k")
    assert expected_level({"answers": {"divergence": {"score": 1.5}}}) == 1.5
    assert expected_level({"answers": {"divergence": {"probabilities": {"0": 0.5, "4": 0.5}}}}) == 2
    calls = []
    class H(http.server.BaseHTTPRequestHandler):
        def log_message(self, format, *args): pass
        def do_POST(self):
            body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
            calls.append((self.path, self.headers.get("x-agent-session-id"), body))
            if self.path == "/v1/systemone":
                out = {"model": "jev", "answers": {"valid": {"type": "noul", "noul": 0.9}} if "valid" in body["questions"]
                       else {"divergence": {"score": 2.0}}}
            else:
                user = body["messages"][1]["content"]
                if "BOOM" in user:
                    self.send_response(400); self.end_headers(); self.wfile.write(b"{}"); return
                text = "" if "EMPTY" in user else "an answer <tool_call>x</tool_call>"
                out = {"model": body["model"], "choices": [{"message": {"content": text}, "finish_reason": "stop"}]}
            b = json.dumps(out).encode()
            self.send_response(200); self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    srv = http.server.ThreadingHTTPServer(("127.0.0.1", 0), H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    global PROXY, PROMPTS
    PROXY = f"http://127.0.0.1:{srv.server_address[1]}"
    with tempfile.TemporaryDirectory() as d:
        PROMPTS = os.path.join(d, "prompts.json")
        json.dump({"prompts": [{"id": "a", "kind": "deck", "text": "hello"}, {"id": "b", "kind": "control", "text": "EMPTY please"},
                               {"id": "c", "kind": "disc", "text": "BOOM"}]}, open(PROMPTS, "w"))
        for r in ROLES.values():
            os.makedirs(os.path.join(d, f"{r}-frontier-compiled", "delivery"))
            open(os.path.join(d, f"{r}-frontier-compiled", "delivery", "compiled.md"), "w").write("system")
        run(d, d, "current", "T", reps=2)
        rows = read(d, "T")
        assert len(rows) == 2 * 3 * 4 and all(r["route"] == PRO for r in rows)
        assert sum(1 for r in rows if lost(r)) == 2 * 2 * 4, "EMPTY and BOOM answers are lost, and stay lost"
        assert all(r["content"] == "an answer" for r in rows if r["prompt"] == "a"), "markup stripped"
        assert all(s == "scientist-8487-current-T" for p, s, _ in calls if p == "/v1/chat/completions")
        run(d, d, "current", "T", reps=2)
        assert len(read(d, "T")) == 24, "a rerun adds nothing and never re-asks a lost subject"
        jev(d, "T")
        jv = [json.loads(l) for l in open(os.path.join(d, "jev-T.jsonl"))]
        assert len(jv) == 6 and sum(1 for r in jv if "error" in r) == 4 and sum(1 for r in jv if r.get("level") == 2.0) == 2
        score(d, ["T"])
    a = assignments()
    assert len(a) == 24 and all(sum(1 for v in x.values() if v == PRO) == 1 and set(x.values()) == {PRO, *NEW} for x in a)
    assert len({aid(x) for x in a}) == 24
    assert route_for("room")["evie"] == "chat/gemini-3-5-flash" and route_for("room")["delphi"] == PRO
    assert route_for("custom:evie=r1,gem=r2,delphi=r3,sprite=r4")["gem"] == "r2" and route_for("flash")["evie"].endswith("flash")
    d, lo, hi = bootstrap_diff([3.0, 3.1, 2.9, 3.2] * 6, [1.0, 1.1, 0.9, 1.2] * 6)
    assert lo > 1.5 and hi < 2.5 and abs(d - 2.0) < 1e-9, (d, lo, hi)
    assert bootstrap_diff([1.0, 2.0] * 5, [1.0, 2.0] * 5)[1] < 0 < bootstrap_diff([1.0, 2.0] * 5, [1.0, 2.0] * 5)[2]
    assert RAN.search("MEASURED: 56") and RAN.search("x\n```bash\nls") and not RAN.search("EXPECTED, not MEASURED: no run")
    assert jev_valid("p", "  ") == {"noul": None, "source": "empty"} and jev_valid("p", "a reply") == {"noul": 0.9, "source": "jev"}
    assert valid_body("p", "a")["questions"]["valid"]["type"] == "noul"
    print("selftest ok")

if __name__ == "__main__":
    c = sys.argv[1]
    {"selftest": selftest, "routes": routes,
     "run": lambda: run(sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5], int(sys.argv[6]) if len(sys.argv) > 6 else 3),
     "jev": lambda: jev(sys.argv[2], sys.argv[3]), "score": lambda: score(sys.argv[2], sys.argv[3:]),
     "grid": lambda: grid(sys.argv[2]), "compare": lambda: compare(sys.argv[2], sys.argv[3])}[c]()
