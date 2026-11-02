---
title: Refine compact interface from Pencil
type: feature
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '051628cec9b6d223c8d733ad36c4630227255c51'
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

<frozen-after-approval reason="user-provided visual and behavioral intent">

## Intent

**Problem:** The implemented compact interface predates the user's refinements to its Pencil screens.

**Approach:** Bring all Compact screens into line with the updated designs: a quiet instrument, a single power circle as the main action, and status dots with truthful meanings.

## Boundaries & Constraints

**Always:** Inspect all six current Compact frames in `design/xray-mini.pen` through Pencil MCP, without changing that file. Follow their spacing, color roles, surfaces, type, radii, grouping, iconography, and control hierarchy. Use one actual accessible keyboard-operable power button with an appropriate accessible action name, disabled states and visible focus. Distinguish active and selected profiles. Reflect backend state and health separately; mint status dots mean the relevant process is running or a check actually passed. Avoid pulsing. Preserve native frame and existing layout responsiveness, scrolling, imports, diagnostics refresh, status revision ordering, tray behavior and safe stop/retry controls even in busy dialogs.

**Never:** Modify backend/session lifecycle, generated bindings, runtime, tray behavior or the user's untracked design. Do not label errors as success or imply the local proxy is running while its process is stopped. Do not fabricate profiles, logs, status, network checks or a live connection.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Disconnected | Stopped with selected profile | Quiet power circle named Connect; heading Proxy is off; mono local endpoints | No profile or unavailable backend disables Connect |
| Connected / degraded | Running process; actual network health | Circle invokes Disconnect; mint circle and network/status dots only when the check is healthy; running process dot reflects process | Unhealthy/unknown check does not appear verified |
| Transition / recovery | Preparing, stopping, cleanup or retained Error | Same circle exposes Cancel startup, disabled cleanup or Retry stop as appropriate | Error messages and retry remain available, including inside modal |
| Selected != active | A is connected; select B | A remains active and can be stopped; current selection is clear without changing the session | Connect B is possible only after cleanup |
| Diagnostics | Real log level/message and timestamp | Compact mono rows, original timestamp retained, neutral or verified-success dots; refresh on entry | Unavailable/loading/empty are distinct, failures remain identifiable |

</frozen-after-approval>

## Code Map

- `frontend/src/App.tsx`: `connectionDisabled`, `connectionLabel`, `toggleConnection`, status revision handling and modal recovery already implement the safety boundary. Reuse them. Replace decorative power plus wide action with one button; retain state-dependent accessible names. Home heading and subtitle, info icon, Details grouping, labels, profile actions, mono runtime, and log rows require UI changes.
- `frontend/src/style.css`: current compact style; map to Pencil variables (`bg`, `card`, `surface`, `border`, `txt`, `muted`, `dim`, `accent`, `accent-ink`, `accent-dim`, `accent-dim-2`, `danger`) and 4px spacing. Match all six frames without losing focus visibility, long-content scrolling or reduced-motion behavior.
- `frontend/src/assets/`, `licenses/NOTICE.md`: include the official Inter Variable font locally and its copyright/OFL notice in the packaged runtime notice, preserving offline rendering and licensing.
- `frontend/src/App.test.tsx`: preserve existing behavioral tests and update text expectations. Add focused checks for power control and transitions, truthful indicators, event-driven status updates, and actual successful versus ordinary info log rows.
- `internal/core/service.go:279-293,460-471`: `info` includes preparation; only exact `HTTP proxy check passed; other destinations may differ` is a known successful check. Buffer lasts for app-service lifetime, so retain the accurate `since app start` footer despite the sample frame text.
- `main.go` and `frontend/src/bridge.ts`: tray uses Go status and UI subscribes/refreshes on focus. No changes required; retain this integration.

## Tasks & Acceptance

**Execution:**
- [x] `frontend/src/App.tsx` — implement the updated screens, one real circle action and truthful status dots, preserving all lifecycle, import and diagnostic behavior.
- [x] `frontend/src/style.css` — align spacing, color, icons, fonts, surfaces, radii and responsive overflow to the Compact frames.
- [x] `frontend/src/assets/`, `licenses/NOTICE.md` — bundle the design font and its distribution notice.
- [x] `frontend/src/App.test.tsx` — adapt behavioral assertions and add meaningful tests for the matrix and keyboard/accessibility of the circle.

**Acceptance Criteria:**
- Given a stopped or verified running session, when Home renders, then the single circle, heading, endpoints and indicators match the corresponding updated Pencil frame and perform the correct action.
- Given intermediate, degraded, error or cleanup state, when the backend sends an event, then the visual indicators remain honest and the correct stop/cancel/retry action remains reachable.
- Given Details, Profiles, Import or Diagnostics, when navigated and used at compact window size, then the controls, typography and spacing match the new frames and remain usable with long content and a keyboard.

## Implementation Notes

- Updated all six Compact views and kept process, health, selected profile, diagnostic refresh and stop/retry behavior tied to actual backend state.
- Verified the bundled Inter v4.1 font SHA-256 and the complete license against the pinned upstream release. The package notice contains the font license.
- On Windows 11 Home with Go 1.26.10 and Wails 3.0.0: 62 frontend tests, 10 packaging tests, production Go tests and the isolated Wails build passed. The isolated `bin` passed the package verifier. All screens were inspected using synthetic data at 360×570; the native window and local synthetic import were checked with a separate profile store. No external proxy connection was started.
- A pre-existing Xray Mini process continued running; the updated complete package is under `artifacts/compact-instrument-20261102/bin`.

## Plan Change Log

## Review Triage Log

- medium — patch: retry/cancel actions on the circle were exposed only via accessible name and tooltip; a visible contextual action hint is required for transition and recovery states.
- medium — patch: the initial placeholder status could be presented as an actual stopped proxy while backend status was loading or unavailable; show unknown/loading until a real status arrives.
- medium — patch: several borders in the Pencil frames, including input outlines, were omitted in CSS; apply the specified border color without breaking compact sizing.
- medium — patch: Inter was not bundled or installed on the test host, causing a system-font fallback, and the hero heading weight differed from Pencil; bundle the official font offline with its license and use the specified weight.

## Verification

- `npm test` and `npm run build` in `frontend` — existing and focused tests plus production build pass.
- `wails3 task build APP_NAME=XrayMini-compact-updated` — the normal Wails build targets a separate executable because an existing app is running; no parallel npm install/test.
- `python scripts/package_runtime.py --verify <isolated-bin>` — required package contents and pinned runtime verify for the new executable.
- Inspect the six layouts at a 360px-class viewport with synthetic data, including connected, unhealthy, long names/logs and a pending import; inspect the native Windows/WebView2 result using isolated test data if available.
