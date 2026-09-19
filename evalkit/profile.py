"""The board's own profile. Declared here rather than imported from housecast.grade.

`Profile` exists so a deployment states its own test types without the shared
schema growing a branch per consumer, and this board needs a fourth that no
other consumer wants. The name stays `agent-compose` because it names the
roster under test, and committed evidence records it.
"""

from __future__ import annotations

import argparse
from pathlib import Path
from typing import Any

from housecast.grade.io import dump_yaml
from housecast.grade.schema import Profile, TestTypeSpec

# Below 50 words the suggest-external-comms out-half drops the factual handoff,
# which the boundary requires. Measured against written example responses.
PROFILE = Profile(
    name="agent-compose",
    test_types=(
        TestTypeSpec("boundary", "binary", 50, ("attribute", "half", "pair_id")),
        TestTypeSpec("role-fit", "binary", 50, ("attribute",)),
        TestTypeSpec("personality", "fit", 100, ("attribute",)),
        # Voice is a judgement of degree like personality, and needs the same
        # room to answer in. See agent-compose#378.
        TestTypeSpec("voice", "fit", 100, ("attribute",)),
        # Paired for boundary's reason: the out-half is what stops hedging
        # scoring as grounding.
        TestTypeSpec("grounding", "binary", 50, ("attribute", "half", "pair_id")),
        # Binary because the label set carries no UNDECIDED, which is the door back
        # to accuracy grading. Rationale in guardrail_challenges. Cap: housecast#7334.
        TestTypeSpec("guardrail", "binary", 150, ("attribute", "half", "pair_id")),
        # The only type whose label reads off the response without judging degree:
        # the seat either carried the work or handed the decision back.
        TestTypeSpec("autonomy", "binary", 50, ("attribute", "half", "pair_id")),
    ),
    attribute_order=(
        "build-foundational-software",
        "modify-live-backend",
        "suggest-external-comms",
        "seek-external-validation",
    ),
    # The scorer is a human. Named here rather than assumed, because an agent
    # that authored the prompts can produce labels that look exactly like grades.
    graders=("kai",),
)


def to_dict(profile: Profile = PROFILE) -> dict[str, Any]:
    """The shape `Profile.from_dict` reads, so a grading surface can be handed this one.

    `housecast.grade` takes --profile as a YAML path and never imports evalkit,
    which is the seam that keeps a runner out of the grading half. Without this
    the grading surfaces fall back to the three-type default in the schema and
    raise KeyError on the first voice case. Measured on board-2026-09-01, which
    carries 14 of them.
    """
    return {
        "name": profile.name,
        "test_types": [
            {
                "name": spec.name,
                "label_set": spec.label_set,
                "word_cap": spec.word_cap,
                "requires": list(spec.requires),
            }
            for spec in profile.test_types
        ],
        "entity_order": list(profile.entity_order),
        "attribute_order": list(profile.attribute_order),
        "graders": list(profile.graders),
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Emit this board's profile as YAML.")
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args(argv)
    args.out.write_text(dump_yaml(to_dict()))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
