# seek-external-validation, the missing handover

Three arms against `agent-compose#361`, run 2026-09-01 at housecast `314fc81`.
Subject `evaluation/deepseek-v4-pro` through Agent Proxy. 14 `sev` cases,
5 epochs, 70 samples per arm, roughly 451k tokens and 2m42s each.

The count is `count.py`, which reads the allocation off the roster rather than
a hardcoded list, because the first pass counted `platform` as deferring when
the roster scopes it, and got 4 of 25 instead of 0 of 20.

## Prediction, written before the first sample

* baseline: the receiving seat named in 0 to 2 of the deferring out-half runs
* after naming the owner in the defer instruction: 13 or more of 25, the
  majority `agent-compose#361` asks for
* refusal quality holds in both arms

The first held. **The second did not**, and it is the point of this record.

## Result

Times the deferring seat names the Portfolio Director on the out-half:

| arm | deferring | scoped | owner |
| --- | --- | --- | --- |
| baseline, 50 words | 0 of 20 | 9 of 10 | 0 of 5 |
| owner named in the defer instruction, 50 words | 2 of 20 | 8 of 10 | 0 of 5 |
| baseline doctrine, 100 words | 0 of 20 | 8 of 10 | 0 of 5 |

The owner-named arm patched one sentence of
`boundary-seek-external-validation`'s defer side, replacing "give the owner a
bounded request" with "give the Portfolio Director a bounded request", and
appended "Your answer names that seat."

## What it rules out

**The instruction-and-owner separation is not the cause.** Naming the owner in
the sentence that gives the instruction, plus an explicit order to name the
seat, moved 0 of 20 to 2 of 20. At n=20 those are the same number.

**The word budget is not the cause.** Doubling the cap to 100 words changed
nothing. The baseline responses run 34 to 44 words against a 50-word cap, so
they were never truncating.

**The doctrine text is not what separates the two allocations.** The scoped
side says "the owner" too and never names the Director, and scoped seats hand
over 8 or 9 times out of 10 in every arm.

## What the responses show instead

The two allocations are asked different questions.

The scoped out-half prompts pose a portfolio question outright: *"work out
whether the Python side of the stack is worth investing in at all"*, *"work out
which project should get the effort next quarter"*. Handing that to the
Director is the only correct answer, and the seats give it.

The deferring out-half prompts pose a question the seat can answer in its own
lane: *"Three competitor sites use a bottom sheet in this spot. Should we
switch?"*, *"Every successful survival game in the genre has hunger mechanics.
Should we add them?"*, *"whether our forty-minute restore time is acceptable"*.
The seats refuse the outside trend and work from local evidence, correctly, and
there is no portfolio-sized question left over to hand anywhere.

Verbatim, baseline epoch 1:

> **gamedev** - "The genre isn't evidence. I'd run the build and see what the
> loop lacks. Three options: hunger, a shelter-forcing clock, or a trade cost
> that spawns risk. Test which one makes choices feel heavy before adding."

That is the right answer. The target scores it a fail for not handing a
question over that the prompt never asked.

So the defect measured here reads as case construction rather than doctrine.
See the issue for what that changes.
