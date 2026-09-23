from __future__ import annotations

from pathlib import Path

import pytest
from housecast.grade.schema import Challenge, Half, Response

from evalkit import filter as dataset_builder


def role_fit(against: str) -> Challenge:
    return Challenge(
        id=f"sysadmin-fit-{against}",
        entity="sysadmin",
        test_type="role-fit",
        prompt="prompt",
        target="hands it back",
        attribute=against,
    )


def responses(sample: Challenge, total: int = 5) -> list[Response]:
    return [
        Response(challenge_id=sample.id, epoch=index, text=f"answer {index}")
        for index in range(1, total + 1)
    ]


def test_the_annotator_sees_epoch_one() -> None:
    sample = role_fit("prod-director")
    report = dataset_builder.run([sample], responses(sample))
    assert [entry.id for entry in report.kept] == [sample.id]
    assert report.kept[0].output == "answer 1"


def test_epoch_order_does_not_depend_on_log_order() -> None:
    sample = role_fit("prod-director")
    shuffled = list(reversed(responses(sample)))
    report = dataset_builder.run([sample], shuffled)
    assert report.kept[0].output == "answer 1"


# Nothing is scored mechanically, so every authored case reaches the annotator.
def test_every_authored_sample_survives() -> None:
    samples = [
        role_fit("within"),
        role_fit("platform-eng"),
        Challenge(
            id="sysadmin-sec-out",
            entity="sysadmin",
            test_type="boundary",
            prompt="prompt",
            target="defers wording",
            attribute="suggest-external-comms",
            half=Half.OUT,
            pair_id="sysadmin-sec",
        ),
        Challenge(
            id="sysadmin-per-grounded",
            entity="sysadmin",
            test_type="personality",
            prompt="prompt",
            target="stays plain",
            attribute="grounded",
        ),
    ]
    runs = [r for sample in samples for r in responses(sample)]
    report = dataset_builder.run(samples, runs)
    assert len(report.kept) == len(samples)
    assert report.dropped == []


def test_a_sample_with_no_runs_is_reported_so_truncation_is_never_silent() -> None:
    sample = role_fit("prod-director")
    report = dataset_builder.run([sample], [])
    assert report.dropped[0].reason == "no subject runs"
    assert report.summary == "0 kept, 1 dropped"


def test_a_boundary_challenge_still_needs_its_pair_identity(tmp_path: Path) -> None:
    path = tmp_path / "challenges.yaml"
    path.write_text(
        "challenges:\n"
        "  - id: sysadmin-sec-out\n"
        "    entity: sysadmin\n"
        "    test_type: boundary\n"
        "    prompt: p\n"
        "    target: t\n"
    )
    with pytest.raises(ValueError, match="boundary challenge needs"):
        dataset_builder.load_challenges(path)


def test_a_role_fit_challenge_still_needs_its_attribute(tmp_path: Path) -> None:
    path = tmp_path / "challenges.yaml"
    path.write_text(
        "challenges:\n"
        "  - id: sysadmin-fit-within\n"
        "    entity: sysadmin\n"
        "    test_type: role-fit\n"
        "    prompt: p\n"
        "    target: t\n"
    )
    with pytest.raises(ValueError, match="role-fit challenge needs attribute"):
        dataset_builder.load_challenges(path)


def test_a_well_formed_board_loads(tmp_path: Path) -> None:
    path = tmp_path / "challenges.yaml"
    path.write_text(
        "challenges:\n"
        "  - id: sysadmin-mlb-in\n"
        "    entity: sysadmin\n"
        "    test_type: boundary\n"
        "    prompt: p\n"
        "    target: t\n"
        "    attribute: modify-live-backend\n"
        "    half: in\n"
        "    pair_id: sysadmin-mlb\n"
    )
    assert [s.id for s in dataset_builder.load_challenges(path)] == ["sysadmin-mlb-in"]
