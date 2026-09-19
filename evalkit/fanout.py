"""Which case pairs a roster edit actually reaches.

A boundary target is generated, so an edit aimed at one pair fans out along
whichever roster field it touches, and only a scoped grant is per-pair. This
answers the question by perturbing one field and re-deriving the board, so the
answer cannot drift from matrix.py the way a second copy of the fan-out rules
would. See docs/evaluation.md and housecast#7173.

The field is addressed on the person snapshot, the same shape `derive` takes,
never on raw roster.yaml: `boundaries`/`scoped_boundaries` are the snapshot's
spelling of `defers`/`scoped` and only exist after housecast/snapshot.py.
"""

from __future__ import annotations

import argparse
import copy
import json
import sys
from pathlib import Path
from typing import Any

from evalkit.provider import RoleProvider

# Replaces the field outright rather than appending, because a summary is read
# only up to its first comma and an appended marker would show as no change.
SENTINEL = "\x01fanout-probe\x01"


class FieldError(ValueError):
    """A field spec that names nothing in this roster."""


def perturb(person: dict[str, Any], spec: str) -> dict[str, Any]:
    """A copy of the roster with the one addressed field replaced.

    Raises rather than returning the roster untouched, because an unchanged
    copy derives an empty fan-out, which reads as "this edit is safe" and is
    the most expensive wrong answer this module could give.
    """
    probe = copy.deepcopy(person)
    parts = spec.split(".")
    match parts:
        case ["boundary", name, "summary"]:
            if name not in probe.get("boundaries", {}):
                raise FieldError(f"no boundary {name!r} in this roster")
            probe["boundaries"][name]["summary"] = SENTINEL
        case ["role", name, "purpose"]:
            if name not in probe.get("roles", {}):
                raise FieldError(f"no role {name!r} in this roster")
            probe["roles"][name]["purpose"] = SENTINEL
        case ["role", name, "scoped", boundary]:
            if name not in probe.get("roles", {}):
                raise FieldError(f"no role {name!r} in this roster")
            grants = probe["roles"][name].get("scoped_boundaries") or []
            for entry in grants:
                if entry.get("name") == boundary:
                    entry["scope"] = SENTINEL
                    break
            else:
                raise FieldError(f"role {name!r} holds no scoped grant on {boundary!r}")
        case _:
            raise FieldError(
                f"unreadable field {spec!r}, want boundary.<name>.summary, "
                "role.<name>.purpose, or role.<name>.scoped.<boundary>"
            )
    return probe


def moved(person: dict[str, Any], spec: str) -> list[str]:
    """Every challenge id whose target text changes when this field changes."""
    probe = perturb(person, spec)  # first, so an unreadable field fails before any derive
    board = RoleProvider(person).challenges()
    before = {c.id: c.target for c in board}
    after = {c.id: c.target for c in RoleProvider(probe).challenges()}
    return [c.id for c in board if before[c.id] != after.get(c.id)]


def fanout(person: dict[str, Any], spec: str) -> list[str]:
    """The pair ids this field reaches, in board order.

    Only the boundary types carry a pair. A role-fit, personality, or voice
    challenge that moves is reported by `moved` and is absent here, so read the
    two together rather than treating this as the whole blast radius.
    """
    index = {c.id: c.pair_id for c in RoleProvider(person).challenges()}
    reached: list[str] = []
    for challenge_id in moved(person, spec):
        pair = index.get(challenge_id)
        if pair and pair not in reached:
            reached.append(pair)
    return reached


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Name the case pairs a roster edit reaches.")
    parser.add_argument("--roster", type=Path, required=True, help="person.json from the roster")
    parser.add_argument(
        "field",
        help="boundary.<name>.summary, role.<name>.purpose, or role.<name>.scoped.<boundary>",
    )
    args = parser.parse_args(argv)
    person = json.loads(args.roster.read_text())
    try:
        reached = fanout(person, args.field)
        changed = moved(person, args.field)
    except FieldError as error:
        print(f"fanout: {error}", file=sys.stderr)
        return 2
    print(f"{args.field} reaches {len(reached)} pair(s):")
    for pair in reached:
        print(f"  {pair}")
    paired = {c for pair in reached for c in (f"{pair}-in", f"{pair}-out")}
    if loose := [c for c in changed if c not in paired]:
        print(f"and {len(loose)} unpaired challenge(s):")
        for challenge_id in loose:
            print(f"  {challenge_id}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
