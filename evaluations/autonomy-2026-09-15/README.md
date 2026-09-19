# Autonomy board, 2026-09-15

First run of the `autonomy` test type. Ungraded.

## The grades that were here are withdrawn

The science seat graded these 14 cases on 2026-09-15 and the grades are deleted,
not corrected. Kai: "only humans are allowed to grade."

The seat had authored the prompts, authored the targets, and then marked its own
work against them. It also wrote the file by hand rather than through
`housecast grade`, which is why nothing refused it at the time. `Profile.graders`
now does. See `housecast/grade/schema.py`.

What remains here is the run and its responses. Those are evidence. The labels
were not.

## The run

```
14 cases, 3 epochs, evaluation/deepseek-v4-pro via Agent Proxy, 50-word cap
just evalkit-run, EVAL_EPOCHS=3
```

`PREREGISTER.md` was written before the run. All three of its predictions were
wrong, including both controls, and the record carries that:
`teable:coilyco-flight-deck/agentic-os#7688`.

`dataset.yaml` is the epoch-1 response per case, which is what was graded.

## What the automated grader did on the same responses

`evalkit/autonomy.py` abstained on 8 of these 14, and on 15 of the 28 held-out
responses. It is a reading aid, not a scorer, and it writes no annotation file.

Its abstention is its own limit rather than the board's. `platform-aut-in` reads
as `unclear` to the classifier while plainly failing its target to a reader,
because impersonal advice is a response to "takes the edit as its own work"
rather than an absence of one. A human grader would not hesitate there.

## What reading these turned up, which matters more than any grade

`evalkit.coverage` globbed `annotations.yaml` exactly, so every
`annotations.<grader>.yaml` was invisible to it. The report claimed **119
ungraded cases**. 103 of them already carried a label in
`evaluations/split-candidates-2026-09-08/annotations.kai.yaml`.

Corrected in the same commit as these grades:

```
before   ungraded 119   orphaned 0
after    ungraded   3   orphaned 2
```

The three: `gamedev-aut-in` above, plus `platform-fit-analyst` and
`sysadmin-fit-analyst`, whose target seat is archived. The two orphans,
`platform-fit-sysadmin` and `sysadmin-fit-director`, are adjacency edges Kai
graded that the roster no longer derives. That drift was real the whole time and
the narrow glob hid it.

**This retires the argument that built the autonomy grader.** "105 cases waiting
on the operator" was the reason given for automating this grading, in
`housecast#111` and on the tracker record. The board was already graded. The
queue the grader was built to drain did not exist.
