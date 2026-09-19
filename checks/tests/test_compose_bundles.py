"""Cover the bundle composition the release build runs and `just test` did not.

`scripts/compose-bundles.py` is invoked only by `just compose-bundles` and by
`scripts/release-build.sh`, so nothing in the validation gate ever ran it. On
2026-09-08 that gap shipped twice: v2.105.0 and v2.106.0 both passed `just test`,
tagged, and then died in `release-build` with

    ValueError: role 'analyst' is archived and cannot be composed

The tags exist and the releases do not, so every host is still on v2.104.0 and
the rename cannot reach one. A validation step that does not run what the build
runs is the defect, not the archived role.
"""

from __future__ import annotations

import pathlib
import subprocess
import sys

import pytest

from checks import roster
from checks.roster import Roster

REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
COMPOSER = REPO_ROOT / "scripts" / "compose-bundles.py"


@pytest.fixture(scope="module")
def loaded() -> Roster:
    return roster.load()


def test_the_roster_still_carries_an_archived_role(loaded: Roster) -> None:
    """The guard below only means something while one exists."""
    archived = [name for name, role in loaded.roles.items() if role.archived]
    if not archived:
        pytest.skip("no archived role in the roster, so nothing to skip over")
    assert archived


def test_composing_every_bundle_succeeds(tmp_path: pathlib.Path) -> None:
    """The exact call release-build makes, against the real roster."""
    result = subprocess.run(
        [sys.executable, str(COMPOSER), str(tmp_path / "bundles")],
        cwd=REPO_ROOT,
        capture_output=True,
        text=True,
    )
    assert result.returncode == 0, (
        f"release-build would fail here:\n{result.stdout}\n{result.stderr}"
    )


def test_no_archived_role_gets_a_bundle(tmp_path: pathlib.Path, loaded: Roster) -> None:
    out = tmp_path / "bundles"
    subprocess.run(
        [sys.executable, str(COMPOSER), str(out)],
        cwd=REPO_ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    # Bundle names are "{role}-{tier}-{delivery}", and a role slug can itself
    # be hyphenated, so match the full prefix rather than split(...)[0].
    built = {path.name for path in out.iterdir()}
    for name, role in loaded.roles.items():
        has_bundle = any(bundle.startswith(f"{name}-") for bundle in built)
        if role.archived:
            assert not has_bundle, f"archived role {name!r} was given a bundle"
        else:
            assert has_bundle, f"live role {name!r} has no bundle"
