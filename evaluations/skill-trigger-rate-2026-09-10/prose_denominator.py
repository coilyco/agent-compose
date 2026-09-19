"""Trigger rate for the writing family against a pre-named prose denominator.

The advocate seat fixed the clause set before this ran and before any number
existed, and named the blind spot itself: none of the clauses sees prose that
lives only in chat, which is the case the complaint is actually about. Sessions
that never wrote prose to a file are the ones most likely to have skipped the
linter too, so every rate here is a CEILING on the real one.

Clauses, as given:
  1. Write or Edit with a file_path ending .md          (strictest, reported alone)
  2. an outward-channel call: Gmail create_draft, an Artifact publish,
     or a Forgejo/GitHub create_pull-request
  3. a git commit carrying a body rather than a bare subject line
"""

import collections
import json
import os
import pathlib
import re
import sys

TRANSCRIPTS = pathlib.Path(
    os.environ.get("CLAUDE_PROJECTS", pathlib.Path.home() / ".claude" / "projects")
)

# Clause 3. A body reaches git through stdin, a file, or a repeated -m. A single
# -m with no newline in it is a bare subject and does not count.
GIT_COMMIT = re.compile(r"git\s+commit\b")
HEREDOC = re.compile(r"<<-?\s*'?\"?[A-Za-z_][A-Za-z0-9_]*")
DASH_F = re.compile(r"\bgit\s+commit\b[^|;&]*?\s-[a-zA-Z]*F\b|\bgit\s+commit\b[^|;&]*?--file\b")
DASH_M = re.compile(r"\s-[a-zA-Z]*m\b|\s--message\b")


def commit_has_body(cmd: str) -> bool:
    if not GIT_COMMIT.search(cmd):
        return False
    seg = cmd[GIT_COMMIT.search(cmd).start():]
    if DASH_F.search(cmd) or HEREDOC.search(seg):
        return True
    if len(DASH_M.findall(seg)) >= 2:
        return True
    # a single -m whose quoted message spans a newline
    m = re.search(r"-m\s+(['\"])(.*?)\1", seg, re.S)
    return bool(m and "\n" in m.group(2))


def is_outward(name: str, inp: dict) -> bool:
    if name == "Artifact":
        return inp.get("action", "publish") == "publish"
    return bool(name) and (
        name.endswith("create_draft")
        or name.endswith("create_pull-request")
        or name.endswith("create_pull_request")
    )


def is_md_write(name: str, inp: dict) -> bool:
    if name not in ("Write", "Edit"):
        return False
    return str(inp.get("file_path", "")).lower().endswith(".md")


class Sess:
    __slots__ = ("c1", "c2", "c3", "skills", "label")

    def __init__(self):
        self.c1 = self.c2 = self.c3 = False
        self.skills = set()
        self.label = None


def parse(lines):
    s = Sess()
    for line in lines:
        try:
            d = json.loads(line)
        except Exception:
            continue
        if d.get("type") == "agent-name" and s.label is None:
            m = re.search(r"\(([^)]+)\)", d.get("agentName") or "")
            if m:
                s.label = m.group(1)
        msg = d.get("message") or {}
        content = msg.get("content")
        if not isinstance(content, list):
            continue
        for b in content:
            if not (isinstance(b, dict) and b.get("type") == "tool_use"):
                continue
            name, inp = b.get("name"), (b.get("input") or {})
            if is_md_write(name, inp):
                s.c1 = True
            if is_outward(name, inp):
                s.c2 = True
            if name == "Bash" and commit_has_body(str(inp.get("command", ""))):
                s.c3 = True
            if name == "Skill":
                slug = inp.get("skill")
                if slug:
                    s.skills.add(slug)
    return s


def _tu(name, inp):
    return json.dumps(
        {"type": "assistant", "message": {"content": [{"type": "tool_use", "name": name, "input": inp}]}}
    )


