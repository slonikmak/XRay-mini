---
title: 'Wait for the proxy before the initial health request'
type: bugfix
created: '2026-11-01'
status: built
baseline_revision: '0bce7ee87267b16a53f24c995b47daedbcb3e5c4'
route: full
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="User explicitly requested fixing the startup check after the BMAD invocation; autonomous execution remains authorized">

## Intent

Creating the Xray process does not mean its HTTP listener is already open. The health goroutine immediately starts its request, so connection refusal can become Check failed until the next 30-second tick. Fix this demonstrated reachable startup race without hiding actual HTTP/TLS/upstream failures. Keep Connecting/checking while the local proxy becomes ready, then issue the existing real control request.

## Boundaries & Constraints

Always retain the total five-second health-check budget (local readiness plus HTTP), cancellation, serial checks, 30-second cadence and two-failure Degraded policy. A live PID or open port must never produce Connected without the successful control response. Bound readiness polling and release every timer/connection. Keep default TLS verification, fixed loopback proxy and control URL, no direct fallback or HTTP-request retries, no new dependencies/public APIs. Unknown failures remain safe; raw errors/addresses/configs must not be logged. Existing unrelated bindings/tooling and user profile files must be preserved.

## I/O & Edge-Case Matrix

| Scenario | Input/state | Expected behavior | Error handling |
|---|---|---|---|
| Delayed listener | First local dial fails, proxy opens within budget | Same first health check waits and returns healthy after actual204 | No interim false failure or 30s wait |
| Port only | Listener accepts, control response pending | No healthy result before HTTP completion | Cancel/timeout remains effective |
| HTTP failure | Ready proxy returns200/503 or closes request | Check fails, exactly one request | No concealment via retries |
| Never ready | Local port stays closed | Check fails within total5s | No endless goroutine/attempt loop |
| Cancellation | Cancel during local wait or HTTP | Prompt false result, no leftovers | No stale success |
| Normal operation | Listener ready and204 | One request succeeds immediately | Existing manager policy unchanged |

</frozen-after-approval>

## Code Map

- `internal/core/process.go:probeNetwork`: currently builds new HTTP Transport with an ordinary Dialer; refusal fails immediately. Keep public wrapper with fixed production constants. A private helper may accept proxy/control URLs for isolated tests. Bound local TCP establishment within the request context; no extra HTTP request or port-only health success.
- `internal/core/service.go`: existing health goroutine creates5s context and serial30s ticker; Service.loop owns transitions. No manager/state change needed.
- New `internal/core/process_test.go` (or focused equivalent): real ephemeral loopback listener delayed until after the first refused dial, HTTP proxy response gating, HTTP failure, cancellation and deadlines. Do not occupy fixed product ports in ordinary tests.
- `service_test.go`: existing cancellation, stale results and two-failure/recovery coverage; preserve and run. Native transport fixture remains opt-in.

## Tasks & Acceptance

Worker owns only process.go and focused process tests. It is not alone: preserve others' work and do not change service/frontend/storage, bindings, docs or commit. Use stdlib/minimal code. Total estimate with meaningful tests exceeds100 lines, hence full route. Parent owns build, evidence, documentation, optional actual-profile smoke and commit.

- [x] Add bounded cancellable local readiness to the real probe path; a successful TCP connection must then carry the single actual HTTP request.
- [x] Cover every matrix row with isolated runnable tests; no external network or secret data. Ensure delayed-listener case fails with the old immediate-dial behavior.
- [x] Run focused tests, native full production Go suite and vet; report exact commands/outcomes and remaining limits. Do not invoke BMAD entry again or run Wails/npm builds.

Acceptance: Given a delayed proxy listener, when one probe starts before it then no failure is emitted merely for early refusal and its actual204 completes that same probe. Given HTTP failure or missing readiness, when the deadline/cancel arrives then result is false within budget with no HTTP retry. Given pending response, no Connected/healthy may be inferred from TCP alone. Parent will run a bounded native real-core first-health smoke if runtime/ports are available.

## Implementation Notes

The actual first failing request's OS error was not retained, so historical screenshots alone do not prove its cause. The immediate-refusal path is reproducible and the regression must prove it. If filtering native refusal codes, Windows WinSock10061 differs from the synthetic syscall.ECONNREFUSED enum; do not accidentally test only Unix behavior. A bounded readiness dial against the fixed local proxy can remain platform-independent. User approved this fix and prior autonomous handling of dirty-tree/plan gates; no initiative is active, this is loose project work.

## Plan Change Log

## Review Triage Log

## Verification

Focused real-loopback regressions; `go test -tags production -count=1 -timeout=180s ./...`; `go vet -tags production ./...`. Parent: native pinned Service first-health smoke, production Wails rebuild, offline package check, preserve unrelated generated bytes. No Windows10/missing-WebView2 acceptance claims, no user profile edits or secret-bearing outputs.

## Completion — 2026-11-01

Quick review: no findings. All matrix rows have passing real-loopback tests; old immediate-dial overlay fails the delayed-listener regression as expected. Production Go suite/vet passed. Parent native default-probe test passed the first check in all three real Xray sessions (3037/3128/1401ms from Connect); each disconnected and cleaned its runtime/store. Production Wails build and offline package verification passed (file completeness and pinned runtime hash). No frontend/API/state-machine change; existing unrelated bindings preserved. [Evidence](backlog/evidence/proxy-readiness-2026-11-01.md).
