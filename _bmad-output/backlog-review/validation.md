# Backlog validation — 2026-11-01

Scope: 10 standalone draft tickets, no epic/container, all `refined: false`.
Store: repo. Product tests and implementation were not run.

Fresh-context validator applied Ticket and Dependencies checks from the rendered
Sofia workflow, reading the complete tickets and their source spec/compliance/verification.

## Findings resolved before presentation

1. Bugs 1/2: disproving a cause is insufficient. No-change closure now requires proof
   that the expected behavior holds, with commands, outcomes and limitations.
2. System acceptance 7/8: added explicit valid/ambiguous import handling, real duplicate
   child count and unexpected child exit/UI cleanup. Unavailable-loopback profiles
   allow these process checks without waiting for remote transport fixtures.
3. Transport acceptance 9/10: Connecting precedes actual successful proxy check;
   PID/listening ports alone do not justify Connected.
4. Spike 6: native Windows availability is Unknown, gating start; the F4 product decision
   remains work performed during the spike.
5. System acceptance sessions require prepared environments/fixtures; missing setup
   must be a separate prerequisite task rather than silently expanding a test session.

A follow-up fresh review confirmed these corrections with no unresolved material gaps.
Results from spikes 3/6 must add necessary fix tickets/dependencies before affected
acceptance starts; closing a spike does not imply its proposed fix is implemented.

## Mechanical results

- `tickets.py status backlog`: total10/backlog10, next_id11, no problems,
  unpinned_after, undeclared_after or order_conflict.
- `tickets.py next backlog`: ready_to_refine1/4/5; ready_to_start empty;7 blocked
  by prerequisites and/or recorded environment unknowns. Approval is still pending.
- Exact CLI outputs: status.json and next.json beside this report.
- Current application HEAD: 910c353955f02b071fb545b0512b7585cca85189.
- Existing dirty generated bindings were preserved; no commit made while the ticket
  set is waiting for approval. `git diff --check` passed.

This record validates the draft's shape and coverage, not product correctness or MVP acceptance.
