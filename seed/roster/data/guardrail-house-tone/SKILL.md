---
name: guardrail-house-tone
description: Outward text ships only after the voice linter has run on it as a file and its exit is pasted. Use when the advocate role is composed.
---

# Guardrail: house tone

Outward text does not ship until the linter has run on it as a file and you have pasted its exit. Text composed in a message and never written to disk was never linted, and sending it is the violation this guardrail exists to catch. Clear every house-style hit before sending, then hand-check the two things a regex cannot reach.

## The procedure

Write it to a file. The linter takes a path, so outward prose drafted in
a chat message and sent from there bypasses the detector completely. That
is the only way this guardrail actually fails.

Run it, resolving the skill root from the harness this seat is running
under. Paste the violation lines and the exit status beside the draft. A
silent clean run is reported as a clean run rather than as silence.

The two halves of a hit. Eight rules are Kai's own house style, covering
the punctuation she does not use, the emphasis she does not use, the
pronoun slips and the restricted address. Those are not negotiable and
you clear every one. Two of the eight over-flag by design, and they clear
by reading the referent and confirming it is not Kai rather than by
editing. The remaining rules are slop vocabulary compiled from external
sources, where a hit is an instruction to reread the sentence rather than
to rewrite it, and overriding one is fine when you say which and why.

Name a rule by its effect rather than by its id. Quoting an id in prose
trips the rule it names, which is how the first draft of this text failed
its own check.

What the regex cannot reach, and you must: no people's names in public
artifacts, which needs context the linter does not have.

What does not count: a clean run on an earlier draft, a clean run on a
different file, and the linter's silence when you never gave it a path.
