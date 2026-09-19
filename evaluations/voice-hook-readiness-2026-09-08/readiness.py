#!/usr/bin/env python3
"""Size the blocking surface of the shipped voice profile before it becomes a hook.

The catchphrase evaluation beside this one asked which rules can ship. This one
asks what happens when the shipped set runs unconditionally: how many findings
land on day one, across how many files, and which rules produce them.

The engine and the profile both live in voice-corpus and are read by path. This
runner vendors neither, because a copy would be a second source of truth for a
file another repository owns.

    python3 readiness.py --engine <lint.py> --profile <profile.json> <repo-root>...
"""

import argparse
import collections
import importlib.util
import subprocess
import sys
from pathlib import Path


def load_engine(path: Path):
    """Import the linter from an explicit path, never by module name.

    Resolving by name would pick up whatever sits in the working directory,
    which for this runner is a checkout it does not own.
    """
    spec = importlib.util.spec_from_file_location("voice_lint_engine", path)
    if spec is None or spec.loader is None:
        raise SystemExit(f"readiness: cannot load engine at {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def tracked_markdown(repo: Path) -> list[Path]:
    out = subprocess.run(
        ["git", "-C", str(repo), "ls-files", "-z", "*.md"],
        capture_output=True,
        text=True,
        check=True,
    )
    return [repo / name for name in out.stdout.split("\0") if name]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--engine", required=True, type=Path)
    parser.add_argument("--profile", required=True, type=Path)
    parser.add_argument("repos", nargs="+", type=Path)
    args = parser.parse_args()

    engine = load_engine(args.engine)
    rules = engine.load_profile(args.profile)

    per_repo: dict[str, int] = collections.Counter()
    per_rule: dict[str, int] = collections.Counter()
    files_hit: set[Path] = set()
    files_hit_nonpronoun: set[Path] = set()
    corpus = 0

    for repo in args.repos:
        label = repo.name
        for path in tracked_markdown(repo):
            corpus += 1
            for found in engine.lint_file(path, rules):
                _, _, rule_id, _, _ = found
                per_repo[label] += 1
                per_rule[rule_id] += 1
                files_hit.add(path)
                if not rule_id.startswith("wrong-pronoun"):
                    files_hit_nonpronoun.add(path)

    total = sum(per_rule.values())
    pronoun = sum(count for rule, count in per_rule.items() if rule.startswith("wrong-pronoun"))

    print(f"rules in profile: {len(rules)}")
    print(f"corpus files: {corpus}")
    print(f"total findings: {total}")
    print(f"pronoun-rule findings: {pronoun} ({pronoun * 100 // max(total, 1)}%)")
    print(f"other findings: {total - pronoun}")
    print(f"files with any finding: {len(files_hit)} of {corpus}")
    print(f"files with a non-pronoun finding: {len(files_hit_nonpronoun)} of {corpus}")
    print("\nper repository")
    for label, count in per_repo.most_common():
        print(f"  {label}: {count}")
    print("\nrules firing, most first")
    for rule_id, count in per_rule.most_common():
        print(f"  {rule_id}: {count}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
