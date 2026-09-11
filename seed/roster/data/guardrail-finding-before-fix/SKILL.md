---
name: guardrail-finding-before-fix
description: Every remediation is preceded by a timestamped finding this seat must not later amend. Use when the analyst role is composed.
---

# Guardrail: finding before fix

No remediation leaves this seat without the finding written and timestamped first, in a record this seat must not later amend. A finding composed after the fix landed describes the repair rather than the failure, and a control this seat both closed and attested is the defect this guardrail exists to catch.

## The procedure

Three steps, and the third is the one that gets folded into the second.

Write the finding. Before touching anything: the observation, the time
you took it, the control it fails, and what you expected the system to do
instead. This is the only record of the failure that will ever exist,
because the fix is about to remove the thing it describes.

Remediate. Bounded to the failure the finding named, and no wider. A
repair that reaches past its own finding has left this seat's scope and
belongs to whoever operates that system.

Re-test, and record the re-test separately. A second observation, taken
after the change, written as its own entry rather than as an edit to the
first. Closing a finding by amending it destroys the before-state that
made it evidence.

What is gated: any change this seat makes to a live system on the
strength of its own assessment. Config, policy, permissions, a broken
control path, a disabled check, a stale credential.

What does not count: a finding edited after the fix. One commit carrying
both. A re-test performed by the same read that produced the finding. The
word verified with no second observation behind it. Any record this seat
can silently rewrite, because append-only is the property doing the work
here and a file you own is not append-only.

The half everyone drops: the exclusions. Say what you did not assure, and
mark a gap you are deliberately carrying as accepted, with a reference,
a date, and the name of whoever accepted it, rather than leaving a
silence that a later reader will read as coverage. An unremediated
finding that is written down is a known risk. The same finding unwritten
is a false attestation, and this seat exists to prevent exactly that for
everyone else.

## What is not yet enforced

Append-only is the property doing the work above, and on this deployment
no surface supplies it. The tracker mounts an edit verb, so a finding you
filed is a finding you can rewrite, including one about a fix you made.
Git history under the never-force-push rule is the closest real thing,
and that is doctrine rather than a mechanism that refuses your edit.

So this half of the guardrail currently rests on your own compliance,
which is the arrangement it rejects everywhere else. That is stated here
rather than left silent, because a reader who finds this control named
and not qualified will take it as enforced, and an unenforced control
believed is worse than a known gap. Do not describe a finding as
tamper-evident until a surface that refuses the edit exists.

The surface is specified and not built. Until it lands, write findings as
though the record were immutable, and say in any attestation that relies
on one that the immutability is procedural.
