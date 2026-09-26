"""M2 check on the PyLadies Remote room (PREREGISTER.md, AMENDMENT-1.md).

run <base url> <out>              S1 then S2 against the room's public endpoints
score <out> [markup export]       P1-P4 from the saved submits and final snapshot
selftest                          the runner against an in-process fake room

Reads GET /api/room only, and writes POST /api/prompts only. It never holds the
control token, so the room owner opens `submissions` before a run.
"""
import json, os, re, sys, threading, time, urllib.error, urllib.request
from datetime import datetime

HERE = os.path.dirname(os.path.abspath(__file__))
PROMPTS = os.path.join(HERE, "..", "bundle-divergence-2026-09-24", "prompts.json")
TAG = " [m2-probe]"
TERMINAL = {"done", "empty", "failed"}
LEVELS, REPS, DEADLINE, POLL, STOP_SHARE = [1, 5, 10], 2, 180.0, 1.0, 0.2
MARKUP = re.compile(r"<\s*/?\s*(tool_call|function_calls|invoke|antml:|parameter\b)|\"tool_calls\"\s*:|<\|tool|```tool", re.I)

def prompts():
    return json.load(open(PROMPTS))["prompts"]

def http(base, method, path, body=None, timeout=30):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(base + path, data, {"Content-Type": "application/json"}, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.load(e)
        except ValueError:
            return e.code, {}

def ts(v):
    """The contract leaves the clock format open, so take epoch s, epoch ms, or ISO."""
    if v is None:
        return None
    if isinstance(v, (int, float)):
        return v / 1000 if v > 1e12 else float(v)
    return datetime.fromisoformat(str(v).replace("Z", "+00:00")).timestamp()

def settled(snap, room_ids):
    by = {}
    for a in snap.get("answers", []):
        by.setdefault(a["prompt_id"], []).append(a["state"])
    n_sub = len(snap.get("subjects", [])) or 4
    return all(len(by.get(i, [])) >= n_sub and all(s in TERMINAL for s in by[i]) for i in room_ids)

def submit_batch(base, batch, log, deadline, poll):
    """Submit every (row) in the same second, then poll until all settle or the deadline."""
    gate = threading.Barrier(len(batch))
    def one(row):
        gate.wait()
        row["t_submit"] = time.time()
        row["status"], body = http(base, "POST", "/api/prompts", {"text": row["text"], "device": row["device"]})
        row["t_ack"] = time.time()
        row["room_id"] = body.get("id")
        row["reason"] = body.get("reason")
    ts_ = [threading.Thread(target=one, args=(r,)) for r in batch]
    [t.start() for t in ts_]
    [t.join() for t in ts_]
    ids = [r["room_id"] for r in batch if r["room_id"]]
    end = time.time() + deadline
    while ids and time.time() < end:
        _, snap = http(base, "GET", "/api/room")
        if settled(snap, ids):
            break
        time.sleep(poll)
    for r in batch:
        r["t_settled_seen"] = time.time()
        log.write(json.dumps(r) + "\n")
    log.flush()

def run(base, out, deadline=DEADLINE, poll=POLL, levels=LEVELS, reps=REPS):
    os.makedirs(out, exist_ok=True)
    ps = prompts()
    persona = [p for p in ps if p["kind"] != "control"]
    _, snap = http(base, "GET", "/api/room")
    if snap.get("phase") != "submissions":
        sys.exit(f"phase is {snap.get('phase')!r}, not 'submissions': the room owner opens it, the run does not start")
    with open(os.path.join(out, "submits.jsonl"), "w") as log:
        for i, p in enumerate(ps):
            submit_batch(base, [{"stage": "S1", "level": 1, "rep": 0, "pid": p["id"], "kind": p["kind"],
                                 "text": p["text"] + TAG, "device": f"m2-s1-{i}"}], log, deadline, poll)
        k = 0
        for level in levels:
            fails = 0
            for rep in range(reps):
                batch = []
                for j in range(level):
                    p = persona[k % len(persona)]; k += 1
                    batch.append({"stage": "S2", "level": level, "rep": rep, "pid": p["id"], "kind": p["kind"],
                                  "text": p["text"] + TAG, "device": f"m2-s2-l{level}-r{rep}-{j}"})
                submit_batch(base, batch, log, deadline, poll)
                _, snap = http(base, "GET", "/api/room")
                fails += sum(1 for r in batch if prompt_errors(r, snap))
            if fails / (level * reps) > STOP_SHARE:
                print(f"stop rule: {fails}/{level * reps} prompts failed at level {level}, sweep stops here")
                break
    _, snap = http(base, "GET", "/api/room")
    json.dump(snap, open(os.path.join(out, "snapshot.json"), "w"), indent=1)
    print(f"run done: {sum(1 for _ in open(os.path.join(out, 'submits.jsonl')))} prompts, snapshot rev {snap.get('rev')}")

def prompt_answers(r, snap):
    return [a for a in snap.get("answers", []) if a["prompt_id"] == r.get("room_id")]

def prompt_errors(r, snap):
    """Error per AMENDMENT-1: non-2xx submit other than 429, a failed answer, or fewer than four terminal."""
    if r["status"] == 429:
        return False
    if not 200 <= r["status"] < 300:
        return True
    ans = prompt_answers(r, snap)
    n_sub = len(snap.get("subjects", [])) or 4
    return len(ans) < n_sub or any(a["state"] not in TERMINAL or a["state"] == "failed" for a in ans)

def timing(r, snap):
    at = next((ts(p["at"]) for p in snap.get("prompts", []) if p["id"] == r.get("room_id")), None)
    fin = [t for t in (ts(a.get("finished_at")) for a in prompt_answers(r, snap) if a["state"] in TERMINAL) if t is not None]
    if at is None or len(fin) < (len(snap.get("subjects", [])) or 4):
        return None
    return {"all4": max(fin) - at, "first": min(fin) - at, "spread": max(fin) - min(fin)}

def pct(v, q):
    v = sorted(v); k = (len(v) - 1) * q; f = int(k); c = min(f + 1, len(v) - 1)
    return v[f] + (v[c] - v[f]) * (k - f)

def score(out, export=None, write=True):
    rows = [json.loads(l) for l in open(os.path.join(out, "submits.jsonl"))]
    snap = json.load(open(os.path.join(out, "snapshot.json")))
    lines, res = [], {}
    say = lines.append
    say(f"prompts submitted {len(rows)}, snapshot rev {snap.get('rev')}, subjects {len(snap.get('subjects', []))}")
    say(f"submit statuses {sorted({r['status'] for r in rows})}, 429s {sum(1 for r in rows if r['status'] == 429)}")
    for level in sorted({r["level"] for r in rows if r["stage"] == "S2"}):
        rs = [r for r in rows if r["stage"] == "S2" and r["level"] == level]
        t = [x["all4"] for x in (timing(r, snap) for r in rs) if x]
        f = [x["first"] for x in (timing(r, snap) for r in rs) if x]
        err = sum(1 for r in rs if prompt_errors(r, snap))
        res[level] = (err, pct(t, .95) if t else None)
        say(f"S2 level {level}: n={len(rs)} errors={err} all4 p50={pct(t, .5):.1f}s p95={pct(t, .95):.1f}s max={max(t):.1f}s first p50={pct(f, .5):.1f}s" if t
            else f"S2 level {level}: n={len(rs)} errors={err} no settled prompt")
    s1 = [r for r in rows if r["stage"] == "S1"]
    t1 = [x["all4"] for x in (timing(r, snap) for r in s1) if x]
    if t1:
        say(f"S1 sequential: n={len(s1)} all4 p50={pct(t1, .5):.1f}s p95={pct(t1, .95):.1f}s")
    top = max(res) if res else None
    p1 = top == 10 and res[10][0] == 0 and res[10][1] is not None and res[10][1] <= 90
    p95 = res.get(top, (0, None))[1]
    say(f"P1 load: {'PASS' if p1 else 'FAIL'} (top level reached {top}, errors {res.get(top, (None,))[0]}, p95 all4 {'n/a' if p95 is None else f'{p95:.1f}s'})")
    ans = [a for r in rows for a in prompt_answers(r, snap)]
    empty = sum(1 for a in ans if a["state"] == "empty")
    empty_ok = bool(ans) and empty / len(ans) <= 0.05
    if export:
        texts = [json.loads(l).get("text") or "" for l in open(export)]
        mk = sum(1 for t in texts if MARKUP.search(t))
        say(f"P2 answers: {'PASS' if empty_ok and mk == 0 else 'FAIL'} (empty {empty}/{len(ans)}, markup {mk}/{len(texts)} from export)")
    else:
        say(f"P2 answers: {'FAIL' if not empty_ok else 'empty half PASS, markup UNMEASURED'} (empty {empty}/{len(ans)}, no export)")
    # Only stance scores sit on the level/4 scale (comment 20132). A lexical fallback is counted, not averaged in.
    done = [d for d in snap.get("divergence", []) if d.get("state") == "done"]
    div = {d["prompt_id"]: d.get("score") for d in done if d.get("method", "stance") == "stance"}
    say(f"divergence methods on done prompts: {sorted({d.get('method', 'stance') for d in done})}, lexical {sum(1 for d in done if d.get('method') == 'lexical')}")
    per = {r["pid"]: div.get(r.get("room_id")) for r in s1}
    kinds = {r["pid"]: r["kind"] for r in s1}
    pers = [v for k, v in per.items() if kinds[k] != "control" and v is not None]
    ctl = [v for k, v in per.items() if kinds[k] == "control" and v is not None]
    if pers and ctl:
        gap = sum(pers) / len(pers) - sum(ctl) / len(ctl)
        p3 = gap >= 0.25 and all(c <= 0.25 for c in ctl) and len(ctl) == 2
        say(f"P3 divergence: {'PASS' if p3 else 'FAIL'} (gap {gap:+.3f}, persona n={len(pers)}, controls {ctl})")
    else:
        say(f"P3 divergence: UNMEASURED (persona scores {len(pers)}, control scores {len(ctl)})")
    tt = [x for x in (timing(r, snap) for r in rows) if x]
    vis = sum(1 for x in tt if x["spread"] >= 5)
    say(f"P4 progress: {'PASS' if tt and vis / len(tt) >= 0.8 else 'FAIL'} ({vis}/{len(tt)} settled prompts with first-to-fourth spread >= 5s)")
    # AMENDMENT-3: descriptive, no threshold. What a prompt with a bad answer does to the rest of its row.
    dstate = {d["prompt_id"]: d for d in snap.get("divergence", [])}
    bad = [r for r in rows if any(a["state"] in ("empty", "failed") for a in prompt_answers(r, snap))]
    stuck = [r for r in rows if any(a["state"] not in TERMINAL for a in prompt_answers(r, snap))]
    say(f"stall: {len(bad)} prompts with an empty or failed answer, {len(stuck)} prompts with an answer never terminal")
    for r in bad + [r for r in stuck if r not in bad]:
        ans = prompt_answers(r, snap)
        d = dstate.get(r.get("room_id"), {})
        t = timing(r, snap)
        say(f"  {r['stage']} L{r['level']} {r['pid']}: states {[a['state'] for a in ans]} reasons {[a.get('reason') for a in ans if a.get('reason')]}"
            f" divergence {d.get('state', 'absent')}/{d.get('method', '-')}{' ' + str(d.get('reason')) if d.get('reason') else ''}"
            f" all4 {'%.1fs' % t['all4'] if t else 'never'}")
    text = "\n".join(lines)
    if write:
        open(os.path.join(out, "score.txt"), "w").write(text + "\n")
    print(text)
    return text

class FakeRoom:
    """Implements the comment-20132 attendee surface closely enough to test the runner."""
    def __init__(self, delay, spread, rules=None):
        from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
        self.lock, self.phase, self.rev, self.seq = threading.Lock(), "submissions", 0, 0
        self.prompts, self.answers, self.divergence, self.last = [], [], [], {}
        self.delay, self.spread, self.rules = delay, spread, rules or {}
        room = self
        class H(BaseHTTPRequestHandler):
            def log_message(self, format, *args): pass
            def reply(self, code, body):
                b = json.dumps(body).encode()
                self.send_response(code); self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
            def do_GET(self):
                with room.lock:
                    self.reply(200, room.snapshot()) if self.path == "/api/room" else self.reply(404, {})
            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                self.reply(*room.submit(body))
        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), H)
        self.base = f"http://127.0.0.1:{self.srv.server_address[1]}"
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()
    def snapshot(self):
        return {"rev": self.rev, "phase": self.phase, "round": None,
                "subjects": [{"id": f"s{i}", "label": f"Subject {i}"} for i in range(4)],
                "prompts": list(self.prompts), "answers": [dict(a) for a in self.answers],
                "divergence": list(self.divergence), "rounds": []}
    def submit(self, body):
        with self.lock:
            if self.phase != "submissions":
                return 409, {"reason": "closed"}
            text, dev = body.get("text", ""), body.get("device")
            if not text.strip() or len(text) > 280:
                return 422, {"reason": "length"}
            if time.time() - self.last.get(dev, -1e9) < 20:
                return 429, {"reason": "rate"}
            self.last[dev] = time.time(); self.seq += 1; self.rev += 1
            pid = f"p{self.seq}"
            self.prompts.append({"id": pid, "seq": self.seq, "text": text, "at": datetime.now().astimezone().isoformat()})
            for i in range(4):
                self.answers.append({"prompt_id": pid, "subject_id": f"s{i}", "state": "queued"})
        threading.Thread(target=self.fan, args=(pid, text), daemon=True).start()
        return 201, {"id": pid}
    def fan(self, pid, text):
        base = text.replace(TAG, "")
        for i in range(4):
            time.sleep(self.delay + self.spread * i)
            state = self.rules.get((base, i), "done")
            with self.lock:
                a = next(a for a in self.answers if a["prompt_id"] == pid and a["subject_id"] == f"s{i}")
                a.update(state=state, finished_at=time.time() * 1000)
                self.rev += 1
        with self.lock:
            score = 0.05 if base in ("What is 7 times 8?", "What is the capital of France?") else 0.5
            method = "lexical" if base == "Describe your ideal weekend in one sentence." else "stance"
            self.divergence.append({"prompt_id": pid, "state": "done", "score": 0.9 if method == "lexical" else score, "method": method})
            self.rev += 1

