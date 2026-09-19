"""Project the person snapshot into the entity roster the annotator renders.

The shared layer prints an entity's charter and knows nothing about how this
deployment composes one, so owns, defers, scoped, and traits are spelled here
rather than there. See docs/evaluation.md.

The projection is JSON because `housecast grade annotate` reads --roster with
json.loads. It is a temp-dir handoff rather than a committed artifact, so the
repo rule preferring YAML for anything a human reads does not reach it.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def to_entity_roster(person: dict[str, Any]) -> dict[str, Any]:
    order = list(person.get("role_order", []))
    boundaries = person.get("boundaries", {})
    entities: dict[str, Any] = {}
    for name in order:
        spec = person["roles"][name]
        notes: list[str] = []
        owned = [b for b, meta in boundaries.items() if meta.get("owner") == name]
        if owned:
            notes.append("owns: " + ", ".join(sorted(owned)))
        scoped = spec.get("scoped_boundaries") or []
        for entry in scoped:
            notes.append(f"scoped {entry['name']}: {entry['scope']}")
        if spec.get("boundaries"):
            notes.append("defers: " + ", ".join(spec["boundaries"]))
        # One line each rather than joined: act texts run to 189 characters and
        # carry their own commas. `tool` must already appear inside `text`.
        for act in spec.get("acts", []):
            notes.append(f"act: {act['text']}")
        if spec.get("personalities"):
            notes.append("traits: " + ", ".join(spec["personalities"]))
        for adjacent in spec.get("adjacents", []):
            notes.append(f"adjacent {adjacent['role']}: {adjacent['reason']}")
        entities[name] = {
            "display_name": spec.get("display_name", name),
            "purpose": spec.get("purpose", ""),
            "notes": notes,
        }
    return {"entity_order": order, "entities": entities}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Project person.json into an entity roster.")
    parser.add_argument("--person", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args(argv)

    projected = to_entity_roster(json.loads(args.person.read_text()))
    args.out.write_text(json.dumps(projected, indent=2, ensure_ascii=False) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
