"""The generated half of a seat's linter profile.

Byte parity with agent-compose's Go engine is asserted there, in
`checks/tests/test_parity.py`. These cover the behaviour that parity cannot
see, because both engines would be wrong together.
"""

from __future__ import annotations

import re

from checks import roster, voiceprofile


def _rule(rules: list[dict[str, object]], rule_id: str) -> dict[str, object]:
    return next(r for r in rules if r["id"] == rule_id)


def _advocate() -> list[dict[str, object]]:
    loaded = roster.load()
    return voiceprofile.generated(voiceprofile.banks(loaded, "advocate"))


def test_the_role_bank_keeps_a_term_a_personality_also_claims() -> None:
    rules = _advocate()
    ids = [r["id"] for r in rules]
    assert len(ids) == len(set(ids))
    assert _rule(rules, "avoid-circle-back")["hint"] == "on the Developer Advocate avoid bank"


def test_a_generated_rule_does_not_fire_inside_a_longer_word() -> None:
    """The over-flag that makes a generated rule worth skipping rather than reading."""
    pattern = re.compile(str(_rule(_advocate(), "avoid-just")["pattern"]), re.I)
    for clean in ("adjust the value", "unjust", "justify it"):
        assert not pattern.search(clean), clean
    assert pattern.search("just run it")


def test_a_phrase_matches_across_a_line_break() -> None:
    pattern = re.compile(str(_rule(_advocate(), "avoid-circle-back")["pattern"]), re.I)
    assert pattern.search("we should circle\nback on this")


def test_go_quote_meta_rather_than_re_escape() -> None:
    """re.escape also escapes space, #, &, - and ~, and one extra backslash breaks parity."""
    assert voiceprofile._quote_meta("well-known") == "well-known"
    assert voiceprofile._quote_meta("a.b") == "a\\.b"


def test_empty_and_unslugged_terms_are_skipped() -> None:
    class _Voice:
        def __init__(self) -> None:
            self.avoid = ["  ", "", "leverage", "Leverage", "!!!"]

    rules = voiceprofile.generated([("Role", _Voice())])  # type: ignore[list-item]
    assert [r["id"] for r in rules] == ["avoid-leverage"]


def test_a_carried_rule_wins_a_collision_and_sorts_first() -> None:
    carried = [{"id": "avoid-leverage", "pattern": "x", "hint": "hand written"}]
    built = voiceprofile.build("roster:core:advocate", carried, _advocate())
    assert built is not None
    rules = built["rules"]
    assert rules[0]["id"] == "avoid-leverage"
    assert rules[0]["hint"] == "hand written"
    assert [r["id"] for r in rules].count("avoid-leverage") == 1


def test_nothing_to_lint_against_builds_no_document() -> None:
    assert voiceprofile.build("roster:core:none", [], []) is None
