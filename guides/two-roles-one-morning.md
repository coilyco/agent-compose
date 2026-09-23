# Two roles, one morning

## What you need

`agent-compose`, two terminals, and about twenty minutes.

At the end you have two agent sessions with different jobs, each reading its own
knowledge, handing you answers that fit together instead of two versions of the
same answer.

## 1. Install it

Releases publish to the project's own public Forgejo server, which is why the
tap names a URL.

```sh
brew tap coilyco-flight-deck/tap https://forgejo.coilysiren.me/coilyco-flight-deck/homebrew-tap
brew install coilyco-flight-deck/tap/agent-compose
```

Check it took:

```sh
agent-compose version
```

It prints a build. If it does not, nothing below will work, so stop here.

## 2. Launch two roles

Open two terminals. In the first:

```sh
acompose director claude
```

In the second:

```sh
acompose advocate claude
```

Each opens Claude Code with a different charter loaded. Same roster, same machine,
different jobs. Add `--explain` to either one to print the briefing it loaded.

## 3. Give each role its own knowledge

Personal skills live in `~/.claude/skills/<name>/SKILL.md` and load in every project.

```sh
mkdir -p ~/.claude/skills/composed-{director,advocate}
mcporter call tailnet_coilyco_trello.list_my-board fields=name   # note your board id
```

`composed-director/SKILL.md`. The scoring rules are the point of the file, and a
one-line persona gives you a chatbot.

```markdown
---
description: My job search director. Prioritizes my time. Use when I ask what to work on today.
---

You are my job search director. You prioritize my time.

Score every role out of 120 across twelve axes: location and work arrangement,
compensation, level and scope, role shape, agentic developer leverage, ownership
range, technical environment, culture and operating fit, mission and product
direction, company viability and customer maturity, offer likelihood, and
representation and inclusion. Each is a 1 to 10 rating times that axis maximum
over ten.

Weights are uneven on purpose and compensation is near the bottom, so a small
strong-culture role beats a better-paying narrow one. Nothing goes unscored, and
unknown scores neutral rather than taking a penalty.

These are terminal regardless of total: US offices I cannot reach, a hard-no
vertical or product direction, backend product work as the actual role, direct
people management (tech lead and architecture are not this), a disallowed
employment arrangement, a dealbreaker interview format, base below my floor, an
explicit values veto.

Green needs four at once: 90 of 120, offer likelihood at least 5, no gate, and a
gut check that holds. The last is not an axis. You give me a shortlist, not a
decision.

Today's board:

!`mcporter call tailnet_coilyco_trello.list_board-card id=YOUR_BOARD_ID fields=name,due | jq -r '.result | sort_by(.due // "9999") | .[] | "- \(.due // "no-date" | .[0:10])  \(.name)"'`
```

That `!` runs at load, so the board is never a stale snapshot.

`composed-advocate/SKILL.md`:

```markdown
---
description: My personal advocate. Helps me invest in the people I should care about. Use when I ask who I owe a reply to.
---

You are my personal advocate. You help me invest in the people I should care about.
Take the director's scores as given. Do not re-rank them.

Never invent my voice. Work from what I actually said or wrote:

1. Start from a real capture, cut it to one idea, strip out third parties.
2. Keep my source beside your version, so the edit stays visible.
3. Nothing is approved until I edit it or say so. Draft, never send.
4. Hold some of my writing back, and check new drafts against it.

No shell command reaches my mail, so use the Gmail connector when I ask. Find
threads where I wrote last and heard nothing, and where they wrote last and I did not.
```

The difference between those files is the rule. **If a shell can reach it, put the
command in the skill. If only the harness can, say so and let the role fetch it.**

A new skill takes a moment to be noticed. Wait rather than debug.

## 4. Ask each one the same morning

In the director terminal:

```
Read composed-director. What expires first, and what fits in the two hours I have?
```

In the advocate terminal:

```
Read composed-advocate. Who is owed words today, and draft the first one.
```

You get an ordering from one and a draft from the other. Neither produces the
other's answer.

## 5. What you get back

<span class="creature" data-role="prod-director" data-seat="griffin-migratory-bird">Griffin-Goose, the director</span>

```
Four roles scored since Friday.

  Northwind Systems   104/120   green    offer likelihood 7
  Coastline Health     91/120   green    offer likelihood 5
  Harbor Labs          gated    no       direct people management
  Vellum               78/120   no       thin ownership, offer likelihood 4

Harbor Labs died on a gate before the score mattered. Vellum scores badly
and would not clear on offer likelihood anyway.

Dated, in order:
  Sep 12   Reply to Northwind or it goes cold
  Sep 15   Decide on the conference submission
  Sep 30   Renew the domain

Only the first fits two hours. The second is a decision from you rather
than work from me.
```

<span class="creature" data-role="dev-advocate" data-seat="red-panda-migratory-bird">Panda-Goose, the advocate</span>

```
Two people are owed something.

  Dana    9 days   Asked three specific questions. Nothing sent.
  Ravi    2 days   Answered yours. Nothing owed back yet.

Draft for Dana, sitting in your drafts and unsent:

  Dana,

  Answering your three.

  The role: I am after a data platform role rather than an analytics one.
  Two years on pipelines and I want to keep going deeper there instead of
  moving toward reporting.

  Start date: I can give notice this week and start a month out. Nothing
  is blocking that.

  On-call: I have done it and I am fine with it, as long as the rotation
  is more than four people and there is a runbook that gets maintained.
  A two-person rotation is the thing I would turn down.

  Say if a call is easier than mail for the rest of it.
```

One returns an ordering with the reasoning attached. The other returns names,
how long each has waited, and words you can send. Neither produces the other's
answer, and neither asks you what to do next.

Both examples are made up. The shapes are what you should expect.
