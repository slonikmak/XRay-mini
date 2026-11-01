# Engineering tour

This is a guide to the implemented client, not a claim of completed production acceptance. Start with a normal Connect/Disconnect, then follow one failure scenario.

## 1. UI and desktop boundary

[frontend/src/App.tsx](../frontend/src/App.tsx) owns presentation state, dialogs, subscription previews and separate locks for connection commands and profile mutations. The same Stop/Retry stop action is available inside dialogs. [frontend/src/bridge.ts](../frontend/src/bridge.ts) adapts generated Wails bindings; [app.go](../app.go) is the Go facade.

Read the initial listen-before-snapshot sequence and revision guards. Explain why an older response must not overwrite a newer state event, and why command acceptance is not connection success. Generated files under `frontend/bindings/` are outputs, not the place to edit the API.

## 2. Single-writer connection manager

[internal/core/service.go](../internal/core/service.go): `Service.loop` selects between commands and session-tagged worker events. It alone changes active session state, revision, health-failure counts and the bounded diagnostic history. `GetStatus` reads an atomic snapshot. Repository I/O uses a separate lock and does not execute inside the loop.

A Connect reserves a unique session before background preparation starts. Another Connect is rejected until StopCompleted releases ownership. Look at reserve-delete/release-delete for the race between deleting a profile and connecting to it.

**Discuss:** why use a single writer here instead of several goroutines mutating status under a shared mutex? Which shared resources still legitimately need a mutex?

## 3. Session supervisor and termination

In the same file, `run` owns configuration preparation, preflight, the working child, its single waiter, health-check lifetime and cleanup. Session context cancels work, but cancellation alone is not evidence that the process has exited.

`ProcessExited` records a terminal fact. Late ConfigValidated, ProcessStarted, ProxyReady or HealthFailed events cannot resurrect that session. `StopCompleted` is later: process/probe/output/file cleanup is confirmed. A failed Kill retains ownership and waits for actual exit or one bounded explicit retry signal.

**Regression story:** `ProcessExited -> ProxyReady` used to produce Connected despite the dead child. Session ID checks were insufficient because both events belonged to the same session. The fix is an irreversible lifecycle transition; the race detector alone cannot prove this property.

## 4. Windows process adapter

[process_windows.go](../internal/core/process_windows.go) implements contained process creation using a Job Object and explicit inherited handles. [process.go](../internal/core/process.go) supplies the runner abstraction, output draining and network probe; [runtime.go](../internal/core/runtime.go) contains the runtime pin and verification rules.

Preflight and working processes use the same verified/contained creation path. Explain ownership of the job, process and pipe handles, and why output readers must finish without filling an unbounded log buffer. The deferred preflight-Kill recovery case is documented in [deferred work](../_bmad-output/deferred-work.md); do not imply working-child retry also covers that case.

## 5. Health is separate from process existence

`probeNetworkVia` in [process.go](../internal/core/process.go) uses an explicit proxy, a shared deadline and a real HTTPS request. Listener readiness consumes the same budget and does not itself mean success. Probes are serial; two failures cause Degraded, success resets the count. One control destination cannot establish reachability of every destination or prove a system-wide tunnel.

## 6. Configuration, profiles and import

[config.go](../internal/core/config.go) builds a new selected-only runtime config rather than retaining arbitrary source routing. [store.go](../internal/core/store.go), platform protection/permission adapters and the OS storage lock manage profiles and runtime residue.

[subscription.go](../internal/core/subscription.go) parses the supported VLESS subset and bounds explicit HTTPS subscription requests. In [app.go](../app.go), generation checks and cancellation prevent replaced previews from becoming current. Preview metadata crosses the UI bridge; raw candidate configuration remains in Go. Persistent save happens outside the preview mutex.

## 7. Tests worth opening during an interview

[service_test.go](../internal/core/service_test.go): `TestExitedSessionRejectsLateEventsUntilCleanup`, `TestUnexpectedExitFactSurvivesStopCommandBeforeEvent`, `TestRepeatedStopBeforeFailureNotificationRetriesOnce`, `TestExplicitStopRetriesFailedKillWithoutAnotherWait`, cancellation, stale sessions and slow-subscriber tests.

[process_windows_test.go](../internal/core/process_windows_test.go): native containment/cleanup cases. [native_transport_windows_test.go](../internal/core/native_transport_windows_test.go): opt-in real-core local fixtures, not external-server acceptance. [frontend/src/App.test.tsx](../frontend/src/App.test.tsx): held profile operations with reachable Stop and revision handling.

Tests using fakes establish control-flow properties. Native loopback establishes a different layer. Neither alone establishes the entire Windows 11 GUI or every remote deployment.

## A compact explanation

> The Go backend is a supervisor, not the networking engine. One event loop owns connection state; per-session workers own I/O and resources. Commands express intent, events establish facts. UI notifications may be coalesced, lifecycle events may not. A session is reusable only after confirmed cleanup, and an exited process cannot become healthy again because of a late result.

Explain the actual code and its tradeoffs. Do not substitute pattern names or AI-tool names for understanding.
