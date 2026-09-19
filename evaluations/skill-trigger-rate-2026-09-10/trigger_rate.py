"""Per-skill trigger rate from historical Claude Code transcripts.

Observational. It counts Skill invocations that already happened rather than
generating anything, so it costs no inference and needs no agentic runner.

The corpus is live: sessions write into it while this runs. Every reported
number is stamped with the corpus size that produced it.

Transcript bodies never leave this script. It emits skill slugs, role labels
and counts, and no session id, path, or message content.
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

# A role label maps to a slug only where a Skill(role-*) call was observed under
# that label. Unconfirmed labels are reported and excluded, never guessed.
ROLE_SLUG = {
    "Portfolio Director": "role-director",
    "Systems Administrator": "role-sysadmin",
    "Applied Scientist": "role-science",
    "Developer Advocate": "role-advocate",
    "Platform Engineer": "role-platform",
    "Frontend Engineer": "role-frontend",
    "Game Developer": "role-gamedev",
    "Engineer": "role-engineer",
    "DevOps": "role-ops",
    "Executive Strategist": "role-exec",
    "AI Engineer": "role-ai",
    "Director": "role-director",
    "Technical Program Manager": "role-tpm",
    "QA": "role-qa",
}


class Session:
    __slots__ = ("label", "skills", "skill_order", "n_tool", "first_publish")

    def __init__(self):
        self.label = None
        self.skills = set()
        self.skill_order = {}
        self.n_tool = 0
        self.first_publish = None


def parse(lines):
    """One transcript's lines to a Session. Tool ordinal is the clock."""
    s = Session()
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
            s.n_tool += 1
            name, inp = b.get("name"), (b.get("input") or {})
            if name == "Skill":
                slug = inp.get("skill")
                if slug:
                    s.skills.add(slug)
                    s.skill_order.setdefault(slug, s.n_tool)
            elif name == "Artifact":
                if inp.get("action", "publish") == "publish" and s.first_publish is None:
                    s.first_publish = s.n_tool
    return s


def _ev(kind, **kw):
    return json.dumps({"type": kind, **kw})


def _tu(name, inp):
    return json.dumps(
        {"type": "assistant", "message": {"content": [{"type": "tool_use", "name": name, "input": inp}]}}
    )


def known_answers():
    """Five checks. The module refuses to report if any fails."""
    checks = []

    a = parse([_ev("agent-name", agentName="Evie [she] (Applied Scientist) x1"),
               _tu("Skill", {"skill": "role-science"})])
    checks.append(("label parsed", a.label == "Applied Scientist"))
    checks.append(("skill parsed", a.skills == {"role-science"}))

    b = parse([_tu("Skill", {"skill": "artifact-design"}), _tu("Artifact", {"file_path": "x.html"})])
    checks.append(("skill precedes publish", b.skill_order["artifact-design"] < b.first_publish))

    c = parse([_tu("Artifact", {"file_path": "x.html"}), _tu("Skill", {"skill": "artifact-design"})])
    checks.append(("publish precedes skill", c.skill_order["artifact-design"] > c.first_publish))

    # A non-publish Artifact action must not enter the denominator.
    d = parse([_tu("Artifact", {"action": "read", "url": "u"})])
    checks.append(("read is not a publish", d.first_publish is None))

    bad = [k for k, ok in checks if not ok]
    for k, ok in checks:
        print(f"  {'ok  ' if ok else 'FAIL'}  {k}")
    if bad:
        print(f"\nrefusing to report: {len(bad)} known-answer check(s) failed")
        sys.exit(1)


def rate(n, d):
    return f"{n/d:.3f}" if d else "  n/a"


