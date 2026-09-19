"""Acts, the primitive agent-compose#388 asked for.

An attribute phrased entirely as attitude does not fire, so every role,
personality, and boundary side names three things a seat can actually run. These
mirror the Go loader's checks: the parity suite proves the two engines render
identically, and it cannot prove the roster is complete.
"""

from __future__ import annotations

import pytest

from checks import render, roster
from checks.roster import Roster

ACTS_PER_ATTRIBUTE = 3
SIDES = ("own", "scoped", "defer")

# Surfaces that exist on one estate and nowhere else, since a named tool that
# is absent reads to a seat as an instruction already satisfied (agentic-os#1381).
ESTATE_PREFIXES = ("aosguard", "mcp__", "ward ", "acompose", "housecast")


@pytest.fixture(scope="module")
def loaded() -> Roster:
    return roster.load()


def test_every_role_and_personality_names_three_acts(loaded: Roster) -> None:
    for name in loaded.role_order:
        assert len(loaded.roles[name].acts) == ACTS_PER_ATTRIBUTE, name
    for name, binding in loaded.personalities.items():
        assert len(binding.acts) == ACTS_PER_ATTRIBUTE, name


def test_every_boundary_side_names_its_own_three(loaded: Roster) -> None:
    for name in loaded.boundary_order:
        boundary = loaded.boundaries[name]
        for side in SIDES:
            assert len(boundary.acts_for_side(side)) == ACTS_PER_ATTRIBUTE, f"{name}/{side}"


def test_an_act_uses_the_tool_it_names(loaded: Roster) -> None:
    """The structured field and the prose cannot drift apart.

    Without this the coverage check passes against text nobody can run.
    """
    everything = [
        *(a for name in loaded.role_order for a in loaded.roles[name].acts),
        *(a for b in loaded.personalities.values() for a in b.acts),
        *(a for b in loaded.boundaries.values() for a in b.acts),
    ]
    assert everything
    for act in everything:
        assert act.tool in act.text, act


def test_no_act_names_an_estate_only_tool(loaded: Roster) -> None:
    for act in [
        *(a for name in loaded.role_order for a in loaded.roles[name].acts),
        *(a for b in loaded.personalities.values() for a in b.acts),
        *(a for b in loaded.boundaries.values() for a in b.acts),
    ]:
        assert not act.tool.startswith(ESTATE_PREFIXES), act.tool


def test_the_card_carries_the_side_the_seat_holds(loaded: Roster) -> None:
    """A deferred boundary is a different act, never the owner's withheld."""
    boundary = loaded.boundaries["seek-external-validation"]
    card = render.instructions(loaded, "platform")
    # platform scopes this boundary, so it owes the scoped acts and not the own.
    for act in boundary.acts_for_side("scoped"):
        assert act.text in card
    for act in boundary.acts_for_side("own"):
        assert act.text not in card
