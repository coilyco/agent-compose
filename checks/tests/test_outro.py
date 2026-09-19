"""Every role closes with something, and it is not melded.

Role only is the departure from agent-compose#359, which asked for role and
personality. Melded, science reads as three sentences, and a closing banner is
read in half a second by somebody already leaving.
"""

from __future__ import annotations

import pytest

from checks import render, roster
from checks.roster import Roster


@pytest.fixture(scope="module")
def shipped() -> Roster:
    return roster.load()


def test_every_role_closes_both_ways(shipped: Roster) -> None:
    for name in shipped.role_order:
        outro = shipped.roles[name].outro
        assert outro is not None, f"{name} has no outro"
        assert outro.clean and outro.failure, f"{name} is missing a variant"


def test_the_two_variants_differ(shipped: Roster) -> None:
    """A close that reads the same either way is not carrying tone."""
    for name in shipped.role_order:
        outro = shipped.roles[name].outro
        assert outro is not None
        assert outro.clean != outro.failure, f"{name} closes the same either way"


def test_no_personality_carries_one(shipped: Roster) -> None:
    """Not melded, so the banner stays one sentence."""
    for personality in shipped.personalities.values():
        assert not hasattr(personality, "outro") or getattr(personality, "outro", None) is None


def test_it_stays_off_the_identity_card(shipped: Roster) -> None:
    """The outro is a closing surface, not doctrine the seat reads at load."""
    card = render.identity_card(shipped, "science")
    outro = shipped.roles["science"].outro
    assert outro is not None
    assert outro.clean not in card
