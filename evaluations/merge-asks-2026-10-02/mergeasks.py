"""Why seats ask Kai to merge: runtime denial or doctrine miss, from transcripts.

  mergeasks.py extract PROJECTS_DIR HOURS OUT.jsonl   one structured row per session, no message text
  mergeasks.py classify OUT.jsonl                     aggregate counts, printed
  mergeasks.py selftest

Transcripts are private, so extract keeps only event kinds, timestamps, tool
names and PR coordinates, never message text. An ask is classed (a) when a merge
of the same session was denied before it, (b) when the session asked with no
merge attempt before it, and (c) when an attempt failed for another reason.
"""

from __future__ import annotations

import collections
import glob
import json
import os
import re
import sys
import time

DENIAL = "denied by the Claude Code auto mode classifier"
MERGE_CMD = re.compile(r"\b(pr merge|merge_pull|/pulls/\d+/merge|forgejo pr merge)\b")
ASK = re.compile(r"(want me to|should i|shall i|ok to|okay to|go ahead and|do you want|want to)\b[^.?\n]{0,80}\bmerge"
                 r"|\bmerge\b[^.\n]{0,60}\?", re.I)
KAI_MERGE = re.compile(r"\bmerge\b", re.I)
PR_OPEN = re.compile(r"\b(pr create|create_pull-request|forgejo pr create)\b")
ERR_KINDS = (("ci_not_green", r"status check|checks? (are )?(pending|failing|required)|not (yet )?green|ci"),
             ("not_mergeable", r"not mergeable|conflict|mergeable.: ?false|405"),
             ("auth_or_permission", r"401|403|permission|forbidden|unauthori"),
             ("not_found", r"404|not found"),
             ("tool_denied_other", r"denied|blocked"))


def err_kind(text: str) -> str:
    t = text.lower()
    return next((k for k, pat in ERR_KINDS if re.search(pat, t)), "other")


def blocks(msg: dict) -> list:
    c = msg.get("content")
    return c if isinstance(c, list) else ([{"type": "text", "text": c}] if isinstance(c, str) else [])


def is_human(text: str) -> bool:
    t = text.lstrip()
    return bool(t) and not t.startswith(("<", "[from ", "Another Claude session")) and "cross-session-message" not in t


def pr_of(tool: str, inp: dict) -> str | None:
    if "merge_pull" in tool:
        return f"{inp.get('owner')}/{inp.get('repo')}#{inp.get('index')}"
    m = re.search(r"repos/([\w.-]+/[\w.-]+)/pulls/(\d+)", json.dumps(inp))
    return f"{m.group(1)}#{m.group(2)}" if m else None


def session_events(path: str) -> list[dict]:
    events, pending = [], {}
    for line in open(path, encoding="utf-8", errors="ignore"):
        try:
            e = json.loads(line)
        except json.JSONDecodeError:
            continue
        msg, ts, role = e.get("message") or {}, e.get("timestamp"), e.get("type")
        for b in blocks(msg):
            kind = b.get("type")
            if role == "assistant" and kind == "tool_use":
                name, inp = b.get("name", ""), b.get("input") or {}
                if "create_pull-request" in name or (name == "Bash" and PR_OPEN.search(str(inp.get("command", "")))):
                    events.append({"ts": ts, "ev": "pr_open"})
                elif "merge_pull" in name or (name == "Bash" and MERGE_CMD.search(str(inp.get("command", "")))):
                    pending[b.get("id")] = len(events)
                    events.append({"ts": ts, "ev": "attempt", "pr": pr_of(name, inp), "result": None})
                elif name == "AskUserQuestion" and re.search(r"\bmerge", json.dumps(inp), re.I):
                    events.append({"ts": ts, "ev": "ask", "via": "question_tool"})
            elif role == "assistant" and kind == "text" and ASK.search(b.get("text", "")):
                events.append({"ts": ts, "ev": "ask", "via": "text"})
            elif role == "user" and kind == "tool_result" and b.get("tool_use_id") in pending:
                text = json.dumps(b.get("content"))
                i = pending.pop(b["tool_use_id"])
                events[i]["result"] = "denied" if DENIAL in text else ("error" if b.get("is_error") else "ok")
                if events[i]["result"] == "error":
                    events[i]["err"] = err_kind(text)
            elif role == "user" and kind == "text" and is_human(b.get("text", "")) and KAI_MERGE.search(b["text"]):
                events.append({"ts": ts, "ev": "human_merge_prompt", "short": len(b["text"]) <= 120})
    return events


def classify_asks(events: list[dict]) -> list[str]:
    out = []
    for i, e in enumerate(events):
        if e["ev"] != "ask":
            continue
        prior = [x for x in events[:i] if x["ev"] == "attempt"]
        if any(x["result"] == "denied" for x in prior):
            out.append("a_denied_first")
        elif prior and all(x["result"] == "ok" for x in prior):
            out.append("b_asked_after_merging_ok")
        elif not prior:
            out.append("b_asked_without_trying")
        else:
            out.append("c_other_failure_first")
    return out


