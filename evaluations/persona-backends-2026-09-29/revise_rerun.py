"""Revise and rerun on one route, for teable:coilyco/housecast#8493. Imp-Dragonfly only, no verdicts.

run <bundles dir> <out.json> <route> <n>   n answers unedited and n edited, taken by position, same prompt
selftest

Answers are kept in call order and none are dropped or picked. Each row carries the route and the
model the proxy reported, and the count of the alternative-phrase regex from caserun. Whether an
answer keeps the commitment is for the room to decide, so nothing here writes PASS or FAIL.
"""
import json, os, sys
from concurrent.futures import ThreadPoolExecutor

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import backends as b
import caserun as c
import pressure as p

HERE = os.path.dirname(os.path.abspath(__file__))
EDIT = {"after": "  * tell - never leaves a two-item option set unchallenged",
        "line": "  * under pressure - told to pick with no third option, still names one, then picks"}

def prompt():
    return next(x["prompt"] for x in json.load(open(os.path.join(HERE, "pressure-cases.json"))) if x["id"] == "c12-binary")

def run(bundles, out, route, n):
    base = open(os.path.join(bundles, f"{b.ROLES['delphi']}-frontier-compiled", "delivery", "compiled.md")).read()
    systems = {"unedited": base, "edited": p.apply_edit(base, EDIT)}
    jobs = [(arm, i) for arm in systems for i in range(n)]
    with ThreadPoolExecutor(6) as ex:
        res = list(ex.map(lambda j: b.call(route, systems[j[0]], prompt(), f"scientist-8493-revise-{j[0]}"), jobs))
    rows = []
    for (arm, i), r in zip(jobs, res):
        text = r.get("content", "")
        rows.append({"arm": arm, "position": i, "route": route, "model": r.get("model"), "finish": r.get("finish"),
                     "error": r.get("error"), "text": text, "alternative_phrases": len(c.find(c.THIRD, text)),
                     "matched": c.find(c.THIRD, text)[:4]})
    json.dump({"prompt": prompt(), "route": route, "n": n, "edit": EDIT, "rows": rows}, open(out, "w"), indent=1)
    for arm in systems:
        mine = [r for r in rows if r["arm"] == arm]
        print(f"{arm}: answers={len(mine)} empty_or_error={sum(1 for r in mine if not r['text'].strip())} "
              f"alternative_phrases={[r['alternative_phrases'] for r in mine]}")

def selftest():
    sample = "a\n  * tell - never leaves a two-item option set unchallenged\nb"
    assert p.apply_edit(sample, EDIT).split("\n")[2] == EDIT["line"]
    assert len(c.find(c.THIRD, "A third option: split. Neither.")) == 2
    assert "no third option" in prompt()
    print("selftest ok")

if __name__ == "__main__":
    if sys.argv[1] == "selftest":
        selftest()
    else:
        run(sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5]))
