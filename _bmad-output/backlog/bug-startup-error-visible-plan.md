---
title: 'Visible safe startup failures'
type: bugfix
ticket: 11
created: '2026-11-01'
status: built
baseline_revision: 'ee4832faa679f632b0bda850fe6c6f288dedcd24'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="Necessary spike6 follow-up authorized by autonomous execution">

## Intent

Keep the desktop façade alive when the core cannot initialize, so the window can
display the fixed safe startup cause. Core methods must still reject operations.
Apply GetStatus independently of failed/pending profile/log refresh, keeping alive,
generation and status revision guards. Preserve current layout, native host checks,
secret handling and command cancellation. No diagnostic polling enhancement is needed.

Given an exclusively owned temp data directory, startup should expose Error with
safe cause and reject mutations; given failing/pending ancillary refresh, fresh status
must become visible and older snapshots must not override newer events.

</frozen-after-approval>

## Implementation Notes

Estimated change under100 lines including two focused regressions; direct oneshot.
Files: app.go safe nonfatal initialization/GetStatus; app_test.go temp owner regression;
frontend/src/App.tsx independent snapshot; App.test.tsx rejection/pending regression.
SDK source and pre-existing native artifact both show startup abort. User delegated
routine decisions/checkpoints; preserve existing diagnostic refresh model for F4.

## Review Triage Log

- high / patch: APPDATA-only isolation is Windows-specific. Restricted the native
  façade regression to Windows and registered shutdown before assertions so a failed
  test cannot open an unisolated user store on another platform or leak a service.
- medium / patch (native observation): the cold UI can see host injection late;
  failure of profiles then left available=false and falsely labelled native startup
  error as browser preview. Detect availability as soon as desktop() succeeds,
  independently of ancillary data, with a late-host-injection regression.

## Verification

- `go test -tags production .` — façade startup regression passes on Windows11.
- `npm run typecheck` and `npm test` in frontend — all UI/bridge cases pass.
- Later production rebuild and isolated native window check record actual outcome.

## Completion, 2026-11-01

Both review patches verified: Windows-only APPDATA isolation and late native host availability. Final frontend typecheck and 23 tests pass, production Go suite/vet pass, production Wails build succeeds. Final isolated native startup-error window displays the safe cause, disables Connect and has no browser-preview banner.
