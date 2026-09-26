# PyLadies Remote room, M2 check

The latency, concurrency, and divergence check on the room's dev deploy, at
milestone M2 of the plan on `teable:coilyco-bridge/inbox#4019`. The build is
`teable:coilyco-flight-deck/housecast#8168`.

* [`PREREGISTER.md`](PREREGISTER.md) - the claim, setup, measures, and pass
  thresholds, fixed before the room existed.
* [`PREDICTION.txt`](PREDICTION.txt) - the numbers expected, timestamped with it.
* [`AMENDMENT-1.md`](AMENDMENT-1.md) - access and measures fitted to the M1 room contract, before any run.
* [`AMENDMENT-2.md`](AMENDMENT-2.md) - which provider served each answer, and how SigNoz shows it.
* [`AMENDMENT-3.md`](AMENDMENT-3.md) - what an empty or failed answer does to its row.

* [`room_m2.py`](room_m2.py) - the runner and scorer. `just evalkit-room-m2 selftest`
  runs it against an in-process fake room built from the M1 contract.

`results/` lands with the M2 run, against the dev deploy. `just evalkit-room-m2 run
<base url> <out>` runs it, then `score <out> [export]`.
