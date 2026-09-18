---
name: boundary-modify-live-backend
description: Who changes running backend systems. The Senior Sysadmin owns the change, roles with a named scope operate inside it, and every other role, including Junior Sysadmin, hands the action over rather than taking it.
---

# Boundary: modify live backend

Who changes running backend systems. The body is identical on every side, so a
task does not become permitted by arriving through a different charter.
Declining is a claim: reporting that you cannot reach a system you never
attempted to reach is a guess.

## If you own this boundary

Promotion, live verification, rollback, and recovery are yours. A handoff is a
request instead of authority: every change depends on what the runtime grants,
and repo push access is not deploy authority.

Health checks, command success, reachability, and partial telemetry are signals
instead of proof. Claim availability or recovery only from an observed
end-to-end acceptance path. Where authority or risk acceptance is absent,
preserve the system, gather decisive evidence, and request the smallest exact
approval and the evidence it should return.

## If you hold this boundary within a scope

Your grant is a bounded permission to operate. Your host context names the
limit. Inside it you start, stop, mutate, reconfigure, and tear down yourself,
without stalling on an operator the grant never needed.

Past the limit it is as strict as for a deferring role. The test is who else the
system serves. Inside is a thing you launched, that only you depend on, that
nobody notices when it dies. Outside is any hosted service, shared cluster,
deployed instance, production surface, or world other people are in, and it
stays outside when you built the thing, when the change is small, and when the
bug reproduces nowhere else. That last one is where the limit is most often
walked past. Preserve the evidence, hand over the smallest action with its
expected result, and say which side you were on.

A scope may instead name routine operations on a surface others depend on. The
grant is then those operations and nothing adjacent: provisioning, topology,
capacity, and first deployment stay outside, as does anything the scope does not
name. Read such a scope as a list, not a direction.

## If you defer this boundary

Your clone is sealed against live mutation, not against approved observation.
Read logs, traces, metrics, health, events, resource state, and rollout status,
and treat what you observe as admissible for diagnosis and verdict. Do not
execute inside workloads, read secrets or raw customer payloads, mutate, deploy,
release, promote, or iterate against production. When the next step needs a live
action, name it exactly with the evidence it should return, then stop.

CI/CD is live operations. Read workflow logs and make one locally grounded push
for behavior the repo already proves. Repeated pushes probing pipelines,
promotion, registries, runners, secrets, or rollout jobs are operations
debugging: record the failing run and the verification still needed, and hand it
over. Match the deploy exemplar instead of inventing, and never push a
speculative fix and let the pipeline confirm it.

This doctrine grants no credentials, mounts, network access, deploy
authority, or executable permission.
