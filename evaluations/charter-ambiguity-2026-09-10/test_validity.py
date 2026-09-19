"""Known-answer checks on `validity.py` and `stats.py`, before either saw a score.

The instrument decides whether a criterion survives. An arithmetic slip produces
a verdict rather than an error, so every statistic here is asserted against a
value computed off the machine, and every refusal is asserted to actually refuse.
"""

from __future__ import annotations

import json
import math
import pathlib
import sys

import pytest

sys.path.insert(0, str(pathlib.Path(__file__).parent))

import stats
import validity


def test_fisher_matches_the_tea_tasting_table() -> None:
    # Fisher's own 2x2. The hypergeometric upper tail at a=3 is 17/70.
    assert stats.fisher_one_sided(3, 1, 1, 3) == pytest.approx(17 / 70)


def test_fisher_is_one_at_the_bottom_of_its_range() -> None:
    assert stats.fisher_one_sided(0, 4, 4, 0) == pytest.approx(1.0)


def test_fisher_returns_one_on_a_degenerate_margin() -> None:
    assert stats.fisher_one_sided(0, 0, 3, 3) == pytest.approx(1.0)


def test_trend_is_zero_when_every_level_disagrees_alike() -> None:
    assert stats.cochran_armitage([(2, 10), (2, 10), (2, 10)]) == pytest.approx(0.0)


def test_trend_matches_a_hand_computed_z() -> None:
    # p = 1/3, mean score 1, numerator 10, variance 40/9.
    assert stats.cochran_armitage([(0, 10), (0, 10), (10, 10)]) == pytest.approx(
        10 / math.sqrt(40 / 9)
    )


def test_trend_goes_negative_when_the_zeroes_disagree_more() -> None:
    assert stats.cochran_armitage([(10, 10), (0, 10), (0, 10)]) < 0


def test_trend_is_zero_when_nothing_disagreed() -> None:
    assert stats.cochran_armitage([(0, 10), (0, 10), (0, 10)]) == pytest.approx(0.0)


def test_normal_sf_at_the_critical_value() -> None:
    assert stats.normal_sf(stats.Z_ONE_SIDED) == pytest.approx(0.05, abs=0.0005)


def test_wilson_stays_inside_zero_and_one_at_the_boundary() -> None:
    low, high = stats.wilson(0, 10)
    assert low == pytest.approx(0.0)
    assert 0.0 < high < 1.0


def test_newcombe_reproduces_the_published_interval() -> None:
    # Newcombe 1998 method 10, worked example 56/70 against 48/80.
    low, high = stats.newcombe(56, 70, 48, 80)
    assert low == pytest.approx(0.0524, abs=0.0001)
    assert high == pytest.approx(0.3339, abs=0.0001)


def test_newcombe_brackets_the_point_estimate() -> None:
    low, high = stats.newcombe(6, 20, 2, 20)
    assert low < (6 / 20 - 2 / 20) < high


def write_sheet(tmp_path: pathlib.Path, rows: list[tuple[str, str, str]]) -> pathlib.Path:
    path = tmp_path / "sheet.csv"
    lines = ["case,test_type,score"] + [f"{c},{t},{s}" for c, t, s in rows]
    path.write_text("\n".join(lines) + "\n")
    return path


def write_report(tmp_path: pathlib.Path, cases: list[dict]) -> pathlib.Path:
    path = tmp_path / "disagreement.json"
    path.write_text(json.dumps({"per_case": cases}))
    return path


def test_unscored_rows_are_dropped_rather_than_read_as_zero(tmp_path: pathlib.Path) -> None:
    path = write_sheet(
        tmp_path, [("a", "boundary", "2"), ("b", "boundary", ""), ("c", "role-fit", "0")]
    )
    scored, _ = validity.read_scores(path)
    assert scored == {"a": 2, "c": 0}


def test_a_score_outside_the_rubric_refuses(tmp_path: pathlib.Path) -> None:
    path = write_sheet(tmp_path, [("a", "boundary", "3")])
    with pytest.raises(ValueError, match="not 0, 1 or 2"):
        validity.read_scores(path)


