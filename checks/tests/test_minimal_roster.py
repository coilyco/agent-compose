"""The smallest roster that loads, and the proof that it is the smallest.

Deleting any key in `housecast/data/minimal-roster.yaml` has to fail. That is what keeps
the example honest: a field nothing needs cannot sit in it unnoticed, and a
newcomer copying it is copying the floor rather than somebody's roster.
"""

from __future__ import annotations

import copy
import pathlib
from typing import Any

import pytest
import yaml

from checks import roster as roster_module
from checks.roster import RosterError, load

MINIMAL = pathlib.Path(roster_module.DATA).parent / "minimal-roster.yaml"


def _paths(node: Any, prefix: tuple[str, ...] = ()) -> list[tuple[str, ...]]:
    """Every mapping key in the document, deepest last."""
    found = []
    if isinstance(node, dict):
        for key, value in node.items():
            found.append((*prefix, key))
            found += _paths(value, (*prefix, key))
    elif isinstance(node, list):
        for index, value in enumerate(node):
            found += _paths(value, (*prefix, str(index)))
    return found


def _without(document: dict[str, Any], path: tuple[str, ...]) -> dict[str, Any]:
    pruned = copy.deepcopy(document)
    node: Any = pruned
    for step in path[:-1]:
        node = node[int(step)] if isinstance(node, list) else node[step]
    del node[path[-1]]
    return pruned


def test_the_minimal_roster_loads() -> None:
    roster = load(MINIMAL)
    assert roster.role_order == ["reader"]
    assert roster.roles["reader"].active_boundaries(roster.boundaries) == ["boundary-read-the-log"]
    grounded = roster.personalities["personality-grounded"]
    assert roster.roles["reader"].favorite_color == grounded.color


@pytest.mark.parametrize("path", _paths(yaml.safe_load(MINIMAL.read_text())), ids=".".join)
def test_every_key_in_it_is_load_bearing(path: tuple[str, ...], tmp_path: pathlib.Path) -> None:
    """A key whose absence still loads is one the example should not be teaching."""
    document = yaml.safe_load(MINIMAL.read_text())
    candidate = tmp_path / "roster.yaml"
    candidate.write_text(yaml.safe_dump(_without(document, path)), encoding="utf-8")
    with pytest.raises((KeyError, TypeError, RosterError)):
        load(candidate)
