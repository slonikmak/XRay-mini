---
title: Keep the application window stationary
type: bugfix
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: 'b6252aae0c7004109b5cf78018afc17c06411267'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user requested that scrolling no longer shifts the window UI">

## Intent

**Problem:** At compact heights the document can grow taller than the desktop window, so the whole interface moves when scrolled.

**Approach:** Keep the application and its navigation fixed to the window. Make the normal compact layouts fit, and confine necessary overflow to the relevant content areas, preserving access to long profile lists, diagnostics, import input, warnings and actions. Do not change connection or import behavior.

</frozen-after-approval>

## Implementation Notes

The document's `min-height: 100vh` permits page overflow. Adjust the viewport layout and local overflow in `frontend/src/style.css`; if needed, group dynamic notices in `frontend/src/App.tsx` so they cannot crowd out the current screen. Preserve accessible Rename/Delete dialogs and the flexible import textarea. The untracked design belongs to the user and is outside scope.

At compact heights the normal Details and Import screens fit without scrolling. Long content has bounded local scrolling; critical session and import errors are brought into view instead of being clipped.

## Plan Change Log

## Review Triage Log

- Medium, fixed: an owned-session error could appear below older notifications in the bounded message area. It is now shown first and the message area returns to its top when the error arrives; a regression test covers the ordering.
- False: removing the textarea resize handle was claimed to remove working manual resizing. A Chromium drag against the baseline stylesheet changed the inline height from 179 to 284px but the flex layout kept its rendered height at 179px. The editor still expands with the window and scrolls its own text.
- Medium, fixed during visual verification: with an active session and invalid JSON at 360×520, the import error was partly clipped. Errors now appear at the top of the form, the editor can shrink as needed and the compact form keeps its action visible.
## Verification

- Windows amd64, Go 1.26.10 and Wails 3.0.0: 75 frontend tests passed and `wails3 build` passed, including TypeScript and Vite.
- Chromium with a mocked desktop bridge: normal screens stay fixed at heights 520, 530, 570 and 610px; 40 profiles, 90 log entries, long input, import acknowledgement and long status messages remain accessible through local scrolling. A 360×520 active-session import error and its action both remain visible.
- `python scripts/package_runtime.py --verify bin` passed for the complete package and pinned runtime. No real proxy connection was exercised.