def selftest():
    import tempfile
    assert pct([1, 2, 3, 4, 5], .5) == 3 and abs(pct(list(range(1, 21)), .95) - 19.05) < 1e-9
    assert ts(1700000000) == 1700000000.0 and ts(1700000000000) == 1700000000.0
    assert ts("2026-09-26T04:00:00Z") == ts("2026-09-26T04:00:00+00:00")
    ms, s = ts("2026-09-26T04:52:10.123Z"), ts("2026-09-26T04:52:10Z")  # the room's confirmed format
    assert ms is not None and s is not None and abs(ms - s - 0.123) < 1e-6
    assert MARKUP.search('<tool_call>{"name": "x"}</tool_call>') and MARKUP.search('{"tool_calls": []}')
    assert not MARKUP.search("I would never deploy on a Friday.") and not MARKUP.search("my favourite colour is #14d5b8")
    # A room that answers in 0.1s with a 2s spread: every prompt settles, P4 passes at the 5s bar only if spread >= 5.
    room = FakeRoom(0.02, 0.01, rules={("Do you want ice cream", 2): "empty", ("What is 7 times 8?", 3): "failed"})
    with tempfile.TemporaryDirectory() as out:
        run(room.base, out, deadline=5, poll=0.02, levels=[1, 2], reps=2)
        rows = [json.loads(l) for l in open(os.path.join(out, "submits.jsonl"))]
        assert len(rows) == 10 + (1 + 2) * 2, len(rows)
        assert len({r["device"] for r in rows}) == len(rows), "one device per simulated submitter"
        assert all(r["text"].endswith(TAG) and r["status"] == 201 for r in rows)
        s = score(out, write=False)
        assert "P1 load: FAIL (top level reached 2" in s, s  # the sweep here stops short of 10 by design
        assert "empty 2/64" in s and "markup UNMEASURED" in s, s
        assert "P3 divergence: PASS (gap +0.450, persona n=7" in s and "lexical 1" in s, s
        assert "P4 progress: FAIL" in s, s
        assert "stall: 3 prompts with an empty or failed answer, 0 prompts with an answer never terminal" in s, s
        assert "S1 L1 ctl-mult: states ['done', 'done', 'done', 'failed']" in s and "divergence done/stance" in s, s
        exp = os.path.join(out, "export.jsonl")
        open(exp, "w").write(json.dumps({"text": "fine"}) + "\n" + json.dumps({"text": "<invoke name='x'>"}) + "\n")
        assert "markup 1/2 from export" in score(out, exp, write=False)
    # The stop rule: a room where every answer fails stops the sweep at the first level.
    bad = FakeRoom(0.01, 0.0, rules={(p["text"], i): "failed" for p in prompts() for i in range(4)})
    with tempfile.TemporaryDirectory() as out:
        run(bad.base, out, deadline=5, poll=0.02, levels=[1, 2], reps=2)
        rows = [json.loads(l) for l in open(os.path.join(out, "submits.jsonl"))]
        assert max(r["level"] for r in rows if r["stage"] == "S2") == 1, "stop rule did not fire"
    # A closed room: the run refuses to start.
    shut = FakeRoom(0.01, 0.0); shut.phase = "holding"
    with tempfile.TemporaryDirectory() as out:
        try:
            run(shut.base, out, deadline=1, poll=0.02)
            raise AssertionError("ran against a closed room")
        except SystemExit as e:
            assert "not 'submissions'" in str(e)
    # The rate limit: a reused device inside 20s gets 429, which is reported but is not an error.
    assert room.submit({"text": "x", "device": "m2-s1-0"})[0] == 429
    print("selftest ok")

if __name__ == "__main__":
    c = sys.argv[1]
    {"selftest": selftest, "run": lambda: run(sys.argv[2].rstrip("/"), sys.argv[3]),
     "score": lambda: score(sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else None)}[c]()
