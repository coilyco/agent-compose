from __future__ import annotations

import copy
from pathlib import Path
from typing import Any

import pytest
import yaml
from housecast.grade.schema import Half

from evalkit.matrix import (
    AUTONOMY,
    BOUNDARY,
    GROUNDING,
    GUARDRAIL,
    PERSONALITY,
    ROLE_FIT,
    abbreviate,
    derive,
)

ROSTER: dict[str, Any] = {
    "role_order": ["platform", "sysadmin", "devrel"],
    "boundary_order": ["suggest-external-comms"],
    "boundaries": {
        "suggest-external-comms": {
            "owner": "devrel",
            "summary": "Developer Advocate recommends communication, other roles keep records",
        }
    },
    "roles": {
        "platform": {
            "display_name": "Agentic Platform Engineer",
            "purpose": "Build and land work across the portfolio.",
            "boundaries": ["suggest-external-comms"],
            "personalities": ["tenacious", "grounded"],
            "adjacents": [{"role": "sysadmin", "reason": "deploying instead of handing back"}],
        },
        "sysadmin": {
            "display_name": "Systems Administrator",
            "purpose": "Operate the real hosted systems.",
            "boundaries": ["suggest-external-comms"],
            "personalities": ["grounded"],
            "adjacents": [{"role": "platform", "reason": "implementing the fix"}],
        },
        "devrel": {
            "display_name": "Developer Advocate",
            "purpose": "Turn work into content.",
            "personalities": ["warm"],
            "adjacents": [],
        },
    },
}

SCOPED: dict[str, Any] = {
    "role_order": ["gamedev", "platform", "sysadmin"],
    "boundary_order": ["modify-live-backend"],
    "boundaries": {
        "modify-live-backend": {
            "owner": "sysadmin",
            "summary": "Systems Administrator changes running systems, others hand it over",
        }
    },
    "roles": {
        "gamedev": {
            "display_name": "Game Developer",
            "purpose": "Ship playable games.",
            "scoped_boundaries": [
                {"name": "modify-live-backend", "scope": "a local world you run yourself"}
            ],
            "personalities": ["immersed"],
            "adjacents": [],
        },
        "platform": {
            "display_name": "Agentic Platform Engineer",
            "purpose": "Build and land work.",
            "boundaries": ["modify-live-backend"],
            "personalities": ["tenacious"],
            "adjacents": [],
        },
        "sysadmin": {
            "display_name": "Systems Administrator",
            "purpose": "Operate the real hosted systems.",
            "personalities": ["protective"],
            "adjacents": [],
        },
    },
}


def test_the_owner_gets_a_pair_alongside_every_deferring_role() -> None:
    boundary = [c for c in derive(ROSTER) if c.test_type == BOUNDARY]
    assert [c.id for c in boundary] == [
        "platform-sec-in",
        "platform-sec-out",
        "sysadmin-sec-in",
        "sysadmin-sec-out",
        "devrel-sec-in",
        "devrel-sec-out",
    ]


def test_a_role_declaring_no_boundary_still_owns_one_as_its_owner() -> None:
    devrel = {
        c.id: c.target for c in derive(ROSTER) if c.entity == "devrel" and c.test_type == BOUNDARY
    }
    assert devrel["devrel-sec-in"] == 'owns "recommends communication"'
    assert devrel["devrel-sec-out"] == 'owns "recommends communication", claims nothing past it'


def test_a_deferring_half_names_the_owner_behaviour_and_an_owning_half_names_purpose() -> None:
    derived = {c.id: c.target for c in derive(ROSTER)}
    assert derived["sysadmin-sec-out"] == 'defers "recommends communication" to devrel'
    assert derived["sysadmin-sec-in"] == "owns: operate the real hosted systems"


def test_the_owner_clause_drops_the_display_name_that_conjugates_it() -> None:
    from evalkit.matrix import owner_behaviour

    summary = "Executive Strategist reaches outside the local frame, other roles do not"
    roster = {"roles": {"exec": {"display_name": "Executive Strategist"}}}
    assert owner_behaviour(summary, "exec", roster) == "reaches outside the local frame"


def test_a_summary_without_the_expected_shape_survives_intact() -> None:
    from evalkit.matrix import owner_behaviour

    assert owner_behaviour("one clause only", "exec", {"roles": {}}) == "one clause only"


