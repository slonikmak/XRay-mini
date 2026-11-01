---
title: 'Xray Mini: runtime lifecycle and interfaces'
type: codebase
repository: 'C:/Users/kp/Documents/projects/XRay-mini'
commit: '910c353955f02b071fb545b0512b7585cca85189'
branch: main
status: complete
validation: normal
claims: {"verified": 22}
created: '2026-10-31'
updated: '2026-10-31'
---

# Xray Mini: runtime lifecycle and interfaces

This report records source behavior at one immutable commit. It is evidence for a
developer planning changes and a reviewer checking a claim. It does not report a new
test run. Locators, analyzers and verifiers used separate contexts; analyzers received
factual questions without the specification or desired changes.

## Open questions

- How does the packaged application behave on Windows 10 and Windows 11 under
  actual crash, DPAPI/DACL, WebView2 and network scenarios? Static research and the
  saved Linux fixture result do not answer this; it requires a separate execution task.
- What app revision produced the saved Linux integration result? The result has no
  app source commit; this research establishes only its presence in 910c353.
- What happens under actual denied runtime deletion, an indefinitely blocked callback,
  failed child termination and interrupted notice copying? Source paths establish
  conditional behavior, not observed failure frequency.

## Questions

1. How are profiles imported, stored and transformed into runtime configuration?
2. How are connection commands, session events and cleanup handled?
3. How are child processes owned, Windows lifecycle handled and runtime binaries delivered?
4. How do health checks and diagnostic logs work?
5. How does UI/API synchronization, profile management and diagnostics display work?
6. How is this behavior tested and what result artifacts exist?

The ledger tracks 22 independently verified key claims; the remaining source citations were mechanically anchored. Symbol lookup beyond an 80-line window was independently checked where flagged.

Budgets: 15 files per question, at most 10 history entries if needed; validation normal.
No historical why-question was asked, so no historian or proposed historical ADR is needed.

## 1. Profiles and runtime configuration

Import requires one selected VLESS outbound with streamSettings, rejects ambiguous
selection, and drops other top-level sections when rebuilding the configuration.
The UI obtains a preview and requires acknowledgment when anything is dropped.
`internal/core/config.go:45-146 (selectOutbound) @ 910c353`;
`frontend/src/App.tsx:90-105 (importProfile) @ 910c353`.

PrepareConfig constructs two loopback inbounds on ports 10808/10809, one selected
VLESS outbound and explicit routing from both inbound tags to it.
`internal/core/config.go:29-42 (PrepareConfig) @ 910c353`.

Windows profile records are encrypted with current-user DPAPI. The service decrypts
the profile, rebuilds the configuration and writes a separate session runtime file.
`internal/core/protect_windows.go:22-43 (crypt) @ 910c353`;
`internal/core/store.go:123-146 (add) @ 910c353`;
`internal/core/service.go:355-398 (run) @ 910c353`.

Directory ACLs allow the current user and SYSTEM; the exclusive store lock precedes
orphan runtime cleanup. These are source mechanisms, not an observed Windows ACL test.
`internal/core/permissions_windows.go:15-48 (secureDirectory) @ 910c353`;
`internal/core/store.go:32-62 (newStore) @ 910c353`.

Input checks reject oversized/deep JSON, duplicate keys, unsupported selected fields,
external dependencies and local resource references. Protocol validity is deferred
to Xray preflight. The inspected UI imports pasted JSON; no file-reader or subscription
parser was found in scoped app/core/frontend searches.
`internal/core/config.go:50-123 (selectOutbound) @ 910c353`;
`internal/core/config.go:152-180 (unsafeDependency) @ 910c353`;
`internal/core/config.go:185-240 (rejectDuplicateKeys) @ 910c353`;
`frontend/src/App.tsx:90-105 (importProfile) @ 910c353`.

## 2. Session ownership and cleanup

Service.loop owns connection state; commands use channels, GetStatus reads an atomic
snapshot, and disk/process/network work runs outside the owner loop.
`internal/core/service.go:97-116 (request) @ 910c353`;
`internal/core/service.go:187-228 (loop) @ 910c353`.