def classify_prompts(events: list[dict]) -> list[str]:
    """What came right before each human 'merge' message in the same session."""
    out = []
    for i, e in enumerate(events):
        if e["ev"] != "human_merge_prompt":
            continue
        prev = [x for x in events[:i] if x["ev"] in ("ask", "attempt", "pr_open")]
        last = prev[-1] if prev else None
        if last is None:
            out.append("no_prior_pr_event")
        elif last["ev"] == "ask":
            out.append("after_seat_ask")
        elif last["ev"] == "attempt" and last["result"] == "denied":
            out.append("after_denial")
        elif last["ev"] == "attempt" and last["result"] != "ok":
            out.append("after_failed_attempt")
        elif last["ev"] == "pr_open":
            out.append("after_pr_open_no_merge_try")
        else:
            out.append("after_successful_merge")
    return out


def extract(projects: str, hours: str, out: str) -> None:
    cutoff = time.time() - float(hours) * 3600
    with open(out, "w") as fh:
        for path in glob.glob(os.path.join(projects, "**", "*.jsonl"), recursive=True):
            if os.path.getmtime(path) < cutoff:
                continue
            ev = session_events(path)
            if ev:
                fh.write(json.dumps({"session": os.path.basename(path)[:8], "events": ev,
                                     "asks": classify_asks(ev), "prompts": classify_prompts(ev)}) + "\n")


def classify(path: str) -> None:
    rows = [json.loads(l) for l in open(path)]
    asks = collections.Counter(a for r in rows for a in r["asks"])
    att = [e for r in rows for e in r["events"] if e["ev"] == "attempt"]
    res = collections.Counter(e["result"] for e in att)
    by_repo = collections.defaultdict(collections.Counter)
    for e in att:
        by_repo[(e["pr"] or "?#").split("#")[0]][e["result"]] += 1
    prompts = sum(e["ev"] == "human_merge_prompt" for r in rows for e in r["events"])
    print(json.dumps({"sessions_with_events": len(rows), "merge_attempts": len(att), "results": dict(res),
                      "asks": dict(asks), "asks_total": sum(asks.values()), "human_merge_prompts": prompts}))
    errs = collections.Counter(e.get("err") for e in att if e["result"] == "error")
    pr = collections.Counter(p for r in rows for p in r.get("prompts", []))
    short = sum(e.get("short", False) for r in rows for e in r["events"] if e["ev"] == "human_merge_prompt")
    opens = sum(e["ev"] == "pr_open" for r in rows for e in r["events"])
    print(json.dumps({"error_kinds": dict(errs), "pr_opens": opens, "human_prompt_context": dict(pr), "human_prompts_short": short}))
    print(json.dumps({"attempts_by_repo": {k: dict(v) for k, v in sorted(by_repo.items())}}))


def selftest() -> None:
    ev = [{"ev": "attempt", "result": "denied"}, {"ev": "ask"}]
    assert classify_asks(ev) == ["a_denied_first"]
    assert classify_asks([{"ev": "ask"}]) == ["b_asked_without_trying"]
    assert classify_asks([{"ev": "attempt", "result": "ok"}, {"ev": "ask"}]) == ["b_asked_after_merging_ok"]
    assert classify_asks([{"ev": "attempt", "result": "error"}, {"ev": "ask"}]) == ["c_other_failure_first"]
    assert ASK.search("Do you want me to merge PR 12 once it's green?")
    assert ASK.search("Should I merge it?") and not ASK.search("Merged PR 12 after CI passed.")
    assert MERGE_CMD.search("gh pr merge 12 --squash") and MERGE_CMD.search("aosguard ops forgejo pr merge 3")
    assert pr_of("mcp__tailnet_coilyco_forgejo__merge_pull-request", {"owner": "o", "repo": "r", "index": "5"}) == "o/r#5"
    assert is_human("merge it") and not is_human("<system-reminder>merge") and not is_human("[from x] merge")
    assert classify_prompts([{"ev": "pr_open"}, {"ev": "human_merge_prompt"}]) == ["after_pr_open_no_merge_try"]
    assert classify_prompts([{"ev": "attempt", "result": "denied"}, {"ev": "human_merge_prompt"}]) == ["after_denial"]
    assert classify_prompts([{"ev": "ask"}, {"ev": "human_merge_prompt"}]) == ["after_seat_ask"]
    assert err_kind("405 Method Not Allowed: not mergeable") == "not_mergeable" and err_kind("401 Unauthorized") == "auth_or_permission"
    print("selftest ok: 13 checks")


if __name__ == "__main__":
    cmd, args = sys.argv[1], sys.argv[2:]
    {"extract": extract, "classify": classify, "selftest": selftest}[cmd](*args)
