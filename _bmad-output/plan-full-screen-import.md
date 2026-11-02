---
title: Full-screen profile import
type: feature
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '013fb48f3d31a952946f652e8def2e762ed05a1e'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user requested the updated Pencil import screen">

## Intent

**Problem:** Profile import is constrained to a modal, unlike the application's other compact screens.

**Approach:** Implement the Compact Import frame from `design/xray-mini.pen` as a full-window screen with a back arrow, flexible JSON textarea, privacy note and full-width primary action. Return to the screen from which import was opened. Preserve JSON preview and acknowledgement, subscription/VLESS import, safe clearing and cancellation, and access to connection and retry-stop controls during ongoing operations.

</frozen-after-approval>

## Implementation Notes

Use the existing screen navigation, React form and CSS tokens in `frontend/src/App.tsx` and `frontend/src/style.css`; keep rename and delete dialogs. Adjust `frontend/src/App.test.tsx` for screen navigation and verify no loss of secret clearing, stale-preview rejection, import validation or stop recovery. This is a frontend-only adaptation of an existing flow. Do not alter the user's untracked Pencil design.

During a pending import, Back is disabled to keep the in-flight operation visible; connection actions remain independent. Returning while a subscription preview is loading clears it and rejects late results.

## Plan Change Log

## Review Triage Log

- Medium, fixed: when an import operation was pending without an active session, the selected profile could not be connected. A conditional connection action now remains available, with a focused regression test.
## Verification

- Windows amd64, Go 1.26.10 and Wails 3.0.0: frontend tests passed (74 tests), and `wails3 build` passed including TypeScript and Vite.
- Inspected JSON and subscription forms in Chromium with a mocked desktop bridge at 360×570, 360×740 and 320×420. The textarea grows with height and the small window scrolls without horizontal overflow.
- `python scripts/package_runtime.py --verify bin` passed for the full package and pinned runtime. No real proxy connection was exercised.