Connect rejects an active or cleaning session. Matching StopCompleted releases it;
ProcessExited alone does not. Events from old session IDs are ignored.
`internal/core/service.go:234-261 (loop) @ 910c353`;
`internal/core/service.go:286-318 (loop) @ 910c353`.

Cancellation kills the child and waits for the registered waiter; deferred cleanup
joins health and diagnostics, then attempts runtime-directory removal before emitting
StopCompleted. Removal failure changes the completion message to a cleanup error,
but StopCompleted still releases active and permits a later Connect.
`internal/core/service.go:343-354 (run) @ 910c353`;
`internal/core/service.go:400-460 (run) @ 910c353`;
`internal/core/service.go:290-302 (loop) @ 910c353`.

Close waits for manager completion, store close and publisher completion. There is no
deadline around the publisher wait; a callback that never returns can hold Close.
`internal/core/service.go:86-95 (publishLoop) @ 910c353`;
`internal/core/service.go:174-184 (Close) @ 910c353`.

Deletion reserves an ID through the manager before separate store I/O, preventing
connect/delete overlap for that profile. Invalid/deleting/active IDs are rejected.
`internal/core/service.go:153-167 (DeleteProfile) @ 910c353`;
`internal/core/service.go:263-274 (loop) @ 910c353`.

## 3. Child processes and delivery

Every production invocation locks and validates the adjacent pinned Windows amd64
runtime before creating the child. Hash mismatch and arbitrary paths are rejected.
`internal/core/process.go:35-46 (startVerifiedContained) @ 910c353`;
`internal/core/runtime.go:28-58 (validateBinary) @ 910c353`;
`internal/core/process_windows.go:226-235 (lockExecutable) @ 910c353`.

Windows start passes a kill-on-close Job Object through JOB_LIST to CreateProcessW;
the job handle is not inherited, and only explicit standard-stream handles are inherited.
Both preflight and working execution use this path.
`internal/core/process_windows.go:99-208 (startContained) @ 910c353`;
`internal/core/process.go:24-46 (Test) @ 910c353`.

windowsChild.Wait is guarded by sync.Once, waits for exit, closes process/job handles
and joins output drains. Its OS wait is infinite; cancellation callers ignore Kill errors.
`internal/core/process_windows.go:56-95 (Wait) @ 910c353`;
`internal/core/service.go:458-461 (run) @ 910c353`.

The package task builds the Windows GUI and installs the separately downloaded,
pinned Xray 26.10.29 executable and notices after archive/executable hash checks.
The packaging script does not execute the downloaded binary. Notice copying follows executable
replacement, so later copy failure can leave incomplete ancillary package contents.
`Taskfile.yml:8-46 (bindings) @ 910c353`;
`scripts/package_runtime.py:19-36 (main) @ 910c353`.

## 4. Health and diagnostics

The probe sends HTTPS GET through the local HTTP proxy to generate_204, accepts only
HTTP 204, rejects redirects and runs with a five-second deadline. One serial worker
checks immediately and then on a 30-second ticker; shutdown cancels and joins it.
`internal/core/process.go:106-120 (probeNetwork) @ 910c353`;
`internal/core/service.go:428-455 (run) @ 910c353`.

The first failed check leaves the existing Connecting or Connected state but sets
health unhealthy; the second consecutive failure gives Degraded. Success resets the
failure count and gives Connected. Process lifetime remains independent of this result.
`internal/core/service.go:319-334 (loop) @ 910c353`.

Child output bytes are discarded after fixed-buffer draining; only stdout/stderr
presence flags are retained. The manager accepts five fixed diagnostic categories,
and retains the newest 200 log entries. This does not provide raw Xray error text.
`internal/core/process.go:49-87 (outputDrain) @ 910c353`;
`internal/core/service.go:198-227 (loop) @ 910c353`;
`internal/core/service.go:308-315 (loop) @ 910c353`.

No automatic restart, alternate health destination or startup-health failure deadline
was found in the inspected service/process code. An unexpected exit terminates the
session through cleanup with a fixed error.
`internal/core/service.go:429-475 (run) @ 910c353`.

## 5. UI and API synchronization

