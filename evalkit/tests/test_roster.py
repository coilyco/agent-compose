"""The annotator reads --roster with json.loads, so the projection must be JSON."""

from __future__ import annotations

import ast
import json
from pathlib import Path

from evalkit import roster
from evalkit.coverage import project

ROSTER = Path(__file__).resolve().parents[2] / "checks" / "data" / "roster.yaml"

PERSON = {
    "role_order": ["sysadmin", "tpm"],
    "boundaries": {"modify-live-backend": {"owner": "sysadmin"}},
    "roles": {
        "sysadmin": {
            "display_name": "Systems Administrator",
            "purpose": "Operate the real hosted systems.",
            "boundaries": ["suggest-external-comms"],
            "personalities": ["protective", "grounded"],
            "scoped_boundaries": [{"name": "build-foundational-software", "scope": "own estate"}],
            "adjacents": [{"role": "platform", "reason": "implementing the fix itself"}],
        },
        "tpm": {"display_name": "Portfolio Director", "purpose": "Decide what is next."},
    },
}


def test_projection_carries_what_the_annotator_renders() -> None:
    projected = roster.to_entity_roster(PERSON)
    assert projected["entity_order"] == ["sysadmin", "tpm"]
    spec = projected["entities"]["sysadmin"]
    assert spec["display_name"] == "Systems Administrator"
    assert spec["purpose"]
    joined = " ".join(spec["notes"])
    assert "owns: modify-live-backend" in joined
    assert "scoped build-foundational-software: own estate" in joined
    assert "defers: suggest-external-comms" in joined
    assert "traits: protective, grounded" in joined
    assert "adjacent platform: implementing the fix itself" in joined


def test_written_file_parses_as_json(tmp_path: Path) -> None:
    person = tmp_path / "person.json"
    person.write_text(json.dumps(PERSON))
    out = tmp_path / "entities.json"
    assert roster.main(["--person", str(person), "--out", str(out)]) == 0
    reloaded = json.loads(out.read_text())
    assert reloaded == roster.to_entity_roster(PERSON)


def _keys_read_off_a_role_spec() -> set[str]:
    """Every literal key this module reads off a role spec, by parsing it.

    Parsed rather than listed, so a read added tomorrow is covered tomorrow.
    """
    tree = ast.parse(Path(roster.__file__).read_text(encoding="utf-8"))
    found: set[str] = set()
    for node in ast.walk(tree):
        if (
            isinstance(node, ast.Call)
            and isinstance(node.func, ast.Attribute)
            and node.func.attr == "get"
            and isinstance(node.func.value, ast.Name)
            and node.func.value.id == "spec"
            and node.args
            and isinstance(node.args[0], ast.Constant)
            and isinstance(node.args[0].value, str)
        ):
            found.add(node.args[0].value)
        if (
            isinstance(node, ast.Subscript)
            and isinstance(node.value, ast.Name)
            and node.value.id == "spec"
            and isinstance(node.slice, ast.Constant)
            and isinstance(node.slice.value, str)
        ):
            found.add(node.slice.value)
    return found


def test_every_key_the_projection_reads_exists_on_the_snapshot() -> None:
    keys = _keys_read_off_a_role_spec()
    # Never a vacuous pass: rename the local and this finds nothing, then
    # reports success while asserting over an empty set. housecast#7183.
    assert {"boundaries", "scoped_boundaries", "purpose"} <= keys, keys

    roles = project(ROSTER)["roles"]
    missing = sorted({key for spec in roles.values() for key in keys if key not in spec})

    assert not missing, (
        f"evalkit/roster.py reads {missing} off a role spec and the person snapshot does "
        "not carry it. housecast/snapshot.py is the seam that renames roster.yaml's "
        "defers and scoped: rename one side only and this is what fails. Assert against "
        "the snapshot, never raw roster.yaml, where these are absent by design."
    )


def test_the_charter_carries_the_seat_s_acts() -> None:
    """What a grader is shown for a boundary case, decided on housecast#7184."""
    person = {
        "role_order": ["qa"],
        "boundaries": {},
        "roles": {
            "qa": {
                "display_name": "Quinn",
                "purpose": "checks things",
                "boundaries": ["ship-it"],
                "acts": [
                    {"tool": "wc", "text": "wc -l before calling it many"},
                    {"tool": "git", "text": "git rev-parse and quote the ref"},
                ],
            }
        },
    }

    notes = roster.to_entity_roster(person)["entities"]["qa"]["notes"]

    assert "act: wc -l before calling it many" in notes
    assert "act: git rev-parse and quote the ref" in notes
    # Acts sit with the work-shape block, after what the seat defers.
    assert notes.index("defers: ship-it") < notes.index("act: wc -l before calling it many")


def test_a_role_with_no_acts_gains_no_act_lines() -> None:
    person = {
        "role_order": ["qa"],
        "boundaries": {},
        "roles": {"qa": {"display_name": "Quinn", "purpose": "checks things"}},
    }

    notes = roster.to_entity_roster(person)["entities"]["qa"]["notes"]

    assert not [note for note in notes if note.startswith("act: ")]
