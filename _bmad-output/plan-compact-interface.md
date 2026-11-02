---
title: Compact interface
type: feature
ticket: ''
created: '2026-11-01'
status: built
baseline_revision: '2e2181b04d709ee360643cd4d6d1eafb7b05843e'
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

<frozen-after-approval reason="user intent">

## Intent

**Problem:** The wide desktop interface does not follow the user's Compact designs in Pencil.

**Approach:** Adapt the desktop window and the existing React UI to the Compact home, connected, details, profiles, import and diagnostics screens, keeping the real desktop behaviors usable.

## Boundaries & Constraints

**Always:** Use the Compact screens in `design/xray-mini.pen` as the visual reference; access the file only with Pencil MCP. Maintain the real Go status/revision as the authority, distinguish selected and active profiles, and keep error/recovery actions and existing import flows accessible. The window, screens, forms and long lists must remain usable by keyboard and at small sizes; allow scrolling. Keep the native window frame and OS controls.

**Never:** Change backend lifecycle, authentication, proxy configuration, runtime or generated bindings; fabricate profiles, logs, network health or connection status. Do not modify the user's design file or unrelated changes.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Connect or recover | Existing selected profile and status from Go | Home shows actual process and network status and provides the correct connect, cancel, disconnect or retry action | Retained Error session permits Retry stop; other errors stay visible |
| Select another profile | Profile A active, choose B in Profiles | Selection changes only; A remains active; Disconnect controls A, a later Connect uses B after cleanup | Never delete active A or silently start B |
| Import | JSON or subscription modal open, possibly during active session | Existing preview, acknowledgement, candidate selection, cancellation and secret clearing work | Disconnect/retry remains available even when a profile operation is pending |
| Details and diagnostics | Addresses, runtime version and bounded logs from backend | Navigate back, copy both addresses, view runtime and logs | Show empty or unavailable states honestly |

</frozen-after-approval>

## Code Map

- `frontend/src/App.tsx`: existing connection/status synchronization, profile CRUD, JSON/subscription import and dialogs. Keep their concurrency guards and revision ordering; replace wide layout with screen navigation.
- `frontend/src/style.css`: current wide layout; replace with compact dark visual language from Pencil. The Compact frames are `Ldm2t`, `tEbx1`, `xMtM8`, `E5EIx`, `RiHqd`, `s2agP9`. Pencil `get_app_state` and `execute`/`TakeScreenshot` can inspect them.
- `frontend/src/App.test.tsx`: existing behavioral regression tests. Update navigation assumptions without deleting the safety assertions; add a two-profile active-vs-selected check and navigation checks.
- `main.go`: Wails window currently starts at 1100 × 760 with 800 × 600 minimum. Use a compact resizable native window, around 380 × 620–640 with minimum around 360 × 560; support overflow and system scaling. Wails has `Windows: application.WindowsWindow{Theme: application.Dark}` for the native dark title bar. Do not recreate native controls in HTML.
- `build/appicon.png`, `build/windows/icon.ico`: export the Compact title-bar mark from Pencil and generate the Windows icon with Wails; retain the exported source so the icon can be regenerated.
- `docs/xray-mini-spec-v0.2.ru.md` FR-07/08 and ADR-008 plus §5: process/health are distinct; session ownership and Retry stop survive until cleanup; UI state comes from Go.

## Tasks & Acceptance

**Execution:**
- [x] `frontend/src/App.tsx` — implement compact home, details, profiles and diagnostics navigation while preserving all profile and connection actions and the existing modal workflows.
- [x] `frontend/src/style.css` — implement the six Compact visual states and responsive scrolling.
- [x] `main.go` — choose compact resizable window dimensions and retain native controls.
- [x] `build/appicon.png`, `build/windows/icon.ico` — use the Compact mark for the native application icon.
- [x] `frontend/src/App.test.tsx` — adapt tests to the new navigation and cover the matrix's important behaviors.

**Acceptance Criteria:**
- Given a profile and actual status, when the app opens, then the compact home shows the selected profile, state, separate process/health checks and the appropriate action.
- Given the compact home, when the user opens Details, Profiles, Import or Diagnostics, then each screen resembles its Pencil reference, relevant existing actions work, and navigation back is available.
- Given an active profile and another selected profile, when the user changes selection, then the active connection does not change and cleanup is required before a new connect.
- Given unavailable backend, a long name, long diagnostics or a small window, when the UI renders, then messages and controls remain reachable and do not claim fabricated success.

## Implementation Notes

- Compact screens follow the Pencil references while preserving backend status ownership, profile/import operations, and stop/retry availability. Log timestamps are displayed in local time and retain the original value.
- The Compact app mark was exported from Pencil and converted to a Windows icon using the Wails generator.
- Frontend 55 tests, frontend production build, production Go tests and Wails build passed. The package contents were verified. Native Windows UI and synthetic Chromium layouts were inspected with isolated test data; no external connection was started.
- Review fixes refresh diagnostics on entry, keep log reads independent and ordered, distinguish an empty buffer from unavailable data, and describe the buffer's service lifetime accurately.

## Plan Change Log

## Review Triage Log

- medium — patch: opening the new Diagnostics screen did not request a fresh log snapshot; diagnostic-only backend events do not publish status events.
- medium — patch: an unavailable or failed log read rendered the same empty-state text as a successful empty result, and a profile-read failure could discard a successful log read.
- medium — patch: the new footer said logs belonged to the current connection session, while the backend buffer lasts for the service lifetime.

## Verification

- `npm test` in `frontend` — frontend behavioral tests pass.
- `npm run build` in `frontend` — typecheck and production bundle pass.
- Check visual layout against the six Compact references and verify scrolling, navigation and dialog interaction in a 360px-class viewport; inspect a native build when available.
