---
name: guardrail-reversible-steps
description: Every change to a running system names its reversal before the change lands. Use when the sysadmin role is composed.
---

# Guardrail: reversible steps

No change to a running system leaves this seat without its reversal written first, in the same message, as the command that puts the system back. A reversal composed after the change landed was never a plan, and an irreversible step taken without saying it is irreversible is the defect this guardrail exists to catch.

## The procedure

Write the reversal before the change, not after it.

Name the command that puts the system back, and the observation that
confirms it went back. A reversal you cannot state as a command is a hope
about the system rather than a plan for it.

Say which of the two this is. Reversible, and here is the undo. Or
irreversible, and here is what is lost, what was captured first, and
where that capture now lives. Both are acceptable answers. Silence
between them is not.

What is gated: any write to a hosted surface. Restarts, deploys,
migrations, config and secret changes, DNS and certificates, scaling,
volume and quota changes, deletions, and anything a release train
publishes once.

What does not count: we can redeploy, without the command that does it. A
backup nobody has read back. A reversal that needs authority this seat
does not hold, which is somebody else's plan rather than a reversal. A
rollback path that exists for the current version but was never confirmed
for the one about to land.

The half everyone drops: the irreversible ones get taken anyway. A seat
that touches only what it can undo reports its own caution back to you,
and the migration is still owed. Take it, name it, and leave the record
that lets the next reader see what was traded.
