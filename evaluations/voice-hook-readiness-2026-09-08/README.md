# Voice profile as a hook, day-one blocking surface

Run 2026-09-08 at housecast `8adbe44`, against voice-corpus `2976701`,
`agentic-os` `131a9454`, `agentic-os-kai`, `agent-compose` and housecast itself.
Deterministic regex, no inference, so the run reproduces exactly.

The catchphrase evaluation beside this one asked which rules can ship. Kai then
asked for the linter to stop being a retrieved skill and start being a hook. A
retrieved linter fires when a description matches. A hook fires always. So the
question this run answers is what "always" costs against the estate as it
stands today.

## Method

`readiness.py` reads the shipped engine and the shipped profile by path from
voice-corpus and applies all 293 rules to every tracked Markdown file in four
repositories. It vendors neither the engine nor the profile, because a copy
would be a second source of truth for files another repository owns.

## Prediction, written before the run

150 to 400 findings estate-wide, with self-reference over 30% of them. The
catchphrase run had produced 52 raw hits from 20 rules, and self-reference was
its loudest single artifact.

**Wrong by roughly 5x, and wrong about the cause.** Self-reference is not the
dominant artifact. A single broken rule is.

## Result

* rules in profile: 293
* corpus: 698 tracked Markdown files
* total findings: 1,830
* files with at least one finding: 440 of 698, or 63%

| rule | findings |
| --- | --- |
| wrong-pronoun-they | 1102 |
| prose-semicolon | 277 |
| italics-asterisk | 100 |
| prose-table | 86 |
| foundational | 34 |
| italics-underscore | 33 |
| leverage | 29 |
| em-dash | 22 |

Forty-one further rules fire between 1 and 13 times each. Full list in
`run.txt`.

## The pronoun rules are broken, and they are 61% of the output

`wrong-pronoun-they` is `\b(they|them|their)\b`, case-insensitive, with no test
for who the sentence is about. Its hint says to use she/her if the subject is
Kai. Nothing in the rule establishes that the subject is Kai.

The operating base requires the neutral pronoun for every person whose pronouns
have not been stated. So the rule fires on prose that is following the doctrine
correctly, and it fires 1,102 times. Sampled hits are generic references in
supply-chain audit checklists and CI troubleshooting notes, none of them about
Kai. `wrong-pronoun-he` has the same shape and adds 13.

Together they are 1,115 of 1,830 findings, or 61%. This is not a tuning problem.
A rule whose predicate is "the subject is Kai" cannot be expressed as a word
match, and the correct pronoun in most of these files is the one being flagged.

The catchphrase run never surfaced this because it measured candidate phrase
rules, and both pronoun rules were already in the shipped profile under the
`house-style` family, outside the battery under test. A rule that ships without
passing through the battery is a rule nobody measured.

## What this means for the hook

Blocking is not available today, on either accounting.

* with the pronoun rules: 440 of 698 files fail, 63% of the corpus
* without them: 715 findings across 231 files, still 33% of the corpus

So the hook ships in one of two shapes. Warning-only now, which fires on every
commit touching a third of the estate and trains people to scroll past it. Or
blocking on a cleared subset, which needs the pronoun rules pulled and a
remediation pass on the rest.

The subset worth blocking first is the four rules that are unambiguous, cheap,
and already doctrine: `prose-semicolon`, `italics-asterisk`, `italics-underscore`
and `em-dash`. Those are 432 findings, all mechanical, none requiring a judgment
about who a sentence is about.

## Engine gaps a rollout still needs

* **No self-reference exemption.** The engine has no path or fence mechanism for
  rule-defining prose, so a style guide trips every rail it documents. The
  catchphrase run carried the exemption in `probe.py` as a measurement
  expedient. It is not in the shipped engine.
* **This file trips its own subject matter.** Documenting a rule means naming it.
  That is the same gap, observed once more.

## Handover

Every fix named here lands in voice-corpus or in the hook wiring, which is
foundational software this seat specifies rather than builds. Three records
carry the buildable definitions and their acceptance conditions, and this
evaluation is the evidence under all three.

* `teable:coilyco-bridge/voice-corpus#7150` - pull the two pronoun rules
* `teable:coilyco-flight-deck/agentic-os#7151` - install the linter as a hook,
  blocked on 7150
* `teable:coilyco-bridge/voice-corpus#7152` - consolidate the twelve voice
  skills, which is the retrieval half of the same problem