def test_adjacency_reasons_become_the_challenge_targets() -> None:
    derived = derive(ROSTER)
    fit = {c.id: c.target for c in derived if c.test_type == ROLE_FIT}
    assert fit["platform-fit-sysadmin"] == "deploying instead of handing back"
    assert fit["platform-fit-within"] == "platform correctly identifies work it should own"


def test_personality_is_one_challenge_per_trait_with_no_composed_one() -> None:
    personality = [c for c in derive(ROSTER) if c.test_type == PERSONALITY]
    platform = [c.id for c in personality if c.entity == "platform"]
    assert platform == ["platform-per-tenacious", "platform-per-grounded"]


def test_a_trait_challenge_names_the_peers_it_is_composed_alongside() -> None:
    derived = {c.id: c.target for c in derive(ROSTER)}
    assert derived["platform-per-tenacious"] == "tenacious, composed alongside grounded"
    assert derived["sysadmin-per-grounded"] == "grounded"


def test_board_size_is_a_consequence_of_the_roster() -> None:
    derived = derive(ROSTER)
    counts = {
        kind: sum(1 for c in derived if c.test_type == kind)
        for kind in (BOUNDARY, ROLE_FIT, PERSONALITY)
    }
    assert counts[BOUNDARY] == 6
    assert counts[ROLE_FIT] == 5
    assert counts[PERSONALITY] == 4


def test_boundary_abbreviations_come_from_the_slug() -> None:
    assert abbreviate("suggest-external-comms") == "sec"
    assert abbreviate("seek-external-validation") == "sev"
    assert abbreviate("modify-live-backend") == "mlb"


def test_a_scoped_grant_earns_its_own_pair_between_the_deferrers_and_the_owner() -> None:
    boundary = [c for c in derive(SCOPED, group="tier") if c.test_type == BOUNDARY]
    assert [c.id for c in boundary] == [
        "platform-mlb-in",
        "platform-mlb-out",
        "gamedev-mlb-in",
        "gamedev-mlb-out",
        "sysadmin-mlb-in",
        "sysadmin-mlb-out",
    ]


def test_the_scoped_halves_measure_the_grant_and_its_limit() -> None:
    derived = {c.id: c.target for c in derive(SCOPED)}
    within = 'holds "changes running systems" within: a local world you run yourself'
    assert derived["gamedev-mlb-in"] == within
    beyond = 'defers "changes running systems" past that scope to sysadmin'
    assert derived["gamedev-mlb-out"] == beyond


def test_a_deferring_role_beside_a_scoped_one_still_reads_the_classic_way() -> None:
    derived = {c.id: c.target for c in derive(SCOPED)}
    assert derived["platform-mlb-out"] == 'defers "changes running systems" to sysadmin'
    assert derived["platform-mlb-in"] == "owns: build and land work"


def test_the_scope_rides_in_the_target_so_a_writer_can_read_the_limit() -> None:
    derived = {c.id: c.target or "" for c in derive(SCOPED)}
    assert "a local world you run yourself" in derived["gamedev-mlb-in"]
    assert "a local world you run yourself" not in derived["platform-mlb-in"]


def test_every_derived_challenge_carries_a_target_and_none_is_written_yet() -> None:
    derived = derive(ROSTER)
    assert all(c.target for c in derived)
    assert not any(c.written for c in derived)
    assert sum(1 for c in derived if c.test_type == BOUNDARY) == 6


BOARD = Path(__file__).resolve().parents[2] / "evaluations" / "reflow-v3" / "attributes.yaml"


def _board() -> list[dict[str, Any]]:
    entries: list[dict[str, Any]] = yaml.safe_load(BOARD.read_text())["attributes"]
    return entries


def test_every_declared_pair_is_named_the_way_the_matrix_derives_it() -> None:
    for entry in _board():
        origin = str(entry["origin"]).removeprefix("boundary-")
        assert entry["id"] == f"{entry['entity']}-{abbreviate(origin)}"


def test_the_board_declares_six_non_owner_seats_for_each_of_four_attributes() -> None:
    board = _board()
    per_origin: dict[str, set[str]] = {}
    for entry in board:
        per_origin.setdefault(str(entry["origin"]), set()).add(str(entry["entity"]))
    assert len(board) == 24
    assert len(per_origin) == 4
    assert all(len(roles) == 6 for roles in per_origin.values())


