# Verification — native Windows update, 2026-11-01

Windows 11 Home amd64 10.0.26300.9711, normal user. Go 1.26.10, Wails 3.0.0; actual owned WebView2 child 156.0.4314.58. The dirty source HEAD, exact artifact/source hashes and scenario outcomes are preserved in the [execution report](../_bmad-output/backlog/evidence/autonomous-execution-2026-11-01.md), not attributed to the old clean baseline.

PASS: native production Go suite and vet, frontend typecheck/23 tests, Python packaging/provenance 25 tests, production Wails build and offline complete-package check. Native real-core loopback passed all three transports through both proxies, stopped-peer no-bypass, wrong TLS leaf pin, duplicate Connect, health recovery and cleanup. Packaged GUI startup error, import, unhealthy running status, forced parent exit, persisted profile/relaunch and orphan cleanup were inspected on this host. [Windows system detail](../_bmad-output/backlog/evidence/windows-11-system.md) and [transport detail](../_bmad-output/backlog/evidence/native-windows-loopback-2026-11-01.md) distinguish their scope.

**Current scope, changed by the user on 2026-11-01:** Windows 11 x64 with WebView2 installed. Windows 10 and missing-WebView2 tests are excluded, not passed. **Full MVP acceptance is pending:** another account, external peers/egress and remaining full native GUI/stress scenarios are not passed. See [current acceptance scope](ACCEPTANCE-SCOPE.md). Native race instrumentation is unavailable without a Windows C toolchain. Existing Linux race results below remain historical Linux evidence.

Repeat package check: `python scripts/package_runtime.py --verify bin`. Repeat native loopback using the exact opt-in commands in its linked evidence; ordinary tests intentionally skip that case. The test executable is a verification artifact and is moved outside the distributable bin after the run.

---

# Historical verification after audit fixes · 2026-10-30

## Actual environment and verified checks

Linux amd64. Official Go **1.26.10**, Wails Go module/CLI/npm runtime **3.0.0**. CLI alone built with `-tags server` to avoid GTK requirements; the Windows GUI is built with production, not server.

- Full core `go test -race -count=2 -v ./internal/core` passed after manager/config/process/store integration. Coverage includes strict routing/import, three transport-field preservation fixtures, session lifecycle, two-failure health recovery, stale-session rejection, bounded logs, slow callback and store-I/O responsiveness, canceled real helper processes, output flooding, rename, hash rejection and live/crashed instance cleanup.
- Official pinned Windows Xray archive downloaded and verified; exact executable hash verified. `python scripts/package_runtime.py bin` produced `bin/runtime/xray.exe` and required notices. The executable is not run on Linux.
- Frontend TypeScript/Vitest/Vite checks passed (20 tests); generated bindings contain 1 service, 9 methods, 4 models and `connection:state-changed`.
- Linux core vet and Windows production vet passed. Windows amd64 core test executable cross-compilation passed. Full `wails3 build` passed (production GUI, Windows resources, generated bindings, npm ci and frontend build). Cross-compilation is **not Windows execution**.
- The original audit regression file was retained in `internal/core/review_spec_v02_test.go`; all seven original scenarios passed unchanged (gofmt only), independently repeated by the review worker.
- Final frontend cancellation regression blocks profile I/O after accepted Connect and verifies Cancel startup remains usable.
- Actual pinned Xray26.10.29 Linux loopback integration passed **12 checks**: RAW+REALITY, XHTTP+REALITY, XHTTP+TLS × HTTP/SOCKS5 success and stopped-server no-bypass. See [transport evidence](TRANSPORT_TESTS.md) for exact harness, pin, commands and limitations. This is not native Windows or an external-server acceptance pass.
- Authoritative spec/audit copies and traceability are in [AUDIT-FIXES.md](AUDIT-FIXES.md).

No screenshots or visual acceptance are claimed. No personal server credentials or Windows system-network settings were used.

## Repeatable commands

```sh
go test -race -count=2 -v ./internal/core
go vet ./internal/core
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet -tags production ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/core -o /tmp/xray-core.test.exe
cd frontend && npm run typecheck && npm test && npm run build
cd .. && wails3 build
python scripts/package_runtime.py bin
```

`wails3 build` invokes npm ci, binding generation, frontend build, Windows resource generation and production GUI cross-build. Do not run another npm test/install concurrently with it. `wails3 task package` additionally packages the pinned runtime; ship the entire bin directory, not the GUI exe alone. Source Git excludes binaries.

## Required native Windows acceptance — pending, not passed

Current campaign: run on **Windows11 x64 with WebView2 installed**, as a normal user. Windows10 and missing-WebView2 cases in the original checklist were withdrawn by the user on 2026-11-01. Use Microsoft's supported WebView2 installation flow. Do not disable system security for an unsigned binary.

1. `go test -v ./internal/core` on Windows: run the native Job Object harness (`TestWindowsJobKillsChildOnParentExit`) and executable-lock test. The harness checks preflight/runtime children after abrupt parent exit. Also stress parent death during child creation, cancel during preflight/start, repeated start/stop and nested-job execution.
2. Verify application starts without elevation, actual WebView2 window renders, buttons/events/snapshot recovery work after reopening/focus; keyboard, scaling and screen-reader/visual behavior need real-device inspection.
3. Import each of RAW+REALITY, XHTTP+REALITY, XHTTP+TLS against controlled authorized test servers with pinned26.10.29; make real HTTP **and SOCKS5** requests. Confirm remote egress, no direct fallback and both listeners only127.0.0.1. Local Linux integration does not replace this platform check.
4. Verify DPAPI profile round-trip and that another Windows account cannot decrypt; inspect directory/file DACLs. Confirm normal disconnect deletes runtime JSON, force-kill GUI then relaunch removes orphan data, and a second live instance cannot delete it.
5. Disconnect during preflight, creation and probe. Confirm no xray.exe/child descendants remain after Disconnect, GUI exit or forced GUI termination; verify ports released and exactly one Wait/cleanup path.
6. Occupy each proxy port before Connect; check clear error and cleanup. Test missing/modified bundled binary rejection **before execution**. Verify official binary passes, version UI matches package and license/notice present.
7. Controlled network loss/restoration: first failed probe reports unhealthy, second Degraded, success Connected; lastCheckedAt advances, no overlapping probes or restart. A blocked control URL must not be reported as proof the server failed.
8. Search application diagnostic exports/buffer for synthetic UUID/URL/password sentinels; no raw Xray output should appear. Confirm bounded memory under output flood.

## Deliberately not supplied

Installer/signature, auto-updater, TUN, system proxy changes, kill switch and automatic server subscriptions are not MVP requirements. The build remains an unsigned experimental Windows package until target-platform acceptance above is completed.
