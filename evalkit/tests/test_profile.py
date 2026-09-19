import pathlib

from housecast.grade.io import load_profile
from housecast.grade.schema import AGENT_COMPOSE

from evalkit.profile import PROFILE, main, to_dict


def test_the_profile_round_trips_through_the_yaml_a_grading_surface_reads(
    tmp_path: pathlib.Path,
) -> None:
    out = tmp_path / "profile.yaml"
    assert main(["--out", str(out)]) == 0
    assert load_profile(out) == PROFILE


def test_the_shipped_default_is_missing_the_types_this_board_derives() -> None:
    """The reason eval-annotate.sh hands a profile over. Measured on board-2026-09-01."""
    declared = {spec.name for spec in PROFILE.test_types}
    fallback = {spec.name for spec in AGENT_COMPOSE.test_types}
    assert declared - fallback == {"voice", "grounding", "guardrail", "autonomy"}


def test_every_declared_type_carries_a_label_set_the_schema_knows() -> None:
    from housecast.grade.schema import LABEL_SETS

    assert all(spec.label_set in LABEL_SETS for spec in PROFILE.test_types)


def test_to_dict_covers_every_test_type_rather_than_the_first(tmp_path: pathlib.Path) -> None:
    assert len(to_dict()["test_types"]) == len(PROFILE.test_types)
