---
title: Clarify connection status headings
type: feature
ticket: ''
created: '2026-11-02'
status: built
baseline_revision: '19b00708654dbf049bbb916e44540f533ba7caee'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user-approved status wording">

## Intent

**Problem:** The home heading does not distinguish a running process from a successful connection check.

**Approach:** Display the agreed headings: Proxy is off, Starting proxy…, Checking connection…, Proxy is ready, Connection check failed, Disconnecting…, Connection error, Checking status… and Status unavailable. For a failed network check, show Check failed in the top pill even if Go still reports Connected. Keep backend state, process/health indicators, session ownership, connection actions, retry/cleanup behavior and event ordering intact.

</frozen-after-approval>

## Implementation Notes

Update the display mapping in `frontend/src/App.tsx` and existing focused behavioral tests in `frontend/src/App.test.tsx`. Retained-session errors take precedence over stopping; stopping and retained cleanup precede normal running/checking; distinguish process startup and check-in-progress. A new start request after a released error displays startup while its result is pending. Never infer a successful check from a running process. Give unknown network health a truthful unverified fallback. Keep changes frontend-only and leave the untracked design untouched.

## Plan Change Log

## Review Triage Log

- Medium, fixed: preserve `Degraded` in the pill after two consecutive failures, as required by ADR-007; a first failure still shows `Check failed`.
- Medium, fixed: after a released error, a pending new Connect request previously still showed `Connection error`; it now shows the starting state, with a focused test.
- Medium, fixed: exercise the retained-error heading during a retry and the cleanup heading after a status event, not just from an initial snapshot.

## Verification

- Windows amd64, Go 1.26.10 and Wails 3.0.0: `npm test -- --reporter=dot` passed (71 tests).
- `wails3 build` passed, including frontend TypeScript and Vite production build. `python scripts/package_runtime.py --verify bin` passed after refreshing the bundled NOTICE from `licenses/NOTICE.md`.
- State and event tests cover the headings, first-failed-check pill, `Degraded`, retry stop, pending cleanup, and a new request after a released error.
