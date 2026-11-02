# Architecture after v0.2 audit

## Single writer and worker ownership

`Service.loop` selects over commands and session-scoped worker events. It alone mutates active session, state, monotonic revision, health failure count and bounded log history. Immutable snapshots are atomically published; GetStatus never waits on disk, DPAPI, process startup, network or a UI callback. The event publisher has one goroutine and one coalescing slot, not a goroutine per event. Close waits at most 250 ms for publication after session/storage cleanup and reports an undelivered-notification error if a callback remains blocked. That single callback may outlive Close until it returns; no replacement publisher is spawned.

Connect accepts only a profile ID, reserves a unique random session ID and returns command acceptance. A worker reads the profile under a separate repository lock, builds private runtime config, checks ports, performs verified preflight and starts the supervised process. Every worker event carries the session ID; stale events cannot update a later session. Deletion reserves the profile ID in the manager before file I/O, preventing a connect/delete race. Rename and other repository operations never hold the manager.

The session worker owns the child; one waiter calls Wait once. ProcessStarted is sent only after ownership and waiter registration. Cancellation kills then waits; even a late successful Start is cleaned up. ProcessExited does not permit reconnect: only StopCompleted after reader/probe/process/file cleanup does. Failed file cleanup publishes a safe Error and retains the session. Connect in this state requests one cleanup attempt and returns a fixed retry error; after confirmed cleanup, a later Connect can start a child. Disconnect or Close can also retry cleanup. There is no automatic polling or restart. Close waits at most five seconds for session/storage cleanup and retains ownership when stopping is unconfirmed; a later Close can retry. The instance lock is released only after actual cleanup. Native Kill terminates the owned Job Object; the single Wait retains native handles until exit.

## Network and state

Both generated inbounds bind 127.0.0.1. The sole selected VLESS outbound has a fixed tag and both inbound tags route explicitly to it. No original routing, direct fallback, DNS or policy is copied. The selected standalone transport subset rejects dependent references. The core, not Go/React, carries user traffic.

A serial checker begins immediately after startup, then on a 30-second ticker. Within a shared 5-second deadline it waits for the local proxy TCP connection in bounded 50 ms intervals and uses the successful connection for one real HTTPS control request. No overlapping checks or HTTP-request retry burst; TCP readiness alone cannot report health. Two consecutive failures produce Degraded; success resets failures and produces Connected. Process state and network health remain separate. Last check time is included in status. Failure may mean the control resource is unavailable, not necessarily that the VLESS server failed.

React subscribes to `connection:state-changed` before loading the initial snapshot, drops older revisions and resynchronizes on focus/visibility. Reopening a window does not recreate backend connection state.

ProcessExited is terminal within its session: the manager records stopped/unknown, cancels remaining work, and rejects subsequent operational events. The waiter captures whether exit was unexpected before publishing; later stop commands cannot erase that fact. SessionID remains present until StopCompleted, then is cleared, including on final Error. A stopped process and a released session are different facts. After a failed Kill of the working child, the worker waits for actual exit or one explicit Disconnect/Close retry; it never spins on a canceled context. A bounded retry signal also retains a repeated stop command that overtakes StopFailed. Wait and resources keep the same owner. The native adapter's initial cancellation Kill remains separate from subsequent explicit supervisor retries.

Profile mutations and connection commands have separate UI locks. Every open profile modal exposes the active session's Stop/Retry stop action, including during a pending save. Error with retained sessionID permits recovery; ordinary stopping remains disabled. A cleaned connection failure permits profile repair and another Connect. Focus/visibility recovery also registers a missing late-host subscription before requesting its snapshot; subsequent events need no further focus. There is no claim of host discovery without one of those existing recovery triggers.

The Wails facade retains safe core startup errors while allowing the native window to open. GetStatus exposes Error and the pinned runtime version; mutations still fail. Status refresh applies independently of failed or pending profile/log requests. Native host availability is detected independently of ancillary API success, including late host injection.

## Trust, storage and crash containment

Only the bundled pinned Windows x64 Xray is allowed. SHA-256 is checked before every invocation including preflight; the version and executable pin are in runtime.go, the archive pin is in scripts/package_runtime.py, and release provenance is in licenses/NOTICE.md. The runtime file is not a sandbox; this does not defend against an attacker already able to modify the user's process/memory or files concurrently.

Windows profile encryption is DPAPI; directories have restricted DACLs. A crash-released exclusive OS file lock owns the data directory. Stale runtime directories are removed only while holding this lock, before connecting. Live second instances fail without deleting anything. Normal cleanup removes plaintext runtime files but secure erasure is not claimed.

Native Windows CreateProcess receives STARTUPINFOEX with JOB_LIST and an explicit inheritable HANDLE_LIST for standard streams. The kill-on-close job handle itself is not inherited. Thus both preflight and working processes are born inside the job; no suspended-create/assign gap. Pipe readers use fixed-size buffers and discard bytes, exposing only two fixed presence categories; no raw text, addresses or credentials reach logs. Reader lifetime is joined with Wait.

## Remaining acceptance

Code and cross-compilation are not native execution evidence. Windows 11 with installed WebView2, nested-job behavior, crash races, DPAPI, DACL and real transport integration are tracked separately in VERIFICATION.md and ACCEPTANCE-SCOPE.md. TUN, system settings, auto-restart/update and arbitrary Xray configuration are deliberately out of scope.
