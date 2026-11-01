---
title: 'Manual subscription import and profile selection'
type: feature
created: '2026-11-01'
status: built
baseline_revision: 'c0e412d67569c018b9c309cccdb559623f3a4d39'
route: full
route_source: auto
risk: high
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="User approved the proposed feature and requested BMAD delivery; prior autonomous execution remains authorized">

## Intent

The supplied HTTPS subscription returns five Base64-encoded VLESS XHTTP/TLS links, while the desktop accepts only complete Xray JSON. Add manual subscription loading, metadata-only candidate selection, explicit import of one profile and the existing Connect flow. Also accept one VLESS link through the same new input. The personal URL and response remain outside repository artifacts.

## Boundaries & Constraints

Always reuse Service.ImportProfile, PrepareConfig, DPAPI, pinned preflight, manager state and fixed loopback proxy policy. Fetch only explicit user-entered HTTPS sources, with normal TLS validation, bounded total time/body/count and cancellation. Reject ambiguous/unsupported parameters rather than silently change behavior. Preserve XHTTP path/mode/extra and TLS/REALITY settings required by supported profiles. UI receives candidate names/opaque IDs/transport only; source and configs stay transient. One ephemeral preview, bounded lifetime, invalidated on source change, cancel, replacement, successful import and shutdown; late results cannot restore it.

Never add automatic refresh, persistent subscription URLs, bulk import, background subscriptions, new dependencies, OS proxy/CA changes, raw HTTP/config errors in diagnostics or secret-bearing evidence. Keep existing JSON import and its discarded-section acknowledgement. Windows 10 and missing-WebView2 testing remain excluded.

## I/O & Edge-Case Matrix

| Scenario | Input/state | Expected behavior | Error handling |
|---|---|---|---|
| Subscription | Valid HTTPS Base64/plain VLESS list | Bounded candidate list, explicit selection, one saved profile | No automatic Connect |
| Single link | Supported VLESS URI | Same selection/import path, no fetch | Fixed validation cause |
| Mapping | RAW/REALITY or XHTTP/REALITY/TLS, IPv6 and escaped fields | Xray config preserves required values; current validator accepts | Unknown/duplicate/unsafe fields rejected |
| Network failure | Timeout, cancel, non-200, oversized body, invalid cert or unsafe redirect | No partial import; UI usable | Fixed secret-free category |
| Invalid content | Empty/malformed Base64, malformed URI/extra, unsupported transport/protocol, excessive count | No silent loss or import | Safe explanation without input echo |
| Lifetime | Edited source, old ID, expired preview, cancel/reload/shutdown or late result | Old candidate rejected, cache discarded | No stale selection/import |
| Existing flow | Complete JSON | Existing preview/acknowledgement/DPAPI path unchanged | Existing regressions pass |

</frozen-after-approval>

## Code Map

- `app.go`: Wails facade owns ephemeral preview and cancellation; no core manager changes. Startup currently discards context; derive app-lifetime context and cancel at shutdown. Guard new operations against closing. Expose preview metadata, import opaque candidate, clear/cancel; no config-returning binding.
- `internal/core/subscription.go` (new): stdlib fetch/decoder/URI conversion; validate with PreviewImport/PrepareConfig. Reuse config duplicate-key/dependency checks. Bound decoded and encoded payloads, profile count and parameter shape. No OS or storage side effects.
- `internal/core/service.go:ImportProfile`, `store.go:add`: already enforce validation/private encrypted storage and shutdown. Reuse without edits.
- `frontend/src/App.tsx`: add JSON/link choice inside import modal; preserve JSON acknowledgement and state revision guards. Clear source/candidates on edit/save/cancel/unmount. Loading cancellation must work without disabling connection controls indefinitely.
- `frontend/src/bridge.ts`: typed facade API; bindings generated only by Wails task. Existing dirty generated files are backed up and must not be wholesale staged.
- `frontend/src/App.test.tsx`, `bridge.test.ts`, `app_test.go`: existing test harnesses, Windows APPDATA isolation for native storage tests.

## Tasks & Acceptance

- [x] `internal/core/subscription.go`, focused tests: bounded HTTPS load and strict supported URI mapping, synthetic fixtures covering matrix; no personal URL in tests.
- [x] `app.go`, native facade tests: one transient preview, opaque IDs, expiry/generation/cancellation, explicit chosen import through existing service; no raw configs in serialized DTOs.
- [x] `frontend/src/App.tsx`, `bridge.ts`, tests and minimal existing CSS additions: accessible load/select/import/cancel flow with independent connection controls and secret clearing.
- [x] Generate bindings via `wails3 task bindings`; preserve original dirty bytes, stage only semantic feature changes.
- [x] Update canonical spec FR-02 and README for manual import; auto-refresh remains excluded. Record commands, exact source revision/dirty, hashes and limited real-world outcome without credentials.

