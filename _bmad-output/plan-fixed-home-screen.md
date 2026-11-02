---
title: Prevent home screen scrolling
type: bugfix
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '36353b41283d84989c62fdd97c3cbb560066dd86'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user showed that the home screen still moves on scroll">

## Intent

**Problem:** The previous fix disabled document scrolling but left the home screen scrollable, so its profile card and connection control still move beneath the header.

**Approach:** Make the home screen itself stationary under wheel and programmatic scrolling. Keep the profile, connection or retry action, state indicators and relevant messages visible at compact window sizes, including with long names and notices. Preserve access to complete information and keep necessary scrolling in long lists, logs and configuration input.

</frozen-after-approval>

## Implementation Notes

Reproduced with a bounded notifications area, long selected/active profiles and a retained-session error: `.home-screen` was 322px high with 453px of content and the wheel moved its scroll position by 131px. Adjust `frontend/src/style.css` and, where necessary, home-screen markup in `frontend/src/App.tsx`; keep connection and import semantics unchanged. The untracked Pencil design is user-owned.

The home content is clipped against scrolling rather than made into a scroll container. Compact layout responds to the space remaining after notifications; full names are available in titles and Profiles, and the full error is keyboard-accessible through Details.

## Plan Change Log

## Review Triage Log

- Medium, fixed: a height just above the compact-layout threshold could clip status indicators with long names. Names now have bounded layout and the hero can shrink; boundary viewport tests cover both sides of the threshold.
- Medium, fixed: a new connection failure could be hidden below an existing refresh error in the bounded notification area. Connection errors now appear first and the area returns to the top; a regression test covers the ordering.
- Medium, fixed: a truncated home error was not directly operable with a keyboard. The error is now a keyboard-accessible control that opens the full message in Details.
## Verification

- Chromium with mocked desktop data: real wheel and programmatic-scroll attempts leave the card, control, indicators and status fixed at 360px width and heights 520–650px, including long names, stacked notices and retained-session errors. Profile lists, logs and configuration input retain local scrolling.
- Windows amd64, Go 1.26.10 and Wails 3.0.0: 77 frontend tests passed, `wails3 build` passed, and `python scripts/package_runtime.py --verify bin` passed.
- An isolated hidden WebView2 instance started, but exposed no targetable window or debugging endpoint; no native wheel interaction is claimed.
