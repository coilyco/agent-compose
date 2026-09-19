"""Self-contained compose tests, with no Go engine in reach.

The differential oracle against Go lives in agent-compose, which is the only
place both engines exist. These are the checks that must keep working after
agent-compose#339 deletes it, so they assert shape and internal consistency
rather than bytes against a reference.
"""

from __future__ import annotations

import json
import pathlib

import pytest

from checks import compose, roster, snapshot
from checks.roster import Roster


@pytest.fixture(scope="module")
def loaded() -> Roster:
    return roster.load()


def _composable() -> list[str]:
    """Role order minus the archived, which compose refuses by design."""
    loaded_roster = roster.load()
    return [n for n in loaded_roster.role_order if not loaded_roster.roles[n].archived]


def _cases() -> list[tuple[str, str]]:
    loaded_roster = roster.load()
    return [
        (name, tier)
        for name in _composable()
        for tier in loaded_roster.roles[name].supported_model_tiers
    ]


@pytest.mark.parametrize(("role", "tier"), _cases())
def test_native_bundle_is_internally_consistent(
    loaded: Roster,
    tmp_path: pathlib.Path,
    role: str,
    tier: str,
) -> None:
    out = compose.compose(loaded, role, tier, tmp_path / "bundle")
    manifest = json.loads((out / "manifest.json").read_text())
    trace = json.loads((out / "trace.json").read_text())

    assert manifest["model_tier"] == tier
    assert manifest["role_skill"] == f"role-{role}"
    selected = [
        d["subject"].removeprefix("skill:")
        for d in trace["decisions"]
        if d["kind"] == "skill" and d["outcome"] == "selected"
    ]
    on_disk = sorted(p.parent.name for p in out.rglob("SKILL.md"))
    assert sorted(selected) == on_disk

    provider = trace["providers"][0]
    assert provider["skills"] == len(selected)
    measured = sum(p.stat().st_size for p in out.rglob("SKILL.md"))
    assert provider["context_bytes"] == measured
    assert provider["approximate_tokens"] == (measured + 3) // 4
    assert manifest["delivery"]["body_bytes"] == len(
        (out / "content" / "instructions.md").read_bytes()
    )


@pytest.mark.parametrize("role", _composable())
def test_compiled_bundle_carries_every_body(
    loaded: Roster,
    tmp_path: pathlib.Path,
    role: str,
) -> None:
    out = compose.compose(loaded, role, "frontier", tmp_path / "bundle", "compiled")
    document = (out / "delivery" / "compiled.md").read_text()
    assert document.startswith((out / "content" / "instructions.md").read_text())
    for body in out.rglob("SKILL.md"):
        assert body.read_text() in document


def test_every_role_owns_exactly_one_boundary(loaded: Roster) -> None:
    """The allocation rule the whole roster language rests on."""
    for name in loaded.role_order:
        owned = [b for b, spec in loaded.boundaries.items() if spec.owner == name]
        assert len(owned) <= 1, f"{name} owns {owned}"
    owners = {spec.owner for spec in loaded.boundaries.values()}
    assert owners <= set(loaded.roles)


def test_snapshot_round_trips_through_json(loaded: Roster) -> None:
    text = snapshot.dumps(loaded)
    parsed = json.loads(text)
    assert parsed["role_order"] == loaded.role_order
    assert set(parsed["roles"]) == set(loaded.roles)
    for name in loaded.role_order:
        assert parsed["roles"][name]["favorite_color"] == loaded.roles[name].favorite_color


def test_favorite_colors_are_distinct(loaded: Roster) -> None:
    # A color_twin is meant to collide with its source, not be told apart
    # from it - agent-compose#7847.
    colors = [
        loaded.roles[n].favorite_color for n in loaded.role_order if not loaded.roles[n].color_twin
    ]
    assert len(set(colors)) == len(colors)


def test_a_guardrail_reaches_the_bundle_both_eagerly_and_as_a_skill(tmp_path: pathlib.Path) -> None:
    """The defect that made the primitive first-class. As a field on a role it derived
    eight board cases and entered no bundle at all, so four seats were graded against an
    instruction none of them was ever given. Nothing failed, because nothing looked."""
    loaded = roster.load()
    out = compose.compose(loaded, "science", "frontier", tmp_path / "bundle")

    guardrail = loaded.guardrails[loaded.roles["science"].guardrail]
    instructions = (out / "content" / "instructions.md").read_text()
    assert guardrail.card.strip() in instructions
    assert guardrail.skill in instructions
    assert [p.parent.name for p in out.rglob("SKILL.md") if p.parent.name == guardrail.skill]

    manifest = json.loads((out / "manifest.json").read_text())
    assert manifest["guardrail"] == "provable-results"
    ids = [entry["id"] for entry in manifest["content"]]
    assert f"{loaded.source}:skill:{guardrail.skill}" in ids


def test_a_role_without_a_guardrail_renders_no_guardrail_section(tmp_path: pathlib.Path) -> None:
    """The negative control. Three roles are deliberately without one, so the eager
    section has to be absent rather than empty, and the skill must not be emitted."""
    loaded = roster.load()
    assert not loaded.roles["platform"].guardrail
    out = compose.compose(loaded, "platform", "frontier", tmp_path / "bundle")

    assert "## Guardrail" not in (out / "content" / "instructions.md").read_text()
    assert not [p for p in out.rglob("SKILL.md") if p.parent.name.startswith("guardrail-")]
    assert "guardrail" not in json.loads((out / "manifest.json").read_text())
