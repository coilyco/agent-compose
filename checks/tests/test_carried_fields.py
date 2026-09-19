"""Fields the #333 port dropped silently, and the round trip that proves they stay.

Role methods and seat channels are the sharpest of the gaps in agent-compose#373
precisely because no shipped role or seat carries either. Nothing renders
differently today, so the parity suite against Go cannot see the loss, and a
roster that added one would lose it with no error. These tests carry the fixture
the shipped roster does not.
"""

from __future__ import annotations

import pytest
import yaml

from checks import render, roster
from checks.roster import Roster

METHODS = ["adversarial-verification", "state-machine-verification"]
CHANNEL = "sirens-deep"


@pytest.fixture(scope="module")
def carrying(tmp_path_factory: pytest.TempPathFactory) -> Roster:
    """The shipped roster with methods and a channel injected onto one role."""
    doc = yaml.safe_load(roster.DATA.read_bytes())
    role = doc["roles"]["science"]
    role["methods"] = list(METHODS)
    role["seats"][0]["channel"] = CHANNEL
    path = tmp_path_factory.mktemp("roster") / "roster.yaml"
    path.write_text(yaml.safe_dump(doc, sort_keys=False))
    return roster.load(path)


def test_role_methods_survive_the_loader(carrying: Roster) -> None:
    assert carrying.roles["science"].methods == METHODS


def test_seat_channel_survives_the_loader(carrying: Roster) -> None:
    assert carrying.roles["science"].seats[0].channel == CHANNEL


def test_the_identity_card_renders_role_methods(carrying: Roster) -> None:
    """Go emits this line between the role skill and the boundaries."""
    card = render.identity_card(carrying, "science")
    assert "**Role methods // `adversarial-verification` // `state-machine-verification`**" in card
    skill_at = card.index("**Role skill //")
    methods_at = card.index("**Role methods //")
    assert skill_at < methods_at
    if "**Boundaries //" in card:
        assert methods_at < card.index("**Boundaries //")


# Negative control. Without the fields the card must not grow the line, or the
# test above passes on a renderer that emits it unconditionally.
def test_the_shipped_roster_renders_no_methods_line() -> None:
    loaded = roster.load()
    assert loaded.roles["science"].methods == []
    assert loaded.roles["science"].seats[0].channel is None
    assert "**Role methods //" not in render.identity_card(loaded, "science")


@pytest.fixture(scope="module")
def archived(tmp_path_factory: pytest.TempPathFactory) -> Roster:
    """The shipped roster with one role archived.

    These three tests used to read the flag off whichever shipped role happened
    to carry it, which was analyst and only analyst. Un-archiving that role
    deleted the coverage rather than failing it, so the fixture makes its own
    archived role the way the rest of this file makes its own methods and
    channel.

    It archives platform rather than analyst for the same reason, from the other
    direction. Analyst, psych and reporter were archived on 2026-09-15, which
    left the live half of these assertions with no subject. Platform is the seat
    least likely to retire, so the pair stays testable whichever way the roster
    moves.
    """
    doc = yaml.safe_load(roster.DATA.read_bytes())
    doc["roles"]["platform"]["archived"] = True
    path = tmp_path_factory.mktemp("archived") / "roster.yaml"
    path.write_text(yaml.safe_dump(doc, sort_keys=False))
    return roster.load(path)


def test_archived_defaults_false_and_survives_the_loader(archived: Roster) -> None:
    """Absent means live, so every roster authored before the field still loads."""
    assert roster.load().roles["platform"].archived is False
    assert archived.roles["platform"].archived is True


def test_the_projection_carries_archived(archived: Roster) -> None:
    """evalkit reads person.json rather than the YAML, so the flag has to cross."""
    from checks import snapshot

    assert snapshot.person_snapshot(archived)["roles"]["platform"]["archived"] is True
    assert snapshot.person_snapshot(roster.load())["roles"]["platform"]["archived"] is False


def test_compose_refuses_an_archived_role(archived: Roster) -> None:
    """The Go resolver refuses too, and the two engines owe the same semantics."""
    from checks import compose as compose_module

    with pytest.raises(ValueError, match="archived"):
        compose_module.compose(archived, "platform", "frontier", "/tmp/unused-bundle")
