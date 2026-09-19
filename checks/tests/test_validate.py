"""Negative controls for every rule agent-compose#333 names.

Each test mutates a roster that passes, so a rule that silently stopped firing
fails here instead of passing quietly. The positive baseline is asserted once,
which is what makes the mutations meaningful.
"""

from __future__ import annotations

import dataclasses
import pathlib

import pytest

from checks import color, roster, validate
from checks.roster import Roster


@pytest.fixture
def loaded() -> Roster:
    return roster.load()


def test_the_shipped_roster_passes(loaded: Roster) -> None:
    validate.check_boundary_ownership(loaded)
    validate.check_personality_bindings(loaded)
    validate.check_definition_set(loaded)
    validate.check_personality_colors(loaded)
    validate.check_grounding_lanes(loaded)
    validate.check_guardrails(loaded)
    validate.check_skill_frontmatter(loaded)
    validate.check_copy_contract(loaded)
    validate.check_prose_floors(loaded)


def test_a_role_with_no_grounding_lane_is_rejected(loaded: Roster) -> None:
    """A missing lane derives no grounding pair, and coverage cannot see the hole."""
    loaded.roles["science"].grounding = ""
    with pytest.raises(roster.RosterError, match="has no grounding lane"):
        validate.check_grounding_lanes(loaded)


def _rail(**kw: object) -> roster.Guardrail:
    """A guardrail with the binding already correct, so a test varies one thing."""
    fields: dict[str, object] = {
        "name": "provable-results",
        "skill": "guardrail-provable-results",
        "role": "science",
        "card": "the rail",
        "body": "the procedure",
        "attests_in": "pastes it",
        "attests_out": "marks what it cannot show",
    }
    fields.update(kw)
    return roster.Guardrail(**fields)  # type: ignore[arg-type]


def test_a_role_with_no_guardrail_is_accepted_because_absence_is_a_decision(
    loaded: Roster,
) -> None:
    """Kai scoped the primitive to four roles. Frontend, platform and gamedev are
    deliberately without one, so absence derives no case and that is correct."""
    assert [name for name, role in loaded.roles.items() if not role.guardrail]
    validate.check_guardrails(loaded)


def test_a_guardrail_with_no_card_half_is_rejected(loaded: Roster) -> None:
    """A card with no text renders an empty eager section, and the correction that was
    supposed to fire without being loaded fires not at all."""
    loaded.guardrails["provable-results"] = _rail(card="  ")
    with pytest.raises(roster.RosterError, match="no card half"):
        validate.check_guardrails(loaded)


def test_a_guardrail_with_no_body_half_is_rejected(loaded: Roster) -> None:
    """A card with no body points the seat at a skill that carries no procedure."""
    loaded.guardrails["provable-results"] = _rail(body="")
    with pytest.raises(roster.RosterError, match="no body half"):
        validate.check_guardrails(loaded)


def test_the_shipped_guardrails_declare_the_kinds_the_board_derives(loaded: Roster) -> None:
    """Both kinds exist in the shipped roster, so neither branch is derived-but-never-run.

    The four are Kai's. Frontend, platform and gamedev are deliberately absent.
    The analyst, psych and reporter guardrails went with those seats' deprecation
    on 2026-09-15, so this is the scope decision written where a change to it has
    to pass.
    """
    kinds = {g.role: g.reproducible for g in loaded.guardrails.values()}
    assert kinds == {
        "senior-sysadmin": False,
        "science": False,
        "director": False,
        "advocate": True,
    }


def test_an_attested_guardrail_without_both_targets_is_rejected(loaded: Roster) -> None:
    """There is no generic attested target. Without one authored here the deriver would
    have to invent it, and every attested role would grade in one role's vocabulary."""
    loaded.guardrails["provable-results"] = _rail(attests_out="")
    with pytest.raises(roster.RosterError, match="without both attests targets"):
        validate.check_guardrails(loaded)


def test_a_reproducible_guardrail_authoring_attests_targets_is_rejected(
    loaded: Roster,
) -> None:
    """The deriver generates both targets from the detector, so authored text beside one
    is parsed and never read, which is the drift a kind enum would have introduced."""
    loaded.guardrails["provable-results"] = _rail(detector="lint.py --strict")
    with pytest.raises(roster.RosterError, match="would never read"):
        validate.check_guardrails(loaded)


