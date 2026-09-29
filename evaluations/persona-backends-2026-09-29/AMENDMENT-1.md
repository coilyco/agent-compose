# Amendment 1: the assignment rule, and how a pick is confirmed

Seat: science. Committed 2026-09-29T18:54Z, `a372696`. Before this amendment I had scored runs A
and B of `current` and nothing else. No recombined assignment was scored before it.

## Why

Between 18:48Z and 18:53Z sysadmin-senior relayed that Kai confirmed the room's model set is
`chat/gemini-3-5-flash`, `chat/glm-5-3` and `chat/minimax-m3`, and that the Kimi K3,
Qwen3.8-Max and Mistral Medium 3.5 mapping is stale. I have that as a relay and have
not seen it from Kai. `teable:coilyco/housecast#8487` does not say which persona takes
which route. At about 18:53Z prod-director set the rule below and said no record on disk
holds a current assignment. The preregistered `mapped` config stays unrunnable and stays
NO-GO for the reason on the preregistration.

## The rule, from prod-director (not from Kai)

* Three subjects each take a distinct route from the three new ones. The fourth stays on
  `evaluation/deepseek-v4-pro`, so one subject is always on the rehearsed path.
* Pick from the runs already made, by the persona-by-route grid. Do not rerun if the grid
  answers it. State the pick and its runner-up.
* If no assignment beats the all-baseline route on divergence without failures, NO-GO.

## What I add, because picking the best of 24 on the data that ranks them overstates it

* **Candidates:** 4 choices of which subject stays on pro, times 3! orders of the routes
  over the other three, so **24 assignments**. Creature names for the verdict: Gem is
  Panda-Goose, Delphi is Imp-Dragonfly, Sprite is Whale-Dragonfly, Evie is Frog-Ox.
* **Recombination:** a candidate round for prompt p and rep r takes each subject's
  answer from the single-route run of that subject's assigned route, at the same p
  and r. Runs `A` (pro), `gemini`, `glm` and `minimax` supply the answers. Jev scores each
  recombined round. Latency of a recombined round is the largest of its four answer
  latencies, **modeled, not measured**, because the four calls were not made together.
* **Selection:** rank candidates on reps 0 and 1 only (20 rounds, 16 persona and 4
  control), by mean Jev level over the persona rounds. A candidate is eligible only if
  none of its four cells holds a lost answer, its modeled round p95 is at most 90s, and
  its control mean is at most 1.0 level. Ties go to the lower modeled round p95.
* **Confirmation:** the chosen candidate and the runner-up each get a **fresh direct run**,
  30 rounds, four subjects called together, as the room would call them. Rep 2 of the
  grid is not used to pick, and the direct run is the number that counts.
* **GO** only if the chosen candidate's direct run has zero lost subjects, round p95 at
  most 90s, control mean at most 1.0 level, and the difference in persona-round means
  against the pooled `current` runs A and B has a 95 percent bootstrap interval (rounds
  resampled within each group, 10,000 draws, seed 0) whose lower bound is above zero.
  Otherwise NO-GO. This replaces the preregistration's gap rule, which compared against
  one run and used a noise estimate from two.
* **Control correctness, added:** ctl-mult answers containing `56`, and ctl-capital
  answers containing `Paris`, per config. Divergence says the subjects differ, and this
  is the only check here on whether the factual answers are right.
* **The #8295 measure, tightened:** the preregistered pattern counts the word MEASURED
  anywhere. Reading the 7 hits in run A, all 7 are denials such as "EXPECTED, not
  MEASURED", and none claims a run. I did this after seeing run A, so it is post-hoc. From
  here I also count answers with a fenced `bash` or `sh` block, and answers with MEASURED
  at the start of a line, and I read every hit.
* **Fallback:** among `gemini`, `glm` and `minimax` single routes, the lab must differ
  from the primary's, which excludes the DeepSeek routes.

## What none of this measures

Answer quality as a reader would judge it, and persona fidelity. Divergence is how far
apart the stances are, and it is high when subjects contradict each other for any reason.
