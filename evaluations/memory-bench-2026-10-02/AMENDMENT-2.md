# Amendment 2: the model moves to ministral-3-14b

Written 2026-10-02 at about 09:10Z, before any lore case existed or ran.

Kai was running a game on the same GPU and expected `ornith:35b` to crawl beside
it. She asked to size down, or move to a Qwen3 model.

* **No Qwen3 route exists.** `/v1/models` lists 18 routes, none of them Qwen.
  Adding one is a change to the Deploy repo, which is not this seat's to make.
* **Every local route is on kai-tower-3026.** At deploy `ca7d09df` those are
  `ornith:35b`, `ornith:9b` and `ministral-3:14b`. `ornith:9b` is reachable only
  through `sirens-echo/default`, a service route that falls back to `ornith:35b`.
* **So the bench uses `evaluation/ministral-3-14b`.** It is a local Ollama
  route with no fallbacks. It replaces ornith-35b for case writing and for both
  conditions, as `model` and `small_model`. `BENCH_ROUTE` sets it in
  `casegen.py` and `bench.py`.

The ornith-35b case generation was stopped after 18.5 minutes and wrote nothing,
so no lore case came from ornith. Smoke on ministral, one synthetic sentinel each
(MEASURED, with the game running):
* **C1:** correct, source cited, 4 tool calls, 65,384 input tokens, 174.7 s. It
  ran first, so the time likely includes the model load.
* **C0:** correct, source cited, 2 tool calls, 44,915 input tokens, 14.6 s.

The predictions in `PREDICTION.txt` were written for ornith-35b and stand as
written. A 14B model may lower both conditions, so the comparison between them is
the number that carries over, not either absolute.