def test_a_guardrail_whose_role_does_not_name_it_back_is_rejected(loaded: Roster) -> None:
    """Half a binding is the failure that made the primitive undeliverable: an element
    the composer never reaches, or a role pointing at nothing. Both were representable."""
    loaded.guardrails["provable-results"] = _rail(role="platform")
    with pytest.raises(roster.RosterError, match="does not name it back"):
        validate.check_guardrails(loaded)


def test_a_role_naming_an_unknown_guardrail_is_rejected(loaded: Roster) -> None:
    """The other half of the same binding."""
    loaded.roles["platform"].guardrail = "no-such-rail"
    with pytest.raises(roster.RosterError, match="unknown guardrail"):
        validate.check_guardrails(loaded)


def test_no_two_shipped_attested_guardrails_share_a_target(loaded: Roster) -> None:
    """Science's MEASURED tag means nothing to a seat whose guardrail is reversibility.
    A shared target is the tell that one role's vocabulary was applied to another."""
    targets = [
        half
        for g in loaded.guardrails.values()
        if not g.reproducible
        for half in (g.attests_in, g.attests_out)
    ]
    assert len(targets) == 6
    assert len(set(targets)) == 6


def test_owner_may_not_declare_the_boundary_it_owns(loaded: Roster) -> None:
    owned = "build-foundational-software"
    assert loaded.boundaries[owned].owner == "platform"
    loaded.roles["platform"].defers.append(owned)
    with pytest.raises(roster.RosterError, match="also declares it"):
        validate.check_boundary_ownership(loaded)


def test_owner_may_not_scope_the_boundary_it_owns(loaded: Roster) -> None:
    owned = "build-foundational-software"
    loaded.roles["platform"].scoped.append(roster.Scoped(owned, "anywhere at all"))
    with pytest.raises(roster.RosterError, match="also scopes it"):
        validate.check_boundary_ownership(loaded)


def test_missing_personality_binding_is_rejected(loaded: Roster) -> None:
    loaded.roles["director"].personalities.append("unbound")
    with pytest.raises(roster.RosterError, match="has no catalog binding"):
        validate.check_personality_bindings(loaded)


def test_a_role_with_no_personalities_is_rejected(loaded: Roster) -> None:
    loaded.roles["director"].personalities.clear()
    with pytest.raises(roster.RosterError, match="activates no personalities"):
        validate.check_personality_bindings(loaded)


def test_mismatched_definition_set_is_rejected(loaded: Roster) -> None:
    loaded.boundary_order.append("no-such-boundary")
    with pytest.raises(roster.RosterError, match="mismatched set"):
        validate.check_definition_set(loaded)


def test_a_role_deferring_an_unknown_boundary_is_rejected(loaded: Roster) -> None:
    loaded.roles["director"].defers.append("no-such-boundary")
    with pytest.raises(roster.RosterError, match="defers unknown boundary"):
        validate.check_definition_set(loaded)


def test_an_orphan_personality_is_rejected(loaded: Roster) -> None:
    orphan = dataclasses.replace(loaded.personalities["decisive"], name="orphan")
    loaded.personalities["orphan"] = orphan
    with pytest.raises(roster.RosterError, match="bound to no role"):
        validate.check_definition_set(loaded)


@pytest.mark.parametrize("bad", ["#ffffff", "#000000", "#a0a0a0"])
def test_out_of_band_personality_color_is_rejected(loaded: Roster, bad: str) -> None:
    loaded.personalities["decisive"] = dataclasses.replace(
        loaded.personalities["decisive"], color=bad
    )
    with pytest.raises(roster.RosterError):
        validate.check_personality_colors(loaded)


def test_out_of_band_colors_are_actually_out_of_band() -> None:
    """The parametrized colors above must fail `legible` for the stated reason.

    #a0a0a0 rather than #808080: mid gray lands at L 0.5983 and trips the
    lightness floor first, so it never exercises the chroma floor at all.
    """
    with pytest.raises(color.ColorError, match="lightness"):
        color.legible("#ffffff")
    with pytest.raises(color.ColorError, match="lightness"):
        color.legible("#000000")
    with pytest.raises(color.ColorError, match="too gray"):
        color.legible("#a0a0a0")
    color.legible("#5fa87a")


def test_an_overlong_role_body_is_rejected(loaded: Roster) -> None:
    role = loaded.roles["director"]
    frontmatter, body = validate.split_frontmatter(role.body, role.skill)
    role.body = f"---\n{frontmatter}\n---\n" + body + ("\n\nword" * 500)
    with pytest.raises(roster.RosterError, match="maximum is 1200"):
        validate.check_copy_contract(loaded)


