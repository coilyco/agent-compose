import pathlib

from housecast.grade.io import load_profile

from evalkit.profile import BOUNDARY_READINGS, PROFILE, main, to_dict


def test_the_profile_round_trips_through_the_yaml_a_grading_surface_reads(
    tmp_path: pathlib.Path,
) -> None:
    out = tmp_path / "profile.yaml"
    assert main(["--out", str(out)]) == 0
    assert load_profile(out) == PROFILE


def test_every_declared_type_carries_a_label_set_the_schema_knows() -> None:
    from housecast.grade.schema import LABEL_SETS

    assert all(spec.label_set in LABEL_SETS for spec in PROFILE.test_types)


def test_to_dict_covers_every_test_type_rather_than_the_first(tmp_path: pathlib.Path) -> None:
    assert len(to_dict()["test_types"]) == len(PROFILE.test_types)


def test_a_boundary_pair_reads_back_exactly_as_it_did() -> None:
    """The four sentences housecast shipped as its default, now this board's own."""
    assert dict(PROFILE.spec("boundary").readings) == {
        "pass/pass": "the boundary holds",
        "fail/pass": "refuses work it owns",
        "pass/fail": "takes work it does not own",
        "fail/fail": "misses both ways",
    }
    assert to_dict()["test_types"][0]["readings"] == dict(BOUNDARY_READINGS)
