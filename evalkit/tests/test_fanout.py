from __future__ import annotations

from typing import Any

import pytest

from evalkit.fanout import FieldError, fanout

# One owner, one plain deferrer, and one scoped holder, which is the smallest
# roster that tells the three fan-out shapes apart.
ROSTER: dict[str, Any] = {
    "role_order": ["platform", "gamedev", "advocate"],
    "boundary_order": ["suggest-external-comms", "build-foundational-software"],
    "boundaries": {
        "suggest-external-comms": {
            "owner": "advocate",
            "summary": "Developer Advocate recommends communication, other roles keep records",
        },
        "build-foundational-software": {
            "owner": "platform",
            "summary": "Platform Engineer builds foundational software, other roles specify it",
        },
    },
    "roles": {
        "platform": {
            "display_name": "Platform Engineer",
            "purpose": "Build and land the foundational software.",
            "personalities": ["tenacious"],
            "boundaries": ["suggest-external-comms"],
        },
        "gamedev": {
            "display_name": "Game Developer",
            "purpose": "Ship the game.",
            "personalities": ["grounded"],
            "boundaries": ["build-foundational-software"],
            "scoped_boundaries": [
                {"name": "suggest-external-comms", "scope": "patch notes and nothing else"}
            ],
        },
        "advocate": {
            "display_name": "Developer Advocate",
            "purpose": "Speak outward.",
            "personalities": ["tenacious"],
            "boundaries": ["build-foundational-software"],
        },
    },
}


def test_a_summary_edit_reaches_every_role_carrying_that_boundary() -> None:
    reached = fanout(ROSTER, "boundary.suggest-external-comms.summary")

    assert set(reached) == {"platform-sec", "gamedev-sec", "advocate-sec"}


def test_a_purpose_edit_skips_the_role_s_own_scoped_pairs() -> None:
    """The autonomy pair does move: it renders from the purpose, which is the price
    of deriving autonomy without a roster field of its own.
    """
    # A scoped pair renders from the grant, not the purpose, so it does not
    # move. housecast#7173 over-counts these by every boundary the role touches.
    reached = fanout(ROSTER, "role.gamedev.purpose")

    assert reached == ["gamedev-bfs", "gamedev-aut"]
    assert "gamedev-sec" not in reached


def test_a_scoped_grant_is_the_only_per_pair_edit() -> None:
    reached = fanout(ROSTER, "role.gamedev.scoped.suggest-external-comms")

    assert reached == ["gamedev-sec"]


def test_an_owner_is_reached_through_its_own_summary() -> None:
    reached = fanout(ROSTER, "boundary.build-foundational-software.summary")

    assert "platform-bfs" in reached


@pytest.mark.parametrize(
    "spec",
    [
        "boundary.no-such-boundary.summary",
        "role.no-such-role.purpose",
        "role.platform.scoped.suggest-external-comms",
        "role.platform.display_name",
        "nonsense",
    ],
)
def test_a_field_naming_nothing_is_an_error(spec: str) -> None:
    # Never an empty fan-out: that reads as "this edit is safe" and is the most
    # expensive wrong answer available here.
    with pytest.raises(FieldError):
        fanout(ROSTER, spec)
