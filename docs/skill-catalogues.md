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

Each item carries the whole stable object for its kind, so the emitted JSON is
the field list rather than a copy of it here. Personality items add role
`affinities`, role items role-skill provenance and `background`, seat items
`role`, and expression items are plain strings.

### Deterministic export

`agent-compose bundle export <bundle-dir> --out <file>.tar.gz` verifies the
bundle before opening the output. The exporter sorts safe relative paths,
rejects links and non-regular entries, normalizes gzip and tar metadata, and
includes `manifest.json`, so identical verified trees export byte-identical.

### Content-aware diff

`agent-compose diff <left-bundle> <right-bundle>` reports resolver-decision
changes, logical content changes, and changed bundle artifacts. Logical content
uses the manifest's stable IDs and SHA-256 digests, so a content change stays
visible even when it alters no resolver decision.

## Local skill catalogues

Agent Compose accepts an AOS-emitted local catalogue manifest:

```yaml
skill_catalog_manifest: ~/.config/aos/catalogues.json
```

This is the Agent Compose v2 ownership boundary. AOS owns remote selection, Git
access, locking, cache freshness, offline fallback and host paths, and must
complete them before convergence runs. Agent Compose opens only the resulting
local JSON: it never fetches a source, and never mutates MCP or approval config.

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
its own. Both keys take the URL form config is written in or the bare host a
record carries. **A bare source with no forge to take is an error** rather than
one that picks a side: `coilyco-flight-deck/agentic-os` is real on both
`forgejo.coilysiren.me` and `github.com`, one canonical and one a PR-gated
mirror whose content may lag, and Forgejo is canonical fleet-wide except the
GitHub-canonical profile repository.

A record names the host, as
`forgejo.coilysiren.me/coilyco-gaming/enshrouded/.agents/skills@main`, and a
skill in it drops the catalogue path. Source travels with the path rather than
being checked and dropped, so a warning names the repository to fix.

Entries retain declaration order, and one catalogue listed twice is refused as
a malformed manifest rather than ruled on as a duplicate. Two **different**
catalogues offering one name dedupe on equal content and are **fatal on
divergent content**, naming both. Unowned files at a load point still win.

An entry marked `"private": true` keeps its content and loses its identity: its
skills project as any other catalogue's, and its source renders as `private
catalogue <n>` everywhere agent-compose writes. `Source.Reveal()` returns the
real string for a message staying on the host that holds the catalogue, and is
the one call that must never reach shipped output. That is what lets a private
focus reach a public lane without the public image learning it exists.
