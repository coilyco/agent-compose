"""A seat's legal name is authored, never derived from the model.

Deriving it would let identity assert a model selection, and the identity
contract grants nothing (agent-compose#396). Absent stays absent.
"""

from __future__ import annotations

import pytest

from checks import roster
from checks.roster import Roster


@pytest.fixture(scope="module")
def shipped() -> Roster:
    return roster.load()


def test_every_seat_names_itself(shipped: Roster) -> None:
    for name in shipped.role_order:
        for seat in shipped.roles[name].seats:
            assert seat.legal_name, f"{name} seat {seat.key} has no legal name"


def test_a_non_model_seat_answers_with_its_own_name(shipped: Roster) -> None:
    """mixpost is not a model, so it is not Claude."""
    seats = {s.key: s.legal_name for s in shipped.roles["advocate"].seats}
    assert seats["mixpost"] == "Mixpost"
    assert seats["discord"] == "Discord"
    assert seats["claude"] == "Claude"


def test_absent_stays_absent(shipped: Roster) -> None:
    """The loader must not backfill, or identity starts asserting."""
    seat = shipped.roles["science"].seats[0]
    bare = type(seat)(key=seat.key, harness=seat.harness)
    assert bare.legal_name is None
