---
title: 'Native Windows real-core loopback verification'
type: feature
ticket: 12
created: '2026-11-01'
status: built
baseline_revision: 'ee4832faa679f632b0bda850fe6c6f288dedcd24'
route: full
route_source: auto
risk: high
review: quick
review_source: auto
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="Autonomous preparation of available native acceptance fixtures">

## Intent

Add and run an opt-in native Windows core test using real pinned Xray, production
Service imports/session worker and native supervisor, fixed10808/10809 and local
controlled VLESS peers for RAW/REALITY, XHTTP/REALITY, XHTTP/TLS. This is explicitly
partial Windows11 evidence, not Windows10/GUI/external-egress acceptance.

## Boundaries & Constraints

Only loopback listeners, ephemeral synthetic keys/UUID/cert and owned temporary data.
No external endpoints, OS proxy/routes/CA edits, admin, unknown downloads, dependency
or allowInsecure. Verify pinned runtime before fixture commands. Preserve production
code; test-only hooks may point probe to real local204 through HTTP proxy and shorten
health interval, with scope documented. Use Service.New/Import/Connect and its real
execRunner; an observing wrapper may count actual launches without faking them.
Call Wait once on each fixture child and join every helper. Refuse busy product ports
without touching another owner's process. Emit no private config/keys in test output.

## I/O & Edge-Case Matrix

| Scenario | State/input | Expected | Failure handling |
|---|---|---|---|
| Three transports | Real peer + actual imported client | HTTP/SOCKS target response,6 successes | Bound every request |
| Probe not yet complete | Worker running, test-only probe gate | Connecting not Connected | Gate release in teardown |
| Stopped peer | Target remains healthy | Both proxies fail, no new target hits, Degraded | No direct fallback/client restart |
| Peer restored | Same peer configuration | Real probe restores Connected, client launch count stays1 | Deterministic bounded wait |
| Wrong TLS pin | Live TLS peer and target | TLS requests fail (pin enforced) | No CA trust change |
| Cleanup/real exit | Kill working child or Disconnect | Correct Error/Disconnected after cleanup, no leftovers | Finally handles all outcomes |

</frozen-after-approval>

## Code Map

- `internal/core/service.go` real worker, recent cleanup/Close fixes; no edits owned here.
- `internal/core/process.go` execRunner verifies/contains all invocations; probeNetwork
  uses fixed external URL. Substitute only s.probe for isolated local HTTP204 test.
- `internal/core/runtime.go` only adjacent runtime/xray.exe and exact digest allowed.
- `integration/transports.py` existing Linux fixture protocol fields/freedom finalRules;
  read as example, do not edit or execute Linux artifact on Windows.
- `internal/core/config.go` preserves supported stream settings including leaf pin.
- Test executable can be built into bin next to existing verified runtime so the
  production path policy remains intact.

## Tasks & Acceptance

Worker owns only a new `internal/core/native_transport_windows_test.go` and a scoped
new evidence note if useful. You are not alone: preserve other edits, do not change
production Go, existing test files, frontend, scripts or ticket files. No commit.

- [x] Use Go stdlib certificate/X25519 generation and net/http explicit HTTP/SOCKS5
  proxies (Transport supports socks5; avoid hand-written protocol if stdlib works).
- [x] Server fixtures may call real verified execRunner with direct server configs.
  Client goes through real Service. TLS fixture uses exact generated leaf SHA256 pin
  plus explicit localhost serverName; add wrong-pin negative proof.
- [x] Keep test opt-in with an explicit environment variable; ordinary core tests
  must not commandeer fixed ports or fail because packaged runtime is absent.
- [x] Build native test binary adjacent to runtime, execute matrix on this host,
  record exact command, cases, limits and cleanup. Return actual counts/outcomes.

Pinned TLS semantics verified from official v26.10.29 source and documentation:
https://github.com/XTLS/Xray-core/blob/v26.10.29/transport/internet/tls/config.go
https://xtls.github.io/en/config/transports/tls.html
Leaf pin performs exact certificate verification (no allowInsecure); do not describe
it as ordinary OS trust-chain acceptance. No new product-level TLS policy is added.

## Implementation Notes

User authorized autonomous execution including needed test setup. The Windows10
environment, another account, missing-WebView2 environment and external authorized
servers remain unavailable; don't fabricate their evidence. Parent packages GUI
separately and preserves pre-existing generated bindings.

## Plan Change Log

## Review Triage Log

## Verification

- `go test -c ./internal/core -o bin/core-native.test.exe`
- Set the chosen opt-in variable only for child execution, then
  `bin/core-native.test.exe -test.v -test.run TestNativeWindows -test.timeout 180s`.
- Confirm actual runtime pin and release of fixed listeners/fixture processes.

## Completion, 2026-11-01

Review findings patched: failed peer stop can be retried and owned material is retained if exit cannot be confirmed; wrong-pin negative checks live peer before/after and correct-pin traffic afterward. Final real-core matrix passes in 18.14s, including three duplicate-Connect checks. See evidence/native-windows-loopback-2026-11-01.md.
