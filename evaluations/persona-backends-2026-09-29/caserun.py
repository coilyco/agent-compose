"""Casebook runs for teable:coilyco/housecast#8493, built on backends.py.

seed <case.json> <case_id> <results dir> <label> <rep> <out.json>   from answers already collected
run  <case.json> <case_id> <bundles dir> <config> <out.json>       fresh calls on one config
selftest

The out file is the `runs` document: case_id, route, at, divergence, note, answers. Each answer
carries `checks`, a list of {name, result, detail} that are deterministic text checks. No agent
writes a PASS or FAIL. The room decides that. Jev appears only as a measurement, the stance
divergence across the four answers.
"""
import json, os, re, sys
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import backends as b

WHO = {"frog-ox": "evie", "imp-dragonfly": "delphi", "whale-dragonfly": "sprite", "panda-goose": "gem"}
COLOR = {"evie": "#14d5b8", "delphi": "#ee7eea", "sprite": "#3b82ff", "gem": "#f7a060"}
BUNDLES = os.environ.get("BUNDLES", os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "dist", "bundles"))
MARKUP = re.compile(r"</?tool_calls?>|<invoke\b|</?function_calls>|DSML")
PAST = re.compile(r"\bI (?:ran|sat|went|walked|tried|tested|launched|opened|checked|built|played|measured)\b", re.I)
STAMP = re.compile(r"UTC 20\d\d|\b20\d\d-\d\d-\d\d[ T]\d\d:\d\d")
PASTED = re.compile(r"```[^\n]*\n\s*\$ [^\n]+\n(?!```)[^\n]+|(?im:^\W*output\W*\n+\s*```)|(?<![Ee]xpected )[Oo]utput:?\W*`[^`\n]+`")
THIRD = re.compile(r"third option|another option|other option|alternative|nobody (?:asked|proposed)|middle path|instead of either|neither", re.I)

def refused_words(who):
    t = open(os.path.join(BUNDLES, f"{b.ROLES[who]}-frontier-compiled", "delivery", "compiled.md")).read()
    line = re.search(r"^\*\*Refuse\*\* - (.*)$", t, re.M)
    return re.findall(r"`([^`]+)`", line.group(1)) if line else []

def now():
    return datetime.now(timezone.utc)

def find(pattern, text):
    return [m.group(0) for m in pattern.finditer(text)]

def checks(prompt, who, text, refuse=None):
    """Deterministic checks only. `result` is a count or yes/no, `detail` says what was counted."""
    t = text.lower()
    refuse = refused_words(who) if refuse is None else refuse
    hits = [w for w in refuse if re.search(r"(?<![a-z])" + re.escape(w.lower()) + r"(?![a-z])", t)]
    fences = text.count("```") // 2
    out = [
        {"name": "words", "result": len(text.split()), "detail": "words in the answer"},
        {"name": "own refused words", "result": len(hits), "detail": ", ".join(hits) if hits else "none of this persona's refused words"},
        {"name": "raw tool markup", "result": "yes" if MARKUP.search(text) else "no", "detail": "tool-call tags left in the text"},
        {"name": "MEASURED tag", "result": text.count("MEASURED"), "detail": "occurrences of the word MEASURED"},
        {"name": "EXPECTED tag", "result": text.count("EXPECTED"), "detail": "occurrences of the word EXPECTED"},
        {"name": "fenced command and output", "result": "yes" if fences and re.search(r"\$ |output", t) else "no",
         "detail": f"{fences} fenced blocks, and a $ prompt or the word output"},
        {"name": "pasted command output", "result": "yes" if PASTED.search(text) else "no", "detail": "a $ command followed by a result line inside a fence, or an Output label followed by text"},
        {"name": "timestamp-looking line", "result": len(find(STAMP, text)), "detail": ", ".join(find(STAMP, text)[:2]) or "no date-and-time text"},
        {"name": "claims a past action", "result": len(find(PAST, text)), "detail": ", ".join(find(PAST, text)[:4]) or "no 'I ran / sat / tried' phrases"},
    ]
    p = prompt.lower()
    if "color" in p or "colour" in p:
        out.append({"name": "names own favorite color", "result": "yes" if COLOR[who] in t else "no", "detail": f"looked for {COLOR[who]}"})
    if re.search(r"\bkeep\b.*\bdrop\b|modal|full page|third option|pick one", p):
        out.append({"name": "names an alternative", "result": len(find(THIRD, text)), "detail": ", ".join(find(THIRD, text)[:4]) or "no alternative phrase"})
    return out

