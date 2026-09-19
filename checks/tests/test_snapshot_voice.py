"""person.json must carry voice, because the board's voice tier reads it there.

The tier was registered in the profile, coded in `evalkit.matrix`, and wired
into `derive`, and it still produced nothing: `snapshot.py` emitted every field
evalkit read except this one, so `spec.get("voice")` was None for all seven
seats and every voice case was skipped before it was built. Nothing failed. The
board just reported 91 cases instead of 105, and a tier that derives zero looks
exactly like a tier that does not exist.
"""

from __future__ import annotations

import json
from typing import Any

import pytest

from checks import roster, snapshot
from checks.roster import Roster


@pytest.fixture(scope="module")
def shipped() -> Roster:
    return roster.load()


@pytest.fixture(scope="module")
def emitted(shipped: Roster) -> dict[str, Any]:
    payload: dict[str, Any] = json.loads(snapshot.dumps(shipped))
    return payload


def test_every_role_emits_its_voice(emitted: dict[str, Any], shipped: Roster) -> None:
    for name in shipped.role_order:
        assert emitted["roles"][name].get("voice"), f"{name} lost its voice in the snapshot"


def test_every_personality_emits_its_voice(emitted: dict[str, Any], shipped: Roster) -> None:
    for name, personality in shipped.personalities.items():
        if personality.voice is None:
            continue
        assert emitted["personalities"][name].get("voice"), f"{name} lost its voice"


def test_the_banks_survive_the_round_trip(emitted: dict[str, Any], shipped: Roster) -> None:
    """A bank flattened to [] would derive a diction case with an empty rule."""
    science = shipped.roles["science"].voice
    assert science is not None
    carried = emitted["roles"]["science"]["voice"]
    assert carried["prefer"] == list(science.prefer)
    assert carried["avoid"] == list(science.avoid)
    assert carried["tell"] == science.tell
    assert carried["person"] == science.person


def test_empty_fields_are_dropped_rather_than_emitted_empty(shipped: Roster) -> None:
    """An absent bank must be absent, so nothing derives a case with no rule."""
    bare = snapshot._voice(roster.Voice(summary="only a summary"))
    assert bare == {"summary": "only a summary"}


def test_acts_are_emitted_because_the_charter_reads_them(emitted: dict[str, Any]) -> None:
    """`snapshot.py` emits what evalkit reads, and evalkit.roster now renders acts.

    Decided by Kai on housecast#7184: a grader judging a boundary case sees what
    the seat is supposed to run. Reverting the renderer without reverting this
    silently returns the charter to owns/defers only.
    """
    acts = emitted["roles"]["science"]["acts"]
    assert acts, "science declares acts in the roster and the snapshot must carry them"
    assert all(a["text"] for a in acts)


def test_the_act_projection_is_not_narrowed_to_what_the_charter_renders(
    emitted: dict[str, Any],
) -> None:
    """`tool` rides along unread on purpose.

    agent-compose#1848 is the precedent: a projection narrowed to the keys one
    consumer reads dropped `creature` outright, and nothing noticed until it was
    measured against the live roster.
    """
    assert all(a["tool"] for a in emitted["roles"]["science"]["acts"])
