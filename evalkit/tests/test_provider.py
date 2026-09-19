"""A-3: the role board derived through the provider interface does not move.

The interface is worth nothing if fitting the role deriver behind it changed a
single byte of what the board already produces. These compare the two paths
directly rather than trusting that an adapter which only forwards cannot drift.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

import pytest
import yaml
from housecast.grade.provider import Provider

from evalkit.coverage import project
from evalkit.matrix import derive
from evalkit.profile import PROFILE
from evalkit.provider import RoleProvider
from evalkit.roster import to_entity_roster

# The shipped roster rather than a stub, because byte-identity on a three-role
# fixture would not exercise the ordering the `role` group applies.
PERSON = project(Path("checks/data/roster.yaml"))


def _serialise(challenges: list[Any]) -> str:
    payload = [c.model_dump(mode="json", exclude_none=True) for c in challenges]
    return yaml.safe_dump({"challenges": payload}, sort_keys=False, width=100)


@pytest.mark.parametrize("group", ["tier", "role"])
def test_board_through_the_provider_is_byte_identical(group: str) -> None:
    direct = _serialise(derive(PERSON, group))
    through = _serialise(RoleProvider(PERSON).challenges(group))
    assert through == direct


def test_entities_through_the_provider_are_byte_identical() -> None:
    direct = json.dumps(to_entity_roster(PERSON), indent=2, ensure_ascii=False)
    through = json.dumps(RoleProvider(PERSON).entities(), indent=2, ensure_ascii=False)
    assert through == direct


def test_profile_is_the_same_object() -> None:
    assert RoleProvider(PERSON).profile() is PROFILE


def test_role_provider_satisfies_the_protocol() -> None:
    assert isinstance(RoleProvider(PERSON), Provider)


def test_the_board_under_test_is_not_empty() -> None:
    """Negative control: byte-identity between two empty lists proves nothing."""
    board = RoleProvider(PERSON).challenges()
    assert len(board) > 20
    assert len({c.entity for c in board}) > 1