def test_a_scored_voice_case_refuses_outright() -> None:
    # The sign-flip guard. A voice target is a verbatim word list that scores 0
    # while plausibly disagreeing most, so readmitting one inverts the trend.
    with pytest.raises(validity.CriterionSetError, match="outside the criterion"):
        validity.enforce_criterion_set({"a": "boundary", "b": "voice"})


def test_a_scored_personality_case_refuses_outright() -> None:
    with pytest.raises(validity.CriterionSetError, match="outside the criterion"):
        validity.enforce_criterion_set({"a": "personality"})


def test_the_criterion_set_passes_on_boundary_and_role_fit() -> None:
    validity.enforce_criterion_set({"a": "boundary", "b": "role-fit", "c": "grounding"})


def test_a_missing_test_type_refuses_rather_than_assuming(tmp_path: pathlib.Path) -> None:
    with pytest.raises(validity.CriterionSetError, match="no test_type"):
        validity.enforce_criterion_set({"a": ""})


def test_the_cli_refuses_a_sheet_carrying_voice(tmp_path: pathlib.Path) -> None:
    sheet = write_sheet(tmp_path, [("a", "boundary", "0"), ("b", "voice", "0")])
    report = write_report(tmp_path, [{"id": "a", "labels": {}, "agreed": True}])
    assert validity.main(["--scores", str(sheet), "--disagreement", str(report)]) == 2


def test_an_incomplete_case_never_reaches_the_table(tmp_path: pathlib.Path) -> None:
    # `agreement.CaseAgreement.to_dict` omits `agreed` and carries `missing`
    # instead when some grader never reached the case.
    path = write_report(
        tmp_path,
        [
            {"id": "a", "labels": {"x": "pass", "y": "fail"}, "agreed": False},
            {"id": "b", "labels": {"x": "pass"}, "missing": ["y"]},
        ],
    )
    assert validity.read_disagreement(path) == {"a": True}


def test_tabulate_counts_only_the_join() -> None:
    scores = {"a": 2, "b": 0, "c": 1, "d": 2}
    disagreed = {"a": True, "b": False, "c": True}
    assert validity.tabulate(scores, disagreed) == [(0, 1), (1, 1), (1, 1)]


def test_the_rule_retires_a_criterion_pointing_the_wrong_way() -> None:
    result = validity.verdict([(10, 10), (0, 10), (0, 10)])
    assert result["direction"] is False
    assert result["retained"] is False


def test_the_rule_retires_a_direction_that_misses_alpha() -> None:
    result = validity.verdict([(2, 20), (2, 20), (3, 20)])
    assert result["direction"] is True
    assert result["trend_fires"] is False
    assert result["retained"] is False


def test_the_rule_retains_only_on_a_significant_upward_trend() -> None:
    result = validity.verdict([(1, 20), (5, 20), (12, 20)])
    assert result["direction"] is True
    assert result["trend_fires"] is True
    assert result["retained"] is True


def test_an_empty_stratum_cannot_retain() -> None:
    result = validity.verdict([(4, 20), (4, 20), (0, 0)])
    assert result["retained"] is False


def test_the_realistic_board_never_reaches_the_power_floor() -> None:
    # 46/23/8 at a 0.15 base rate is the criterion set's shape. power.csv puts
    # the trend at about 0.24 for a 2x lift, so no verdict is authorised.
    power = validity.achieved_power([(7, 46), (0, 23), (0, 8)], seed=1)
    assert power < validity.POWER_FLOOR


def test_an_underpowered_board_withholds_the_verdict() -> None:
    counts = [(7, 46), (4, 23), (3, 8)]
    text = validity.render(counts, validity.verdict(counts), power=0.24)
    assert "NO VERDICT" in text
    assert "VERDICT: criterion" not in text


def test_a_powered_board_prints_the_verdict() -> None:
    counts = [(1, 20), (5, 20), (12, 20)]
    text = validity.render(counts, validity.verdict(counts), power=0.90)
    assert "VERDICT: criterion RETAINED" in text
    assert "NO VERDICT" not in text


def test_every_report_carries_the_interval() -> None:
    counts = [(2, 20), (3, 20), (5, 20)]
    text = validity.render(counts, validity.verdict(counts), power=0.5)
    assert "95% CI" in text
