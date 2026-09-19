"""The coverage check, against rosters small enough to state the whole board.

`housecast/data/minimal-roster.yaml` derives four cases, so a fixture here can
name every one of them rather than asserting on a count.
"""

from __future__ import annotations

import dataclasses
import pathlib

import yaml

from checks import roster as roster_module
from evalkit import coverage
from evalkit.profile import PROFILE

MINIMAL = roster_module.DATA.parent / "minimal-roster.yaml"
CHALLENGES = coverage.ROOT / "challenges.yaml"


def _challenges(tmp_path: pathlib.Path, ids: list[str]) -> pathlib.Path:
    path = tmp_path / "challenges.yaml"
    path.write_text(
        yaml.safe_dump({"challenges": [{"id": case, "prompt": "p"} for case in ids]}),
        encoding="utf-8",
    )
    return path


def _graded(tmp_path: pathlib.Path, run: str, ids: list[str]) -> pathlib.Path:
    root = tmp_path / "evaluations"
    (root / run).mkdir(parents=True, exist_ok=True)
    (root / run / "annotations.yaml").write_text(
        yaml.safe_dump(
            {"grader": "kai", "annotations": [{"id": case, "label": "pass"} for case in ids]}
        ),
        encoding="utf-8",
    )
    return root


def _with_second_boundary(tmp_path: pathlib.Path) -> pathlib.Path:
    """The minimal roster plus one boundary the same role owns."""
    document = yaml.safe_load(MINIMAL.read_text())
    existing = document["boundaries"]["boundary-read-the-log"]
    added = dict(existing)
    added["skill"] = "boundary-quote-the-line"
    added["body"] = existing["body"].replace("boundary-read-the-log", "boundary-quote-the-line")
    document["boundaries"]["boundary-quote-the-line"] = added
    document["boundary_order"].append("boundary-quote-the-line")
    path = tmp_path / "roster.yaml"
    path.write_text(yaml.safe_dump(document), encoding="utf-8")
    return path


def test_the_minimal_roster_derives_a_board_small_enough_to_name(
    tmp_path: pathlib.Path,
) -> None:
    report = coverage.build(MINIMAL, _challenges(tmp_path, []), tmp_path / "none")
    assert report.derived == {
        "reader-brtl-in",
        "reader-brtl-out",
        "reader-fit-within",
        "reader-per-personality-grounded",
        "reader-gnd-in",
        "reader-gnd-out",
        "reader-aut-in",
        "reader-aut-out",
    }


def test_adding_a_boundary_reports_its_newly_implied_cases_as_unauthored(
    tmp_path: pathlib.Path,
) -> None:
    """The board is derived, so a roster edit is what makes a case exist."""
    covered = coverage.build(MINIMAL, _challenges(tmp_path, []), tmp_path / "none")
    authored = _challenges(tmp_path, sorted(covered.derived))
    assert not coverage.build(MINIMAL, authored, tmp_path / "none").unauthored

    after = coverage.build(_with_second_boundary(tmp_path), authored, tmp_path / "none")
    assert after.unauthored == {"reader-bqtl-in", "reader-bqtl-out"}


def test_a_graded_record_for_a_removed_role_is_orphaned_rather_than_dropped(
    tmp_path: pathlib.Path,
) -> None:
    """`devrel` was a role and is not one now. Its evidence still exists."""
    evaluations = _graded(tmp_path, "board-2026-01-01", ["devrel-fit-within", "reader-fit-within"])
    report = coverage.build(MINIMAL, _challenges(tmp_path, []), evaluations)
    assert report.orphaned == {"evaluations/board-2026-01-01": {"devrel-fit-within"}}


def test_a_retired_run_is_not_reported_forever(tmp_path: pathlib.Path) -> None:
    """Its roster is gone, so its cases orphan permanently rather than pending."""
    evaluations = _graded(tmp_path, "retired-run", ["devrel-fit-within"])
    config = coverage.Config(retired_runs=("evaluations/retired-run",))
    report = coverage.build(MINIMAL, _challenges(tmp_path, []), evaluations, config)
    assert not report.orphaned


def test_an_authored_case_with_no_annotation_is_ungraded(tmp_path: pathlib.Path) -> None:
    authored = _challenges(tmp_path, ["reader-fit-within", "reader-brtl-in"])
    evaluations = _graded(tmp_path, "board", ["reader-brtl-in"])
    report = coverage.build(MINIMAL, authored, evaluations)
    assert report.ungraded == {"reader-fit-within"}
    assert not report.stale


def test_an_authored_case_the_roster_no_longer_derives_is_stale(
    tmp_path: pathlib.Path,
) -> None:
    report = coverage.build(MINIMAL, _challenges(tmp_path, ["devrel-fit-within"]), tmp_path / "n")
    assert report.stale == {"devrel-fit-within"}


