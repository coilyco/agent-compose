"""Count out-half runs that name the receiving seat, split by allocation.

Deferring and scoped are different states and the board derives them
separately, so counting them together hides which one produces a handover.
"""

import re
import sys
from collections import defaultdict

from inspect_ai.log import read_eval_log

from housecast import roster as R

BOUNDARY = "seek-external-validation"
roster = R.load()
owner = roster.boundaries[BOUNDARY].owner
NAMES = re.compile(
    rf"{roster.roles[owner].display_name}|\b{owner}\b|\bdirector\b", re.I
)
deferring = [n for n in roster.role_order if BOUNDARY in roster.roles[n].defers]
scoped = [n for n in roster.role_order if any(e.name == BOUNDARY for e in roster.roles[n].scoped)]

log = read_eval_log(sys.argv[1])
per_case = defaultdict(list)
for sample in log.samples or []:
    per_case[str(sample.metadata.get("challenge_id") or sample.id)].append(
        sample.output.completion or ""
    )

for label, seats in (("deferring", deferring), ("scoped", scoped), ("owner", [owner])):
    named = total = 0
    print(f"--- {label} ---")
    for entity in seats:
        runs = per_case.get(f"{entity}-sev-out", [])
        hits = sum(1 for t in runs if NAMES.search(t))
        named += hits
        total += len(runs)
        print(f"  {entity}-sev-out  {hits} of {len(runs)}")
    print(f"  = {named} of {total}\n")
