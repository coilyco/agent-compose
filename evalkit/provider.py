"""The role roster as one provider behind the core's interface.

This adapts rather than reimplements. `derive` and `to_entity_roster` still hold
every rule about what a roster implies, and the board they produce through here
is byte-identical to the board they produce when called directly, which is the
acceptance criterion this module exists to satisfy.

It sits above `matrix` and `roster` rather than beside them, so the dependency
runs one way and no module here imports this one back.
"""

from __future__ import annotations

import json
from typing import TYPE_CHECKING, Any

from evalkit.matrix import derive
from evalkit.profile import PROFILE
from evalkit.roster import to_entity_roster

if TYPE_CHECKING:
    from pathlib import Path

    from housecast.grade.schema import Challenge, Profile


class RoleProvider:
    """Entities and challenges derived from a rendered person."""

    def __init__(self, person: dict[str, Any]) -> None:
        self._person = person

    @classmethod
    def from_path(cls, person: Path) -> RoleProvider:
        return cls(json.loads(person.read_text()))

    @property
    def name(self) -> str:
        return "role"

    def profile(self) -> Profile:
        return PROFILE

    def entities(self) -> dict[str, Any]:
        return to_entity_roster(self._person)

    def challenges(self, group: str = "tier") -> list[Challenge]:
        return derive(self._person, group)
