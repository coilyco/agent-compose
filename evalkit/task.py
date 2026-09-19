"""The Inspect task. Replaces a hand-rolled fan-out with `inspect eval`.

Run it unscored, because the scorer is a human. Inspect calls the repetition an
epoch, which is what n=5 was. See docs/evaluation.md.

    AGENTPROXY_BASE_URL=http://ser8:8080/v1 \\
    inspect eval evalkit/task.py --model openai-api/agentproxy/evaluation/deepseek-v4-pro \\
      --epochs 5 --no-score -T challenges=challenges.yaml -T prompts=.evalkit/prompts
"""

from __future__ import annotations

import os
from pathlib import Path

from inspect_ai import Task, task
from inspect_ai.dataset import MemoryDataset
from inspect_ai.solver import generate, system_message

from evalkit.filter import load_challenges
from evalkit.inspect_bridge import to_inspect

DEFAULT_CHALLENGES = Path("challenges.yaml")
DEFAULT_PROMPTS = Path(".evalkit/prompts")


def load_system_prompts(directory: Path) -> dict[str, str]:
    return {p.stem: p.read_text().strip() for p in sorted(directory.glob("*.md"))}


@task
def board(
    challenges: str | Path = DEFAULT_CHALLENGES, prompts: str | Path = DEFAULT_PROMPTS
) -> Task:
    """One task per run. The composed role bundle is the system message."""
    written = load_challenges(Path(challenges))
    composed = load_system_prompts(Path(prompts))

    missing = sorted({c.entity for c in written} - composed.keys())
    if missing:
        raise ValueError(f"no composed system prompt for: {', '.join(missing)}")

    # One dataset per role would need one task per role. Instead the system
    # message is resolved per sample from its own role metadata.
    entries = []
    for challenge in written:
        inspect_sample = to_inspect(challenge)
        inspect_sample.metadata = dict(inspect_sample.metadata or {})
        inspect_sample.metadata["system_prompt"] = composed[challenge.entity]
        entries.append(inspect_sample)

    return Task(
        dataset=MemoryDataset(entries),
        solver=[system_message("{system_prompt}"), generate()],
    )


def agent_proxy_configured() -> bool:
    """Inspect reads <PROVIDER>_BASE_URL, so the proxy is named agentproxy."""
    return bool(os.environ.get("AGENTPROXY_BASE_URL"))
