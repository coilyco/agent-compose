"""Pressure-variant cases for teable:coilyco/housecast#8493, on the room's call shape.

run <cases.json> <bundles dir> <out dir> <config> <reps> [case_id ...]   four personas per round, checked
summary <out dir> <config> [config ...]                                 counts of each deterministic check
seed <out dir> <config> <case_id> <rep> <out.json>                      one round as a casebook runs doc
selftest

No agent grades an answer here. The rows carry deterministic checks from caserun.checks, and
the summary counts them. Whether an answer keeps its commitment is for the room to decide.
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
                    rec["lost"] = not rec["text"].strip()
                    rec["checks"] = [] if rec["lost"] else c.checks(case["prompt"], w, rec["text"])
                    fh.write(json.dumps(rec) + "\n")
                fh.flush()
    print(f"{config}: {sum(1 for _ in open(path))} answers in {path}")

def flagged(chk):
    return chk["result"] == "yes" or (isinstance(chk["result"], int) and chk["result"] > 0 and chk["name"] != "words")

def summary(cases, out, configs):
    for config in configs:
        rows = [json.loads(l) for l in open(os.path.join(out, f"pressure-{config}.jsonl"))]
        print(f"\n== {config}")
        for case in cases:
            rs = [r for r in rows if r["case"] == case["id"]]
            if not rs:
                continue
            n = len({r["rep"] for r in rs})
            print(f"  {case['id']} rounds={n} lost subjects={sum(1 for r in rs if r['lost'])}")
            for who in NAME.values():
                mine = [r for r in rs if r["who"] == who and not r["lost"]]
                names = [k["name"] for k in mine[0]["checks"] if k["name"] != "words"] if mine else []
                counts = {k: sum(1 for r in mine for x in r["checks"] if x["name"] == k and flagged(x)) for k in names}
                counts = {k: v for k, v in counts.items() if v}
                print(f"    {who:16s} answers={len(mine)} checks that fired: {counts if counts else 'none'}")

def seed(cases, out, config, case_id, rep, dest):
    case = next(x for x in cases if x["id"] == case_id)
    rows = {r["who"]: r for r in map(json.loads, open(os.path.join(out, f"pressure-{config}.jsonl")))
            if r["case"] == case_id and r["rep"] == rep}
    texts = {w: rows[NAME[w]]["text"] for w in b.ROLES}
    div = c.stance(case, texts, f"pressure|{config}|{case_id}|{rep}")
    route = ", ".join(sorted({r["route"] for r in rows.values()}))
    note = f"Answers from a fresh run, arm {config}{TAG}, round {rep}, at the room's call shape. Checks are deterministic text checks and carry no verdict. Divergence is the Jev stance level 0 to 4."
    valid = c.validity(case, texts) if os.environ.get("VALID_RESPONSE") else None
    doc = c.build(case_id, case, texts, route, c.now(), note + (c.VALID_NOTE if valid else ""), div, valid)
    json.dump(doc, open(dest, "w"), indent=1)
    print(dest, f"div={div}")

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
