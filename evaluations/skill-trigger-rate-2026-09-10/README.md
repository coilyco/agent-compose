# Per-skill trigger rate, measured from transcripts rather than estimated

Run `2026-09-10T19:29:57Z` at housecast `46b37e2`, against the live transcript
corpus at 882 transcripts and 145,686 tool calls. `trigger_rate.py` is the
whole instrument, `run.txt` is the run, and five known-answer checks gate the
report.

The advocate seat asked for a per-skill trigger rate so a prose refactor could
be evidence-led rather than a reshuffle with no before-measurement. This is that
rate for the two families where a denominator is machine-checkable.

**Amended after running.** [`AMENDMENT.md`](AMENDMENT.md) supplies the missing
denominator for the writing family, which this file could only report as a raw
count. The advocate seat fixed the clause set and named its blind spot before
any number existed. Against the strictest clause the family fires at 0.025,
while the same sessions load some other skill at 0.741.

## The number this was built on does not have a source

The request cited roughly 50% retrieval on the voice family, from the
description comment on `teable:coilyco-flight-deck/agentic-os#7151`. That
comment says it plainly:

    Measured behaviour of retrieval on the voice family is roughly 50% for
    the full set, and the failure is always under-triggering.

It names its evidence as `housecast:evaluations/voice-hook-readiness-2026-09-08/`.
That row measures something else. It applies 293 regex rules to 698 tracked
Markdown files and reports 1,830 findings. It contains no retrieval measurement,
no invocation count, and no denominator. Grepping it for a rate returns six
hits, all prose framing, none a number.

Across five repositories on disk, tracked files only, nothing matches a
retrieval or trigger rate either. The only two hits are the previous row saying
the instrument cannot measure one, and a script in `agentic-os-kai` that replays
a hook detector over transcripts to get a fire rate. That script is the shape of
the idea. Nobody had pointed it at skills.

**So 50% is EXPECTED, not MEASURED, and this row supersedes it.** It is not
close to the measured rate for either family below, and the direction of the
error is different for each. Recording this because the figure has already been
carried into a refactor plan as a premise.

## The instrument

Claude Code writes every session to `~/.claude/projects/**/*.jsonl`, and a
`Skill` invocation appears there as an ordinary `tool_use` block carrying the
slug. So a trigger rate is recoverable by replay, at no inference cost and with
no agentic runner. The previous row concluded that no load event was observable.
That was true of `evalkit` and false of the harness, and this row is the
correction.

Two structural facts make the unit clean.

**A skill loads at most once per transcript.** In the scouting run every
skill's call count equalled its distinct-transcript count, 199 of 199. So the
observation is a per-session binary rather than a count, which is what a rate
needs.

**A denominator exists wherever the trigger condition is machine-checkable.**
Two families qualify. `artifact-design` must load before an Artifact publish,
and the publish is a tool call. A role skill must load in a session assigned
that role, and the role is in the transcript's `agent-name` event.

Transcript bodies never leave the script. It emits slugs, role labels and
counts, and no session id, path, or message content.

## Result: the rate is not uniform, and the spread is the finding

    artifact-design, sessions publishing an Artifact       101/123   0.821
    role skill, pooled over 13 assigned roles              106/479   0.221
    any writing-* skill, Portfolio Director sessions         0/84    0.000

Both of the first two are mandatory rules with the word MUST or its equivalent
in always-loaded context. They differ by 0.60.

Per role, the mandatory role skill ranges from 0.029 to 0.750:

    role-tpm        6/8    0.750        role-frontend    9/45   0.200
    role-exec       7/12   0.583        role-science    10/58   0.172
    role-qa         1/2    0.500        role-platform    7/50   0.140
    role-ops       12/28   0.429        role-gamedev     1/23   0.043
    role-ai         4/11   0.364        role-engineer    1/34   0.029
    role-advocate  13/53   0.245
    role-sysadmin  14/60   0.233
    role-director  21/95   0.221

The three smallest cells are 2, 8 and 11 sessions and carry no weight. The
extremes that do carry weight are `role-engineer` at 1 of 34 and `role-director`
at 21 of 95.

**This answers the question that was actually asked.** The advocate seat needed
to know whether the misses are uniform or concentrated, because uniform means
rewrite descriptions and concentrated means move rules to another tier. They are
concentrated. A pooled family figure averages over a 0.00 to 0.82 range and
describes no skill in it.

## The writing family is not under-triggering, it is barely triggering

    writing-email                       12
    writing-kai-voice                    5
    writing-social-technical             2
    writing-system-improvement-vocab     1
    total                               20 invocations across 882 transcripts

