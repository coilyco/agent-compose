"""The field reference, rendered from the dataclasses rather than typed beside them.

A hand-written reference is a second copy of `housecast/roster.py`, and this repo
has already been bitten by that shape: the version lived in two places and drifted
twice inside one migration, with the tag saying 0.1.3 and the metadata 0.1.1 and
nothing failing. So there is no committed copy at all. `housecast fields` renders
it on demand, off the loaded types and the minimal roster.
"""

from __future__ import annotations

import dataclasses
import pathlib

import yaml

from checks import roster

MINIMAL = pathlib.Path(__file__).parent / "data" / "minimal-roster.yaml"

# Top-level first, then what it contains, so the listing reads the way the YAML
# is entered rather than the way the module happens to define it.
ORDER = (
    roster.Roster,
    roster.Role,
    roster.Personality,
    roster.Boundary,
    roster.Act,
    roster.Voice,
    roster.Emblem,
    roster.Outro,
    roster.Guardrail,
    roster.Seat,
    roster.Scoped,
    roster.Adjacent,
)

# Required-key status comes off the minimal roster, not the dataclass default:
# `Role.scoped` has no default and the loader still reads it with `.get`.
SECTIONS: dict[str, str | None] = {
    "Roster": None,
    "Role": "roles",
    "Personality": "personalities",
    "Boundary": "boundaries",
}

# A field whose YAML key is not its own name, with the reason. A test asserts
# none of these is also a key, so the note cannot outlive what it describes.
CARRIED_DIFFERENTLY = {
    ("Roster", "raw"): "the file's own bytes",
    ("Role", "name"): "the mapping key under `roles`",
    ("Role", "favorite_color"): "derived from the meld by resolve_favorite_colors",
    ("Role", "identity_name"): "identity.name",
    ("Role", "identity_pronouns"): "identity.pronouns",
    ("Personality", "name"): "the mapping key under `personalities`",
    ("Boundary", "name"): "the mapping key under `boundaries`",
}

PREAMBLE = """Every field the loader builds, off housecast/roster.py.

A required key is one the loader raises without. For the four keyed types that
is measured against housecast/data/minimal-roster.yaml, whose own test deletes
each key in turn to prove none is spare. The nested types report their dataclass
default instead, which is where the loader takes theirs."""


def required_keys(cls_name: str) -> set[str] | None:
    """The required keys, read off the roster proven to carry only those."""
    if cls_name not in SECTIONS:
        return None
    document = yaml.safe_load(MINIMAL.read_text(encoding="utf-8"))
    section = SECTIONS[cls_name]
    if section is None:
        return set(document)
    return {key for spec in document[section].values() for key in spec}


def _note(cls_name: str, spec: dataclasses.Field[object], keys: set[str] | None) -> str:
    carried = CARRIED_DIFFERENTLY.get((cls_name, spec.name))
    if carried:
        return f"not a key of its own, read from {carried}"
    if keys is not None:
        return "required key" if spec.name in keys else "optional key"
    defaulted = (
        spec.default is not dataclasses.MISSING or spec.default_factory is not dataclasses.MISSING
    )
    return "optional key" if defaulted else "required key"


def render() -> str:
    lines = [PREAMBLE, ""]
    for cls in ORDER:
        lines.append(f"{cls.__name__}")
        keys = required_keys(cls.__name__)
        for spec in dataclasses.fields(cls):
            lines.append(f"  {spec.name}: {spec.type} - {_note(cls.__name__, spec, keys)}")
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"
