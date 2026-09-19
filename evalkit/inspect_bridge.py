"""Translate the shared schema to and from Inspect's Sample.

This is the seam the shared layer deliberately does not cross. `housecast.grade`
carries no runner and no model client, so the mapping onto Inspect's carrier
lives here, in the repo whose runner is Inspect.
"""

from __future__ import annotations

from enum import StrEnum

from housecast.grade.schema import Challenge
from inspect_ai.dataset import Sample as InspectSample

# Inspect's Sample carries only input, target, id, and metadata, so every
# domain field rides in metadata.
METADATA_FIELDS = (
    "entity",
    "test_type",
    "attribute",
    "half",
    "pair_id",
    # The runner reads this back off the sample to check the answer's shape, so
    # a field that does not ride here is a shape expectation nothing enforces.
    "required_tool",
)


def to_inspect(challenge: Challenge) -> InspectSample:
    # An unwritten challenge has no prompt, so the runner is where that stops.
    if not challenge.written:
        raise ValueError(f"{challenge.id}: an unwritten challenge cannot be run")
    metadata = {
        key: (value.value if isinstance(value, StrEnum) else value)
        for key, value in ((name, getattr(challenge, name)) for name in METADATA_FIELDS)
        # An empty string is an absent expectation rather than an empty one, and
        # carrying it would put `required_tool: ""` on every sample.
        if value is not None and value != ""
    }
    return InspectSample(
        id=challenge.id,
        input=str(challenge.prompt),
        target=str(challenge.target),
        metadata=metadata,
    )


def from_inspect(inspect_sample: InspectSample) -> Challenge:
    metadata = dict(inspect_sample.metadata or {})
    return Challenge(
        id=str(inspect_sample.id),
        prompt=str(inspect_sample.input),
        target=str(inspect_sample.target),
        **metadata,
    )
