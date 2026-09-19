# LLM catchphrase rules, false positives against Kai's own writing

Twenty candidate voice rules against three arms, run 2026-09-08 at housecast
`7e27dd2`, against `voice-corpus` `6022f53`, `agentic-os` `67565212`,
`agent-compose` `bf7574f` and `agentic-os-kai`. Deterministic regex, no
inference, so the run reproduces exactly.

Kai reported that `Worth stating plainly` reached a draft the advocate seat
wrote. The linter engine and Kai's profile both already exist and the profile
carries no phrase rule at all, so this measures which phrase rules can ship
without flagging Kai's real prose. Records: `teable:coilyco-bridge/agentic-os-kai#7129`
and `teable:coilyco-flight-deck/agent-compose#7130`.

## Arms

* **A authentic-kai** - `source.text` where `authored_by_kai` is true. n=65
  segments, 19,255 chars. A hit here is a false positive.
* **B kai-approved** - `transformation.post_text` at status approved or
  published. n=33, 13,164 chars. Machine-transformed and then endorsed, so a
  hit is a phrase Kai signed off on.
* **C agent-written** - tracked Markdown across four fleet repositories. n=640
  docs, 1,829,358 chars. A hit is agent prose, which is what a rule should
  catch.

## Prediction, written before the run

* the bundled `robust|seamless|leverage|holistic|synergy|streamline` rule
  over-flags Arm A, because those are ordinary technical words
* `just` and `easy`, if generated straight off the seat avoid lists, flood
* Arm C carries the observed phrase at least twice, from the two tracked
  instances already grepped

The third held. **The first did not.** Arm A fired nothing at all.

## Result

Rules that fired. Arm C is adjusted for self-reference, described below.

| rule | A authentic | B approved | C agent-written | C per 10k |
| --- | --- | --- | --- | --- |
| announce-plainly | 0 | 0 | 1 | 0.01 |
| worth-noting | 0 | 0 | 2 | 0.01 |
| announce-answer | 0 | 0 | 1 | 0.01 |
| not-x-but-y | 0 | **1** | 7 | 0.04 |
| robust-seamless | 0 | 0 | 26 | 0.14 |
| moreover | 0 | 0 | 3 | 0.02 |
| hedge-tends | 0 | 0 | 1 | 0.01 |
| em-dash | 0 | 0 | 4 | 0.02 |

Twelve of the twenty fired nowhere, `flatter-rare` among them once its own
definition was excluded.

**Arm A fired zero of twenty.** Every rule, including the bundled one predicted
to over-flag, is clean against 19,255 chars of Kai's reviewed writing. The
`leverage` and `robust` worry was wrong: Kai does not use those words.

## Self-reference, and why a rollout needs the exemption

A doctrine file that bans a phrase has to quote it. `writing-kai-voice`
therefore trips every rail it defines, and the linter's own COMPOSED.md trips
the em-dash rule with its worked example. Seven of Arm C's fifty-two raw
hits are that artifact.

`flatter-rare` is the clearest case. Two raw hits, both
`writing-kai-voice/COMPOSED.md` lines 15 and 16, zero real usages. Reported
raw, the loudest finding in the estate is the style guide describing itself.

So a shipped linter needs a path exemption or a fenced convention for
rule-defining prose. `probe.py` carries the exemption as `SELF_REFERENCE`,
which is a measurement expedient rather than the shipping design.

## What ships, and what does not

**Ship.** `announce-plainly`, `worth-noting` and `announce-answer` cover the
reported failure and cost nothing in either Kai arm. The twelve zero-fire rules
carry no evidence against them either, though see the caveat.

**Split before shipping.** `robust-seamless` is 26 hits, of which 24 are the
single word `leverage` and the other 2 are `robust`, with the cluster
concentrated in `tooling-scout-*` at 12 and `tooling-tpm-*` at 5. Bundling six
words behind one id hides that. One rule per word reports what a reader can act
on, and `leverage` is already on the advocate seat's own avoid list.

**Drop.** `not-x-but-y` is the only rule with evidence against it: one hit in
Kai-approved output, and its seven Arm C hits read as legitimate contrast
rather than slop, including a deliberate doctrine line repeated across three
skills. It has the worst precision in the battery and it is the rule most
likely to make a seat learn to skip the linter.

## Caveat on the twelve, and why Kai overruled it

Zero hits in Arm A at 19,255 chars is weak evidence that a rare phrase is one
Kai would ever write. I read that as grounds to hold the zero-fire rules back.

**Kai inverted it, and she is right.** The question a blocklist answers is not
"is this rule safe" but "is this phrase absent and do we want it to stay
absent". A phrase with zero hits in both Kai's corpus and the estate is the
cheapest rule there is: it costs nothing today and it holds the line against
drift tomorrow. The rules needing caution are the ones already firing on real
prose, which is the opposite of the set I hesitated over.

So the shipping test became: a candidate ships **unless** it fires against
Kai's own reviewed writing.

## Run two, 2026-09-08, the compiled battery

295 candidates compiled from five sources, at housecast `b4cb06f`. Sources are
named in the shipped profile's own `sources` block: Wikipedia's WikiProject AI
Cleanup, the Kobak et al. excess-vocabulary study in Science Advances 2025, and
three 2026 wordlists.

`battery_terms.py` is the source list, `battery-gen.py` compiles it to
`battery-rules.json`, and `run-2.txt` is the raw output. The generator
reproduces the rules file byte for byte.

**Arm A: 0 of 295. Arm B: 0 of 295.**

Four candidates fired and were cut, each confirmed as real usage by reading the
surrounding sentence rather than by count alone.

* `that said` - her own email decline, used as a real pivot.
* `on the same page` - the subject of a LinkedIn post, and it survived into the
  approved transformation, so it is endorsed rather than tolerated.
* `harness` - "curiosity with a test harness" in approved output, and 344 hits
  across 118 estate files where it means the agent harness. Estate vocabulary,
  not slop.
* `at scale` - hers.

## Two failure classes the second run added

**A rule can collide with an estate proper noun.** `foundational` looked like
the highest-frequency slop in the battery at 38 hits, until the hits turned out
to be the `build-foundational-software` boundary slug, 47 occurrences of it
across tracked Markdown. Narrowed with a lookaround rather than dropped, it
falls to 11 real hits. This is distinct from self-reference: the document is not
about the rule, it just contains the string.

**A straight apostrophe never matches curly-quoted prose.** Every phrase rule
carrying `'` had to accept `’` as well, and curly quotes are themselves on the
tell list, so the prose most likely to need these rules is the prose they would
have silently skipped.

## What a better pattern bought

Run one recommended dropping `not-x-but-y`, the one rule with evidence against
it. Run two keeps the family and fixes the pattern instead. Requiring a comma
or dash rather than allowing a sentence break separates the LLM reframe from
legitimate contrast:

* `It's not just a linter, it's a contract.` fires
* `This is not a complete model. It is a useful one.` does not, and that
  sentence is the Arm B hit that condemned the loose version

A recommendation to drop a rule is sometimes a recommendation to rewrite it.

## Widening Arm A

Still the obvious follow-up and still the limit: `data/raw/` holds one demo
batch and `data/evals/` is empty. Arm A is also email and social register
rather than technical prose, so a word absent from it may simply be absent from
that register. The estate-vocabulary check in Arm C is what covers that gap,
and it is what caught `harness`.