UI subscribes before loading a snapshot, rejects older status revisions and resyncs
on focus/visibility changes. A separate generation counter rejects obsolete refreshes.
`frontend/src/App.tsx:48-75 (refresh) @ 910c353`;
`frontend/src/App.tsx:107-112 (toggleConnection) @ 910c353`.

The command lock is released before an unawaited refresh. Backend Connect acknowledges
acceptance without waiting for preparation or process/network I/O; the scheduled worker
may begin I/O before the acknowledgment is delivered.
`frontend/src/App.tsx:77-88 (perform) @ 910c353`;
`internal/core/service.go:234-282 (loop) @ 910c353`.

Profile metadata, rename/delete controls, separate process/health display, pinned
version, last-check time and latest 100 log entries are exposed in the single window.
Browser mode requires a native host before enabling operations.
`frontend/src/App.tsx:119-176 (App) @ 910c353`;
`frontend/src/bridge.ts:41-45 (desktop) @ 910c353`.

If the UI bridge remains callable after service startup failure, ListProfiles rejection
causes refresh's Promise.all to discard the fallback GetStatus result and show a generic
synchronization warning. This source read does not establish whether Wails displays the
window after ServiceStartup returns an error. Diagnostic-only events
update the backend log snapshot without emitting status; the UI sees those rows on
a later refresh trigger.
`app.go:21-96 (ServiceStartup) @ 910c353`;
`frontend/src/App.tsx:48-60 (refresh) @ 910c353`;
`internal/core/service.go:308-316 (loop) @ 910c353`.

No frontend interval polling, call timeout or browser-storage operation matched the
analyzer's scoped App/bridge/generated API search.
`frontend/src/App.tsx:48-88 (refresh) @ 910c353`;
`frontend/src/bridge.ts:30-45 (api) @ 910c353`.

## 6. Test definitions and saved results

Frontend has 20 test definitions: 16 App cases and four bridge cases. They use mocked
APIs/jsdom, so they do not establish native Wails behavior. The locator's initial
17-App-case estimate was corrected by the analyzer's exact count.
`frontend/src/App.test.tsx:23-198 (describe) @ 910c353`;
`frontend/src/bridge.test.ts:11-46 (describe) @ 910c353`.

Service tests replace the process runner and probe. Process tests use a helper
executable that does not open sockets. Windows-only definitions test abrupt-parent
Job Object cleanup and executable locking, using helper children rather than transport
traffic. These are definitions, not passes observed in this research.
`internal/core/service_test.go:17-65 (fakeChild) @ 910c353`;
`internal/core/process_test.go:13-93 (TestMain) @ 910c353`;
`internal/core/process_windows_test.go:18-103 (TestWindowsJobKillsChildOnParentExit) @ 910c353`.

Linux integration invokes real PrepareConfig then substitutes ephemeral inbound
ports; it uses loopback server/client/TLS fixtures, bypassing the Windows service and
supervisor. Both proxy protocols are tested with the server running and stopped;
cleanup asserts that fixture children were reaped.
`integration/prepareconfig/main.go:11-21 (main) @ 910c353`;
`integration/transports.py:119-195 (main) @ 910c353`.

The saved Linux artifact records 12 PASS entries, six requests and six no-bypass
checks, plus successful cleanup. It identifies engine pins and time, but not the
application source commit or a Windows result. Its engine commit is not the app commit.
`integration/results-linux-amd64.json:2-71 (engine) @ 910c353`.

## Where the answers meet

PrepareConfig is used both at import and before startup; it rejects unsupported
configuration before runtime creation. This links questions 1 and 2.
`internal/core/store.go:116 (add) @ 910c353`;
`internal/core/service.go:365 (run) @ 910c353`.

Service.loop rejects stale session IDs and rejects Connect while active; health,
process and UI status publication all pass through it. This links questions 2, 4 and 5.
`internal/core/service.go:234-255 (loop) @ 910c353`;
`internal/core/service.go:286-334 (loop) @ 910c353`.

## Prior documents

No prior research run or standalone adr-*.md was found. The parent separately read
Architecture after v0.2 audit, Verification after audit fixes, Real transport integration
tests and the historical v0.2 audit. They were excluded from analyzer inputs and are
not substituted for source evidence in this report.
