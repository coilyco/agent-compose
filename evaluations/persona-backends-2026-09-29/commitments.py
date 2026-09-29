import json, re, sys, glob, os, collections
B = "ac/dist/bundles"
ROLE = {"evie": "scientist", "delphi": "frontend-eng", "sprite": "game-dev", "gem": "dev-advocate"}
NAME = {"evie": "Frog-Ox", "delphi": "Imp-Dragonfly", "sprite": "Whale-Dragonfly", "gem": "Panda-Goose"}
refuse, color = {}, {}
for who, role in ROLE.items():
    t = open(f"{B}/{role}-frontier-compiled/delivery/compiled.md").read()
    refuse[who] = re.findall(r"`([^`]+)`", re.search(r"^\*\*Refuse\*\* - (.*)$", t, re.M).group(1))
    color[who] = re.search(r"Favorite color // `(#[0-9a-f]{6})`", t).group(1)
pat = {w: {term: re.compile(r"(?<![A-Za-z])" + re.escape(term) + r"(?![A-Za-z])", re.I) for term in terms} for w, terms in refuse.items()}
labels = sys.argv[1:]
print("colors:", color)
for lab in labels:
    rows = [json.loads(l) for l in open(f"results/answers-{lab}.jsonl")]
    print(f"\n=== {lab}")
    for who in ROLE:
        rs = [r for r in rows if r["who"] == who and r.get("content")]
        hit, terms = 0, collections.Counter()
        for r in rs:
            found = [t for t, p in pat[who].items() if p.search(r["content"])]
            hit += bool(found); terms.update(found)
        col = [r for r in rs if r["prompt"] == "deck-purple"]
        cm = sum(1 for r in col if color[who].lower() in r["content"].lower())
        print(f"  {who:6s} {NAME[who]:16s} refused-word answers {hit}/{len(rs)}  top {terms.most_common(4)}  | purple answer names own color {cm}/{len(col)}")
