# Why a skill that was in the bundle did not reach the artifact: the pre-registration

Written 2026-09-10 for `teable:coilyco-flight-deck/agentic-os#7325`, at housecast
`3c3a2e4`, agentic-os `f5a531dc`, agentic-os-kai `b3eda1a7`, agent-compose
`4677dc2`, voice-corpus `a6504f3`. Nothing here is a result. No generation has
run, and the design below is fixed before one does.

The advocate seat ghostwrote three recruiter drafts and two of them broke a
fixed rail in `writing-coilyco-voice`. That seat handed over two hypotheses and
asked which one carries the effect, because they have different fixes in
different repos and picking wrong costs a cycle while the failure stays live.

**Amended before running.** [`AMENDMENT.md`](AMENDMENT.md) carries what the three
real drafts changed. The primary outcome below, the held-out regex pair, is dead:
one of the three real violations is reachable by no pattern, so the pair measures
string avoidance rather than generalisation. Read the amendment before acting on
the grading section.

## What I checked before designing anything

The seat that reported this said plainly that it is the subject as well as the
reporter, and asked to be treated as a source rather than a finding. So every
claim below names the file it came from.

**The skill was present, eager, and unloaded.**
`agentic-os-kai:docs/context-budget-advocate-current.yaml:215` reads
`voice/writing-coilyco-voice: {class: ordinary, eager: 55, lazy: 789,
resources: 1}`. Its frontmatter was in the advocate system prompt at 55 tokens.
This was not an install gap or a promotion gap, which is the cheapest
explanation and had to be eliminated first.

**Its description already carries the task word.** Verbatim from
`voice-corpus:.agents/skills/writing-coilyco-voice/SKILL.md:3`:

    Write or edit in coilyco's established house style. Use for ghostwriting,
    substantive rewrites, register adaptation, public prose, messages, docs, or
    text that sounds generically assistant-written.

The seat was ghostwriting. `ghostwriting` is the fourth word of the use clause.
So H1 cannot be that there was no vocabulary to match on. Whatever H1 is about,
it is the `Triggers - ...` header convention rather than the keywords, and those
are different fixes.

**The family splits across two classes, and the reported count pooled them.**
In the advocate bundle, 5 writing skills are `role-composed` and 5 are
`ordinary`. Composed skills are promoted after role selection rather than
discovered, which is exactly why `role-`, `boundary-` and `personal-` were
correctly excluded from the original comparison. Half the writing family belongs
in that excluded group.

Recomputed by class, over the repos on disk and by parsing frontmatter rather
than by counting names:

    ordinary        60/76  = 0.79 carry a Triggers header
    role-composed   34/99  = 0.34

**The reported observation survives the correction.** Among the 5 ordinary
writing skills the advocate bundle actually resolves, 1 carries a Triggers
header, against an ordinary-class base rate of 0.79. The family is a real
outlier in the class where the convention holds. The corrected number is 1 of 5
against 0.79, not 2 of 9 against a pooled rate.

**H2 is confirmed verbatim.** `agent-compose:seed/roster/data/role-advocate/`
holds `role.yaml` and `SKILL.md`. Neither names a `writing-` skill, house style,
voice guidance, or ghostwriting. The one grep hit is a `voice:` key, which is
the next finding rather than a counterexample.

## H3, which neither hypothesis names

`role-advocate/role.yaml` ships an inline `voice:` block: `prefer` and `avoid`
word lists, a cadence line, a person line, a tell. It is delivered eagerly in
the role instructions on every session. The identity card in the composed system
prompt renders the same block.

That block carries word-level preferences and no behavioural rails. The rail
that broke lives only in the lazy skill body, at
`writing-coilyco-voice/SKILL.md:15-19`. So the seat always holds voice guidance
that is complete-looking and incomplete in exactly the dimension that failed.

This predicts the reported experience precisely. The seat answered the question
correctly from first principles without knowing a written rule existed, which is
what having partial guidance and no prompt to seek more feels like from inside.
An eager partial that satisfies the felt need is a different mechanism from a
lazy complete that is hard to find, and it has a third fix.

H3 is a hypothesis. It is stated here so it is falsifiable rather than
discovered later as an explanation for whatever the numbers do.

## The instrument cannot measure a load rate today

`evalkit/task.py:48` is the whole solver:

    solver=[system_message("{system_prompt}"), generate()]

One system message, one generation, no tools. `grep -rniE 'skill' evalkit/*.py`
returns nothing across 941 lines. There is no Skill tool in the loop, so there
is no load event to observe and no transcript to count invocations in. Skills
reach the run only as frontmatter already baked into the composed system prompt.

Measuring "how often does a matching skill load" needs an agentic runner with a
tool surface and invocation logging. That is a build, it is inside this seat's
scope, and it is not free. Ordering it before knowing whether loading matters is
the mistake this row exists to avoid.

## The outcome is compliance, not loading

The question was framed as a load rate. The failure is a rail breach. Those come
apart, and the reporter's own account is the demonstration: three drafts, two
breaches, and the clean one complied without the skill ever loading.

A design that measures loading alone would recommend whatever raises loading,
including when compliance does not follow. That is the cosmetic outcome the
reporter was right to worry about, arriving through the measurement rather than
around it.

So compliance is primary and loading is the mechanism variable, measured second
and only if the first result licenses the build.

## E0 bounds the payoff before anyone picks a lever

