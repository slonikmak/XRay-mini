---
title: 'Session cleanup and bounded shutdown'
type: bugfix
ticket: 1
created: '2026-11-01'
status: built
baseline_revision: 'ee4832faa679f632b0bda850fe6c6f288dedcd24'
route: full
route_source: auto
risk: high
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="User approved backlog and autonomous execution on 2026-11-01">

## Intent

Fix the session lifecycle findings in approved tickets 1 and 2 and answer spike 3.
An undeleted runtime configuration must prevent another working child until cleanup
is confirmed. A blocked status callback must not indefinitely hold application close.
Determine the failed-Kill/unreturned-Wait behavior and implement the minimal necessary
safe native fix as authorized follow-up; document synthetic versus native evidence.

## Boundaries & Constraints

Always keep the manager as single state writer, session-ID filtering, exactly one
Wait, private files, exclusive store ownership and late-start cleanup. No disk I/O
in the manager loop. Keep bounded/coalescing publication and separate process/health.
Keep ownership until cleanup succeeds; recovery must be explicit, finite per request
and usable with the existing Connect/Disconnect API. Report fixed safe errors.
Never solve shutdown by announcing success while Xray still lives or dropping Wait.
Use stdlib/native mechanisms, avoid new dependencies or a generic task framework.

## I/O & Edge-Case Matrix

| Scenario | Input/state | Expected behavior | Error handling |
|---|---|---|---|
| Cleanup failure | Windows denies deletion of owned runtime | Error remains visible, new child not started | Safe cleanup category, no raw path/secret |
| Recovery | Deletion block removed, documented user action | Cleanup confirmed, later Connect works | No busy retry/restart loop |
| Normal stop/cancel | Running or preparing session canceled | One Wait, no runtime/process, stale events ignored | Existing safe errors retained |
| Blocked callback | Callback never returns, child cooperative | Close ends within documented finite bound | Explicit result distinguishes undelivered notification from process failure |
| Child termination failure | Kill error, Wait not completed | No false Disconnected, no second Wait; safe ownership strategy | Record fake versus actual Windows behavior |

</frozen-after-approval>

## Code Map

- `internal/core/service.go`: Service.loop owns active; StopCompleted currently clears it even when run's RemoveAll failed. run owns child, waiter, health and temp directory.
- `internal/core/service.go`: publishLoop calls callback synchronously; Close waits publisherDone after session/store cleanup. All callback calls must stay outside manager.
- `internal/core/process_windows.go`: Kill uses TerminateProcess; Wait closes kill-on-close job only after INFINITE wait. Native job ownership can provide a stop path independent of direct-process termination rights.
- `internal/core/service_test.go`: fixture/fakeRunner/fakeChild, late-start, old-event, slow-callback and once-only-Wait tests; reuse helpers.
- `internal/core/process_windows_test.go`: native job crash/executable lock tests; add real file-share deletion refusal and native containment regression as needed.

## Tasks & Acceptance

Execution ownership: worker owns `internal/core/service.go`, `internal/core/process_windows.go`,
their focused core test files and `_bmad-output/backlog/evidence/child-termination-failure.md`.
The worker is not alone: preserve all other edits, do not commit, and accommodate
other agents. Do not change frontend, packaging, integration harness, or ticket files.

- [x] Implement minimal failed-cleanup admission/recovery and safe Close result.
- [x] Bound notification shutdown; document any unavoidable one-callback lifetime limit.
- [x] Reproduce synthetic termination failure; use native job cleanup to remove the
  direct-process rights dependency if necessary, with a meaningful native check.
- [x] Cover every matrix row; temporary blocks and fake processes must be releasable
  in test teardown even after an assertion fails.
- [x] Write spike evidence: observed checks, fake/native distinction, outcome and risks.

Acceptance: approved tickets 1/2 hold, regression tests run on this Windows 11 host;
spike3 records whether further work is needed. Do not claim Windows10 acceptance.

## Implementation Notes

Authorization: user explicitly requested autonomous completion; plan/dirty-tree
checkpoints are covered by that instruction. Pre-existing generated binding changes
are preserved and outside worker ownership. Tasks 1→2→3 intentionally share one code
lane; they may be implemented together to keep the lifecycle coherent.

## Plan Change Log

## Review Triage Log

- medium / patch: CleanupFailed uses Error/stopped, so the existing UI calls Connect
  rather than the advertised Disconnect retry. Verified against frontend active guard.
  Existing API Connect must request one cleanup retry without admitting a child;
  preserve retained ownership and once-only Wait. Sent to the same implementer.

## Verification

- `go test -count=1 -timeout=120s ./internal/core` — all core cases pass natively.
- Focused native cleanup/callback/job tests must execute, not merely compile.
- `go vet ./internal/core` — no reported findings.
- Race instrumentation may require a Windows C toolchain; report its availability
  honestly, never substitute a non-race pass for race evidence.

## Completion, 2026-11-01

Review recovery finding patched: Connect requests exactly one retained-session cleanup retry without launching a child. Final native production-tag suite and vet passed; native cleanup denial, bounded callback and limited-process-rights tests execute. See evidence/child-termination-failure.md and autonomous-execution-2026-11-01.md.
