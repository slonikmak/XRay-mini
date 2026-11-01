# Locator: internal/

Snapshot: 910c353. Search only, no working-tree sources.

- Q1: `internal/core/config.go:29 (PrepareConfig) @ 910c353`, selection/duplicate/dependency checks; `internal/core/store.go:32 (newStore) @ 910c353`, storage/recovery; Windows protection and permissions adapters.
- Q2: `internal/core/service.go:189 (loop) @ 910c353`, commands, worker events and transitions; `internal/core/service.go:343 (run) @ 910c353`, worker cleanup; platform store locks.
- Q3: `internal/core/process.go:37 (startVerifiedContained) @ 910c353`; `internal/core/process_windows.go:102 (startContained) @ 910c353`; `internal/core/runtime.go:28 (validateBinary) @ 910c353`.
- Q4: `internal/core/process.go:106 (probeNetwork) @ 910c353`, outputDrain and port checks; service loop/run.
- Q5: `internal/core/service.go:116 (GetStatus) @ 910c353`, atomic snapshot, callback publisher.
- Q6: seven core test files: config, store, service, process, process_windows, runtime and review_spec_v02.

No Wails/server/frontend integration matched within internal/. Runtime downloading not found there.