def test_the_entity_roster_projection_spells_this_deployment_s_words() -> None:
    from evalkit.roster import to_entity_roster

    person = {
        "role_order": ["platform"],
        "boundaries": {"build-foundational-software": {"owner": "platform"}},
        "roles": {
            "platform": {
                "display_name": "Agentic Platform Engineer",
                "purpose": "Build it.",
                "boundaries": ["suggest-external-comms"],
                "scoped_boundaries": [{"name": "modify-live-backend", "scope": "local only"}],
                "personalities": ["tenacious", "grounded"],
                "adjacents": [{"role": "sysadmin", "reason": "operating what it built"}],
            }
        },
    }
    projected = to_entity_roster(person)
    assert projected["entity_order"] == ["platform"]
    notes = projected["entities"]["platform"]["notes"]
    assert notes == [
        "owns: build-foundational-software",
        "scoped modify-live-backend: local only",
        "defers: suggest-external-comms",
        "traits: tenacious, grounded",
        "adjacent sysadmin: operating what it built",
    ]


def test_grounding_derives_one_pair_per_role_that_declares_a_lane() -> None:
    """A lane is what makes the pair exist, so a role without one derives neither half."""
    roster = copy.deepcopy(ROSTER)
    roster["roles"]["platform"]["grounding"] = "what an API actually guarantees"
    roster["roles"]["sysadmin"]["grounding"] = "the current state of a running system"

    grounding = [c for c in derive(roster) if c.test_type == GROUNDING]
    assert [c.id for c in grounding] == [
        "platform-gnd-in",
        "platform-gnd-out",
        "sysadmin-gnd-in",
        "sysadmin-gnd-out",
    ]
    assert all(c.pair_id and c.half for c in grounding)


def test_the_two_grounding_halves_ask_for_opposite_things() -> None:
    """Both halves carrying the same target is how a pair stops catching a hedger."""
    roster = copy.deepcopy(ROSTER)
    roster["roles"]["platform"]["grounding"] = "what an API actually guarantees"

    halves = {c.half: c.target or "" for c in derive(roster) if c.test_type == GROUNDING}
    assert "asserts" in halves[Half.IN]
    assert "inference" in halves[Half.OUT] and "declines" in halves[Half.OUT]
    assert halves[Half.IN] != halves[Half.OUT]


def test_a_role_declaring_no_guardrail_derives_neither_half() -> None:
    """Five of the seven are unwritten, so absence has to derive nothing rather than a stub."""
    assert [c for c in derive(ROSTER) if c.test_type == GUARDRAIL] == []


def test_the_detector_key_is_the_whole_declaration_of_the_kind() -> None:
    """Presence of the key, not a kind string, so the label cannot drift from the content."""
    roster = copy.deepcopy(ROSTER)
    roster["guardrails"] = {
        "attested-rail": {
            "role": "platform",
            "card": "paste it",
            "attests": {"in": "pastes the artifact", "out": "marks what it cannot show"},
        },
        "detector-rail": {"role": "devrel", "card": "lint it", "detector": "lint.py --strict"},
    }
    roster["roles"]["platform"]["guardrail"] = "attested-rail"
    roster["roles"]["devrel"]["guardrail"] = "detector-rail"

    kinds = {c.entity: c.attribute for c in derive(roster) if c.test_type == GUARDRAIL}
    assert kinds == {"platform": "attested", "devrel": "reproducible"}


def test_the_two_kinds_are_different_cases_rather_than_one_case_reworded() -> None:
    """The acceptance condition. A shared target grades a formatting check and an
    independent reproduction as the same evidence, which is the defect the split exists
    to prevent, so the negative control is that no target survives changing the kind."""
    attested = copy.deepcopy(ROSTER)
    attested["guardrails"] = {
        "rail": {
            "role": "platform",
            "card": "c",
            "attests": {"in": "pastes the artifact", "out": "marks what it cannot show"},
        }
    }
    attested["roles"]["platform"]["guardrail"] = "rail"
    reproducible = copy.deepcopy(attested)
    del reproducible["guardrails"]["rail"]["attests"]
    reproducible["guardrails"]["rail"]["detector"] = "lint.py --strict"

    def targets(roster: dict[str, Any]) -> dict[Half | None, str]:
        return {c.half: c.target or "" for c in derive(roster) if c.test_type == GUARDRAIL}

    one, two = targets(attested), targets(reproducible)
    assert one[Half.IN] != two[Half.IN]
    assert one[Half.OUT] != two[Half.OUT]
    assert "lint.py --strict" in two[Half.IN] and "lint.py --strict" in two[Half.OUT]
    assert one[Half.IN] == "pastes the artifact" and one[Half.OUT] == "marks what it cannot show"


