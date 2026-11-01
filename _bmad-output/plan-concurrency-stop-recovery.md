---
title: 'Reliable session termination and explicit recovery'
type: bugfix
ticket: ''
created: '2026-11-01'
status: built
baseline_revision: 'eed355017638e1834a957808438148cff1a011eb'
route: full
route_source: auto
risk: high
review: thorough
review_source: pinned
lenses_ran: [blind-hunter, edge-case-hunter, verification-gap, intent-alignment]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="User requested this complete BMAD repair cycle here; prior autonomous execution approval covers routine plan and dirty-input gates">

## Intent

Reliable termination must survive out-of-order events, a failed Kill and an outstanding profile save. Resolve R3-01/02/03 from `xray-mini-review-concurrency-eed3550.md`, restore subscription after late bridge discovery, and reconcile the documented Error/Close contract. Preserve the report as the original review input. Work remains loose, with no active initiative, as previously agreed.

## Boundaries & Constraints

Always keep Service.loop the sole session-state writer, one Wait per child, session-tagged events, cancellation, bounded retries and session/storage ownership until confirmed StopCompleted. Process exit and cleanup completion are separate facts. Preserve a safe unexpected-exit reason through cleanup and explicit stop commands. One additional Kill attempt is permitted per explicit recovery request after failure; no automatic retry/restart loop. No secrets or raw OS errors in status/evidence.

Preserve current profile/import behavior and independent manager responsiveness. Provide an accessible Disconnect/Retry stop inside modal dialogs while profile I/O remains pending. Retain command duplicate protection and older-revision rejection. Windows 11 with installed WebView2 remains scope; no new dependencies or public backend API. Close retains bounded waits and honest ownership on error; arbitrary callbacks cannot be forcibly joined.

## I/O & Edge-Case Matrix

| Scenario | Input/state | Expected behavior | Error handling |
|---|---|---|---|
| Exit ordering | ProcessExited then Started/Ready/Failed of same session | Process stays stopped, no healthy transition, ownership retained | Unexpected-exit reason survives final cleanup |
| Cancellation/stale | Disconnect then late success; old events after new session | No resurrection or new-session mutation | Cleanup still joins one Wait |
| Stop failure | Kill fails, then explicit Disconnect/Close | Error permits retry; next request retries once and successful exit cleans | New Connect and competing store remain blocked meanwhile |
| Repeated failure | Multiple failed explicit attempts | Bounded attempts, no spin or false stopped/release | Safe Error remains recoverable |
| Pending profile save | Rename/import held with modal open while session active | Modal Stop reaches backend before save resolves | Profile lock cannot suppress stop |
| Retained stopped session | Error/stopped with sessionID during cleanup | UI offers cleanup recovery, never treats it as released | Connect remains backend-gated |
| Late bridge | Host absent initially, appears on existing recovery event | Subscribe once before snapshot; later events update without another focus | Unmount unsubscribes and stale events ignored |

</frozen-after-approval>

## Code Map

- `internal/core/service.go`: session, loop, run; reuse existing events, retryCleanup and Close error reporting. Add manager-owned terminal/stop-failure state and explicit supervisor retry signal. Never cancel away the unexpected-exit cause.
- `internal/core/service_test.go`: fixture/fakeRunner/fakeChild, one-Wait, ownership, stale-event and late-start tests; add focused deterministic recovery tests, correct waits that conflate process exit with completed cleanup.
- `frontend/src/App.tsx`, `frontend/src/App.test.tsx`: perform/lock, toggleConnection, Modal, subscription effect; separate connection command gating, modal stop action, sessionID ownership and late subscription recovery. Preserve subscription preview cancellation and revision guards.
- `docs/xray-mini-spec-v0.2.ru.md`, `docs/ARCHITECTURE.md`, `docs/VERIFICATION.md`: reconcile terminal/recovery/Close semantics; link exact new verification evidence. Root spec remains original input.

## Tasks & Acceptance

- [x] Implementation agent owns service.go and lifecycle tests: reproduce failures first, fix terminal ordering and explicit Kill retries, add matrix regressions and targeted repeated stress.
- [x] The same agent then owns App.tsx and App.test.tsx sequentially: separate command lock, reachable modal stop/retry, bridge re-subscription and held-Promise tests. No docs/build/commit edits. It is not alone; preserve others' work.
- [x] Parent owns docs, review-input preservation, evidence, full checks, production package and local commit. Do not revert others' changes or push.

Given any late event after exit/cancel, when processed, then it cannot publish a running/healthy session before cleanup. Given a failed Kill, when another explicit stop is requested, then it can terminate the owned child without a second Wait. Given a pending save, when Stop is clicked inside its dialog, then backend Disconnect runs before save resolves. Given a discovered late bridge, when a subsequent status event arrives, then UI updates without another recovery event.

