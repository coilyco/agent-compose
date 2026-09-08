"""Guard the one hazard of running two engines side by side.

housecast/data/roster.yaml restates seed/roster/data while both exist, so a
change to the Go data silently staled the YAML once already: the science seat was
retitled Applied Scientist and the only symptom was a byte-diff in a parity
test that named neither file. These tests fail on the drift itself, so the
message points at the file to regenerate.

They delete themselves with the Go side under #339.
"""

from __future__ import annotations

import json
import pathlib
import shutil
import subprocess
from typing import Any

import pytest
from housecast import roster
from housecast.roster import Roster

GO_DATA = pathlib.Path(__file__).resolve().parents[2] / "seed" / "roster" / "data"


@pytest.fixture(scope="module")
def loaded() -> Roster:
    return roster.load()


def _skills(loaded: Roster) -> list[tuple[str, str]]:
    entries = [(r.skill, r.body) for r in loaded.roles.values()]
    entries += [(p.skill, p.body) for p in loaded.personalities.values()]
    entries += [(b.skill, b.body) for b in loaded.boundaries.values()]
    return sorted(entries)


GONE = "the Go person data is gone, so drift cannot occur"


@pytest.mark.skipif(not GO_DATA.is_dir(), reason=GONE)
def test_every_skill_body_matches_the_go_source(loaded: Roster) -> None:
    stale = [
        skill
        for skill, body in _skills(loaded)
        if (GO_DATA / skill / "SKILL.md").read_text() != body
    ]
    assert not stale, f"regenerate housecast/data/roster.yaml, these drifted: {stale}"


@pytest.mark.skipif(not GO_DATA.is_dir(), reason=GONE)
def test_the_invariant_matches_the_go_source(loaded: Roster) -> None:
    assert loaded.invariant == (GO_DATA / "invariant" / "INVARIANT.md").read_text()


@pytest.mark.skipif(not GO_DATA.is_dir(), reason=GONE)
def test_every_go_role_is_present(loaded: Roster) -> None:
    go_roles = {p.name.removeprefix("role-") for p in GO_DATA.glob("role-*") if p.is_dir()}
    assert go_roles == set(loaded.roles)


@pytest.mark.skipif(not GO_DATA.is_dir(), reason=GONE)
def test_every_go_personality_is_present(loaded: Roster) -> None:
    go_names = {p.name.removeprefix("personality-") for p in GO_DATA.glob("personality-*")}
    assert go_names == set(loaded.personalities)


REPO = pathlib.Path(__file__).resolve().parents[2]


@pytest.fixture(scope="module")
def go_snapshot(tmp_path_factory: pytest.TempPathFactory) -> dict[str, Any]:
    """The Go engine's own view of the roster, which the YAML must agree with."""
    if shutil.which("go") is None:
        pytest.skip("no Go toolchain, so the snapshot is unavailable")
    binary = tmp_path_factory.mktemp("go") / "agent-compose"
    subprocess.run(
        ["go", "build", "-o", str(binary), "./cmd/agent-compose"],
        cwd=REPO,
        check=True,
        capture_output=True,
    )
    out = tmp_path_factory.mktemp("roster") / "roster"
    subprocess.run([str(binary), "roster", "--out", str(out)], check=True, capture_output=True)
    loaded: dict[str, Any] = json.loads((out / "person.json").read_text())
    return loaded


def test_role_metadata_matches_the_go_snapshot(loaded: Roster, go_snapshot: dict[str, Any]) -> None:
    """Body drift is not the only drift: the science retitle moved display_name too."""
    stale = []
    assert loaded.role_order == go_snapshot["role_order"]
    for name, role in loaded.roles.items():
        go_role = go_snapshot["roles"][name]
        for field, mine, theirs in (
            ("display_name", role.display_name, go_role["display_name"]),
            ("purpose", role.purpose, go_role["purpose"]),
            ("skill", role.skill, go_role["skill"]),
            ("skill_source", role.skill_source, go_role["skill_source"]),
            ("archived", role.archived, go_role.get("archived", False)),
            (
                "adjacents",
                [[a.role, a.reason] for a in role.adjacents],
                [[a["role"], a["reason"]] for a in go_role.get("adjacents") or []],
            ),
            ("stance", role.stance, go_role["stance"]),
            ("tiers", role.supported_model_tiers, go_role["supported_model_tiers"]),
            ("defers", role.defers, go_role["boundaries"]),
            ("personalities", role.personalities, go_role["personalities"]),
            (
                "identity",
                [role.identity_name, role.identity_pronouns],
                [go_role["identity"]["name"], go_role["identity"]["pronouns"]],
            ),
            (
                "seats",
                [[s.key, s.harness, s.tier] for s in role.seats],
                [[s.get("key"), s["harness"], s.get("tier")] for s in go_role["seats"]],
            ),
            (
                "scoped",
                [[s.name, s.scope] for s in role.scoped],
                [[s["name"], s["scope"]] for s in go_role.get("scoped_boundaries") or []],
            ),
        ):
            if mine != theirs:
                stale.append(f"{name}.{field}")
    assert not stale, f"regenerate housecast/data/roster.yaml, these drifted: {stale}"


def test_boundary_and_personality_metadata_matches(
    loaded: Roster,
    go_snapshot: dict[str, Any],
) -> None:
    stale = []
    for name, boundary in loaded.boundaries.items():
        go_boundary = go_snapshot["boundaries"][name]
        if [boundary.skill, boundary.owner, boundary.summary] != [
            go_boundary["skill"],
            go_boundary["owner"],
            go_boundary["summary"],
        ]:
            stale.append(f"boundary {name}")
    for name, personality in loaded.personalities.items():
        go_personality = go_snapshot["personalities"][name]
        if [
            personality.skill,
            personality.color,
            personality.motif,
            personality.emblem.names,
            personality.emblem.emoji,
        ] != [
            go_personality["skill"],
            go_personality["color"],
            go_personality["motif"],
            go_personality["emblem"]["names"],
            go_personality["emblem"]["emoji"],
        ]:
            stale.append(f"personality {name}")
    assert not stale, f"regenerate housecast/data/roster.yaml, these drifted: {stale}"
