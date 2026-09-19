"""The linter profile a composed seat lints its own prose against.

Mirrors agent-compose's `internal/voiceprofile` byte for byte, because
`checks/tests/test_parity.py` there asserts the two engines compose identical
bundles. Merge rules and anchoring live in that repository's
`docs/manifest-schema.md`.
"""

from __future__ import annotations

import re
from typing import Any

from checks.roster import Roster, Voice

FORMAT = "agent-compose.voice-profile"

# Go's regexp.QuoteMeta set, which is not Python's re.escape set: re.escape
# also escapes space, #, &, - and ~, and one extra backslash breaks parity.
_GO_META = set("$()*+.?[\\]^{|}")

_NOT_SLUG = re.compile(r"[^a-z0-9]+")


def _quote_meta(value: str) -> str:
    return "".join("\\" + c if c in _GO_META else c for c in value)


def _slug(term: str) -> str:
    return _NOT_SLUG.sub("-", term.lower()).strip("-")


def _is_word_char(c: str) -> bool:
    return c == "_" or (c.isascii() and c.isalnum())


def _pattern(term: str) -> str:
    """Anchor on word-character ends, and let internal spacing match a break."""
    fields = term.split()
    if not fields:
        return ""
    body = r"\s+".join(_quote_meta(f) for f in fields)
    collapsed = re.sub(r"\s+", " ", term)
    if _is_word_char(collapsed[0]):
        body = r"\b" + body
    if _is_word_char(collapsed[-1]):
        body += r"\b"
    return body


def banks(roster: Roster, role_name: str) -> list[tuple[str, Voice]]:
    """Melds role first, then personalities, matching render's Refuse line."""
    role = roster.roles[role_name]
    out: list[tuple[str, Voice]] = []
    if role.voice:
        out.append((role.display_name, role.voice))
    for name in role.personalities:
        binding = roster.personalities[name]
        if binding.voice:
            out.append((_display_slug(name), binding.voice))
    return out


def _display_slug(name: str) -> str:
    from checks.render import display_slug

    return display_slug(name)


def generated(sources: list[tuple[str, Voice]]) -> list[dict[str, Any]]:
    """One rule per avoid term, keeping the first bank that claims it."""
    seen: set[str] = set()
    rules: list[dict[str, Any]] = []
    for label, voice in sources:
        for raw in voice.avoid:
            term = raw.strip()
            rule_id = _slug(term)
            if not rule_id or rule_id in seen:
                continue
            body = _pattern(term)
            if not body:
                continue
            seen.add(rule_id)
            rules.append(
                {
                    "id": "avoid-" + rule_id,
                    "pattern": body,
                    "hint": f"on the {label} avoid bank",
                    "flags": ["i"],
                    "source": "voice:" + label,
                }
            )
    return rules


def build(
    name: str, carried: list[dict[str, Any]], gen: list[dict[str, Any]]
) -> dict[str, Any] | None:
    """Merged document, or None when the seat has nothing to lint against.

    A carried rule wins a collision on id and sorts first. The caller renders
    it through `go_json`, so the bytes match what the Go engine marshals.
    """
    claimed: set[str] = set()
    rules: list[dict[str, Any]] = []
    for rule in carried:
        if rule["id"] in claimed:
            continue
        claimed.add(rule["id"])
        rules.append(rule)
    rules.sort(key=lambda r: r["id"])
    kept: list[dict[str, Any]] = []
    for rule in gen:
        if rule["id"] in claimed:
            continue
        claimed.add(rule["id"])
        kept.append(rule)
    kept.sort(key=lambda r: r["id"])
    rules += kept
    if not rules:
        return None
    return {"format": FORMAT, "name": name, "rules": rules}
