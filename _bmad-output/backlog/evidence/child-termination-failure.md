# Child termination failure and shutdown evidence

Date: 2026-11-01. App baseline: `ee4832faa679f632b0bda850fe6c6f288dedcd24`, with the uncommitted lifecycle changes in `internal/core/service.go`, `internal/core/process_windows.go` and their focused tests. No commit was created by this work.

Environment: Microsoft Windows 11 Home, version `10.0.26300`, build `26300`; `go version go1.26.10 windows/amd64`. This is native Windows 11 evidence, not Windows 10 acceptance.

## Outcome and recovery contract

A failed runtime-directory deletion leaves the manager's active session reserved. The worker retains the exact directory and waits without polling. Connect cannot start another child and deletion of that session's profile remains rejected. An explicit Connect while cleanup is blocked requests one cleanup attempt and returns the fixed safe retry-requested error; it never starts a child in that request. Status reports a fixed cleanup error without the path, configuration or OS error. After removing the external block, use the desktop **Connect once** action (or call Disconnect) to request one cleanup attempt; wait for Disconnected, then call Connect again to start a child. The existing Error/stopped desktop action therefore reaches recovery without a new API. Another failed attempt returns to the visible cleanup error. There is no automatic restart or busy retry loop.

Close requests cancellation and waits at most five seconds for confirmed session/storage cleanup, then at most 250 ms for the publisher. A cleanup failure or failed Kill is reported as an error; an unreturned worker also produces an unconfirmed-stop error after the five-second bound. Those failures retain the instance lock and the worker's single Wait. A later Close explicitly retries failed file cleanup. Successful late cleanup releases the lock outside the manager loop. Close prevents all later Connect/import work even if its shutdown attempt returned an error.

On successful session cleanup, the publisher normally drains the final coalesced notifications. A callback that does not return yields the distinct fixed notification-delivery error after 250 ms. Go cannot forcibly stop an arbitrary callback: at most the one already executing callback and its publisher goroutine can remain until the callback returns (potentially for the application's remaining lifetime). No replacement callback goroutines are spawned; after that callback returns the publisher exits without delivering queued notifications. A notification error alone does not mean the child or its files remain alive.

## Synthetic observation

`TestFailedKillRetainsWaitAndOwnershipUntilActualExit` uses a fake child whose Kill returns an error and whose Wait remains blocked until the test explicitly releases it. Close reports unconfirmed stop promptly; status remains Error/stopping, a second working child is rejected, the instance lock stays exclusive, and Wait is called exactly once. Releasing the fake process permits cleanup and a successful later Close. Teardown also releases the fake on assertion failure.

This reproduces the lifecycle hazard deterministically. It does **not** assert that an ordinary native child on this host spontaneously denied termination or hung in Wait.

## Native Windows observation and minimal fix

The old Kill used `TerminateProcess` on the process handle while Wait kept the kill-on-close job handle open until process exit. Therefore denial of direct process termination could leave Wait waiting without using the independently owned job termination capability.

Kill now uses `TerminateJobObject` on the existing owned job handle. It keeps both process and job handles until the existing single Wait confirms exit and releases them. No new dependency, alternate executable path, second Wait or process-rights fallback was introduced. Both preflight and runtime already use the same contained creation path, so the fix applies to cancellation of either process.

`TestWindowsJobTerminationDoesNotRequireProcessTerminateRight` launches a real contained helper process using the existing test child CLI, opens a real process handle with SYNCHRONIZE and QUERY_LIMITED_INFORMATION only, and observes native `TerminateProcess` returning ERROR_ACCESS_DENIED. It substitutes that limited handle as the owned process handle, invokes the real job-based Kill and observes the single Wait return within five seconds with both native handles released. The job and process are real; only the absence of terminate rights is deliberately induced. This is not a fake Kill/Wait result and does not claim a production full-rights handle naturally loses its granted rights.

The existing native `TestWindowsJobKillsChildOnParentExit` also passes for both preflight and runtime. It exits the parent without Wait/Kill/defer cleanup and observes native child exit. The atomic job assignment, non-inherited job handle, executable lock, pinned digest validation and private runtime creation are retained.

`TestWindowsRuntimeCleanupFailureRetainsAdmissionAndAllowsExplicitRetry` holds the actual worker config file with a real Windows handle that omits FILE_SHARE_DELETE. Native RemoveAll fails. Connect and Disconnect recovery followed by a later successful Connect, and failed Close followed by explicit Close recovery, pass. A Connect request while the file is still blocked returns the fixed retry-requested result without a new child; after that single failed attempt, revisions stay unchanged without another request. After releasing the block, Connect requests cleanup only and a later Connect starts the new child. While blocked, no new child/profile deletion is admitted, status remains a safe error, no retry revisions appear without a request, and failed Close retains the store lock. The handle is unconditionally released in teardown. The lifecycle child is fake in this test; the filesystem refusal and store lock are native.

## Checks actually executed

Review follow-up: only `go test -v -count=1 -timeout=120s ./internal/core -run '^TestWindowsRuntimeCleanupFailureRetainsAdmissionAndAllowsExplicitRetry$'` was rerun for the Connect recovery adjustment; all three recovery paths passed. The complete-suite/vet results below precede that adjustment; final full verification is performed separately by the parent task.

- `go test -count=1 -timeout=120s ./internal/core`: PASS natively, latest complete run `ok .../internal/core 4.986s`.
- `go test -v -count=1 -timeout=120s ./internal/core -run 'TestWindowsRuntimeCleanup|TestCloseBoundsBlocked|TestFailedKill|TestWindowsJobTermination|TestWindowsJobKillsChild'`: PASS; native cleanup, process-rights/job and parent-crash cases executed, along with synthetic callback and failed-Kill cases. A later complete run also includes the final callback-drain refinement.
- Normal running stop, canceled preparation, late process start after cancellation, unique session IDs, stale events, bounded logs and health cancellation all remain covered by the complete core suite. Normal/late starts assert exactly one Wait and removal of runtime files.
- `go vet ./internal/core`: PASS, no reported findings.
- `go test -race -count=1 -timeout=120s ./internal/core`: unavailable; default environment reports that -race requires cgo. Retrying with `CGO_ENABLED=1` fails in runtime/cgo because `gcc` is not found in PATH. No race pass is claimed.

## Remaining limits and spike decision

The directly reproduced process-rights dependency is removed using the already owned native job. No further native implementation is required for that demonstrated case. If the job termination API itself fails or a native Wait never confirms exit, the service reports failure, keeps its worker/instance ownership and never admits another child; it does not manufacture successful cleanup. A failed Close does not itself force the host application to remain alive: on actual process exit the existing kill-on-close job remains the containment mechanism, and a residual runtime must pass startup orphan cleanup before a future service owns the store.

This work does not prove Windows 10 behavior, full GUI/WebView2 close behavior, real Xray transport operation, DPAPI/DACL acceptance across users, real-world pathological kernel termination, or a race-instrumented run. Arbitrary blocked injected startup/probe implementations may outlive Close as retained workers; production adapters retain their existing context/time limits. Unrelated generated bindings and other dirty work were preserved.