def main():
    print("known-answer checks")
    known_answers()
    print()

    files = sorted(TRANSCRIPTS.rglob("*.jsonl"))
    sessions = [parse(f.read_text(errors="replace").splitlines()) for f in files]

    n_tool_calls = sum(s.n_tool for s in sessions)
    n_skill_calls = sum(len(s.skills) for s in sessions)
    labelled = [s for s in sessions if s.label in ROLE_SLUG]

    print(f"corpus stamp")
    print(f"  transcripts scanned          {len(sessions)}")
    print(f"  tool calls                   {n_tool_calls}")
    print(f"  Skill invocations            {n_skill_calls}")
    print(f"  distinct skills invoked      {len({x for s in sessions for x in s.skills})}")
    print(f"  transcripts with any Skill   {sum(1 for s in sessions if s.skills)}")
    print(f"  transcripts with a role tag  {sum(1 for s in sessions if s.label)}")
    print()

    # A skill is invoked at most once per transcript. Verified, not assumed:
    # the per-skill call count equalled the per-skill transcript count in the
    # scouting run, which is why a per-session binary is the right unit.

    print("=" * 66)
    print("A. artifact-design - hard denominator, machine-checkable trigger")
    print("   rule: MUST load before writing the file Artifact publishes")
    print("=" * 66)
    den = [s for s in sessions if s.first_publish is not None]
    any_ = [s for s in den if "artifact-design" in s.skills]
    before = [s for s in any_ if s.skill_order["artifact-design"] < s.first_publish]
    print(f"  sessions publishing an Artifact      {len(den)}")
    print(f"  loaded artifact-design at all        {len(any_):4d}   rate {rate(len(any_), len(den))}")
    print(f"  loaded it BEFORE the first publish   {len(before):4d}   rate {rate(len(before), len(den))}")
    print()

    print("=" * 66)
    print("B. role skill - hard denominator, mandatory by the identity card")
    print("   rule: load the selected role skill before acting")
    print("=" * 66)
    den_r = collections.Counter()
    num_r = collections.Counter()
    anyskill_r = collections.Counter()
    for s in labelled:
        slug = ROLE_SLUG[s.label]
        den_r[slug] += 1
        if s.skills:
            anyskill_r[slug] += 1
        if slug in s.skills:
            num_r[slug] += 1
    print(f"  {'role slug':18} {'loaded':>6} {'sessions':>9} {'rate':>7}")
    for slug in sorted(den_r, key=lambda x: -den_r[x]):
        print(f"  {slug:18} {num_r[slug]:6d} {den_r[slug]:9d} {rate(num_r[slug], den_r[slug]):>7}")
    tn, td = sum(num_r.values()), sum(den_r.values())
    print(f"  {'POOLED':18} {tn:6d} {td:9d} {rate(tn, td):>7}")
    print()
    unmapped = collections.Counter(s.label for s in sessions if s.label and s.label not in ROLE_SLUG)
    print("  unmapped role labels, excluded rather than guessed:")
    for k, v in unmapped.most_common():
        print(f"    {v:4d}  {k}")
    print()

    print("=" * 66)
    print("C. negative controls")
    print("=" * 66)
    print("  C1. session size does not explain the role rate")
    print(f"      {'min tool calls':>14} {'loaded':>7} {'sessions':>9} {'rate':>7}")
    for floor_n in (0, 5, 10, 20, 50, 100):
        sub = [s for s in labelled if s.n_tool >= floor_n]
        n = sum(1 for s in sub if ROLE_SLUG[s.label] in s.skills)
        print(f"      {floor_n:>14} {n:7d} {len(sub):9d} {rate(n, len(sub)):>7}")
    print()
    sub = [s for s in labelled if s.skills]
    n = sum(1 for s in sub if ROLE_SLUG[s.label] in s.skills)
    print(f"  C2. among sessions that loaded ANY skill: {n}/{len(sub)} = {rate(n, len(sub))}")
    print()
    pub = den
    other = [s for s in sessions if s.first_publish is None]
    mp = sum(len(s.skills) for s in pub) / len(pub) if pub else 0
    mo = sum(len(s.skills) for s in other) / len(other) if other else 0
    print(f"  C3. artifact sessions load more skills generally: mean {mp:.2f} vs {mo:.2f}")
    print("      so a skill-specific effect must beat that inflation:")
    print(f"      {'probe':28} {'publish':>16} {'other':>16}")
    for probe in ("artifact-design", "artifact-capabilities", "role-science",
                  "role-director", "coding-core-git-workflow", "writing-kai-voice"):
        a = sum(1 for s in pub if probe in s.skills)
        b = sum(1 for s in other if probe in s.skills)
        print(f"      {probe:28} {a:5d}/{len(pub):<4} {rate(a,len(pub)):>5} "
              f"{b:5d}/{len(other):<4} {rate(b,len(other)):>5}")
    print()

    print("=" * 66)
    print("D. the writing family, which is the family under discussion")
    print("=" * 66)
    w = collections.Counter()
    for s in sessions:
        for slug in s.skills:
            if slug.startswith("writing-"):
                w[slug] += 1
    for k, v in w.most_common():
        print(f"    {v:4d}  {k}")
    print(f"    total writing-* invocations {sum(w.values())} across {len(sessions)} transcripts")
    print()
    for label in ("Developer Advocate", "Portfolio Director", "Content Creator"):
        sub = [s for s in sessions if s.label == label]
        if not sub:
            continue
        hit = sum(1 for s in sub if any(x.startswith("writing-") for x in s.skills))
        ld = sum(1 for s in sub if s.skills)
        print(f"    {label:20} any writing-* skill {hit:3d}/{len(sub):<4} = {rate(hit,len(sub))}"
              f"   (skill-loading sessions {ld})")
    print()

    print("=" * 66)
    print("E. full per-skill invocation counts")
    print("=" * 66)
    c = collections.Counter()
    for s in sessions:
        for slug in s.skills:
            c[slug] += 1
    for k, v in c.most_common():
        print(f"    {v:5d}  {k}")


if __name__ == "__main__":
    main()
