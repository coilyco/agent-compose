# Can this board be attested after the fact

Asked by the director seat against the wedge-1 plan: whether binding a graded
run to the digest of the bundle it graded works backwards onto runs already
taken, or only forwards onto runs not yet run. That turns on one fact about this
board's artifact. Does its Inspect log carry the composed system prompt text.

**It does, in three places, and one of them is never condensed. Attestation
reaches backwards.**

Read 2026-09-03 against housecast `90c22d0` and the `inspect_ai` 0.3.260 wheel,
the version `uv.lock` pins under the looser `inspect-ai>=0.3.255` spec.

## Where the text is

* **`sample.metadata["system_prompt"]`.** `evalkit/task.py` assigns the whole
  composed bundle per sample, resolved from that sample's own `entity`.
* **`sample.messages[0]`.** The `system_message("{system_prompt}")` solver
  prepends it to the conversation.
* **The event stream**, again, inside the model event inputs.

`EvalSample` in `inspect_ai/log/_log.py` declares `messages` and `metadata` as
model fields, so both serialise into the `.eval`.

## Read metadata. Not the other two

`condense_sample` in `inspect_ai/log/_condense.py` rebuilds exactly `input`,
`messages`, `events`, `error_retries`, `attachments` and `events_data`.
**`metadata` is absent from that set**, so it is never condensed and holds the
text inline unconditionally. That is the carrier to attest against.

The other two are conditioned on a size rule that every composed bundle trips:

* `events_attachment_fn` replaces any text over 100 characters with an
  `attachment://<mm3_hash>` reference, so the event stream holds a pointer.
* `messages_attachment_fn` returns non-data-URI text unchanged, so
  `sample.messages` is inline **today**. It is the weaker guarantee of the two,
  because that function exists to handle images and could grow a length rule
  without anyone thinking about attestation.

`mm3_hash` is MurmurHash3. It is a de-duplication key and is not an attestation
digest.

## The binding is per role, not per bundle

One system prompt resolves per sample from its `entity`, so this board's 105
cases at 5 epochs carry seven distinct system prompts across 525 samples.
Attestation is therefore seven digests, or one digest over the sorted set of
role prompts. A single whole-bundle digest does not describe what was run.

## What this record does not claim

I did not open this board's `.eval`. `.gitignore` carries `.evalkit/` as the
"Eval run cache: composed prompts, Inspect logs, datasets, annotations", so the
log exists only on the host that ran it. This is a measurement of the code path
that produced the run, not of the artifact.

Two facts narrow that gap. `evalkit/task.py` last changed at `29c451b` on
2026-08-27 and this board ran on 2026-09-01, so the code above is the code that
ran it. And `dataset.yaml` records the bundle as `composed at a691f40`, which is
a ref in a comment rather than a machine-checkable digest, which is the gap
wedge-1 exists to close.

The residual risk is the `>=0.3.255` spec resolving to something other than the
locked 0.3.260 in the runner's environment.

## Where the log is not

Searched on 2026-09-03, so the next reader skips these:

* **kais-macbook-pro.** No housecast checkout under `~/projects` at all, and no
  `.evalkit/` directory or `*.eval` file anywhere under `~`.
* **kai-server.** `~/projects/coilyco-flight-deck/` exists and is populated,
  `agentic-os` among it, and housecast is not checked out there.
* **The host behind `http://ser8:8080/v1`**, which the run command in
  `evalkit/task.py` names for Agent Proxy, is unchecked rather than clear. Its
  node-stats readable root excludes home directories, so nothing there was
  ruled out. It is the likeliest place left to look.

Anyone holding the log closes this in one line:

    python3 -c "from inspect_ai.log import read_eval_log; \
      s = read_eval_log('<path>.eval').samples[0]; \
      print(len(s.metadata['system_prompt']))"

Recorded by the science seat at the director seat's request.
