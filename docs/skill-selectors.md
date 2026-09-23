# Skill selectors and load points

How ordinary skills are selected, and where they load.

## Ordinary-skill selectors

A repository declaration in `.agents/roles.kdl` may bound its ordinary
`.agents/skills` catalogue with `skill` children:

```kdl
repositories {
    repository hardware path="example/hardware-knowledge" {
        skill "compute-stack"
        skill "machine-*"
    }
}
roles {
    role platform-eng {
        use-repository hardware
    }
}
```

Patterns use Go path-match syntax. A literal is exact and `*`, `?`, or bracket
forms provide glob matching within one skill ID. Use `skill "*"` to admit a
provider repository's whole ordinary catalogue.

### Fail-closed validation

The KDL and generated-JSON loaders reject an empty pattern or malformed glob.
At composition time, every pattern must
match at least one ordinary skill. No skill may match two configured patterns.
Unmatched or overlapping patterns fail without producing a bundle.

Agent Compose loads and validates the provider's complete ordinary and
composed catalogues before filtering. A selector therefore cannot hide a
malformed source. Selection retains the catalogue's lexical order instead of
pattern order.

### Evidence and budget

The provider report records the patterns and admitted catalogue fraction.
Selected skill decisions carry that selector outcome. Skills outside the
slice remain explicit excluded decisions with a selector reason, so
`agent-compose describe --why skill:<id>` explains why they did not enter the
bundle.

Context bytes and approximate tokens measure only selected trees. Native and
staged projection consume the same immutable bundle and therefore retain the
same selector evidence and budget across Claude, Codex, Goose, and OpenCode.

See the [role-provider example](../examples/role-provider-selector/README.md)
for a minimal configuration fragment.

## An org grants a whole owner

An `org` node names a forge **owner** rather than a path, so which repositories
it holds is the compiled manifest's answer rather than a list kept in config:

```kdl
repositories {
    org gaming owner="acme-games" {
        skill "repo-*"
        skill "game-*"
    }
}
roles {
    role game-dev {
        use-org gaming
    }
}
```

**The grant is open.** A repository added to the org later reaches the bundle
with no config change, so the `skill` children are the whole review surface and
an org declared without one is refused. Patterns keep the semantics above but
apply across the org's whole surface rather than per catalogue. Each
contributing catalogue becomes one bundle source at scope `org` holding only
the named slice, and is a bare skills directory rather than a provider tree, so
it carries no invariant and no role graph.

An org the manifest holds no catalogue for is an error: an org contributing
nothing is an empty selector by another name. A skill offered by two catalogues
in one org is fatal and names both. A skill is a directory carrying `SKILL.md`,
because five repositories keep a `categories.yaml` beside theirs.

## A skill is reachable by address

`skill_requests` in the cascade config names one skill, terse as
`owner/repo/skill` or qualified as `forge/owner/repo/skill`. Resolution happens
against the compiled set at converge time, and an address resolving to nothing
**fails the converge and names the address** rather than starting a lane with a
focus missing. A terse address served by two forges is refused as a bare source
in [skill catalogues](skill-catalogues.md) is. Catalogues still mount whole, so
a request today is resolution and validation rather than reach.

## Global skill load points

`skill_load_points` names the harness-native directory each harness reads for
global skills, and converge links the compiled residency set into each.

### Defaults

Claude and codex are wired by default, matching `load_points`:

* `claude` - `~/.claude/skills`
* `codex` - `~/.agents/skills`

Claude Code reads `.claude/skills` and never the portable `.agents/skills`, so
one shared path cannot serve both. Goose and opencode do read the portable
directory, but like their instruction load points stay opt-in through config.

### Overrides

A configured entry replaces its default. A null or false value opts that
harness out, the same falsy rule `load_points` uses.

Naming one harness leaves the other on its default. Before this, config setting
only `codex` unwired claude silently, and a claude session started with no
global skills while codex had the full set.
