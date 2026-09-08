"""Differential test against the Go engine, which is agent-compose#333's bar.

The oracle disappears when #339 deletes the Go side, so every parity question
has to be settled while this test can still run. It skips rather than fails
where the Go toolchain is absent, and the skip is loud in the CI log.
"""

from __future__ import annotations

import json
import pathlib
import shutil
import subprocess

import pytest
from housecast import compose, roster
from housecast.roster import Roster

REPO = pathlib.Path(__file__).resolve().parents[2]
TIERS = ("frontier", "commodity", "oss")


@pytest.fixture(scope="session")
def go_binary(tmp_path_factory: pytest.TempPathFactory) -> pathlib.Path:
    if shutil.which("go") is None:
        pytest.skip("no Go toolchain, so the differential oracle is unavailable")
    target: pathlib.Path = tmp_path_factory.mktemp("go") / "agent-compose"
    subprocess.run(
        ["go", "build", "-o", str(target), "./cmd/agent-compose"],
        cwd=REPO,
        check=True,
        capture_output=True,
    )
    return target


@pytest.fixture(scope="session")
def loaded() -> Roster:
    return roster.load()


def _go_compose(
    go_binary: pathlib.Path,
    tmp_path: pathlib.Path,
    role: str,
    tier: str,
    delivery: str = "native-skills",
) -> pathlib.Path:
    request = tmp_path / "request.kdl"
    request.write_text(
        f'compose {{\n    role "{role}"\n    delivery "{delivery}"\n    model-tier "{tier}"\n}}\n'
    )
    out = tmp_path / "go-bundle"
    subprocess.run(
        [str(go_binary), "compose", "--out", str(out), str(request)],
        check=True,
        capture_output=True,
    )
    composed: pathlib.Path = next(p for p in out.iterdir() if p.is_dir())
    return composed


def _tree(root: pathlib.Path) -> dict[str, bytes]:
    files = sorted(p for p in root.rglob("*") if p.is_file())
    return {str(p.relative_to(root)): p.read_bytes() for p in files}


def _composable() -> list[str]:
    """Role order minus the archived, which both engines refuse to compose.

    Parity over a refused role has nothing to compare. The refusal itself is
    covered on each side: TestResolveRefusesAnArchivedRole in Go, and
    test_compose_refuses_an_archived_role in housecast.
    """
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
def test_bundle_is_byte_identical_to_go(
    go_binary: pathlib.Path,
    loaded: Roster,
    tmp_path: pathlib.Path,
    role: str,
    tier: str,
) -> None:
    reference = _go_compose(go_binary, tmp_path, role, tier)
    mine = compose.compose(loaded, role, tier, tmp_path / "py-bundle")
    assert _tree(mine) == _tree(reference)


@pytest.mark.parametrize("role", sorted(_composable()))
def test_compiled_delivery_is_byte_identical_to_go(
    go_binary: pathlib.Path,
    loaded: Roster,
    tmp_path: pathlib.Path,
    role: str,
) -> None:
    reference = _go_compose(go_binary, tmp_path, role, "frontier", "compiled")
    mine = compose.compose(loaded, role, "frontier", tmp_path / "py-bundle", "compiled")
    assert _tree(mine) == _tree(reference)


def test_both_delivery_modes_are_covered() -> None:
    """Native and compiled differ in the delivery block and the compiled document."""
    loaded_roster = roster.load()
    native = compose.manifest(loaded_roster, "platform", "frontier", "native-skills")
    compiled = compose.manifest(loaded_roster, "platform", "frontier", "compiled")
    assert native["delivery"]["skills_root"] == "content/skills"
    assert "skills_root" not in compiled["delivery"]
    assert compiled["delivery"]["compiled_context"] == "delivery/compiled.md"


def test_every_tier_is_covered() -> None:
    """A silent drop to one tier would make the matrix look green for nothing."""
    covered = {tier for _, tier in _cases()}
    assert covered == set(TIERS)


def test_go_verify_accepts_the_python_bundle(
    go_binary: pathlib.Path,
    loaded: Roster,
    tmp_path: pathlib.Path,
) -> None:
    mine = compose.compose(loaded, "platform", "frontier", tmp_path / "py-bundle")
    result = subprocess.run(
        [str(go_binary), "verify", str(mine)], check=True, capture_output=True, text=True
    )
    assert "bundle verified" in result.stdout


def test_favorite_colors_match_the_go_snapshot(
    go_binary: pathlib.Path,
    loaded: Roster,
    tmp_path: pathlib.Path,
) -> None:
    out = tmp_path / "roster"
    subprocess.run([str(go_binary), "roster", "--out", str(out)], check=True, capture_output=True)
    snapshot = json.loads((out / "person.json").read_text())
    for name in loaded.role_order:
        assert loaded.roles[name].favorite_color == snapshot["roles"][name]["favorite_color"]
