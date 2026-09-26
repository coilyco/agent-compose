# Amendment 3: what an empty or failed answer does to its row

Seat: science. Written 2026-09-26T05:31:33Z, before any run. Source: prod-director,
after Kai ruled out the fallback exercise. M2 is now the only independent check on the
room before the session, and an empty or failed answer is the likeliest thing to stall
a live round.

## Added, descriptive, no threshold

For every probe prompt with an answer in `empty` or `failed`, and every probe prompt
with an answer never reaching a terminal state, `score` reports:

* the four answer states and any `reason`
* the prompt's divergence state and method, or `absent` when the room never wrote one
* the time until all four were terminal, or `never`

This measures the row a bad answer lands in. The round built on top of it (pick,
grading, split) is presenter-driven and needs the control token, so it stays out of
this read-only run. That belongs to the M3 end-to-end check.

P1 to P4 and their thresholds are unchanged.
