"""What an overlay may append, and what it must refuse.

The refusals matter more than the appends here. An overlay that silently ignored
a misspelled role would produce a bundle missing exactly the acts somebody wrote
it for, and nothing would say so - the seat would just carry its three portable
acts and look correct.
"""

from __future__ import annotations

import pathlib
from typing import Any

import pytest
import yaml

from checks import overlay, render, roster
from checks.roster import Roster, RosterError


@pytest.fixture(scope="module")
def shipped() -> Roster:
    return roster.load()


def write(tmp_path: pathlib.Path, doc: dict[str, Any]) -> pathlib.Path:
    path = tmp_path / "overlay.yaml"
    path.write_text(yaml.safe_dump(doc, sort_keys=False))
    return path


def test_role_acts_append_after_the_portable_ones(shipped: Roster, tmp_path: pathlib.Path) -> None:
    before = list(shipped.roles["senior-sysadmin"].acts)
    path = write(
        tmp_path,
        {"roles": {"senior-sysadmin": [{"tool": "aosguard", "text": "paste the before state"}]}},
    )
    after = overlay.apply(shipped, path).roles["senior-sysadmin"].acts
    assert [a.text for a in after] == [*[a.text for a in before], "paste the before state"]


def test_the_shipped_roster_is_not_mutated(shipped: Roster, tmp_path: pathlib.Path) -> None:
    """apply returns a new roster, so a second call cannot compound the first."""
    count = len(shipped.roles["senior-sysadmin"].acts)
    path = write(tmp_path, {"roles": {"senior-sysadmin": [{"tool": "t", "text": "x"}]}})
    overlay.apply(shipped, path)
    overlay.apply(shipped, path)
    assert len(shipped.roles["senior-sysadmin"].acts) == count


def test_boundary_acts_land_on_the_named_side(shipped: Roster, tmp_path: pathlib.Path) -> None:
    path = write(
        tmp_path,
        {
            "boundaries": {
                "modify-live-backend": {
                    "own": [{"tool": "aosguard", "text": "own act"}],
                    "defer": [{"tool": "SendMessage", "text": "defer act"}],
                }
            }
        },
    )
    boundary = overlay.apply(shipped, path).boundaries["modify-live-backend"]
    assert "own act" in [a.text for a in boundary.acts_for_side("own")]
    assert "defer act" in [a.text for a in boundary.acts_for_side("defer")]
    assert "own act" not in [a.text for a in boundary.acts_for_side("defer")]


def test_scoped_is_a_third_position_rather_than_a_flavour_of_own(
    shipped: Roster, tmp_path: pathlib.Path
) -> None:
    """A scoped seat resolves to `scoped`, so an overlay that only knows own and
    defer would drop its estate acts silently."""
    path = write(
        tmp_path,
        {
            "boundaries": {
                "build-foundational-software": {
                    "scoped": [{"tool": "git", "text": "confirm it stayed in scope"}]
                }
            }
        },
    )
    boundary = overlay.apply(shipped, path).boundaries["build-foundational-software"]
    assert "confirm it stayed in scope" in [a.text for a in boundary.acts_for_side("scoped")]
    assert "confirm it stayed in scope" not in [a.text for a in boundary.acts_for_side("own")]


def test_the_appended_act_reaches_the_identity_card(
    shipped: Roster, tmp_path: pathlib.Path
) -> None:
    """The whole point is that the seat reads it, so render is the real check."""
    path = write(
        tmp_path,
        {"roles": {"science": [{"tool": "aosguard", "text": "the estate act for science"}]}},
    )
    card = render.identity_card(overlay.apply(shipped, path), "science")
    assert "the estate act for science" in card


@pytest.mark.parametrize(
    "doc",
    [
        {"roles": {"sys-admin": [{"tool": "t", "text": "x"}]}},
        {"personalities": {"grounded-ish": [{"tool": "t", "text": "x"}]}},
        {"boundaries": {"modify-live-backends": {"own": []}}},
    ],
    ids=["role", "personality", "boundary"],
)
def test_a_misspelled_name_is_refused_rather_than_ignored(
    shipped: Roster, tmp_path: pathlib.Path, doc: dict[str, Any]
) -> None:
    with pytest.raises(RosterError):
        overlay.apply(shipped, write(tmp_path, doc))


def test_an_unknown_side_is_refused(shipped: Roster, tmp_path: pathlib.Path) -> None:
    doc = {"boundaries": {"modify-live-backend": {"owns": [{"tool": "t", "text": "x"}]}}}
    with pytest.raises(RosterError, match="unknown sides"):
        overlay.apply(shipped, write(tmp_path, doc))


def test_redefinition_is_unexpressible(shipped: Roster, tmp_path: pathlib.Path) -> None:
    """The format carries acts and nothing else, so a purpose or a meld cannot
    be reached even by a caller trying to."""
    doc = {"roles": {"science": [{"tool": "t", "text": "x", "purpose": "something else"}]}}
    with pytest.raises(RosterError, match="unknown keys"):
        overlay.apply(shipped, write(tmp_path, doc))
    with pytest.raises(RosterError, match="unknown top-level keys"):
        overlay.apply(shipped, write(tmp_path, {"personality_order": ["grounded"]}))


def test_an_act_without_a_tool_is_refused(shipped: Roster, tmp_path: pathlib.Path) -> None:
    """`tool` is carried separately so coverage is exact rather than a grep."""
    doc = {"roles": {"science": [{"text": "no tool named"}]}}
    with pytest.raises(RosterError, match="tool is required"):
        overlay.apply(shipped, write(tmp_path, doc))


def test_an_empty_overlay_is_a_no_op(shipped: Roster, tmp_path: pathlib.Path) -> None:
    path = tmp_path / "empty.yaml"
    path.write_text("")
    applied = overlay.apply(shipped, path)
    for name in shipped.role_order:
        assert applied.roles[name].acts == shipped.roles[name].acts
