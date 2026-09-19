# The MCP tool-description loop, first closed run

2026-09-16. Every number this repository quotes about the loop comes from the
files beside this one. They are the record of what was true when the runs
executed and are never rewritten to match a later source. The loop itself is
documented in the module docstrings under `housecast/mcpeval/`, which is where
its walkthrough lives: `docs/` is at its 20-page cap, so no page was added.

## What was held constant

* task `transcribe-video`, 20 prompts
* subject `mediakit-0.4.2` over HTTP MCP at `127.0.0.1:8931/mcp`
* model `evaluation/deepseek-v4-flash`, temperature 0.0, seed 7, no `tool_choice`
* concurrency 8

## What moved

One tool's description. `media_analyze` went from prose claiming it returns
"text, speech, metadata and content" to prose saying it returns metadata only
and naming `media_transcribe` instead. Definition sets `acc3eda0674f` (baseline)
and `69d25b60cb3a` (edited) are in `definitions/`.

## The runs

| run | definitions | mean | seconds |
| --- | --- | --- | --- |
| `run_b5a0f3b9b5` | acc3eda0674f baseline | 0.667 | 26.4 |
| `run_1ffba2f586` | acc3eda0674f baseline | 0.592 | 24.3 |
| `run_5718c6408b` | acc3eda0674f baseline | 0.617 | 24.9 |
| `run_5547856053` | 69d25b60cb3a edited | 0.742 | 22.6 |
| `run_e0babf2bd3` | 69d25b60cb3a edited | 0.800 | 22.8 |

Three baseline runs rather than one, because a single before-number cannot say
whether a delta cleared the noise. The baseline spread is 0.592 to 0.667, and
both edited runs sit above all three.

## Reproducing the comparisons

```sh
housecast mcpeval compare runs/run_5718c6408b.json runs/run_5547856053.json
housecast mcpeval compare runs/run_1ffba2f586.json runs/run_e0babf2bd3.json
```

Output at the time of the run:

```
transcribe-video: baseline -> narrow media_analyze so it stops winning transcription
  11 improved, 4 regressed, 5 held, of 20 paired
  sign test p=0.118 over 15 moved (cannot support a conclusion at this n)
  DECISION promote

transcribe-video: baseline -> narrow media_analyze so it stops winning transcription
  14 improved, 4 regressed, 2 held, of 20 paired
  sign test p=0.031 over 18 moved (clears 0.05)
  DECISION promote
```

## The one thing a person should adjudicate

`p13` and `p14` regressed in both comparisons. Both are the no-audio edge case
(`/media/b-roll-skyline.mp4`), where the correct behaviour is to report the
missing audio track rather than produce a transcript. A consistent regression
across two independent pairs is a trade rather than noise, and the reading worth
checking is that a model no longer reaching for `media_analyze` also no longer
discovers the missing track that way.

## Disclosure

The prose under test, the correctness rules, and the edit were all written by
this seat. A subject and a criterion from one hand is a fact a grader needs and
cannot otherwise see, so it is stated here rather than inferred.
