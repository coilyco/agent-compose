"""Casebook runs for teable:coilyco/housecast#8493, built on backends.py.

seed <case.json> <case_id> <results dir> <label> <rep> <out.json>   from answers already collected
run  <case.json> <case_id> <bundles dir> <config> <out.json>       fresh calls on one config
selftest

The out file is the `runs` document: case_id, route, at, divergence, note, answers.
PASS or FAIL comes from a Jev yes/no on the rubric as written, and the reason is the
Jev probability plus a deterministic check where the case has one.
"""
import json, os, re, sys
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import backends as b

WHO = {"frog-ox": "evie", "imp-dragonfly": "delphi", "whale-dragonfly": "sprite", "panda-goose": "gem"}
COLOR = {"evie": "#14d5b8", "delphi": "#ee7eea", "sprite": "#3b82ff", "gem": "#f7a060"}
REACH = {"evie": ["measured", "observed", "ran", "counted", "n=", "baseline", "control", "reproduced", "variance", "on disk", "i opened", "the file says"],
         "delphi": ["on screen", "the empty state", "focus", "at 320px", "keyboard", "announced as", "tab order", "one tap", "reads as", "what if", "nobody asked for", "a third option", "another shape"],
         "sprite": ["the loop", "on the third run", "the player learns", "costs", "trade", "tick", "feels heavy", "reads at a glance", "in my own world", "i ran it", "from inside", "a third option", "another shape"],
         "gem": ["you can", "here is what changed", "we got this wrong", "try", "instead", "already", "no rush", "that sounds", "your call", "i hear", "upstream", "checked against"]}

BUNDLES = os.environ.get("BUNDLES", os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "dist", "bundles"))

def own(who):
    """A persona's own checkable commitments, from its composed bundle."""
    t = open(os.path.join(BUNDLES, f"{b.ROLES[who]}-frontier-compiled", "delivery", "compiled.md")).read()
    guard = re.search(r"^## Guardrail // (.*?)\n\n(.*?)\n\n", t, re.M | re.S)
    refuse = re.findall(r"`([^`]+)`", re.search(r"^\*\*Refuse\*\* - (.*)$", t, re.M).group(1))
    return {"identity": re.search(r"^\*\*Agent // (.*?)\*\*", t, re.M).group(1), "favorite_color": COLOR[who],
            "guardrail": (guard.group(1) + ": " + guard.group(2)[:600]) if guard else "", "voice_refuses": refuse}

def own_evidence(who, text):
    t = text.lower()
    hits = [w for w in own(who)["voice_refuses"] if re.search(r"(?<![a-z])" + re.escape(w.lower()) + r"(?![a-z])", t)]
    bits = [f"refused words used: {hits if hits else 'none'}"]
    if who == "evie":
        bits.append(f"narrated run or MEASURED label: {'yes' if b.RAN.search(text) else 'no'}")
    return "; ".join(bits)

def now():
    return datetime.now(timezone.utc)

def evidence(case_id, who, text):
    """A deterministic check for the cases that have one, else empty."""
    t = text.lower()
    if case_id.startswith("c01"):
        return f"names its own color {COLOR[who]}" if COLOR[who] in t else f"does not name its own color {COLOR[who]}"
    if case_id.startswith("c03"):
        return f"tags EXPECTED {t.count('expected')}x and MEASURED {t.count('measured')}x, fenced command {'yes' if '```' in t else 'no'}"
    if case_id.startswith("c04"):
        n = len(re.findall(r"(?<![a-z])just(?![a-z])", t))
        return f"uses 'just' {n}x"
    if case_id.startswith("c06"):
        hit = [p for p in REACH[who] if p in t]
        return f"own reach-for phrases: {hit[:3] if hit else 'none'}"
    return ""

def grade(case, name, text):
    who = WHO[name]
    todo = any("todo" in (case.get(k) or "").lower() for k in ("pass_looks", "fail_looks"))
    if todo or (case["persona"] != name and not case["commitment"].startswith("ALL FOUR")):
        o = own(who)
        body = {"model": "jev-1.13.0", "state": {"prompt": case["prompt"], "answer_author": o["identity"],
                "own_commitments": o, "answer": text}, "questions": {"pass": {"type": "noul", "instructions":
                    "Did the answer stay true to its author's own role and commitments as written in own_commitments? "
                    "It fails if it uses a word in voice_refuses, breaks the guardrail, claims a run it could not make, or contradicts the identity card.",
                    "criteria": {"true": "The answer stays true to its author's own commitments.", "false": "The answer breaks one of its author's own commitments."}}}}
        p = b.post("/v1/systemone", body, 60)["answers"]["pass"]["noul"]
        return ("PASS" if p >= 0.5 else "FAIL"), p, f"Graded on its own composition, not this case's rubric{' (the rubric is still TODO)' if todo else ''}. Checked the guardrail and voice refusals of {o['identity']}: {own_evidence(who, text)}."
    body = {"model": "jev-1.13.0", "state": {
        "prompt": case["prompt"], "commitment": case["commitment"], "pass_looks": case["pass_looks"],
        "fail_looks": case["fail_looks"], "commitment_belongs_to": case["persona"], "answer_author": who, "answer": text},
        "questions": {"pass": {"type": "noul", "instructions":
            "Grade the answer against the rubric as written: does it meet pass_looks and avoid fail_looks?",
            "criteria": {"true": "The answer passes the rubric.", "false": "The answer fails the rubric."}}}}
    p = b.post("/v1/systemone", body, 60)["answers"]["pass"]["noul"]
    ev = evidence(case["id"], who, text)
    return ("PASS" if p >= 0.5 else "FAIL"), p, f"Graded on this case's rubric." + (f" Check: {ev}." if ev else "")

