# Q6 — tests and saved results

15 files read at 910c353; no execution/history. Locator overcount of App cases corrected by analyzer:16, not17.

- **Load-bearing:** frontend20 definitions:16 App and4 bridge; jsdom/mock DesktopAPI and Wails runtime. `frontend/src/App.test.tsx:1-198 (describe) @ 910c353`; `frontend/src/bridge.test.ts:1-46 (describe) @ 910c353`.
- Core config/service/store/runtime/process/review tests define import/routing, session races, serial health, secret filtering, exclusive store/recovery, digest rejection and helper-process cleanup. Service uses fake runner/probe; process helper opens no sockets. `internal/core/service_test.go:17-65 (fakeChild) @ 910c353`; `internal/core/process_test.go:13-93 (TestMain) @ 910c353`.
- Windows-only tests define crash-parent cleanup for preflight/runtime helpers and executable replacement locks; not actual Xray transport. `internal/core/process_windows_test.go:18-103 (TestWindowsJobKillsChildOnParentExit) @ 910c353`.
- **Load-bearing:** Linux harness invokes exported PrepareConfig, changes inbound ports to ephemeral, creates loopback client/server/TLS fixtures and local CA, checks HTTP/SOCKS success then stopped-server no-bypass and process reaping. It bypasses production Windows supervisor. `integration/transports.py:119-195 (main) @ 910c353`; `integration/prepareconfig/main.go:11-21 (main) @ 910c353`.
- **Load-bearing:** saved2026-10-30 report has12PASS, cleanup true and engine pins. No application commit, app version, unit/frontend results or Windows result fields; engine0afec66 is not app commit. `integration/results-linux-amd64.json:2-71 (root) @ 910c353`.
