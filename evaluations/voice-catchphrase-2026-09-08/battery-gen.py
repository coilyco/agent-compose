"""Generate the candidate profile, anchoring exactly as the shipped emit does."""
import json, pathlib, re, sys
sys.path.insert(0, str(__import__("pathlib").Path(__file__).parent))
from battery_terms import TERMS, RAW

GO_META = set("$()*+.?[\\]^{|}")

def quote_meta(v):
    return "".join("\\" + c if c in GO_META else c for c in v)

def is_word_char(c):
    return c == "_" or (c.isascii() and c.isalnum())

def pattern(term):
    fields = term.split()
    # A straight apostrophe never matches curly-quoted prose, and curly quotes
    # are themselves on the tell list, so every rule has to accept both.
    body = r"\s+".join(quote_meta(f).replace("'", "['\u2019]") for f in fields)
    collapsed = re.sub(r"\s+", " ", term)
    if is_word_char(collapsed[0]):
        body = r"\b" + body
    if is_word_char(collapsed[-1]):
        body += r"\b"
    return body

rules = []
for rid, term, fam, hint, src in TERMS:
    rules.append({"id": rid, "pattern": pattern(term), "hint": hint,
                  "flags": ["i"], "family": fam, "sources": src})
for rid, pat, fam, hint, src in RAW:
    rules.append({"id": rid, "pattern": pat, "hint": hint,
                  "flags": ["i"], "family": fam, "sources": src})

pathlib.Path(sys.argv[1]).write_text(
    json.dumps({"name": "llm-cliche-candidates", "rules": rules}, indent=2, ensure_ascii=False)
    + "\n"
)
print(f"wrote {len(rules)} rules")
