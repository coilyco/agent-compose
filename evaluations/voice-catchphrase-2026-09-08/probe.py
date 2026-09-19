#!/usr/bin/env python3
"""Measure candidate voice rules against three labelled arms.

Arm A  authentic Kai, voice-corpus `source.text` where `authored_by_kai` is true
Arm B  Kai-approved LinkedIn transformations from the same corpus
Arm C  agent-written prose, tracked Markdown under the repository roots given

A rule firing in A is a false positive against Kai's own writing. A rule firing
in C and not in A is catching agent slop, which is the reason to ship it.

Arm C needs the self-reference adjustment. A doctrine file that bans a phrase
quotes that phrase, so `writing-kai-voice` trips every rail it defines. Those
hits are an artifact of measuring the measurer, and the adjusted count is the
one a rollout should be sized against.

    python3 probe.py rules.json <voice-corpus-root> [fleet-root ...]
"""

import collections
import glob
import json
import os
import re
import sys

# Paths whose hits are the rule's own definition rather than a usage. A shipped
# linter needs the same exemption or its loudest finding is the style guide.
SELF_REFERENCE = (
    "composed/writing-kai-voice/COMPOSED.md",
    "composed/writing-voice-guide-linter/COMPOSED.md",
    "composed/kai-voice-guide-linter/COMPOSED.md",
    "evaluations/voice-catchphrase-2026-09-08/",
)

SKIP_DIRS = {".git", "node_modules", "__pycache__", ".venv", "data"}


def load_rules(path):
    rules = json.load(open(path))["rules"]
    return [(r["id"], r.get("src", ""), re.compile(r["pattern"], re.I)) for r in rules]


def corpus_arms(root):
    authentic, approved = [], []
    for path in sorted(glob.glob(os.path.join(root, "data/curated/*.jsonl"))):
        for line in open(path, encoding="utf-8"):
            line = line.strip()
            if not line:
                continue
            rec = json.loads(line)
            name = f"{os.path.basename(path)}:{rec.get('id')}"
            source = rec.get("source") or {}
            if source.get("text") and source.get("authored_by_kai") is True:
                authentic.append((name, source["text"]))
            trans = rec.get("transformation") or {}
            if trans.get("post_text") and trans.get("status") in ("approved", "published"):
                approved.append((name, trans["post_text"]))
    return authentic, approved


def fleet_arm(roots):
    docs = []
    for root in roots:
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            for filename in filenames:
                if not filename.endswith(".md"):
                    continue
                path = os.path.join(dirpath, filename)
                try:
                    docs.append((path, open(path, encoding="utf-8").read()))
                except (UnicodeDecodeError, OSError):
                    continue
    return docs


def is_self_reference(name):
    return any(marker in name for marker in SELF_REFERENCE)


def measure(docs, rules, label, adjust):
    hits = collections.Counter()
    excluded = collections.Counter()
    docs_hit = collections.Counter()
    examples = collections.defaultdict(list)
    for name, text in docs:
        skip = adjust and is_self_reference(name)
        for rule_id, _, pattern in rules:
            found = pattern.findall(text)
            if not found:
                continue
            if skip:
                excluded[rule_id] += len(found)
                continue
            hits[rule_id] += len(found)
            docs_hit[rule_id] += 1
            if len(examples[rule_id]) < 3:
                examples[rule_id].append((name, pattern.search(text).group(0)[:60]))
    return {
        "label": label,
        "n_docs": len(docs),
        "chars": sum(len(t) for _, t in docs),
        "hits": hits,
        "excluded": excluded,
        "docs_hit": docs_hit,
        "examples": examples,
    }


def report(result, rules):
    print(f"\n=== {result['label']}  n_docs={result['n_docs']}  chars={result['chars']}")
    fired = 0
    for rule_id, src, _ in rules:
        count = result["hits"][rule_id]
        if not count:
            continue
        fired += 1
        rate = count / (result["chars"] / 10000) if result["chars"] else 0.0
        sample = result["examples"][rule_id][0][1]
        dropped = result["excluded"][rule_id]
        note = f"  (+{dropped} self-ref excluded)" if dropped else ""
        print(
            f"  {rule_id:18s} hits={count:<5d} docs={result['docs_hit'][rule_id]:<4d}"
            f" per10k={rate:6.2f}  [{src}]  e.g. {sample!r}{note}"
        )
    print(f"  {fired} of {len(rules)} rules fired. The rest reported zero.")


def main():
    rules = load_rules(sys.argv[1])
    authentic, approved = corpus_arms(sys.argv[2])
    results = [
        measure(authentic, rules, "A authentic-kai", adjust=False),
        measure(approved, rules, "B kai-approved", adjust=False),
    ]
    if sys.argv[3:]:
        results.append(measure(fleet_arm(sys.argv[3:]), rules, "C agent-written", adjust=True))
    for result in results:
        report(result, rules)


if __name__ == "__main__":
    main()
