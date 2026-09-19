"""Report the gap between the board the roster implies and the board that exists.

The board is derived, so a roster edit silently changes which cases exist.
Nothing reported that a boundary was added and its implied cases were never
written, or written and never graded, or that a role was renamed and its graded
evidence went orphaned. This does. See docs/evaluation.md.

It reports and exits zero by default. The gate is deliberate rather than
accidental: the grader is a human by hand, so a blocking check would put every
roster edit behind an annotation session. Flip `blocking` in pyproject.toml.
"""

from __future__ import annotations

import argparse
import json
import sys
import tomllib
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml
from housecast.grade.schema import Profile

from checks import roster as roster_module
from checks import snapshot as snapshot_module
from evalkit.profile import PROFILE
from evalkit.provider import RoleProvider

ROOT = Path(__file__).resolve().parent.parent
CHALLENGES = ROOT / "challenges.yaml"
EVALUATIONS = ROOT / "evaluations"
CONFIG_SECTION = ("tool", "evalkit", "coverage")


@dataclass(frozen=True)
class Config:
    blocking: bool = False
    retired_runs: tuple[str, ...] = ()


@dataclass
class Report:
    derived: set[str]
    authored: set[str]
    unauthored: set[str]
    stale: set[str]
    ungraded: set[str]
    orphaned: dict[str, set[str]] = field(default_factory=dict)
    # Files whose grader is not on the profile roster. Reported rather than
    # skipped: a label the report drops silently is how one gets counted.
    rejected: dict[str, str] = field(default_factory=dict)

    @property
    def clean(self) -> bool:
        return not (
            self.unauthored or self.stale or self.ungraded or self.orphaned or self.rejected
        )


def load_config(pyproject: Path | None = None) -> Config:
    """Read the switch from configuration, so flipping it is not a code change."""
    path = pyproject or ROOT / "pyproject.toml"
    node: Any = tomllib.loads(path.read_text(encoding="utf-8"))
    for key in CONFIG_SECTION:
        node = node.get(key, {}) if isinstance(node, dict) else {}
    return Config(
        blocking=bool(node.get("blocking", False)),
        retired_runs=tuple(str(entry) for entry in node.get("retired_runs", [])),
    )


def project(path: Path) -> dict[str, Any]:
    """The person snapshot, straight from the YAML roster with no file in between."""
    return dict(json.loads(snapshot_module.dumps(roster_module.load(path))))


def authored_ids(path: Path) -> set[str]:
    document = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return {str(entry["id"]) for entry in document.get("challenges", [])}


def graded_runs(
    root: Path, retired: tuple[str, ...] = (), profile: Profile = PROFILE
) -> tuple[dict[str, set[str]], dict[str, str]]:
    """Every run directory holding annotations, keyed the way a reader would type it.

    Relative to the tree's own parent rather than to ROOT, so a caller pointing
    at a fixture gets the same `evaluations/<run>` shape the default produces.

    `annotations.<grader>.yaml` counts. Matching only the bare name read a board
    graded by one named grader as entirely ungraded: on 2026-09-15 this reported
    119 ungraded cases where 103 already carried a label in
    `annotations.kai.yaml`. Two graders on one run union, because either one is
    a label on record and this report is about presence, not agreement.

    A file whose `grader` the profile does not permit contributes nothing and is
    returned separately, so the caller reports it rather than dropping it. A
    label this report ignores quietly is how one ends up counted.
    """
    base = root.resolve().parent
    runs: dict[str, set[str]] = {}
    rejected: dict[str, str] = {}
    for record in sorted(root.rglob("annotations*.yaml")):
        rel = record.parent.resolve().relative_to(base).as_posix()
        if any(rel == entry or rel.startswith(f"{entry}/") for entry in retired):
            continue
        document = yaml.safe_load(record.read_text(encoding="utf-8")) or {}
        named = document.get("grader")
        grader = None if named is None else str(named)
        if not profile.permits(grader):
            rejected[record.resolve().relative_to(base).as_posix()] = grader or "unattributed"
            continue
        ids = {str(entry["id"]) for entry in document.get("annotations", [])}
        runs.setdefault(rel, set()).update(ids)
    return runs, rejected


def build(
    roster_path: Path,
    challenges: Path = CHALLENGES,
    evaluations: Path = EVALUATIONS,
    config: Config | None = None,
    profile: Profile = PROFILE,
) -> Report:
    settings = config or Config()
    derived = {challenge.id for challenge in RoleProvider(project(roster_path)).challenges()}
    authored = authored_ids(challenges)
    runs, rejected = (
        graded_runs(evaluations, settings.retired_runs, profile)
        if evaluations.is_dir()
        else ({}, {})
    )
    everything_graded: set[str] = set().union(*runs.values()) if runs else set()
    return Report(
        derived=derived,
        authored=authored,
        unauthored=derived - authored,
        stale=authored - derived,
        ungraded=(authored & derived) - everything_graded,
        orphaned={run: ids - derived for run, ids in runs.items() if ids - derived},
        rejected=rejected,
    )


def render(report: Report, config: Config) -> str:
    lines = [
        f"{len(report.derived)} cases derived from the roster, "
        f"{len(report.authored)} authored in challenges.yaml",
        "",
    ]
    for title, ids in (
        ("unauthored - derived from the roster, no prompt written", report.unauthored),
        ("stale - authored, but the roster no longer derives it", report.stale),
        ("ungraded - authored and derived, no annotation on record", report.ungraded),
    ):
        lines.append(f"{title}: {len(ids)}")
        lines += [f"  {case}" for case in sorted(ids)]
    total = sum(len(ids) for ids in report.orphaned.values())
    lines.append(f"orphaned - graded, but the roster no longer derives it: {total}")
    for run, ids in sorted(report.orphaned.items()):
        lines.append(f"  {run}: {len(ids)}")
        lines += [f"    {case}" for case in sorted(ids)]
    lines.append(f"rejected - a grader this board does not permit: {len(report.rejected)}")
    for record, grader in sorted(report.rejected.items()):
        lines.append(f"  {record}: grader {grader}")
    lines.append("")
    where = "pyproject.toml [tool.evalkit.coverage]"
    if config.blocking:
        lines.append(f"blocking: on. A non-empty report exits non-zero. Set in {where}.")
    else:
        lines.append(f"blocking: off. Reporting only. Set blocking = true in {where}.")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Report board coverage against the roster.")
    parser.add_argument("--roster", type=Path, default=roster_module.DATA)
    parser.add_argument("--challenges", type=Path, default=CHALLENGES)
    parser.add_argument("--evaluations", type=Path, default=EVALUATIONS)
    parser.add_argument("--config", type=Path, default=None, help="the pyproject.toml to read")
    args = parser.parse_args(argv)

    config = load_config(args.config)
    report = build(args.roster, args.challenges, args.evaluations, config)
    print(render(report, config))
    if config.blocking and not report.clean:
        print("coverage: reporting is blocking and the board is not covered", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