def test_a_role_body_under_three_paragraphs_is_rejected(loaded: Roster) -> None:
    role = loaded.roles["director"]
    frontmatter, _ = validate.split_frontmatter(role.body, role.skill)
    role.body = f"---\n{frontmatter}\n---\n\n# Portfolio Director\n\n" + ("word " * 200)
    with pytest.raises(roster.RosterError, match="at least three paragraphs"):
        validate.check_copy_contract(loaded)


def test_a_skill_whose_frontmatter_names_another_skill_is_rejected(loaded: Roster) -> None:
    role = loaded.roles["director"]
    role.body = role.body.replace("name: role-director", "name: role-somebody-else", 1)
    with pytest.raises(roster.RosterError, match="does not declare name"):
        validate.check_skill_frontmatter(loaded)


def test_a_boundary_missing_a_side_is_rejected(loaded: Roster) -> None:
    boundary = loaded.boundaries["modify-live-backend"]
    loaded.boundaries["modify-live-backend"] = dataclasses.replace(
        boundary, body=boundary.body.replace(validate.BOUNDARY_DEFER_HEADING, "## Something else")
    )
    with pytest.raises(roster.RosterError, match="needs both"):
        validate.check_copy_contract(loaded)


def test_word_count_drops_a_leading_heading() -> None:
    assert validate.word_count("# Title\n\none two three") == 3
    assert validate.word_count("one two three") == 3


def test_paragraph_count_ignores_blank_runs() -> None:
    assert validate.paragraph_count("a\n\n\n\nb") == 2


def test_unsupported_model_tier_is_refused(loaded: Roster) -> None:
    from checks import compose

    assert "oss" not in loaded.roles["director"].supported_model_tiers
    with pytest.raises(ValueError, match="does not support model tier"):
        compose.compose(loaded, "director", "oss", pathlib.Path("/tmp/zq49-unreachable"))


def test_a_thin_role_body_is_rejected(loaded: Roster) -> None:
    role = loaded.roles["director"]
    frontmatter, _ = validate.split_frontmatter(role.body, role.skill)
    role.body = f"---\n{frontmatter}\n---\n\n# Portfolio Director\n\n" + ("word " * 50)
    with pytest.raises(roster.RosterError, match="minimum is 140"):
        validate.check_prose_floors(loaded)


def test_a_thin_personality_body_is_rejected(loaded: Roster) -> None:
    """Go enforces this floor and the Python port dropped it. #7033."""
    personality = loaded.personalities["grounded"]
    frontmatter, _ = validate.split_frontmatter(personality.body, personality.skill)
    thin = f"---\n{frontmatter}\n---\n\n# Grounded\n\n" + ("word " * 40)
    loaded.personalities["grounded"] = dataclasses.replace(personality, body=thin)
    with pytest.raises(roster.RosterError, match="minimum is 120"):
        validate.check_prose_floors(loaded)


def test_a_thin_boundary_side_is_rejected(loaded: Roster) -> None:
    """The thinnest shipped side sits one word above this floor."""
    boundary = loaded.boundaries["modify-live-backend"]
    frontmatter, body = validate.split_frontmatter(boundary.body, boundary.skill)
    head, _, _ = body.partition(validate.BOUNDARY_DEFER_HEADING)
    thin = (
        f"---\n{frontmatter}\n---\n"
        + head
        + validate.BOUNDARY_DEFER_HEADING
        + "\n\nword word word\n"
    )
    loaded.boundaries["modify-live-backend"] = dataclasses.replace(boundary, body=thin)
    with pytest.raises(roster.RosterError, match="defer side has 3 words, minimum is 80"):
        validate.check_prose_floors(loaded)


def test_the_ceiling_check_no_longer_carries_the_role_floor(loaded: Roster) -> None:
    """Go splits them: ceilings bind every package, floors bind the shipped roster."""
    role = loaded.roles["director"]
    frontmatter, _ = validate.split_frontmatter(role.body, role.skill)
    role.body = f"---\n{frontmatter}\n---\n\na\n\nb\n\nc\n"
    validate.check_copy_contract(loaded)
    with pytest.raises(roster.RosterError, match="minimum is 140"):
        validate.check_prose_floors(loaded)
