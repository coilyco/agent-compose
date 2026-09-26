# Aborted M2 start, 2026-09-26 05:42Z

Started 05:42:40Z against http://kai-server:30123 (dev on housecast `7966eb6`),
with the runner at `dd391be`. I stopped it at about 05:43:00Z on the room owner's
"hold M2", so M2 measures the version that ships.

* It submitted 2 of 42 prompts: `57066415` (deck-purple) at 05:42:47.480Z and
  `212213be` (deck-refuse) at 05:42:52.624Z. `submits.jsonl` holds only the first,
  because the runner writes a row after its prompt settles.
* deck-purple settled with all four answers `done`, finishing between 05:42:52.151Z
  and 05:42:52.608Z. That's 4.7 to 5.1s after submit. The room read at 05:43:07Z shows it.

This is n=1 on a build that is being replaced. It's not scored and not a P1 to P4 result.
Both prompts stay in the dev log under their `[m2-probe]` text. The scorer counts only
prompts in its own run's `submits.jsonl`, so the rerun does not count them.
