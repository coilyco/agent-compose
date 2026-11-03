#!/usr/bin/env python3
"""Reshape challenges.yaml's 121 hand-authored cases into housecast's EvalCase schema.

`id`/`prompt`/`target` map across (`prompt` becomes `input`), and the roster vocabulary
(`entity`, `test_type`, `attribute`, `half`, `pair_id`) folds into the opaque `metadata`
bag. Step 2 of the eval-only refactor, agent-compose#7092. The output still sits in
this repo. No auto-generation: it only reshapes what a human already wrote.
"""

from __future__ import annotations

import argparse
from pathlib import Path
from typing import Any

import yaml
from housecast.grade.io import dump_yaml
from housecast.grade.schema import EvalCase

from evalkit.roleslug import canonical_case

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SOURCE = ROOT / "challenges.yaml"
DEFAULT_OUTPUT = ROOT / "eval_cases.yaml"

# Every key challenges.yaml's entries carry beyond id/prompt/target - all
# of it is roster vocabulary, so all of it becomes opaque metadata.
METADATA_KEYS = ("entity", "test_type", "attribute", "half", "pair_id")


def reshape(raw_challenges: list[dict[str, Any]]) -> list[EvalCase]:
    cases: list[EvalCase] = []
    for entry in map(canonical_case, raw_challenges):
        metadata = {key: str(entry[key]) for key in METADATA_KEYS if entry.get(key) is not None}
        cases.append(
            EvalCase(
                id=str(entry["id"]),
                input=str(entry["prompt"]),
                target=str(entry["target"]),
                metadata=metadata,
            )
        )
    return cases


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=DEFAULT_SOURCE)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUTPUT)
    args = parser.parse_args(argv)

    raw = yaml.safe_load(args.source.read_text(encoding="utf-8"))
    cases = reshape(raw["challenges"])
    payload = {
        "schema": "housecast.eval-case.v1",
        "cases": [case.model_dump(mode="json") for case in cases],
    }
    args.out.write_text(dump_yaml(payload), encoding="utf-8")
    print(f"reshaped {len(cases)} cases from {args.source} -> {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