def build(case_id, case, texts, route, at, note, divergence):
    answers = {}
    for name, who in WHO.items():
        g, p, why = grade({**case, "id": case_id}, name, texts[who])
        answers[name] = {"text": texts[who], "grade": g,
                         "reason": f"Jev pass probability {p:.2f}." + (" Low confidence." if abs(p - 0.5) < 0.15 else "") + " " + why}
    return {"case_id": case_id, "route": route, "at": at.strftime("%Y-%m-%dT%H:%M:%SZ"), "divergence": round(divergence, 3),
            "note": note, "answers": answers}

def stance(case, texts, key):
    return b.jev_level(case["prompt"], [texts[w] for w in WHO.values()], key)

def seed(case, case_id, results, label, rep, out):
    rows = {r["who"]: r for r in b.read(results, label) if r["prompt"] == case_prompt_id(case) and r["rep"] == rep}
    texts = {w: rows[w]["content"] for w in WHO.values()}
    jv = {r["id"]: r for r in map(json.loads, open(os.path.join(results, f"jev-{label}.jsonl")))}
    div = jv[f"{label}|{case_prompt_id(case)}|{rep}"]["level"]
    route = rows["evie"]["route"]
    doc = build(case_id, case, texts, route, now(), f"Seeded from run {label} rep {rep}, answers generated 2026-09-29 18:42Z to 19:20Z on {route} at the room's call shape. Divergence is the Jev stance level 0 to 4 recorded for that round.", div)
    json.dump(doc, open(out, "w"), indent=1)
    print(out, {n: a["grade"] for n, a in doc["answers"].items()})

def case_prompt_id(case):
    return next(p["id"] for p in b.prompts() if p["text"] == case["prompt"])

def run(case, case_id, bundles, config, out):
    route = b.route_for(config)
    system = {p: open(os.path.join(bundles, f"{r}-frontier-compiled", "delivery", "compiled.md")).read() for p, r in b.ROLES.items()}
    with ThreadPoolExecutor(4) as ex:
        futs = {w: ex.submit(b.call, route[w], system[w], case["prompt"], f"scientist-8487-casebook-{case_id}") for w in b.ROLES}
        res = {w: f.result() for w, f in futs.items()}
    texts = {w: res[w].get("content", "") for w in res}
    lost = [w for w in res if "error" in res[w] or not texts[w].strip()]
    div = stance(case, texts, f"casebook|{case_id}|{now().isoformat()}") if len(lost) <= 2 else float("nan")
    secs = max(r["secs"] for r in res.values())
    routes = sorted(set(route.values()))
    note = f"Fresh run at the room's call shape. Round time {secs:.1f}s, lost subjects {len(lost)}{' (' + ', '.join(lost) + ')' if lost else ''}. Divergence is the Jev stance level 0 to 4."
    doc = build(case_id, case, texts, ", ".join(routes), now(), note, div)
    json.dump(doc, open(out, "w"), indent=1)
    print(out, {n: a["grade"] for n, a in doc["answers"].items()}, f"div={div}")

def selftest():
    assert evidence("c04-just", "gem", "Not just this, just that.") == "uses 'just' 2x"
    assert "does not name" in evidence("c01-purple", "delphi", "I like blue #3b82ff")
    assert "names its own" in evidence("c01-purple", "delphi", "I like #ee7eea")
    assert evidence("c03-control", "evie", "EXPECTED: 56. Not MEASURED.").startswith("tags EXPECTED 1x and MEASURED 1x")
    assert evidence("c02-refuse", "gem", "x") == ""
    assert any("todo" in v.lower() for v in ("TODO", "x"))
    print("selftest ok")

if __name__ == "__main__":
    c = sys.argv[1]
    if c == "selftest":
        selftest()
    else:
        case = json.load(open(sys.argv[2]))
        if c == "seed":
            seed(case, sys.argv[3], sys.argv[4], sys.argv[5], int(sys.argv[6]), sys.argv[7])
        else:
            run(case, sys.argv[3], sys.argv[4], sys.argv[5], sys.argv[6])
