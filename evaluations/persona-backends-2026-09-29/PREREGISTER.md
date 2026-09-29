# Pre-registration: do per-persona model backends beat the current single route for the room

Seat: science. Written 2026-09-29T18:41Z, before any model call in this evaluation.
`teable:coilyco/housecast#8487`, dispatched by prod-director. The session is Wed
2026-09-30 07:00 PDT. The verdict is due Tue 2026-09-29 18:00 PDT.

## Claim

Routing the room's four subjects to four different hosted models (the mapped
config) answers every round with no lost subject, and diverges at least as much as
the current single route, without a slower tail.

## Sources for the mapping, named

* **Mapped config:** Kai's 2026-09-25 pairing artifact
  (`claude.ai/artifact/WokJsFC1a9XbCgnGUEJYD2`, "updated 2026-09-25"), tracked in
  `teable:coilyco/agent-compose#8190`. Gem carries Mistral Medium 3.5, Delphi
  Kimi K3, Sprite Qwen3.8-Max, Evie DeepSeek V4.1 Flash. The artifact keys models to
  **role seats**, and the four room subjects are the Claude side bundles of those
  role types. So the artifact assigns the model to a seat, and the room use of it is
  the dispatcher's reading.
* **Newer naming:** `teable:coilyco/deploy#8440` (2026-09-28) names Gemini 3.5 Flash,
  GLM-5.3 and MiniMax M3 for the seats prod-manager, eng-junior and sysadmin-junior.
  None of the three is a room subject. No record found assigns MiniMax or Gemini to
  Gem, Delphi, Sprite or Evie.
* **Current single route:** `evaluation/deepseek-v4-pro`, the `ROOM_MODEL` default in
  `housecast/room/cli.py` and the route the M2 preregistration says Kai chose. The
  deploy config that sets the live value is not readable from this seat.

## Setup, frozen

* **Subjects:** the four `frontier`/`compiled` bundles (scientist=Evie,
  frontend-eng=Delphi, game-dev=Sprite, dev-advocate=Gem) from `just compose-bundles`
  at agent-compose `7ee5709`. `bundle-divergence-2026-09-24` used `94e4e11`. The
  room's live `subjects.json` is not readable from this seat.
* **Call shape, copied from `housecast/room/models.py`:** system prompt is the
  bundle, user is the prompt plus `Answer in under 150 words.`, temperature 0.7,
  max_tokens 4000, tool-call markup stripped, 120s deadline, one retry on a
  transient proxy error.
* **Transport:** Agent Proxy at `ser8:8080` only. Header `x-agent-session-id`
  carries `scientist-8487-<config>-<label>`, so SigNoz spans attribute to this run.
* **Prompts:** `../bundle-divergence-2026-09-24/prompts.json`, unchanged. 8 persona
  prompts and 2 factual controls.
* **A round:** one prompt, four subjects called in parallel. Each config runs every
  prompt 3 times, so **30 rounds and 120 answers per config**, rounds one at a time.
* **Divergence:** Jev `jev-1.13.0` stance score per round over the answers that
  came back, on the 0 to 4 level scale, as the room does.
* **Configs:**
  * `current` - all four on `evaluation/deepseek-v4-pro`. Run twice (labels A and B),
    so run-to-run variance is measured and a delta can be told from noise.
  * `mapped` - the four models above. Runs only if every route exists.
  * whole-round single-route candidates, all four subjects on one route:
    `flash` (`evaluation/deepseek-v4-flash`), `gemini` (`chat/gemini-3-5-flash`),
    `minimax` (`chat/minimax-m3`), `glm` (`chat/glm-5-3`), `chatdefault`
    (`chat/default`).
  * `available-mix` - **exploratory, my construction, not the mapped config.** Evie on
    `evaluation/deepseek-v4-flash`, Gem on `chat/gemini-3-5-flash`, Delphi on
    `chat/glm-5-3`, Sprite on `chat/minimax-m3`. It shows what per-persona routing
    does on routes that exist today. It cannot turn a NO-GO on `mapped` into a GO.
* **No Anthropic route is called.** Kai's 2026-09-28 rule.
* **Data:** the bundles are tracked in a public repository and the prompts are
  public-safe. Nothing from the graded layer leaves this seat.
* **Read-only:** no route, config, pod or deploy is changed. A missing route goes to
  sysadmin-senior as a finding.

## Measures, fixed here

* **Route inventory:** the mapped model names against `GET /v1/models`, by substring.
* **Lost subject:** a subject answer that errors after the retry, passes the 120s
  deadline, or is empty after markup stripping. A lost answer is counted and never
  re-asked, so a rerun cannot hide it.
* **Round latency:** from the round starting to the last subject finishing. p50, p95
  and max per label. Per-answer p50 and p95 are reported beside it.
* **Truncation and markup:** `finish_reason` of `length`, and answers whose text
  changed under stripping.
* **Divergence:** mean Jev level over the 24 persona rounds, over the 6 control
  rounds, and the gap. Jev failures are counted, and a round scored lexically is
  reported as such and left out of the mean.
* **Prior finding, `teable:coilyco/agent-compose#8295`:** Evie answers that contain
  the tag `MEASURED` or a fenced `bash` block, counted per config out of 30.
* **Provenance:** SigNoz `litellm.provider.model` on the spans for each session id,
  read-only, to show which backend served each route.

## Decision rules, fixed here

* **GO** for a per-persona config only if all hold. Every route answers through
  Agent Proxy. Zero lost subjects over 30 rounds. Round p95 at most 90s (the M2 P1
  bar). Persona minus control gap at least the `current` gap minus the A to B
  difference in that gap, with both control means at or below 1.0 level.
* **NO-GO** otherwise, and the current route ships unchanged. This is the
  reverse condition prod-director recorded on `teable:coilyco/housecast#8447`, and it
  is deterministic, so it does not go to Jev.
* **Whole-round fallback:** a single-route config qualifies if it has zero lost
  subjects, round p95 at most 90s, and both control means at or below 1.0 level. It
  must be a lab other than the primary's. The pick among qualifiers is a structured
  decision and goes to Jev with the measured rows as state.
* **The verdict wording** goes to Jev as a yes or no on the measured state, and the
  reply reports Jev's probability beside the deterministic rule.

## Disclosure

I wrote this preregistration, the runner, the measures and the thresholds. I did not
build the room or the routes. The science seat wrote `bundle-divergence-2026-09-24`
and its prompt set, which this reuses. The subject bundles share this seat's role doctrine, so
Evie's subject prompt and this measurer come from one hand. The 90s bar and the
gap rule are mine, and the 90s bar is borrowed from a threshold I also wrote for M2.
