# KDL contracts

Agent-compose uses KDL for human-authored requests and policy. Parsers reject
unknown nodes, duplicate facts, missing sources, or an empty selection.

## Compose request and capability sources

A request names a role, a delivery mode, an optional person package, and any
external capability sources:

```kdl
compose {
    person-policy "external-only"
    person-source "person"
    role "platform-eng"
    model-tier "commodity"
    delivery "native-skills"
    source "aos-public" root="agentic-os" required=#true
}
```

`person-source` names a request-relative package and fully replaces the mounted
one, `person-policy "external-only"` requires it and prohibits fallback, and
omitting both selects the mounted `roster:core` unless the host guard supplies
it. The binary carries no roster and resolves one from disk ([FEATURES.md](FEATURES.md)). The role activates its personality set, ordinary skills, and
composed-skill allowlist. `delivery` is `native-skills` or `compiled`.
`model-tier` is `frontier`, `commodity`, or `oss`, defaults to `frontier`, must
be supported by the role, and never changes selected context: every supported
tier receives the complete selection. Legacy `density "full"` is ignored and
other densities fail. Sources run in request order, `root` and `declaration`
only locate files, and optional `identity` renames the composed seat
([identity](identity.md)). Optional `boundary-omit` drops defer-side boundaries
whose owning seat this deployment does not have
([boundary omission](boundary-omission.md)). The public AOS provider needs only its root.
Agent-compose discovers ordinary skills and reads one `.agents/roles.kdl`
graph:

```kdl
repositories {
    repository hardware path="example/hardware-knowledge" {
        skill "machine-*"
    }
}
roles {
    role "platform-eng" {
        use-repository hardware
        composed-skill "coding-*"
    }
    role "dev-advocate" {
        use-repository hardware {
            skill "machine-laptop-*"
        }
    }
}
```

Repository IDs are document-local, paths use `owner/repository`, and `skill`
marks a selected repository as an ordinary-skill provider with a bounded
catalogue, failing closed when its checkout or `.agents/skills` catalogue is
unavailable. A `use-repository` or `use-provider` binding may carry its own
`skill` children, which narrow that one role's reach within what the definition
already admits. The binding runs as a second pass over the definition's
selection, so it can only subtract. A binding pattern reaching past the
definition matches nothing and fails, exactly like any other unmatched pattern.
Overlap and empty-selector rules apply to the binding's own patterns, and
omitting the children leaves the role receiving whatever the definition admits.
Two roles mounting one definition therefore no longer have to receive the same
set, which is what lets a repository hold material one role must not reach. Only trusted roots widen eligibility, imported graphs do not
recurse, and [role selection](role-selection.md) covers resolution and
provenance. Each `composed-skill` admits `.agents/composed/<name>/COMPOSED.md`
by exact name or glob, globs expand lexically, and invalid or overlapping
selections fail. A graph may bind skills to a role the roster does not define:
only the requested role's bindings are read, so the rest stay inert rather than
failing the compose, letting a provider stage a role ahead of the roster that
ships it. A role named in the request must still exist. Materialization renames
admitted entry points to `SKILL.md`, and nested `SKILL.md` files or
ordinary/composed name collisions fail. The same root form works in requests,
roster arguments, and `roster_sources`. Roster sources are optional overlays,
the selected person source always supplies the invariant and bound personality
bodies, and an overlay or another provider can instead declare explicitly:

```kdl
source "aos-public" {
    instruction "foundation" path="content/foundation.md"
    skill "coding-go" path="skills/coding-go"
}
```

The request admits it with `source "aos-public"
declaration="source-public.kdl"`. Paths stay beneath the declaration root.
Symlinks and escaping paths fail. Required missing sources fail, while optional
ones produce trace decisions. During rolling upgrades, identical legacy
invariant and personality copies shadow behind the person source. Different
copies conflict.

## Acts

Every role, personality, and boundary side names three runnable things, because
an attribute phrased as attitude does not fire. The finding is #388.

```kdl
act tool="WebSearch" "WebSearch the claim before ranking it, and report the result count"
act "defer" tool="create_issue" "create_issue naming the outside question"   // boundaries
```

`tool` is carried apart from the sentence so a coverage or portability check
reads it without parsing English, and it has to appear in the text so the two
cannot drift and leave the check passing against text nobody can run. A boundary
act takes a leading side argument, and each side owes its own three: a deferred
boundary is a different action rather than the owner's withheld. Acts are
optional until one attribute declares them, then all do, and they spend no
doctrine word budget.

The shipped roster names no estate-specific tool. An absent one reads as an
instruction already satisfied (agentic-os#1381), so a harness lacking a named
tool substitutes its own and says which. Estate acts go in a private overlay.

## See also

- [manifest-schema.md](manifest-schema.md) - the schema these requests compose into.
