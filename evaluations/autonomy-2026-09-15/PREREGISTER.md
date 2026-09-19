# Pre-registration: first run of the autonomy dimension
Seat: Evie (science) xq49. housecast at the autonomy-grader branch.

## Why this run exists
Two purposes, and they are separable.
  P1  Non-confounded responses to validate the grader on. The transcript-based
      set failed: 7/111 measured the set, not the grader, because the prose
      preceding an AskUserQuestion carries findings and not the deferral.
  P2  The dimension's first reading.

## Setup, frozen before the run
14 cases (7 seats x in/out), 3 epochs = 42 responses.
Model evaluation/deepseek-v4-pro through Agent Proxy. 50-word cap.
NOT Kai's Claude seats. This measures the composed bundle against that model,
so it does not transfer to the seats she actually runs without a second arm.

## Claims, stated before the tally
C4  in-half pass rate below 50%. The census says seats over-ask, and the
    in-half is the half that over-asking fails.
C5  out-half pass rate above 70%. Deferring is the trained behaviour, so the
    half that rewards deferring should pass easily.
C6  grader UNCLEAR rate on these 42 below 30%. If it is higher, the grader
    does not work on board responses either and does not ship as a grader.

## Negative control
C5 is the control for C4. If both halves pass high, the board is not
discriminating and the prompts are the suspect, not the seats.
If the grader labels in-half and out-half identically, it is reading the
prompt's framing rather than the response.
Tue Sep 15 06:44:37 UTC 2026
