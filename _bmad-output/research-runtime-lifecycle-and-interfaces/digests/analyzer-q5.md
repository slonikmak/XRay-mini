# Q5 — UI/API

9 files read at 910c353; no execution/history.

- **Load-bearing:** refresh generation/alive gates prevent obsolete request updates, independent revision gates reject older status; effect subscribes before snapshot and resyncs focus/visibility. `frontend/src/App.tsx:48-75 (refresh) @ 910c353`; `frontend/src/App.tsx:107-112 (toggleConnection) @ 910c353`.
- **Load-bearing:** perform releases command lock before unawaited forced refresh; Connect replies after scheduling worker, before IO. `frontend/src/App.tsx:77-88 (perform) @ 910c353`; `internal/core/service.go:234-282 (loop) @ 910c353`.
- Import is paste-based, preview acknowledgment resets on edit; successful import and cancel clear input. Rename/delete through perform, active-profile delete guarded in UI/backend. `frontend/src/App.tsx:89-105 (importProfile) @ 910c353`; `frontend/src/App.tsx:163-176 (App) @ 910c353`.
- UI shows process/health separately, readiness requires running+healthy, version/timestamp, last100 logs. Core retains200. `frontend/src/App.tsx:119-162 (App) @ 910c353`.
- **Load-bearing:** native host required; browser mode disables operations. Startup backend errors reject ListProfiles; refresh Promise.all then discards fallback GetStatus Error and shows generic sync warning/Retry. `frontend/src/bridge.ts:41-45 (desktop) @ 910c353`; `app.go:21-96 (ServiceStartup) @ 910c353`; `frontend/src/App.tsx:48-60 (refresh) @ 910c353`.
- Diagnostic-only events update log snapshot without publishing status; new log rows await next refresh trigger. `internal/core/service.go:308-316 (loop) @ 910c353`.
- No polling, frontend timeout/cancellation or browser storage found in App/bridge/generated API scoped search.