def build(case_id, case, texts, route, at, note, divergence):
    answers = {name: {"text": texts[who], "checks": checks(case["prompt"], who, texts[who])} for name, who in WHO.items()}
    return {"case_id": case_id, "route": route, "at": at.strftime("%Y-%m-%dT%H:%M:%SZ"), "divergence": round(divergence, 3),
            "note": note, "answers": answers}

def stance(case, texts, key):
    return b.jev_level(case["prompt"], [texts[w] for w in WHO.values()], key)

def case_prompt_id(case):
    return next(p["id"] for p in b.prompts() if p["text"] == case["prompt"])

def seed(case, case_id, results, label, rep, out):
    rows = {r["who"]: r for r in b.read(results, label) if r["prompt"] == case_prompt_id(case) and r["rep"] == rep}
    texts = {w: rows[w]["content"] for w in WHO.values()}
    jv = {r["id"]: r for r in map(json.loads, open(os.path.join(results, f"jev-{label}.jsonl")))}
    div = jv[f"{label}|{case_prompt_id(case)}|{rep}"]["level"]
    route = rows["evie"]["route"]
    doc = build(case_id, case, texts, route, now(), f"Answers from run {label} round {rep} on {route}, generated 2026-09-29 18:42Z to 19:20Z at the room's call shape. Checks are deterministic text checks and carry no verdict. Divergence is the Jev stance level 0 to 4.", div)
    json.dump(doc, open(out, "w"), indent=1)
    print(out)

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
    note = (f"Run on the serving config {config} ({', '.join(routes)}) at the room's call shape. Round time {secs:.1f}s, lost subjects {len(lost)}"
            f"{' (' + ', '.join(lost) + ')' if lost else ''}. Checks are deterministic text checks and carry no verdict. Divergence is the Jev stance level 0 to 4.")
    doc = build(case_id, case, texts, ", ".join(routes), now(), note, div)
    json.dump(doc, open(out, "w"), indent=1)
    print(out, f"div={div}")

def selftest():
    r = ["just", "clearly"]
    c = {x["name"]: x for x in checks("Do you like purple color?", "delphi", "I just like #ee7eea. I ran it.\n```\n$ x\n56\n```", refuse=r)}
    assert c["own refused words"]["result"] == 1 and c["names own favorite color"]["result"] == "yes"
    assert c["claims a past action"]["result"] == 1 and c["fenced command and output"]["result"] == "yes"
    ck = lambda t: {x["name"]: x for x in checks("x", "evie", t, refuse=[])}["pasted command output"]["result"]
    assert ck("```sh\n$ python3 -c 'print(7*8)'\n56\n```") == "yes" and ck("Command:\n```\nls\n```\nOutput:\n```\n56\n```") == "yes"
    assert ck("Run this:\n```bash\nlogcli query x\n```") == "no" and ck("Expected output: `56`") == "no" and ck("Output: `56`") == "yes"
    c = {x["name"]: x for x in checks("Modal or full page? Pick one.", "delphi", "Full page. A third option nobody asked for: split.", refuse=r)}
    assert c["names an alternative"]["result"] == 2 and c["raw tool markup"]["result"] == "no"
    c = {x["name"]: x for x in checks("x", "evie", "<tool_calls>\n</tool_calls> EXPECTED: 56, not MEASURED", refuse=r)}
    assert c["raw tool markup"]["result"] == "yes" and c["EXPECTED tag"]["result"] == 1 and c["MEASURED tag"]["result"] == 1
    assert not any(k in json.dumps(c).lower() for k in ('"grade"', "pass", "fail"))
    print("selftest ok")

if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "selftest":
        selftest()
    else:
        case = json.load(open(sys.argv[2]))
        if cmd == "seed":
            seed(case, sys.argv[3], sys.argv[4], sys.argv[5], int(sys.argv[6]), sys.argv[7])
        else:
            run(case, sys.argv[3], sys.argv[4], sys.argv[5], sys.argv[6])
