---
title: Xray Mini tray menu
type: feature
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '9537934e2c877b75ab5c95f4cf41aa874c5cd4b4'
route: full
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context:
  - AGENTS.md
---

<frozen-after-approval reason="User accepted the proposed tray behavior and authorized implementation in this chat">

## Intent

**Problem:** Closing the only desktop window currently ends Xray Mini and its connection. The user wants a conventional tray icon and menu so the app can stay available in the background.

**Approach:** Closing the window hides it while the active session continues. A native tray menu exposes safe status, opens the existing window, allows stopping the current session, and provides an explicit Exit that runs normal shutdown.

## Boundaries & Constraints

**Always:** Derive tray state from the Go service status, including startup errors and retained cleanup sessions; keep menu actions responsive. The window opens at its normal location. Explicit Exit must call the existing service shutdown and Xray cleanup. Preserve the distinction between an active session and the profile selected only in React.

**Never:** Connect from a tray profile guess, change core session ownership or network behavior, display secrets in tray text, or modify the user's untracked `design/` files. Do not turn the main window into an attached popup.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Window close | Connected or Connecting | Window hides; app and session continue | Tray icon can reopen window |
| Reopen | Hidden window | Same window shows and focuses; React refreshes Go status | No second app or Xray process |
| Tray action | Active, stopping, error with retained session, or idle | Appropriate Disconnect/Cancel/Retry stop action appears only when valid | Native menu does not block while cleanup runs |
| Exit | Any state | Application runs normal `ServiceShutdown` and exits | Existing bounded cleanup/error reporting remains authoritative |

</frozen-after-approval>

## Code Map

- `main.go`: creates the Wails app and native window. Add tray, embedded `build/appicon.png`, native menu, close hook, and state rendering here. Wails v3.0.0 provides `SystemTray.New`, `MenuItem.SetLabel/SetEnabled`, `Menu.Update`, `InvokeAsync`, and `WindowClosing`.
- `app.go`: `ServiceStartup` creates `core.Service`, installs a status callback, and `ServiceShutdown` calls `service.Close`. Reuse that callback for a lightweight tray notification; keep core publication nonblocking.
- `internal/core/service.go`: `GetStatus` reads an atomic Go snapshot. Session ID and process state indicate which stop/retry action is valid. Do not change this file.
- `docs/xray-mini-spec-v0.2.ru.md` FR-05/08/09 and `docs/VERIFICATION.md`: source of existing connection, single-window, shutdown, and Windows acceptance requirements.
- `build/appicon.png`: existing native mark, reuse as embedded tray source. Leave `design/` untouched.

## Tasks & Acceptance

**Execution:**
- [x] `main.go`, `app.go` -- create tray and close behavior, derive menu/tooltip from service snapshots, and keep menu callbacks short.
- [x] `main_test.go` -- verify state-to-menu behavior, especially error with retained session and idle/transition cases.
- [x] `README.md` -- document close-to-tray and explicit Exit behavior for users.

**Acceptance Criteria:**
- Given a running connection, when the user closes and reopens the window, then the same connection and status remain visible.
- Given a hidden window, when the user chooses Open or clicks the icon, then the original window becomes visible and focused.
- Given a retained failed-stop session, when the menu updates, then Retry stop is available without pretending that cleanup completed.
- Given any active session, when the user chooses Exit, then the owned Xray is stopped through the existing shutdown path.

## Implementation Notes

- Embedded the existing `build/appicon.png` as the native tray icon. Window close cancels destruction and hides the original window; Open and icon click show and focus it at its existing location.
- Tray status is rendered from `App.GetStatus()` using fixed, secret-free labels. A retained session in Error exposes Retry stop, while idle and stopping states have no stop action. Core status callbacks send only a coalesced, nonblocking notification; menu updates run on the Wails UI thread.
- Exit invokes Wails `Quit`, which runs the existing `ServiceShutdown` and `service.Close` cleanup path. The tray stop action runs separately from the native menu callback.

## Plan Change Log

## Review Triage Log

- 2026-11-02 quick inspection: 0 findings; no code changes or deferred work.

## Verification

**Commands:**
- `go test -tags production ./...` -- Go tests pass on this Windows host.
- `wails3 build` then `python scripts/package_runtime.py --verify bin` -- complete packaged runtime verifies.

**Manual checks:**
- On Windows 11 with isolated APPDATA, the window opened in Disconnected state. Clicking its native Close button hid it while the owned GUI process remained alive. Computer Use could not target the Windows notification area, so clicking the tray icon, opening its menu, and Exit with an active Xray session remain unverified natively in this run.
