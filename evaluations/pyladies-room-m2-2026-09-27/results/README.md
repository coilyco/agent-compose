# M2 result: P1 and P3 pass, P4 fails, P2 half-measured

2026-09-26, science seat. Dev room at `http://kai-server:30123` on housecast `7966eb6`,
pod `pyladies-remote-dev-86498d46c-bqbtr`, which had 0 restarts through the run. Runner
at `dd391be`. The run went from about 05:43:50Z to 05:47:10Z. `run.json` has the setup,
`submits.jsonl` and `snapshot.json` are raw, and `score.txt` is
`just evalkit-room-m2 score evaluations/pyladies-room-m2-2026-09-27/results`.

42 prompts were submitted, all `201`, with 0 `429`s. That's 168 answers. The aborted
05:42Z start is in `../results-aborted-0542/` and is not counted here.

## Against the preregistered thresholds

* **P1 load: PASS.** At 10 simultaneous submitters, n=20: 0 errors, with time to all four
  at p50 8.6s, p95 16.0s and max 16.7s. The bar was p95 at most 90s.
  * At 5, n=10: p50 8.4s, p95 14.9s. At 1, n=2: p50 7.0s, p95 7.8s.
  * S1 sequential, n=10: p50 8.1s, p95 24.9s. The slowest was disc-review at 30.8s.
* **P2 answers: empty half PASS, markup UNMEASURED.** 0/168 answers were empty. The markup
  half waits on the room owner's post-strip export, per AMENDMENT-1.
* **P3 divergence: PASS.** The persona minus control gap was +0.334 (1.34 Jev levels). The
  bar was at least 0.25. Controls: ctl-mult 0.21, ctl-capital 0.16, both at or under 0.25.
  All 10 S1 prompts were scored `stance`, with 0 `lexical`. Every S1 prompt is n=1.
* **P4 progress: FAIL.** 20/42 prompts had at least 5s between the first and fourth
  answer. The bar was 80%. The spreads sorted: 0.4 to 4.9s for 22 prompts, and 5.0 to
  26.0s for 20.
* **Stall (AMENDMENT-3):** 0 prompts with an empty or failed answer, and 0 with an answer
  that never reached a terminal state.

## Provider (AMENDMENT-2), SigNoz, read-only

* `count_distinct(trace_id)` of agent-proxy `request.chat` with `agent.session_id =
  'pyladies-room-dev'` over 05:43:30 to 05:47:40Z was **168**, equal to the 168 answers.
* `litellm_request` spans in the same window: **168**, all `deepseek/deepseek-v4-pro` on
  `evaluation/deepseek-v4-pro`, none with an error. No other traffic was in the window, so
  the count is exact: **0 served by the Baseten fallback, 0 failed primary attempts.**
  The comparison with `bundle-divergence-2026-09-24` holds on provider.

## Against the prediction

* Time to all four: predicted p50 about 25s at 1 and p95 about 80s at 10. Measured about
  8s and 16.0s. **Wrong, 3 to 5x faster.**
* Errors at 10: 30% chance of at least one. Measured 0.
* Empty answers under 5% only with a raised cap or frame: held at 0/168. The room owner
  reported both.
* Divergence gap about 1.3 levels: **held**, at 1.34. Controls about 0.3 levels:
  **wrong**, measured 0.84 and 0.64 levels. They're under the bar, but it's the closest
  margin in the run.
* P4 pass: **wrong**, it failed.

## Inference, not measured

* The speed-up is most likely the short-answer frame the room adds (a 4000 cap, one or two
  sentences), against `bundle-divergence-2026-09-24`'s long answers at a 1500 cap. The
  provider is the same, so the length of what the model writes is what moved.
* P4 fails for the same reason. With answers this short, the four finish close together:
  22/42 were within 5s, and mostly on deck prompts. Per-subject progress on a screen
  will flash rather than build for about half of prompts. Nothing stalled. Whether that
  matters for the room is a display call. The 5s bar was set when about 15s calls were
  expected, and it isn't relaxed here after the fact.
* The controls sitting at 0.16 to 0.21 means a factual prompt lands mid-table on the
  leaderboard, not at the bottom. That's the thing to watch in the room.
