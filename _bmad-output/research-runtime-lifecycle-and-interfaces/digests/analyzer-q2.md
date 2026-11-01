# Q2 — sessions and cleanup

12 files read at 910c353; no execution/history.

- **Load-bearing:** unbuffered command channel, single loop state writer and atomic status snapshot. `internal/core/service.go:97-116 (request) @ 910c353`; `internal/core/service.go:187-198 (loop) @ 910c353`.
- **Load-bearing:** active session rejects duplicate Connect; Disconnect cancels without releasing reservation. Stale IDs are ignored; ProcessExited alone never releases ownership. Only matching StopCompleted clears active. `internal/core/service.go:234-261 (loop) @ 910c353`; `internal/core/service.go:286-318 (loop) @ 910c353`.
- **Load-bearing:** late Start still registers one waiter, cancellation kills/waits; deferred cleanup joins health/diagnostics and removes runtime before StopCompleted. RemoveAll failure produces fixed cleanup-error message but completion still releases active. `internal/core/service.go:343-354 (run) @ 910c353`; `internal/core/service.go:400-460 (run) @ 910c353`.
- Close waits manager, closes store under repository lock, then waits publisher. Blocking publisher callback can hold Close indefinitely; no shutdown timeout and store close error is discarded. `internal/core/service.go:174-184 (Close) @ 910c353`.
- Store ownership lock precedes stale directory removal; failure aborts startup. `internal/core/store.go:32-62 (newStore) @ 910c353`.
- Deletion reserves profile ID in manager before separate store I/O; active/deleting/invalid IDs fail. `internal/core/service.go:153-167 (DeleteProfile) @ 910c353`; `internal/core/service.go:263-274 (loop) @ 910c353`.
- Completion retains sessionID; Error also retains profileID. `internal/core/service.go:290-302 (loop) @ 910c353`.
- Tests define late-start cancellation, once-only Wait, stale-event rejection, serial health, slow subscriber and blocked-store responsiveness; no execution inferred. `internal/core/service_test.go:201-388 (TestRapidReconnectAndRevision) @ 910c353`.

No cleanup retry or shutdown timeout found by scoped service/store grep. Kill/Wait errors are not forwarded raw to UI; no restart loop.
