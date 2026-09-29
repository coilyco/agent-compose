"""Pressure-variant cases for teable:coilyco/housecast#8493, on the room's call shape.

run <cases.json> <bundles dir> <out dir> <config> <reps> [case_id ...]   four personas per round, graded
summary <out dir> <config> [config ...]                                 fail and low-confidence counts
seed <out dir> <config> <case_id> <rep> <out.json>                      one round as a casebook runs doc
selftest
"""
import json, os, sys, time
from concurrent.futures import ThreadPoolExecutor

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import backends as b
import caserun as c

NAME = {w: n for n, w in c.WHO.items()}
# PRESSURE_EDIT is {"delphi": {"after": "<exact line>", "line": "<line to add>"}}, and PRESSURE_TAG names the arm.
EDIT = json.loads(os.environ.get("PRESSURE_EDIT", "{}"))
TAG = os.environ.get("PRESSURE_TAG", "")

def apply_edit(text, edit):
    lines = text.split("\n")
    match = (lambda l: l.strip() == edit["after"].strip()) if "after" in edit else (lambda l: l.startswith(edit["after_prefix"]))
    i = next(i for i, l in enumerate(lines) if match(l))
    lines.insert(i + 1, edit["line"])
    return "\n".join(lines)

def run(cases, bundles, out, config, reps, only):
    os.makedirs(out, exist_ok=True)
    route = b.route_for(config)
    system = {p: open(os.path.join(bundles, f"{r}-frontier-compiled", "delivery", "compiled.md")).read() for p, r in b.ROLES.items()}
    for w, edit in EDIT.items():
        system[w] = apply_edit(system[w], edit)
    path = os.path.join(out, f"pressure-{config}{TAG}.jsonl")
    done = {json.loads(l)["id"] for l in open(path)} if os.path.exists(path) else set()
    with open(path, "a") as fh, ThreadPoolExecutor(4) as ex:
        for case in cases:
            if only and case["id"] not in only:
                continue
            for rep in range(reps):
                ids = {w: f"{config}{TAG}|{case['id']}|{rep}|{NAME[w]}" for w in b.ROLES}
                if all(i in done for i in ids.values()):
                    continue
                t0 = time.time()
                futs = {w: ex.submit(b.call, route[w], system[w], case["prompt"], f"scientist-8487-pressure-{config}{TAG}") for w in b.ROLES}
                res = {w: f.result() for w, f in futs.items()}
                for w, r in res.items():
                    rec = {"id": ids[w], "config": config, "case": case["id"], "rep": rep, "who": NAME[w], "route": route[w],
                           "t_round": round(time.time() - t0, 3), "secs": r["secs"], "text": r.get("content", ""), "error": r.get("error"), "finish": r.get("finish")}
                    if rec["text"].strip():
                        g, p, why = c.grade(case, NAME[w], rec["text"])
                        rec.update(grade=g, p=p, why=why)
                    else:
                        rec.update(grade="LOST", p=None, why=rec["error"] or "empty after stripping")
                    fh.write(json.dumps(rec) + "\n")
                fh.flush()
    print(f"{config}: {sum(1 for _ in open(path))} answers in {path}")

def summary(cases, out, configs):
    for config in configs:
        rows = [json.loads(l) for l in open(os.path.join(out, f"pressure-{config}.jsonl"))]
        print(f"\n== {config}")
        for case in cases:
            rs = [r for r in rows if r["case"] == case["id"]]
            if not rs:
                continue
            n = len({r["rep"] for r in rs})
            tgt = [r for r in rs if r["who"] == case["persona"]]
            oth = [r for r in rs if r["who"] != case["persona"]]
            low = lambda r: r["p"] is not None and 0.35 <= r["p"] <= 0.65
            print(f"  {case['id']:14s} rounds={n} target {case['persona']}: FAIL {sum(1 for r in tgt if r['grade'] == 'FAIL')}/{len(tgt)} low-conf {sum(1 for r in tgt if low(r))} lost {sum(1 for r in tgt if r['grade'] == 'LOST')}"
                  f" | others FAIL {sum(1 for r in oth if r['grade'] == 'FAIL')}/{len(oth)} low-conf {sum(1 for r in oth if low(r))} lost {sum(1 for r in oth if r['grade'] == 'LOST')}"
                  f" | round secs max {max(r['t_round'] for r in rs):.1f}")

def seed(cases, out, config, case_id, rep, dest):
    case = next(x for x in cases if x["id"] == case_id)
    rows = {r["who"]: r for r in map(json.loads, open(os.path.join(out, f"pressure-{config}.jsonl")))
            if r["case"] == case_id and r["rep"] == rep}
    texts = {w: rows[NAME[w]]["text"] for w in b.ROLES}
    div = c.stance(case, texts, f"pressure|{config}|{case_id}|{rep}")
    doc = {"case_id": case_id, "route": ", ".join(sorted({r["route"] for r in rows.values()})),
           "at": c.now().strftime("%Y-%m-%dT%H:%M:%SZ"), "divergence": round(div, 3),
           "note": f"Seeded from pressure run {config} round {rep}, fresh calls at the room's call shape, graded on 2026-09-29. Divergence is the Jev stance level 0 to 4.",
           "answers": {n: {"text": rows[n]["text"], "grade": rows[n]["grade"],
                           "reason": f"Jev pass probability {rows[n]['p']:.2f}." + (" Low confidence." if abs(rows[n]["p"] - 0.5) < 0.15 else "") + " " + rows[n]["why"]}
                       for n in c.WHO}}
    json.dump(doc, open(dest, "w"), indent=1)
    print(dest, {n: a["grade"] for n, a in doc["answers"].items()}, f"div={div}")

def selftest():
    cases = json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "pressure-cases.json")))
    assert len({x["id"] for x in cases}) == len(cases) == 6 and [x["order"] for x in cases] == list(range(7, 13))
    assert apply_edit("a\n  * b\nc", {"after": "  * b", "line": "  * new"}) == "a\n  * b\n  * new\nc"
    assert apply_edit("a\n**Refuse** - x\nc", {"after_prefix": "**Refuse**", "line": "new"}) == "a\n**Refuse** - x\nnew\nc"
    assert all(len(x["prompt"]) <= 280 and len(x["commitment"]) <= 140 and x["persona"] in c.WHO for x in cases)
    print("selftest ok")

if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "selftest":
        selftest()
    else:
        cases = json.load(open(sys.argv[2] if cmd == "run" else os.path.join(os.path.dirname(os.path.abspath(__file__)), "pressure-cases.json")))
        if cmd == "run":
            run(cases, sys.argv[3], sys.argv[4], sys.argv[5], int(sys.argv[6]), sys.argv[7:])
        elif cmd == "seed":
            seed(cases, sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]), sys.argv[6])
        else:
            summary(cases, sys.argv[2], sys.argv[3:])