`writing-kai-voice` has fired five times in the whole recorded history of the
corpus. The Portfolio Director, 84 sessions and the heaviest prose seat on the
board, has loaded a `writing-*` skill zero times, and 46 of those sessions
loaded some other skill. The Developer Advocate reaches 11 of 53.

A 50% figure and a 0.00 figure imply different work. At 50% the description is
nearly good enough and wants tuning. At 0.00 in the seat that writes most of the
prose, retrieval is not the mechanism at all.

## Negative controls, including the one that came out clean

**Session size does not explain the role rate.** Trivially short sessions
legitimately load nothing, so the pooled rate could have been an artifact of
including them. Conditioning on session size moves it by 0.03 and then flattens:

    min tool calls    0      5     10     20     50    100
    rate           0.221  0.245  0.248  0.254  0.249  0.247

That control was run to kill the finding and did not. Reported because a
measurement programme with no failed control is reporting its own selection.

**Conditioned on the agent loading any skill at all**, the role rate is
106/261 = 0.406. Even among sessions demonstrably in a skill-loading mode, the
mandatory skill is absent in three of five.

**The 0.821 is skill-specific, not a session-type effect.** Artifact-publishing
sessions do load more skills generally, mean 2.63 against 0.99, so a 2.7x
inflation had to be ruled out:

    probe                       publish        other
    artifact-design           101/123 0.821    0/759 0.000
    artifact-capabilities      29/123 0.236    1/759 0.001
    role-science                3/123 0.024    7/759 0.009
    role-director               5/123 0.041   17/759 0.022
    coding-core-git-workflow    0/123 0.000   18/759 0.024

The role skills barely move and `coding-core-git-workflow` moves down. The
inflation is real and far too small to produce the contrast.

## The confound I cannot separate, and it is the whole mechanism question

`artifact-design` and the role skills differ on two axes at once, and this
design moves both.

**Proximity to the action.** The `artifact-design` requirement sits in the
Artifact tool's own description, which the agent reads at the moment it reaches
for the tool. The role requirement sits in a session-start preamble, thousands
of tokens from any action.

**Whether an eager partial already satisfies the felt need.** The composed
identity card renders a summary of every role and personality skill, then says
the summary is not the operative text and to load each one. `artifact-design`
has no such summary anywhere in context, so an agent that does not load it has
nothing.

Both differ in the same direction, so the 0.60 contrast is consistent with
either and this row cannot apportion it. Naming it because the second axis is
exactly H3 from `evaluations/skill-discovery-2026-09-10`, arrived at
independently and from the opposite direction. That row hypothesised that an
eager partial suppresses retrieval of the complete body. Here the only family
with no eager partial is the only family above 0.80.

That is two independent observations pointing at one mechanism, and it is still
not a test of it. Separating the axes needs a role skill whose pointer is moved
to point of action with the summary left intact, against one with the summary
removed and the pointer left alone.

## What this does not measure

**Whether loading a skill changes the artifact.** That is E0 in
`evaluations/skill-discovery-2026-09-10`, pre-registered and unrun. A trigger
rate cannot license a refactor on its own: a fix that raises loading and does
not change compliance is the cosmetic outcome that row exists to prevent. The
0.00 for the writing family raises the value of E0 rather than replacing it,
because it establishes there is a real gap for E0's ceiling to be a ceiling on.

**Whether the missed loads were misses.** The denominator counts sessions where
the rule says load. It does not check that the task needed the skill. For
`artifact-design` the rule is unconditional, so the denominator is exact. For
role skills the identity card is also unconditional, so it is exact against the
written rule, and a reader who thinks the rule is too strong is disagreeing with
the rule rather than the measurement.

**Anything about tier three.** No hook fire rate is measured here.

## Runs I chose not to do

Two families have machine-checkable denominators and I stopped there. The
`mcp-tools-*` family is 26 skills with a plausible denominator in whether that
server's tools were called, and `claude-api` has an explicit trigger clause
matchable against user message text. Both are reachable with this instrument
and neither was needed to answer whether the rate is uniform.

I did not correct for the corpus being live, because it cannot be held still.
It grew by one transcript between the scouting run and this one, 881 to 882,
and by 21 tool calls between two runs 113 seconds apart, both of them this
session writing into the corpus it is measuring. That is why every table here
comes from a single stamped run rather than the readings that preceded it, and
why the stamp is on the row rather than in a commit message.

## Where I was wrong

My first sweep for `Skill` invocations grepped `'"name":"Skill"'` across the
corpus and returned 0, which I nearly reported as the instrument being
unavailable. The pattern was wrong, not the data. Had I stopped there this row
would have confirmed the previous row's conclusion that no load event is
observable, and that conclusion is what this row overturns.

A single empty query is not a negative result, and this is the second time on
this question that one nearly became a finding.
