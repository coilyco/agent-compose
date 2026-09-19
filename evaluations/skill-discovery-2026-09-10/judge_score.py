"""Turn judge verdicts into margins, and margins into the board they imply.

No inference here, deliberately. This is the arithmetic between a judge saying
yes or no and a number anyone would act on, and separating it from the model
call means it was tested before any model produced a verdict.

`wilson` is imported from calibration_size rather than rewritten: a second
implementation of the same interval is the duplication this row keeps catching.
"""

import json
import os
import pathlib
import re
import sys

from calibration_size import wilson

SET = pathlib.Path(__file__).parent / "calibration.jsonl"
PROMPT = pathlib.Path(__file__).parent / "judge_prompt.txt"

# The linter this rail is graded against. HOME is the session shadow rather
# than the operator's home under a native launch, so candidates are searched
# and VOICE_PROFILE overrides. A miss reports that it could not run.
_RELATIVE = "coilyco-bridge/voice-corpus/.agents/skills/coilyco-voice-guide-linter/profile.json"


def _profile():
    override = os.environ.get("VOICE_PROFILE")
    if override:
        return pathlib.Path(override)
    roots = [pathlib.Path.home() / "projects", pathlib.Path("/Users/kai/projects")]
    if os.environ.get("AOS_NATIVE_SESSION_PROJECTS"):
        roots.insert(0, pathlib.Path(os.environ["AOS_NATIVE_SESSION_PROJECTS"]))
    for root in roots:
        candidate = root / _RELATIVE
        if candidate.exists():
            return candidate
    return roots[0] / _RELATIVE


PROFILE = _profile()


def load_set(path=SET):
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def margins(items, verdicts):
    """Sensitivity and specificity with their intervals.

    verdicts maps item id to True when the judge says the text performs the
    move. An item with no verdict is an abstention and is excluded from its
    margin rather than counted as wrong, because a judge that declines is a
    different failure from a judge that is mistaken, and pooling them hides it.
    """
    tp = fn = tn = fp = 0
    skipped = []
    for item in items:
        call = verdicts.get(item["id"])
        if call is None:
            skipped.append(item["id"])
            continue
        positive = item["label"] == "positive"
        if positive and call:
            tp += 1
        elif positive and not call:
            fn += 1
        elif not positive and call:
            fp += 1
        else:
            tn += 1
    se = tp / (tp + fn) if tp + fn else None
    sp = tn / (tn + fp) if tn + fp else None
    return {
        "tp": tp, "fn": fn, "tn": tn, "fp": fp,
        "sensitivity": se,
        "specificity": sp,
        "se_interval": wilson(tp, tp + fn) if tp + fn else None,
        "sp_interval": wilson(tn, tn + fp) if tn + fp else None,
        "abstained": skipped,
    }


def youden(se, sp):
    """se + sp - 1. Zero means the judge carries no information."""
    if se is None or sp is None:
        return None
    return se + sp - 1


def board_multiplier(j):
    """Cost in n, relative to a perfect grader, from attenuation.py's 1/J^2."""
    if j is None or j <= 0:
        return None
    return 1.0 / (j * j)


def report(items, verdicts, stream=sys.stdout):
    m = margins(items, verdicts)
    j = youden(m["sensitivity"], m["specificity"])
    mult = board_multiplier(j)
    w = lambda s: print(s, file=stream)
    w(f"  graded            {m['tp'] + m['fn'] + m['tn'] + m['fp']} of {len(items)}")
    if m["abstained"]:
        w(f"  abstained         {len(m['abstained'])}  {m['abstained'][:6]}")
    w(f"  tp {m['tp']}  fn {m['fn']}  tn {m['tn']}  fp {m['fp']}")
    for name, key, interval in (
        ("sensitivity", "sensitivity", "se_interval"),
        ("specificity", "specificity", "sp_interval"),
    ):
        value, ci = m[key], m[interval]
        if value is None:
            w(f"  {name:16} n/a")
        else:
            w(f"  {name:16} {value:.3f}   95% ({ci[0]:.3f}, {ci[1]:.3f})")
    if j is None:
        w("  Youden J         n/a")
    elif mult is None:
        w(f"  Youden J         {j:.3f}   carries no information, no board rescues it")
    else:
        w(f"  Youden J         {j:.3f}   board cost {mult:.2f}x a perfect grader")
    return m, j, mult


