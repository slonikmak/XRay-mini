---
title: Prevent import screen scrolling
type: bugfix
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '80c0817d414f32dd3ea4d60b14ce4464a4fb9c4a'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user reported that the import screen still scrolls">

## Intent

**Problem:** The import form remains a scroll container, so the screen shifts under the user's mouse wheel.

**Approach:** Keep the import screen stationary while preserving accessible JSON and subscription import, error messages, explicit acknowledgement of discarded sections, and independent connection controls. Long configuration text and other genuinely bounded content may scroll within their own controls; the form itself must not scroll. Keep the design's back navigation and primary action available.

</frozen-after-approval>

## Implementation Notes

The wheel moves `.import-screen` by 137px in a JSON warning state and by 5px in the link state at 360×520 with an active session. A simple overflow clip would hide confirmation and actions. Adapt `frontend/src/App.tsx` and `frontend/src/style.css` so the form fits its states; retain security and concurrency tests in `frontend/src/App.test.tsx`. Do not touch the user-owned untracked Pencil design.

JSON imports with discarded sections use a review state on the same screen. Repeated generic section names are collapsed, acknowledgement remains mandatory, and Edit configuration returns to the unchanged input. The independent session action stays visible in the header; long warning details can scroll within their own keyboard-accessible region.

## Plan Change Log

## Review Triage Log

- Medium, fixed: clipping the form could hide the primary button's keyboard focus outline. Import controls now draw focus inside the form, with a contrasting outline on the mint button; checked in Chromium with keyboard navigation.
- False after browser verification: an 80-character wide candidate name was suspected to expand the form horizontally. Before and after selection, the form, select and submit button remained within the 328px content width.
## Verification

- Chromium with a mocked desktop bridge: wheel and programmatic scroll leave the form fixed at 360px width and 520–620px heights. JSON errors, a warning with repeated categories, review/edit/keyboard confirmation, subscription loading/cancellation/errors, pending saves and retained-session recovery keep their controls visible.
- Windows amd64, Go 1.26.10 and Wails 3.0.0: 77 frontend tests passed; `wails3 build` and `python scripts/package_runtime.py --verify bin` passed. The target executable was not running during the build; no native WebView2 wheel interaction or external connection was exercised.
