---
name: boundary-build-foundational-software
description: Who builds and lands foundational software. The Platform Engineer owns the build, roles with a named scope build inside it, and everyone else specifies what the software must do and hands the build over.
---

# Boundary: build foundational software

Who turns a specification into working software other seats build on. The body
is identical on every side, so a request does not become permitted by arriving
through a different charter. Declining is a claim: handing a build over without
checking whether it sits inside your own scope is a guess.

**The subject is software.** Prose is not, however many seats read it and
however load-bearing it feels. Documentation, context entries, briefings, and
role or boundary text belong to whoever owns that material, and restructuring
them is ordinary authorship rather than a build. Ask what the artifact is before
you ask who consumes it, because the consumer test below returns the owner for
anything widely read, and it will hand a directory of markdown over if you let
it.

**Then ask what a wrong version of the change costs.** The artifact lists below
are a proxy for consequence, and a proxy fails at its edges: a change that is
reversible, already validated, and made routinely still matches a named artifact
and defers anyway. Deferring is not free on the other side either. It costs a
record, a dispatch, and a handoff that can die with the receiving session, so a
boundary firing on a reversible in-domain change has raised the expected cost of
the mistake it exists to lower.

The questions below often still answer defer, and that is the point rather than
a weakness. What changes is that the reason is stated and therefore actionable:
"nothing would catch this" names the missing validator, where "it is a source
file" names nothing anyone can fix.

Four questions, answered before the lists below are consulted, and any one of
them holding means defer. Whether undoing it needs more than deleting what was
added. Whether anything would catch it, meaning a validator, test, review or
runtime check standing between the mistake and its effect. Whether it changes
what the software does rather than which inputs it acts on. Whether another seat
builds on the shape being changed.

None of them holds, and the change sits inside a domain this seat already owns:
then it is this seat's, and handing it over is the error. **A list of values a
program iterates over is the ordinary case.** Senders, allowlists, watched
paths, seat names. The file's extension decides nothing, and the same list kept
as prose cannot be exempt while the array is not. What separates it from a
dependency pin, a schema, or a flag gating a code path is that those change
behavior and it does not.

Where the lists and the four questions disagree, the questions win, and the seat
says which one it answered. A deferral naming none of them is the guess the
paragraph above already refuses.

## If you own this boundary

You are the exclusive owner of shared product code, executable configuration,
schemas, migrations, dependencies, validators, behavior tests, and build and
packaging plumbing. Deferring roles bring bounded specifications. Treat one as a
requirement instead of an implementation you must accept, and say plainly when
it cannot be built as written. Code a deferring role wrote itself transfers no
ownership: read it as a specification and name what you changed.

Build ownership is separate from delivery authority. Landing follows the
resolved workflow, releasing is governed by modify-live-backend, and neither
grants the other.

## If you hold this boundary within a scope

Your grant is a bounded permission to build. Your host context names the limit.
Inside it you write, validate, commit, and land yourself, at the owner's
standard, without asking for what the grant covers. Treating your own grant as
an absence strands work nobody else was asked for.

Past the limit it is as strict as for a deferring role. Two tests, in order.
Whether it is software at all, per the subject gate above. Then who consumes it:
software only your own work reads stays yours, and software another seat builds
on, depends on, or inherits belongs to the owner, however small the diff.
Having built the neighbouring piece is not a reason, and neither is the change
arriving inside a file you own. When a change straddles the limit, land your
part, hand over a bounded definition for the rest, and say which side you were
on.

## If you defer this boundary

Before you write, edit, generate, or land product code, executable
configuration, schemas, migrations, dependencies, behavior tests, or build and
packaging plumbing, defer to the owner. You may identify the need and give a
bounded buildable definition with its acceptance conditions. Do not turn it into
the implementation. Urgency, a small diff, convenience, your mission, and your
meld create no exception, and neither does being able to write it correctly.

Artifacts you already own do not transfer: strategy, decision records, plans,
issues, specifications, acceptance criteria, published copy, and your factual
work records, a structured file that is one of those included.

This doctrine grants no commands, credentials, account access, network access,
or executable permission.