def _simulated(items, se, sp):
    """A judge with exactly these margins, deterministic and order-stable.

    Wrong answers are taken from the front of each class, so a requested rate
    is met exactly rather than approached, and a test can assert equality.
    """
    verdicts = {}
    for label, rate in (("positive", se), ("negative", sp)):
        group = [i for i in items if i["label"] == label]
        correct = round(rate * len(group))
        for index, item in enumerate(group):
            right = index >= len(group) - correct
            verdicts[item["id"]] = right if label == "positive" else not right
    return verdicts


def social_patterns():
    """The rail's own family, or None when the profile is not reachable."""
    if not PROFILE.exists():
        return None

    def walk(node, family=None):
        if isinstance(node, dict):
            here = node.get("family", family)
            if "pattern" in node:
                yield here, node["pattern"]
            for value in node.values():
                yield from walk(value, here)
        elif isinstance(node, list):
            for value in node:
                yield from walk(value, family)

    return [p for f, p in walk(json.loads(PROFILE.read_text())) if f == "social"]


def contamination(items):
    """The prompt must not hand the judge the strings it is grading for.

    Showing a model the phrasings and then grading those phrasings measures
    string avoidance, which is the defect that killed the pre-registered regex
    primary on this row.
    """
    prompt = PROMPT.read_text()
    reused = [i["id"] for i in items if i["text"].lower()[:40] in prompt.lower()]
    patterns = social_patterns()
    if patterns is None:
        print("  SKIP  rail profile not on this host, phrasing check did not run")
        matched = []
    else:
        matched = [p for p in patterns if re.search(p, prompt, re.I)]
        print(f"  {'ok  ' if not matched else 'FAIL'}  judge prompt reuses none of "
              f"the rail's {len(patterns)} social phrasings")
    print(f"  {'ok  ' if not reused else 'FAIL'}  judge prompt contains no calibration item text")
    if matched or reused:
        sys.exit("refusing to report: the judge prompt is contaminated")


def known_answers(items):
    checks = []

    perfect = _simulated(items, 1.0, 1.0)
    m = margins(items, perfect)
    checks.append(("perfect judge scores 1.0 on both margins",
                   m["sensitivity"] == 1.0 and m["specificity"] == 1.0))
    checks.append(("perfect judge has J = 1", youden(1.0, 1.0) == 1.0))

    inverted = {k: not v for k, v in perfect.items()}
    m = margins(items, inverted)
    checks.append(("inverted judge scores 0 on both margins",
                   m["sensitivity"] == 0.0 and m["specificity"] == 0.0))
    checks.append(("inverted judge has J = -1", youden(0.0, 0.0) == -1.0))

    m = margins(items, _simulated(items, 0.5, 0.5))
    checks.append(("coin-flip judge has J = 0", abs(youden(m["sensitivity"], m["specificity"])) < 1e-9))
    checks.append(("a J of 0 has no finite board", board_multiplier(0.0) is None))

    # The identity the whole design rests on, recovered rather than assumed.
    m = margins(items, _simulated(items, 0.9, 0.9))
    checks.append(("margins recover a simulated 0.90 / 0.90",
                   abs(m["sensitivity"] - 0.9) < 1e-9 and abs(m["specificity"] - 0.9) < 1e-9))
    checks.append(("J 0.80 costs 1.5625x", abs(board_multiplier(0.8) - 1.5625) < 1e-9))

    partial = dict(list(perfect.items())[:40])
    m = margins(items, partial)
    checks.append(("an abstention is excluded rather than counted wrong",
                   len(m["abstained"]) == 40 and m["tp"] + m["fn"] + m["tn"] + m["fp"] == 40))

    for label, ok in checks:
        print(f"  {'ok  ' if ok else 'FAIL'}  {label}")
    if not all(ok for _, ok in checks):
        sys.exit("refusing to report: known-answer check failed")


def main():
    items = load_set()
    print(f"calibration set: {len(items)} items, "
          f"{sum(1 for i in items if i['label'] == 'positive')} positive, "
          f"{sum(1 for i in items if i['label'] == 'negative')} negative\n")
    print("contamination checks")
    contamination(items)
    print()
    print("known-answer checks")
    known_answers(items)
    print()
    print("what this set can resolve, at judge margins it might actually hit")
    for rate in (1.0, 0.95, 0.90, 0.85, 0.80, 0.75):
        print(f"\njudge at {rate:.2f} on both margins")
        report(items, _simulated(items, rate, rate))
    print()
    print("No judge has run. Every number above is a simulated judge, present so")
    print("the scoring was tested before a model produced a verdict.")


if __name__ == "__main__":
    main()
