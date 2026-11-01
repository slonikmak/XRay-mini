# Q4 — health and diagnostics

6 files read through git at 910c353; no execution/history.

- **Load-bearing:** actual HTTPS GET through 127.0.0.1:10809 to generate_204, only204 succeeds; redirects rejected, dial3s/TLS5s/client5s. `internal/core/process.go:106-120 (probeNetwork) @ 910c353`.
- Probes run immediately, serially, then on30s ticker with5s context. Cleanup cancels and joins worker. `internal/core/service.go:428-455 (run) @ 910c353`.
- **Load-bearing:** initial failure leaves Connecting/unhealthy; first failure after success leaves Connected/unhealthy; second consecutive failure Degraded; success resets. `internal/core/service.go:319-334 (loop) @ 910c353`.
- Internal event names match ConfigValidated, ProcessStarted, ProxyReady, HealthFailed, ProcessExited, StopCompleted, plus Diagnostic; timestamps/session IDs; stale/canceled events gated. `internal/core/service.go:286-334 (loop) @ 910c353`.
- **Load-bearing:** output readers use4096-byte buffers and retain only stdout/stderr-presence flags; no child text. `internal/core/process.go:49-87 (outputDrain) @ 910c353`.
- Diagnostics polled1s and teardown, once/category/session; five-category allowlist. Logs newest200, GetRecentLogs returns bounded copies; callbacks use coalescing queue. `internal/core/service.go:198-227 (loop) @ 910c353`; `internal/core/service.go:308-315 (loop) @ 910c353`; `internal/core/service.go:413-426 (run) @ 910c353`.
- Scoped search finds no auto restart, startup health-failure deadline, alternative endpoint or persistent diagnostic file. Tests define serial probes, two-failure recovery and redaction, not observed passes.
