# Amendment: the writing family against a denominator, and it is a ceiling

Run `2026-09-10T19:34Z` at housecast `98b310f`, same 882-transcript corpus as
`README.md`. `prose_denominator.py` is the instrument, `prose_run.txt` is the
run, twelve known-answer checks gate it.

`README.md` reported the writing family as a raw count, 20 invocations across
the corpus, because no denominator existed for it. A count is not a rate. This
file supplies the rate.

## The clause set was fixed before the numbers existed

The advocate seat named the trigger condition in advance, unprompted by any
result, and named its blind spot in the same message rather than after seeing
what it produced. Recorded because it is the thing that makes this a
measurement rather than a search for a flattering denominator.

    1. Write or Edit with a file_path ending .md
    2. an outward-channel call: Gmail create_draft, an Artifact publish,
       or a Forgejo or GitHub create_pull-request
    3. a git commit carrying a body rather than a bare subject line

Clause 1 is load-bearing by their reasoning: the house guardrail requires
outward text to be written to a file and linted as a file before it ships, so a
compliant session leaves a `.md` write behind by construction. Clause 3 catches
commit and pull-request prose, which house style governs and which never
becomes a `.md` file.

**Every rate below is a ceiling, and the seat that named the clauses said so
first.** None of the three sees prose that exists only in chat, which is the
case the original complaint is about. The sessions that never wrote prose to a
file are the sessions most likely to have skipped the linter too, so the bias
runs toward overstating compliance. A ceiling is still worth having when it is
this low.

## Result

Clause incidence, separately, over 882 transcripts:

    clause 1  .md Write/Edit        162   18.4% of corpus
    clause 2  outward channel       207   23.5%
    clause 3  commit with body      388   44.0%

**Denominator A, clause 1 alone**, the strictest and the one the advocate seat
preferred. 162 sessions:

    loaded any writing-* skill                 4   0.025
    loaded any voice-family skill (wider)      5   0.031
    loaded writing-kai-voice specifically      3   0.019
    loaded ANY skill at all                  120   0.741

**Denominator B, clauses 1 or 2 or 3**, 440 sessions:

    loaded any writing-* skill                12   0.027
    loaded any voice-family skill (wider)     15   0.034
    loaded writing-kai-voice specifically      4   0.009
    loaded ANY skill at all                  281   0.639

The two denominators differ by 2.7x in size and the rate moves by 0.002. That
stability is the useful part: the answer does not depend on which of the two
clause sets is chosen, so the choice between them stops being worth arguing.

The wider voice family is reported because two voice skills sit outside the
`writing-` prefix, `kai-linkedin-voice` and `kai-voice-guide-linter`. Widening
the numerator can only raise the rate and it raises it by 0.006.

## The control that separates the two available explanations

A rate of 0.025 has two readings. Either these sessions do not retrieve skills,
or they retrieve skills and not these ones.

**They retrieve skills.** In denominator A, 120 of 162 sessions loaded at least
one skill, 0.741. The same sessions that reach for a skill three times in four
reach for a writing skill one time in forty.

The negative control runs the other way and confirms the denominator is doing
work. Among the 442 sessions matching no clause, any-skill loading is 0.170
against 0.741 inside. So the proxy is selecting substantive sessions rather
than slicing arbitrarily.

## By seat, denominator A

    Engineer                   writing-*   0/18   0.000
    Platform Engineer          writing-*   0/14   0.000
    DevOps                     writing-*   0/12   0.000
    Developer Advocate         writing-*   2/11   0.182
    Frontend Engineer          writing-*   1/11   0.091
    Applied Scientist          writing-*   0/9    0.000
    Portfolio Director         writing-*   0/7    0.000
    Systems Administrator      writing-*   0/6    0.000

Six of the eight busiest seats wrote Markdown and loaded a writing skill zero
times. The advocate seat, whose lane this is, reaches 0.182.

## What this does and does not settle

It settles the shape of the problem for the writing family. At 0.025 against a
ceiling, with 0.741 any-skill loading in the same sessions, a description
rewrite is not the intervention. The family is not under-triggering at the
margin, it is outside the retrieval path.

It does not settle that moving the rules raises compliance. That is still E0 in
`evaluations/skill-discovery-2026-09-10`, pre-registered and unrun, and this
amendment does not license a refactor any more than `README.md` did.

## The measurement I did not make

I did not measure prose that exists only in chat, which is the case the
complaint names, because no clause reaches it. Whether a chat-only reply
complied with house style is gradeable, by the same judge E0 needs and against
the same calibration set. It is not gradeable by replay, and nothing in this
row's method extends to it.
