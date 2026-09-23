# Repository plan and policy

Which repositories get a resident checkout, and the policy behind that.

## Repository plan

`agent-compose cascade` writes the compiled repository availability contract at
`~/.agent-compose/repository-plan.yaml`. The format marker is
`agent-compose.repositories.v2`. The file is machine-owned, deterministic YAML.
It remains readable so an operator can inspect the projects root, trusted
inputs, per-role selections, and residency union without replaying
`.agents/roles.kdl`.

```yaml
format: agent-compose.repositories.v2
projects_root: /absolute/projects/root

inputs:
  - identity: example/operating-context
    revision: 0123456789012345678901234567890123456789
    policy:
      path: .agents/roles.kdl
      sha256: sha256:0123456789012345678901234567890123456789012345678901234567890123

roles:
  example-role:
    - identity: example/context
      path: /absolute/projects/root/example/context
      source: example/operating-context
      scope: global
      reason: repository policy makes this repository available to every role

residency:
  - identity: example/context
    path: /absolute/projects/root/example/context
    source: example/operating-context
    scope: role-union
    reason: repository is selected by at least one canonical role
```

The document contains:

* `inputs` - trusted policy source identity, full Git revision, policy path,
  and SHA-256 digest.
* `roles` - the exact sorted selection for every canonical role.
* `residency` - every role-selected repository plus resident-only pins.
* selection provenance - source, scope, reason, and provider selector details.

Consumers validate the same bounded YAML subset Agent Compose writes. The
loader rejects aliases, custom tags, duplicate keys, unknown fields, ambiguous
scalar coercion, unsafe paths, unsorted identities, and incomplete provenance.
Consumers never parse `roles.kdl`. Native skill linking reads `residency`.
Native role launch reads only the selected role.

## Repository policy

Agent Compose is the sole compiler for role-aware repository availability.
Trusted operating-context roots declare policy in `.agents/roles.kdl`. Host
configuration names those roots through `operating_context`. Doctrine sources,
harness load points, and repositories merely present on disk never grant
availability.

### Grammar

The optional top-level `repositories` node declares stable repository ids and their `owner/repository` paths. Optional `skill` children mark a repository as a bounded ordinary-skill provider:

```kdl
repositories {
    repository handbook path="example-org/handbook"
    repository style-guide path="example-org/style-guide"
    repository profile path="example-org/profile"
    repository hardware path="example-org/hardware-notes" {
        skill "compute-stack"
        skill "machine-*"
    }

    global handbook
    resident-only profile
}

roles {
    role dev-advocate {
        use-repository style-guide
    }
    role platform-eng { use-repository hardware }
}
```

Policy has three distinct scopes:

* `global` makes a repository available to every canonical role.
* `use-repository` makes a repository available only to the containing role.
* `resident-only` keeps a checkout on the host without granting it to any role.

Repositories with `skill` children become role skill providers when selected by `use-repository`.
Agent Compose rejects unknown ids, duplicate paths, duplicate use, unsafe paths, conflicting trusted definitions, and provider cycles.

### Compiled plan

`agent-compose cascade` writes `~/.agent-compose/repository-plan.yaml` in
`agent-compose.repositories.v2` format. The file is machine-owned but
intentionally reviewable. It records each trusted policy source identity, full
Git revision, `.agents/roles.kdl` path, and SHA-256 digest before the role and
residency selections. The plan uses a bounded safe-YAML subset and absolute
paths beneath one `projects_root`. Consumers reject unsafe YAML, unsafe paths,
unsorted identities, and incomplete provenance. Consumers never parse
`roles.kdl` themselves. Native skill linking uses the residency projection. A
role launch reads only its role selection. See [Repository
plan](repository-plan.md).

### Verified bundle handoff

Selected repository identities and provenance are sealed into `manifest.json`
and the decision trace. `agent-compose verify` checks ordering, identity
safety, provenance, and manifest-to-trace agreement. `agent-compose bundle
materialize --role <role> --harness <harness>` converges the host and returns a
verified immutable bundle selection without starting a harness. This is the
adapter surface for AOS and other launch consumers. Repository metadata
expresses context reachability only. It grants no runtime authority, network
access, credentials, or mutation permission.