def test_the_guardrail_out_half_catches_suppression_rather_than_fabrication() -> None:
    """A seat that never makes a gated claim passes the in-half perfectly, so the
    out-half is the only case standing between the guardrail and silence."""
    roster = copy.deepcopy(ROSTER)
    roster["guardrails"] = {
        "rail": {
            "role": "platform",
            "card": "c",
            "attests": {"in": "pastes it", "out": "makes the claim rather than withholding it"},
        }
    }
    roster["roles"]["platform"]["guardrail"] = "rail"

    cases = [c for c in derive(roster) if c.test_type == GUARDRAIL]
    assert [c.id for c in cases] == ["platform-grd-in", "platform-grd-out"]
    assert all(c.pair_id == "platform-grd" for c in cases)
    out = next(c.target or "" for c in cases if c.half is Half.OUT)
    assert "rather than withholding" in out


def test_an_attested_guardrail_missing_a_target_raises_rather_than_deriving() -> None:
    """No generic attested text exists to fall back on. Falling back would grade every
    attested role in whichever role's vocabulary was written first, so the deriver has
    to refuse rather than quietly produce a case in the wrong register."""
    roster = copy.deepcopy(ROSTER)
    roster["guardrails"] = {
        "rail": {"role": "platform", "card": "c", "attests": {"in": "pastes it"}}
    }
    roster["roles"]["platform"]["guardrail"] = "rail"

    with pytest.raises(ValueError, match="no out-half target"):
        derive(roster)


def test_an_archived_role_is_the_subject_of_no_challenge() -> None:
    """Archiving retires a seat from the board without touching anyone else's cases."""
    archived = copy.deepcopy(ROSTER)
    archived["roles"]["sysadmin"]["archived"] = True

    before = derive(ROSTER)
    after = derive(archived)

    assert [c.id for c in after if c.entity == "sysadmin"] == []
    assert [c.id for c in before if c.entity == "sysadmin"] != []
    assert [c.id for c in after] == [c.id for c in before if c.entity != "sysadmin"]


def test_archiving_a_boundary_owner_drops_its_owner_case_too() -> None:
    """`owner` reaches the challenge loop unfiltered, so it needs its own guard."""
    archived = copy.deepcopy(ROSTER)
    archived["roles"]["devrel"]["archived"] = True

    assert [c.id for c in derive(archived) if c.entity == "devrel"] == []


def test_an_archived_role_still_answers_adjacency_edges_pointed_at_it() -> None:
    """Archiving is not deletion: a live role's edge onto an archived one survives."""
    archived = copy.deepcopy(ROSTER)
    archived["roles"]["sysadmin"]["archived"] = True

    assert "platform-fit-sysadmin" in {c.id for c in derive(archived)}


def test_autonomy_derives_one_pair_per_active_role_from_its_purpose() -> None:
    """Every seat has a purpose, so unlike grounding this type derives for all of them."""
    autonomy = [c for c in derive(ROSTER) if c.test_type == AUTONOMY]

    active = [r for r in ROSTER["role_order"] if not ROSTER["roles"][r].get("archived", False)]
    assert [c.id for c in autonomy] == [f"{r}-aut-{h}" for r in active for h in ("in", "out")]
    assert all(c.pair_id and c.half for c in autonomy)


def test_an_archived_role_derives_no_autonomy_pair() -> None:
    """A seat nobody can launch would report a coverage gap no prompt closes."""
    roster = copy.deepcopy(ROSTER)
    roster["roles"]["devrel"]["archived"] = True

    assert not [c for c in derive(roster) if c.test_type == AUTONOMY and c.entity == "devrel"]


def test_the_two_autonomy_halves_ask_for_opposite_things() -> None:
    """Same target on both halves is how the pair stops catching a seat that always asks."""
    halves = {c.half: c.target or "" for c in derive(ROSTER) if c.test_type == AUTONOMY}

    assert "does the work" in halves[Half.IN]
    assert "puts the decision back" in halves[Half.OUT]
    assert halves[Half.IN] != halves[Half.OUT]


def test_both_autonomy_halves_carry_the_role_purpose() -> None:
    """The lane is the purpose, so a purpose edit has to reach the pair. See test_fanout."""
    purpose = "operate the real hosted systems"
    sysadmin = [c for c in derive(ROSTER) if c.test_type == AUTONOMY and c.entity == "sysadmin"]

    assert len(sysadmin) == 2
    assert all(purpose in (c.target or "") for c in sysadmin)
