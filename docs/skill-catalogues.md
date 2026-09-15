# Skill catalogues

The catalogue export surface and the local catalogues it reads.

## Catalogues and bundle export

Agent Compose exposes the effective selected profile through deterministic text
and JSON catalogues. Inspection is read-only. It does not select a role,
activate a personality, change authority, or fetch a source.

### Catalogue commands

Every person-dependent command accepts `--person-source` and repeatable
`--personality-library` roots:

```text
agent-compose catalog personalities [--query <cue>] [--json]
agent-compose catalog roles [--json]
agent-compose catalog seats [--role <slug>] [--json]
agent-compose catalog expressions [--json]
```

Text output is unpaged and follows effective catalogue order. Exact normalized
personality slugs win over aliases. An ambiguous alias returns every candidate
in catalogue order.

Every JSON command emits:

```json
{
  "format": "agent-compose.catalog.v1",
  "items": []
}
```

Personality items contain `slug`, `skill`, the one-sentence skill
`description`, `aliases`, `color`, `motif`, `emblem`, `form`,
`source_library`, `digest`, and complete role `affinities`. Role items contain
`slug`, `purpose`, `skill`, role-skill provenance, role `identity`, `seats`, ordered
`personalities`, `favorite_color`, and the derived `background`. Seat items
contain `role` plus the full
stable seat object. Expression items are stable strings.

### Deterministic export

`agent-compose bundle export <bundle-dir> --out <file>.tar.gz` verifies the
bundle before opening the output. The exporter sorts safe slash-separated
relative paths, rejects links and non-regular entries through verification,
normalizes gzip and tar metadata, and includes `manifest.json`. Identical
verified trees produce byte-identical archives.

### Content-aware diff

`agent-compose diff <left-bundle> <right-bundle>` reports resolver-decision
changes, logical content changes, and changed bundle artifacts. Logical content
uses stable IDs and SHA-256 digests from the manifest. The effective role
skill, invariant, personality definitions, evaluation assets, copy contract,
and compact role identity metadata therefore remain visible even when a change
does not alter a resolver decision.

## Local skill catalogues

Agent Compose accepts an AOS-emitted local catalogue manifest:

```yaml
skill_catalog_manifest: ~/.config/aos/catalogues.json
```

This is the Agent Compose v2 ownership boundary. AOS owns remote selection,
Git access, locking, cache freshness, offline fallback, and host paths, and
must complete them before convergence runs. Agent Compose opens only the
resulting local JSON: it never fetches a source, and never mutates MCP or
approval configuration.

### Manifest contract

The document must use `aos.catalogues.v1`:

```json
{
  "format": "aos.catalogues.v1",
  "forge": "https://forgejo.coilysiren.me",
  "catalogues": [
    {
      "source": "owner/repo/.agents/skills@main",
      "path": "/absolute/local/catalogue",
      "commit": "0123456789abcdef0123456789abcdef01234567"
    }
  ]
}
```

Every entry needs a source that resolves to a forge, an absolute existing
directory, and a full 40- or 64-character Git object ID. Unknown fields,
trailing JSON, unsupported formats, missing paths, relative paths, and regular
files all fail before roster or load-point writes.

### Every source names its forge

A source carrying a scheme names its own forge, as
`https://github.com/coilysiren/coilysiren/.agents/skills@main` does. A bare
`owner/repo/path` takes the document's `forge`, which an entry may override with
its own `forge` key. **A bare source with no forge to take is an error** rather
than one that picks a side: `coilyco-flight-deck/agentic-os` is real on both
`forgejo.coilysiren.me` and `github.com`, one canonical and one a PR-gated
mirror whose content may legitimately lag. Forgejo is canonical fleet-wide
except the GitHub-canonical profile repository, so per-entry resolution is a
case the estate has rather than a generality. Both `forge` keys take the URL
form config is written in or the bare host a record carries.

A record names the host,
`forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main`, and a
skill in it drops the catalogue path:
`forgejo.coilysiren.me/coilyco-gaming/enshrouded/sirens-game-enshrouded`.
Source travels with the path rather than being checked and dropped, which is
what lets a warning name the repository to fix instead of a cache directory.

Entries retain declaration order. Two catalogues offering one skill name dedupe
on equal content and are **fatal on divergent content**, naming both sources.
Unowned files at a load point still win over every managed catalogue.