Two conditions, on the instrument that exists.

* **C0** - the composed advocate bundle as it ships, no rail text reachable.
  This is the first-principles compliance rate, and it is the floor.
* **C1** - the same bundle with the rail text placed in the system prompt. This
  is compliance if the skill had loaded every time, and it is the ceiling.

`C1 - C0` is the most that any fix to discoverability, composition, or the
inline voice block can buy, because all three of them do nothing except change
how often the body arrives. Neither H1 nor H2 nor H3 can pay more than the
ceiling, and the contrast costs no new instrument.

If the contrast is small, the argument between the hypotheses is an argument
between cosmetic fixes, and that is a result rather than a failure to find one.

## The grader is contaminated, and the design routes around it

`coilyco-voice-guide-linter/profile.json` is a deterministic regex profile, so
compliance grades without a human. The comparison family holds four rules. Two
of them were widened and two were added at voice-corpus `a6504f3`, dated
`2026-09-10T07:50:14Z`, in response to these exact drafts:

    pre-incident   which-is-rare      \bwhich\s+is\s+rare\b
    pre-incident   most-people-dont   \bmost\s+people\s+don['’]t\b
    a6504f3        which-is-rare      widened to rare|rarer|unusual|uncommon
    a6504f3        most-people-dont   widened to don't|do not
    a6504f3        unlike-most        \bunlike\s+most\b
    a6504f3        not-the-norm       \b(?:that|this|it)\s+is\s+not\s+the\s+norm\b

A grader fitted to the failure it now measures inflates detection on
failure-like text. Both arms are graded by the same profile, so this biases the
absolute rate more than the contrast.

The worse problem is specific to C1. The rail text names `which is rare`,
`most people don't`, `unlike most recruiters` and `that is not the norm`, and
the profile greps those same four. Showing the model the strings and then
grepping the strings measures string avoidance and calls it understanding.

**So the rail is split.** C1 receives the two pre-incident phrasings and is
graded on the two added at `a6504f3`, which it never saw. C0 receives neither
and is graded on the same held-out two. The held-out pair is the primary
outcome and it measures whether the move generalises. The full profile is
reported as a secondary and is never decisive.

The linter is regex, so unlisted phrasings of the same move pass. Measured
violation is a lower bound on real violation, and the bias runs toward
understating both arms.

## What the power buys, and what it does not

`power.py`, stdlib only, Fisher exact one-sided at alpha 0.05, 20000 trials,
checked against four known answers before it produced any of this. Fisher's tea
tasting at 1/70, the 3-1 table at 17/70, the closed form at the reference cell,
and a null contrast that cannot beat alpha.

Minimum detectable effect at 80% power, base C0 = 0.50:

    n= 20   0.39 points
    n= 30   0.33
    n= 40   0.29
    n= 60   0.24
    n= 80   0.21
    n=120   0.17

Null calibration, no true effect: 0.0252 at n=30, 0.0394 at n=60 and n=120. The
rule is conservative rather than anti-conservative, which is the direction to be
wrong in.

The predicted effect is +0.20 to +0.45, so **n=60 per arm sits on the edge and
n=120 per arm is the honest board**. At 120 per arm the run is 240 single-turn
generations, which is affordable in a way an agentic board is not. That
affordability is the reason to run E0 first and is not an argument that E0
answers the original question.

## The gate

E1, the lever separation, runs only if E0's held-out contrast clears +0.10 with
a 95% interval excluding zero. Below that, the finding is that loading the skill
barely changes the artifact, and the correct next move is to ask why the rail
is a lazy body at all rather than which of three ways of finding it to fix.

If E0 clears, E1 gets its own pre-registration, its own runner build, and a
sham-trigger arm. The reporter asked for a negative control and named the reason
honestly, which is that they would believe "triggers matter" too readily right
now. The control that answers it is an arm whose Triggers header carries
keywords that do not match the task. It separates the header convention from the
vocabulary, and Finding 2 above is why that separation is the whole of H1.

## Where I was wrong

`PREDICTION.md` was stamped at `08:13:57Z`, before `power.py` ran.

I predicted 50 to 60 per arm for a 25-point lift. The closed form says 46, which
is outside and low, and the exact Fisher test needs about 55, which is inside. I
did not say which test I meant, and the two disagree by 20%, so I am recording
it as a miss rather than resolving it in my own favour afterwards.

I predicted 120 to 160 for a 15-point lift and the closed form says 134, inside.
I predicted a 30 to 35 point minimum detectable effect at n=30 and measured 33,
inside.

The three predictions about C0, C1 and their contrast are unresolved, because
nothing has run.

## Who owns which fix

Naming this now, because the original handoff routed one of them to the wrong
seat.

* **H1, trigger headers on the writing family** - the words belong to the
  advocate seat, in voice-corpus.
* **H2 and H3, role composition and the inline voice block** - both live in
  `agent-compose`, which `kai-evaluation-stack` places inside this seat's build
  grant entire, including the shipped roster. The original handoff sent this to
  the platform seat. It does not need to go there.
* **The measurement, the runner, and the graders** - this seat.

## Files

* `PREDICTION.md` - stamped before `power.py` and before E0, kept so no number
  below can be read back as the one I expected.
* `power.py` - Fisher exact, the closed form, the MDE search, and the null
  calibration. Four known-answer checks run first and the module refuses to
  print a table if any fails.
* `power.txt` - the run.