def test_the_switch_is_configuration_rather_than_code(tmp_path: pathlib.Path) -> None:
    """Flipping it is a pyproject edit, so a repo can gate without a patch here."""
    reporting = tmp_path / "reporting.toml"
    reporting.write_text("[tool.evalkit.coverage]\nblocking = false\n", encoding="utf-8")
    blocking = tmp_path / "blocking.toml"
    blocking.write_text("[tool.evalkit.coverage]\nblocking = true\n", encoding="utf-8")

    assert coverage.load_config(reporting).blocking is False
    assert coverage.load_config(blocking).blocking is True

    argv = [
        "--roster",
        str(MINIMAL),
        "--challenges",
        str(_challenges(tmp_path, ["devrel-fit-within"])),
        "--evaluations",
        str(tmp_path / "none"),
    ]
    assert coverage.main([*argv, "--config", str(reporting)]) == 0
    assert coverage.main([*argv, "--config", str(blocking)]) == 1


def test_the_shipped_default_is_report(tmp_path: pathlib.Path) -> None:
    """A blocking default would put every roster edit behind an annotation session."""
    assert coverage.load_config().blocking is False
    argv = ["--roster", str(MINIMAL), "--challenges", str(CHALLENGES)]
    assert coverage.main([*argv, "--evaluations", str(tmp_path / "none")]) == 0


def test_a_grader_named_annotations_file_counts_as_graded(tmp_path: pathlib.Path) -> None:
    """Matching only `annotations.yaml` read a whole graded board as ungraded.

    Observed 2026-09-15: 119 cases reported ungraded where 103 carried a label in
    `annotations.kai.yaml`, which is the number the autonomy grader was built to
    reduce and had already been reduced by a human.
    """
    root = tmp_path / "evaluations"
    (root / "run").mkdir(parents=True)
    (root / "run" / "annotations.kai.yaml").write_text(
        yaml.safe_dump(
            {"grader": "kai", "annotations": [{"id": "reader-gnd-in", "label": "pass"}]}
        ),
        encoding="utf-8",
    )

    report = coverage.build(MINIMAL, _challenges(tmp_path, ["reader-gnd-in"]), root)
    assert "reader-gnd-in" not in report.ungraded


def test_two_graders_on_one_run_union_rather_than_overwrite(tmp_path: pathlib.Path) -> None:
    """Either label is a label on record, and this report is about presence.

    Supplies its own profile rather than the board's, whose roster names one
    person. A test that needed a second grader on this board would have to
    invent one.
    """
    two = dataclasses.replace(PROFILE, graders=("kai", "quinn"))
    root = tmp_path / "evaluations"
    (root / "run").mkdir(parents=True)
    for grader, case in (("kai", "reader-gnd-in"), ("quinn", "reader-gnd-out")):
        (root / "run" / f"annotations.{grader}.yaml").write_text(
            yaml.safe_dump({"grader": grader, "annotations": [{"id": case, "label": "pass"}]}),
            encoding="utf-8",
        )

    challenges = _challenges(tmp_path, ["reader-gnd-in", "reader-gnd-out"])
    report = coverage.build(MINIMAL, challenges, root, profile=two)
    assert not report.ungraded


def _annotated(tmp_path: pathlib.Path, grader: str | None, ids: list[str]) -> pathlib.Path:
    root = tmp_path / "evaluations"
    (root / "run").mkdir(parents=True, exist_ok=True)
    payload: dict[str, object] = {}
    if grader is not None:
        payload["grader"] = grader
    payload["annotations"] = [{"id": case, "label": "pass"} for case in ids]
    name = "annotations.yaml" if grader is None else f"annotations.{grader}.yaml"
    (root / "run" / name).write_text(yaml.safe_dump(payload), encoding="utf-8")
    return root


def test_a_grader_the_board_does_not_permit_contributes_no_labels(
    tmp_path: pathlib.Path,
) -> None:
    """The scorer is a human, and on 2026-09-15 an agent wrote a board's labels."""
    root = _annotated(tmp_path, "evie", ["reader-gnd-in"])

    report = coverage.build(MINIMAL, _challenges(tmp_path, ["reader-gnd-in"]), root)
    assert "reader-gnd-in" in report.ungraded


def test_a_refused_file_is_named_rather_than_skipped_quietly(tmp_path: pathlib.Path) -> None:
    """A label this report drops in silence is how one ends up counted."""
    root = _annotated(tmp_path, "evie", ["reader-gnd-in"])

    report = coverage.build(MINIMAL, _challenges(tmp_path, ["reader-gnd-in"]), root)
    assert report.rejected == {"evaluations/run/annotations.evie.yaml": "evie"}
    assert not report.clean


def test_the_same_file_under_a_permitted_grader_counts(tmp_path: pathlib.Path) -> None:
    """The positive control. Without it the check could refuse everything and pass."""
    root = _annotated(tmp_path, "kai", ["reader-gnd-in"])

    report = coverage.build(MINIMAL, _challenges(tmp_path, ["reader-gnd-in"]), root)
    assert "reader-gnd-in" not in report.ungraded
    assert not report.rejected


def test_an_unattributed_file_does_not_pass_a_named_roster(tmp_path: pathlib.Path) -> None:
    """A label whose author is unknown is the case the roster exists to stop."""
    root = _annotated(tmp_path, None, ["reader-gnd-in"])

    report = coverage.build(MINIMAL, _challenges(tmp_path, ["reader-gnd-in"]), root)
    assert report.rejected == {"evaluations/run/annotations.yaml": "unattributed"}
