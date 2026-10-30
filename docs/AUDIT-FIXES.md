# v0.2 audit traceability · 2026-10-30

## Authoritative inputs

The full specification and baseline audit were read through Library, not inferred from snippets, and are copied verbatim alongside this file:

- `xray-mini-spec-v0.2.ru.md`: Library `libfile_a2e867a51144819192e6ba8db717900c`, 27,333 source bytes, created 2026-10-30T12:32:33.590636Z. Audit-reported original SHA256 `261d26ee0a1b57139a6ae1d02711338bb9feb55abee69287ac0436ab8e07df5e`.
- `xray-mini-audit-v0.2.ru.md`: Library `libfile_a22109bb9d808191862d52d0f7e9263b`, 37,316 source bytes, created 2026-10-30T13:37:09.054123Z.

The audit baseline is commit `50b8cb36b311148871d21084c0b83dfba51198ed`, Git tree `c35e5fde5672148f553610513ebda5cb404abce0`. Its old implementation findings are historical; they are not rewritten to imply acceptance.

| Finding | Code change | Evidence / remaining acceptance |
|---|---|---|
| REVIEW-01 routing/import | One VLESS, required streamSettings, duplicate-key/dependency rejection; fixed selected-only runtime/loopback routing; explicit UI discard preview and confirmation. | Config regression tests cover all three transport field sets, missing/ambiguous VLESS, direct-route removal, fixed routing and unsafe dependencies. Pinned Linux real-core integration passed12 transport/proxy/no-bypass checks; see TRANSPORT_TESTS.md. |
| REVIEW-02 trusted runtime | Official Xray26.10.29 exact archive/executable pins; package script; controlled runtime path; validation before each invocation under Windows executable write/delete lock; version UI and licenses. | Official Windows archive downloaded; archive+exe hashes verified; package materialized. Missing/substituted hash tests pass. Native locked execution test written but Windows run pending. |
| REVIEW-03 crash residue | Exclusive cross-process data-directory lock precedes stale runtime cleanup; lock held until service/worker cleanup. | Normal/cancel cleanup, active second-instance protection, actual Linux subprocess owner-crash/restart tests. Windows lock/DACL/crash-recovery still pending. |
| REVIEW-04 child crash window | Native CreateProcessW JOB_LIST + kill-on-close job for preflight and runtime; restricted inherited handles. | Cross-compiles; Windows test harness kills parent without cleanup and checks both children. Actual Windows execution, nested jobs and stage-race stress pending. |
| REVIEW-05 architecture | Channel/select manager single writer; unique sessions; tagged events; disk/DPAPI/process/network outside loop; bounded coalescing publisher and bounded diagnostics. | Race tests, old-session events, slow callback, blocked store I/O, duplicate connect, late cleanup, once-only Wait and repeated reconnect tests. |
| REVIEW-06 health | 5s deadline, serial 30s ticker, two-failure threshold, success reset, timestamps; no restart. | success→failure→failure→success regression with timestamps; cancellation/nonoverlap tests. After first failure state retains Connected but health explicitly unhealthy and UI says check failed. |
| REVIEW-07 diagnostics | Fixed allowlisted stdout/stderr presence categories, fixed-size drains, bounded200 log history, levels, pinned version. | Raw secret output never retained; output-flood process test. Live categories collected by a bounded one-second supervisor ticker and at exit, once per category/session. No claim of rich sanitized raw Xray logs. |
| REVIEW-08 API/UI | RenameProfile; GetRecentLogs(limit); Connect(profileID); connection:state-changed subscription + initial/focus snapshot with revision guard. | Generated Wails3 bindings and frontend20-test regression suite including cancellation despite blocked profile refresh. Native window lifecycle still pending. |
| REVIEW-09 acceptance | Added Windows executable-lock/job-crash harness and documented exact platform/network acceptance matrix. | **NOT fully closed.** Actual Linux real-core transport integration passed; Linux tests/cross-build do not establish Windows10/11, WebView2, DPAPI/DACL, real Windows transport execution or external-server behavior. See VERIFICATION.md and integration results. |

## Completion boundary

Implementation fixes are not equivalent to accepted MVP. Do not describe REVIEW-09 or all of the specification as verified until the pending target-platform runs are actually executed and recorded. No user server credentials, personal secrets, real OS network-setting changes, installer, signature or auto-update are introduced by these changes.