def known_answers():
    checks = [
        ("md Write is clause 1", parse([_tu("Write", {"file_path": "/a/b.md"})]).c1),
        ("py Write is not clause 1", not parse([_tu("Write", {"file_path": "/a/b.py"})]).c1),
        ("md Edit is clause 1", parse([_tu("Edit", {"file_path": "/a/B.MD"})]).c1),
        ("Artifact publish is clause 2", parse([_tu("Artifact", {"file_path": "x.html"})]).c2),
        ("Artifact read is not clause 2", not parse([_tu("Artifact", {"action": "read", "url": "u"})]).c2),
        ("gmail draft is clause 2", parse([_tu("mcp__claude_ai_Gmail__create_draft", {})]).c2),
        ("forgejo PR is clause 2", parse([_tu("mcp__tailnet_coilyco_forgejo__create_pull-request", {})]).c2),
        ("bare -m is not clause 3", not parse([_tu("Bash", {"command": 'git commit -m "subject only"'})]).c3),
        ("heredoc -F is clause 3", parse([_tu("Bash", {"command": "git commit -q -F - <<'EOF'\nsubj\n\nbody\nEOF"})]).c3),
        ("two -m is clause 3", parse([_tu("Bash", {"command": 'git commit -m "subj" -m "body"'})]).c3),
        ("multiline -m is clause 3", parse([_tu("Bash", {"command": 'git commit -m "subj\n\nbody"'})]).c3),
        ("git status is not clause 3", not parse([_tu("Bash", {"command": "git status --porcelain"})]).c3),
    ]
    for k, ok in checks:
        print(f"  {'ok  ' if ok else 'FAIL'}  {k}")
    if [k for k, ok in checks if not ok]:
        print("\nrefusing to report: known-answer check failed")
        sys.exit(1)


WRITING = lambda s: s.startswith("writing-")
VOICE_WIDE = lambda s: s.startswith("writing-") or "voice" in s or s.endswith("-linter")


def rate(n, d):
    return f"{n/d:.3f}" if d else "  n/a"


def report(title, sessions, all_sessions):
    n1 = sum(1 for s in sessions if any(WRITING(x) for x in s.skills))
    n2 = sum(1 for s in sessions if any(VOICE_WIDE(x) for x in s.skills))
    nk = sum(1 for s in sessions if "writing-kai-voice" in s.skills)
    na = sum(1 for s in sessions if s.skills)
    d = len(sessions)
    print(f"\n{title}")
    print(f"  sessions in denominator                {d:5d}   ({d/len(all_sessions):.1%} of corpus)")
    print(f"  loaded any writing-* skill             {n1:5d}   rate {rate(n1, d)}")
    print(f"  loaded any voice-family skill (wider)  {n2:5d}   rate {rate(n2, d)}")
    print(f"  loaded writing-kai-voice specifically  {nk:5d}   rate {rate(nk, d)}")
    print(f"  loaded ANY skill at all                {na:5d}   rate {rate(na, d)}")


def main():
    print("known-answer checks")
    known_answers()
    print()
    files = sorted(TRANSCRIPTS.rglob("*.jsonl"))
    S = [parse(f.read_text(errors="replace").splitlines()) for f in files]
    print(f"corpus stamp: {len(S)} transcripts")

    print("\nclause incidence, separately")
    for lab, sel in (("1  .md Write/Edit", lambda s: s.c1),
                     ("2  outward channel", lambda s: s.c2),
                     ("3  commit with body", lambda s: s.c3)):
        k = sum(1 for s in S if sel(s))
        print(f"  clause {lab:22} {k:5d}   {k/len(S):.1%} of corpus")

    report("DENOMINATOR A - clause 1 alone (strictest, advocate-preferred)",
           [s for s in S if s.c1], S)
    report("DENOMINATOR B - clauses 1 OR 2 OR 3 (union, as specified)",
           [s for s in S if s.c1 or s.c2 or s.c3], S)

    print("\nNEGATIVE CONTROL - sessions matching no clause")
    report("  (a prose skill here would mean the denominator misses prose work)",
           [s for s in S if not (s.c1 or s.c2 or s.c3)], S)

    print("\nby seat, denominator A")
    byl = collections.defaultdict(list)
    for s in S:
        if s.c1 and s.label:
            byl[s.label].append(s)
    for lab in sorted(byl, key=lambda x: -len(byl[x]))[:8]:
        sub = byl[lab]
        n = sum(1 for s in sub if any(WRITING(x) for x in s.skills))
        w = sum(1 for s in sub if any(VOICE_WIDE(x) for x in s.skills))
        print(f"  {lab:26} writing-* {n:3d}/{len(sub):<4} {rate(n,len(sub))}"
              f"   wider {w:3d}/{len(sub):<4} {rate(w,len(sub))}")


if __name__ == "__main__":
    main()
