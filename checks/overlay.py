"""Append estate-specific acts onto a shipped roster, without redefining it.

The core roster stays runnable by a stranger, and a test refuses any act naming a
tool only one estate has. That refusal is what keeps the shipped bundle portable,
and it is also why the sharpest acts had nowhere to live: `aosguard ops kubectl` is
most of what the sysadmin own side actually does, and it cannot ship.

An overlay is where those go. Decided on agent-compose#393:

* **Append, never replace.** A seat on a host carrying both gets six acts per
  attribute, three portable and three estate. Replacing would give it three and
  drop the portable fallback, which is the half that still works when the estate
  tool is absent.
* **Per boundary side.** An appended boundary act names its side, because owning,
  scoping, and deferring are three different acts and the estate tools bite
  hardest on the own side.

The format carries acts and nothing else. Redefining a role, a personality, a
boundary, or a meld is not something an overlay is asked not to do - it is
something the format cannot express.
"""

from __future__ import annotations

import dataclasses
import pathlib
from typing import Any

import yaml

from checks.roster import Act, Roster, RosterError

SIDES = ("own", "scoped", "defer")
SECTIONS = ("roles", "personalities", "boundaries")


def _acts(spec: Any, where: str) -> list[Act]:
    if not isinstance(spec, list):
        raise RosterError(f"overlay {where}: expected a list of acts")
    acts = []
    for index, entry in enumerate(spec):
        if not isinstance(entry, dict):
            raise RosterError(f"overlay {where}[{index}]: expected a mapping")
        unknown = set(entry) - {"tool", "text"}
        if unknown:
            raise RosterError(f"overlay {where}[{index}]: unknown keys {sorted(unknown)}")
        for required in ("tool", "text"):
            if not entry.get(required):
                raise RosterError(f"overlay {where}[{index}]: {required} is required")
        acts.append(Act(tool=str(entry["tool"]), text=str(entry["text"])))
    return acts


def apply(roster: Roster, path: pathlib.Path | str) -> Roster:
    """Return a roster carrying the overlay's acts after its own.

    Every name is checked against the roster rather than tolerated, because a
    typo that silently appends nothing is the failure this whole mechanism
    exists to avoid: an estate act that never reaches the seat looks exactly
    like an estate act that was never written.
    """
    doc = yaml.safe_load(pathlib.Path(path).read_bytes()) or {}
    if not isinstance(doc, dict):
        raise RosterError("overlay: expected a mapping at the top level")

    unknown = set(doc) - {"overlay", *SECTIONS}
    if unknown:
        raise RosterError(f"overlay: unknown top-level keys {sorted(unknown)}")

    roles = dict(roster.roles)
    for name, spec in (doc.get("roles") or {}).items():
        if name not in roles:
            raise RosterError(f"overlay roles: {name!r} is not a role on this roster")
        extra = _acts(spec, f"roles.{name}")
        roles[name] = dataclasses.replace(roles[name], acts=[*roles[name].acts, *extra])

    personalities = dict(roster.personalities)
    for name, spec in (doc.get("personalities") or {}).items():
        if name not in personalities:
            raise RosterError(f"overlay personalities: {name!r} is not a personality")
        extra = _acts(spec, f"personalities.{name}")
        personalities[name] = dataclasses.replace(
            personalities[name], acts=[*personalities[name].acts, *extra]
        )

    boundaries = dict(roster.boundaries)
    for name, spec in (doc.get("boundaries") or {}).items():
        if name not in boundaries:
            raise RosterError(f"overlay boundaries: {name!r} is not a boundary")
        if not isinstance(spec, dict):
            raise RosterError(f"overlay boundaries.{name}: expected a mapping of side to acts")
        bad = set(spec) - set(SIDES)
        if bad:
            raise RosterError(
                f"overlay boundaries.{name}: unknown sides {sorted(bad)}, want {list(SIDES)}"
            )
        sided: list[Act] = []
        for side in SIDES:
            if side not in spec:
                continue
            sided += [
                dataclasses.replace(act, side=side)
                for act in _acts(spec[side], f"boundaries.{name}.{side}")
            ]
        boundaries[name] = dataclasses.replace(
            boundaries[name], acts=[*boundaries[name].acts, *sided]
        )

    return dataclasses.replace(
        roster, roles=roles, personalities=personalities, boundaries=boundaries
    )
