"""Reading tool calls off Inspect's log, and carrying the expectation into it.

Inspect splits a call from its answer across two messages, so these use its real
`ToolCall` and `ChatMessage` types rather than stubs. A stub here would prove
only that the extractor agrees with my picture of the log.
"""

from __future__ import annotations

from typing import Any

from housecast.grade.schema import Challenge
from inspect_ai.model import ChatMessageAssistant, ChatMessageTool
from inspect_ai.tool import ToolCall as InspectToolCall

from evalkit.filter import _calls
from evalkit.inspect_bridge import from_inspect, to_inspect


def challenge(**fields: Any) -> Challenge:
    return Challenge(
        id="fs-selectable-in",
        entity="filesystem.write_file",
        test_type="boundary",
        prompt="save this to notes.md",
        target="calls write_file",
        **fields,
    )


def call(name: str, call_id: str = "1", **arguments: str) -> InspectToolCall:
    return InspectToolCall(id=call_id, function=name, arguments=dict(arguments))


def test_the_expectation_rides_into_the_sample() -> None:
    sample = to_inspect(challenge(required_tool="write_file"))
    assert sample.metadata is not None
    assert sample.metadata["required_tool"] == "write_file"


def test_the_expectation_survives_the_round_trip() -> None:
    restored = from_inspect(to_inspect(challenge(required_tool="write_file")))
    assert restored.required_tool == "write_file"


def test_a_case_with_no_expectation_carries_no_empty_key() -> None:
    """The negative control. An empty expectation on every sample reads as one."""
    sample = to_inspect(challenge())
    assert sample.metadata is not None
    assert "required_tool" not in sample.metadata


def test_calls_are_read_off_the_assistant_message() -> None:
    message = ChatMessageAssistant(content="", tool_calls=[call("write_file", path="notes.md")])
    extracted = _calls(message, [])
    assert [c.name for c in extracted] == ["write_file"]
    assert extracted[0].arguments == {"path": "notes.md"}


def test_a_call_is_paired_with_the_message_that_answered_it() -> None:
    message = ChatMessageAssistant(content="", tool_calls=[call("write_file", call_id="abc")])
    answer = ChatMessageTool(content="wrote 12 bytes", tool_call_id="abc", function="write_file")
    extracted = _calls(message, [answer])
    assert extracted[0].result == "wrote 12 bytes"
    assert extracted[0].error == ""


def test_an_unanswered_call_is_kept() -> None:
    """A call that got nothing back is the observation, not a reason to drop it."""
    message = ChatMessageAssistant(content="", tool_calls=[call("write_file", call_id="abc")])
    extracted = _calls(message, [])
    assert [c.name for c in extracted] == ["write_file"]
    assert extracted[0].result == ""


def test_a_message_with_no_calls_extracts_nothing() -> None:
    assert _calls(ChatMessageAssistant(content="prose only"), []) == ()


def test_the_answer_is_matched_by_id_not_by_position() -> None:
    message = ChatMessageAssistant(
        content="",
        tool_calls=[call("read_file", call_id="one"), call("write_file", call_id="two")],
    )
    answers = [
        ChatMessageTool(content="wrote it", tool_call_id="two", function="write_file"),
        ChatMessageTool(content="read it", tool_call_id="one", function="read_file"),
    ]
    extracted = _calls(message, answers)
    assert [(c.name, c.result) for c in extracted] == [
        ("read_file", "read it"),
        ("write_file", "wrote it"),
    ]