## Implementation Notes

Manager retains terminal exit cause and cancels from ProcessExited; worker no longer cancels before the exit event is observed. StopCompleted clears sessionID; profileID remains failure context for repair. Test waits distinguish process exit from completed cleanup. Parent tightened the real-core exit fixture accordingly and updated README/spec/architecture. Baseline regressions failed as expected; native full Go/vet, frontend typecheck/37 tests, targeted lifecycle matrix x20, failed-Kill test x100 and real-core three-transport loopback (12.22s) passed. Runtime/GUI acceptance limits remain separate.

## Plan Change Log

## Review Triage Log

### Pass 1 — 2026-11-01

Verdicts before grouping: high 0, medium 7, low 5, false 2. All four lenses ran; capacity allowed three concurrent reviewers and the fourth as the first slot freed. No intent or architectural loopback required; surviving changes are direct corrections/tests with no public surface.

| Finding | Verdict | Route and evidence |
|---|---|---|
| Blind 1: exit event queued before stop loses cause | medium | patch: loop classifies using current canceled context; capture safe exit cause in waiter before publishing event. Test the captured-before-command ordering. |
| Blind 2: failed preflight termination has no explicit retry | medium | defer: pre-existing execRunner.Test owns a synchronous Wait; native cancellation ignores Kill error. This repair targets the reported working-child supervisor; record unresolved preflight recovery separately and scope the new retry documentation to the working child. |
| Blind 3: native context watcher violates explicit retry count | false | reject: unchanged watcher makes its initial cancellation attempt only once. Each later recovery request adds at most one supervisor attempt; fake tests establish that supervisor property, not a total native initial-Kill count. |
| Blind 4: modal hides recovery explanation | medium | patch: native modal makes background status inaccessible while save is pending; render safe status with modal recovery action. |
| Blind 5: concurrent operations share feedback | medium | patch: perform clears shared error/notice on both command classes; separate connection error feedback and preserve profile notices. |
| Blind 6: cleanup message says Connect | low | patch: align visible instructions with Retry stop/Disconnect/Close; Connect API cleanup recovery remains supported. |
| Blind 7: synthetic exit does not exercise waiter capture | low | patch, grouped with Blind 1: add captured-exit/command regression alongside manager matrix; native fixture already exercises real Wait/cleanup. |
| Blind 8: unexpected-exit CleanupFailed preservation untested | medium | patch: add real Windows deny-delete cleanup failure/retry case, preserving exit cause, session and single Wait. |
| Blind 9: held delete/subscription saves untested | low | patch: extend held-Promise modal action tests to those existing mutations. |
| Blind 10: verification record not published yet | low | patch completion docs/evidence: parent has actual full/native logs and will attach exact source/artifact identities after final build. No acceptance completion claim is made. |
| Edge 1: recovery command overtakes queued StopFailed | medium | patch: stopBlocked is set only on event consumption; preserve an explicit repeated-stop signal before that event, bounded/coalesced without auto polling. |
| Verification 1: StopFailed after ProcessExited missing | medium | patch: guard exists but matrix omits same-session StopFailed; add it. |
| Verification 2: ordinary stopped-but-owned UI phase missing | low | patch: cover disabled Finishing cleanup followed by released-session Connect. |
| Intent: integrated native failure/save reading differs from fixtures | false | reject as defect: targeted repair and contract reconciliation align. Record synthetic-vs-native coverage and pending integrated GUI/DPAPI/Kill-failure acceptance honestly; do not claim full MVP acceptance. |

## Verification

First run new regressions against the baseline and retain concise failing evidence. Then focused native Go and frontend tests; full `go test -tags production -count=1 -timeout=180s ./...`, `go vet -tags production ./...`, frontend typecheck/test, targeted repeated lifecycle stress. Run native runtime checks supported by existing harness; distinguish fixtures from real Windows process evidence. Thorough independent review follows; fix material findings. Finally `wails3 build` (no concurrent npm) and `python scripts/package_runtime.py --verify bin`. Capture full baseline revision, source/artifact hashes, commands/outcomes and remaining acceptance limits.

## Completion - 2026-11-01

All current-change review patches verified. Capture of unexpected exit and bounded early retry are covered; StopFailed terminal matrix, modal safe feedback, independent operation errors/notices, held delete/subscription saves and ordinary cleanup UI tests pass. Parent Windows deny-delete unexpected-exit/retry regression passes. Final full Go/vet and frontend typecheck/42 tests, real pinned-core three-transport loopback (11.53s), production Wails build and offline package verification passed. One pre-existing preflight-child retry gap is deferred. [Evidence and matrix coverage](backlog/evidence/concurrency-recovery-2026-11-01.md).