Acceptance criteria: Given the provided subscription, when loaded then five selectable XHTTP/TLS profiles appear without raw config disclosure. Given a selected candidate, when imported then exactly that profile persists encrypted and Connect follows normal pinned supervision. Given a real authorized endpoint, when checked through HTTP/SOCKS then record actual PASS/FAIL and cleanup; do not infer connectivity from import/preflight. Given cancellation or stale preview, no late result/import restores invalid state. Given JSON import, all existing behavior still passes regression tests.

## Implementation Notes

No initiative is active; this is one loose feature in the existing project, consistent with the session's standalone work. The user already authorized autonomous execution and approved the immediately preceding feature proposal; routine dirty-tree/plan checkpoints are covered. Preserve unrelated .agents/_bmad/skills lock and root provided spec. Existing generated bindings are dirty before this feature; back them up first. The implementation worker owns the source/test/generated feature edits and README/spec changes listed above; it is not alone and must preserve others' work without reverting it. Do not commit during implementation. Parent owns personal-endpoint testing and final evidence/review/commit; never fetch or look for the private subscription URL. The subagent should run synthetic tests and the production build as needed, then report for parent review. Full route (>100 changed lines); risk high because URL/config inputs contain credentials. Use stdlib, narrow APIs and no subscription manager.

## Plan Change Log

## Review Triage Log

## Verification

- Focused parser/fetch/cache tests with TLS httptest servers, fake clocks or controlled hooks and synthetic credentials.
- `go test -tags production -count=1 -timeout=180s ./...`, `go vet -tags production ./...`.
- Frontend typecheck/Vitest; production `wails3 build` sequentially, no npm concurrency with Wails build.
- `python scripts/package_runtime.py --verify bin`.
- Isolated native GUI load/select/import and bounded real-core HTTP/SOCKS smoke when possible. No actual URL/config in checked-in outputs; distinguish external network failures from implementation failure.

Primary format sources: https://github.com/XTLS/Xray-core/discussions/716 and https://github.com/XTLS/Xray-core/discussions/4113 ; TLS fields: https://xtls.github.io/en/config/transports/tls.html .

## Review pass 1 — triage

Final counts: high 1, medium 4, low/false/maybe-false 0 (including the additional clear-ordering finding). Installed rendered workflow selects the quick lens; other lenses are not configured.

- medium / patch — decoded URI values can contain invalid UTF-8 (path=%FF), then json.Marshal replaces bytes. Reject invalid decoded values; regression must prove no silent mutation.
- medium / patch — candidate name limit is 80 runes while store accepts at most 100 bytes after trim. Align preview names with persistence so offered defaults can be saved; multibyte regression required.
- medium / patch — showModal makes main controls inert during subscription loading despite their enabled flag. Expose the existing connection action within the loading dialog, using existing toggle flow; keep normal modal behavior and cancellation. Test the reachable dialog action instead of only the obscured button's disabled flag.
- high / patch (parent inspection) — ImportSubscriptionCandidate holds previewMu across Service.ImportProfile disk/DPAPI I/O; ServiceShutdown takes the same mutex before bounded Service.Close, so slow storage bypasses the shutdown bound. Consume/reserve the chosen candidate under the mutex, release it before persistence, preserve once-only import and generation safety. A controlled delayed-save regression must show preview clear/shutdown remain responsive.

- medium / patch (parent inspection) — source edits dispatch ClearSubscriptionPreview calls concurrently, then load awaits only the latest promise. Wails uses HTTP runtime calls; an earlier clear can be scheduled after a newer preview. Serialize clear acknowledgements before preview; retain generation guards and add delayed-clear regression.

## Review patch verification

All five findings patched. Invalid UTF-8 rejected without mutation; default names trim and fit 100 bytes; reachable dialog connection controls use the existing handler; candidate IDs are consumed before persistence outside previewMu; queued clear acknowledgements are serialized. Delayed-save, multibyte/UTF-8 and delayed-clear regressions passed. Final full production Go suite/vet PASS; frontend typecheck and 30 tests PASS. Native build and private-endpoint validation follow separately. The final rendered review policy is quick/pinned, regardless of the initial planning preference for thorough.

## Completion — 2026-11-01

Final Go/vet, frontend 30 tests, production build/package and real private-endpoint validation passed. GUI loaded five candidates; one imported, HTTP/SOCKS external HTTPS and egress passed, next production probe reached Connected. Disconnect and normal GUI exit left no child/listener/runtime or owned temporary APPDATA; reopened import was empty. All five review findings resolved. Prior user-interrupted attempt remains a separate FAIL/timeout record. See [native acceptance report](backlog/evidence/subscription-acceptance-2026-11-01.md). Unrelated generated binding bytes preserved; no overall platform acceptance inferred.
