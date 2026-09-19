"""The field reference is rendered, so there is no second copy to drift.

`pyproject.toml` records the version single-sourced from the package because
keeping it in two places drifted twice inside one migration. A hand-written
field reference is that shape again, so nothing here is committed: these assert
that what `reference.render()` renders still describes the loader.
"""

from __future__ import annotations

import dataclasses

import pytest
import yaml

from checks import reference, roster

MINIMAL = reference.MINIMAL


def test_every_field_reaches_the_listing() -> None:
    rendered = reference.render()
    for cls in reference.ORDER:
        assert f"\n{cls.__name__}\n" in f"\n{rendered}"
        for spec in dataclasses.fields(cls):
            assert f"  {spec.name}: " in rendered


def test_every_dataclass_reaches_the_listing() -> None:
    """A new type in roster.py is a type the reference would otherwise omit."""
    defined = {
        obj.__name__
        for obj in vars(roster).values()
        if isinstance(obj, type)
        and dataclasses.is_dataclass(obj)
        and obj.__module__ == roster.__name__
    }
    assert defined == {cls.__name__ for cls in reference.ORDER}


@pytest.mark.parametrize(
    ("cls", "section"),
    [
        (roster.Roster, None),
        (roster.Role, "roles"),
        (roster.Personality, "personalities"),
        (roster.Boundary, "boundaries"),
    ],
    ids=lambda value: value.__name__ if isinstance(value, type) else str(value),
)
def test_a_field_carried_under_another_key_is_not_also_a_key(
    cls: type, section: str | None
) -> None:
    """Both halves would otherwise drift: the note stays while the loader moves."""
    document = yaml.safe_load(MINIMAL.read_text())
    keys = (
        set(document)
        if section is None
        else {key for spec in document[section].values() for key in spec}
    )
    declared = {field for (name, field) in reference.CARRIED_DIFFERENTLY if name == cls.__name__}
    assert not declared & keys, f"{cls.__name__}: declared as carried differently, but is a key"


def test_every_note_names_a_field_that_exists() -> None:
    for name, field in reference.CARRIED_DIFFERENTLY:
        cls = getattr(roster, name)
        assert field in {spec.name for spec in dataclasses.fields(cls)}, (
            f"{name}.{field} is noted in the reference but is not a field"
        )
