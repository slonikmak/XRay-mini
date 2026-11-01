# Q1 — profiles and configuration

Analyzer read 13 files at immutable 910c353; no execution or history.

- **Load-bearing:** JSON object import selects exactly one VLESS and requires streamSettings; the UI previews discarded sections and requires acknowledgment. `internal/core/config.go:45-82 (selectOutbound) @ 910c353`; `frontend/src/App.tsx:90-105 (importProfile) @ 910c353`.
- **Load-bearing:** stored config is rebuilt on import, with one selected-vless outbound and fixed loopback SOCKS/HTTP inbounds plus explicit routing. `internal/core/store.go:112-127 (add) @ 910c353`; `internal/core/config.go:29-42 (PrepareConfig) @ 910c353`.
- **Load-bearing:** Windows encrypts complete profile records using user DPAPI, not machine-wide protection. `internal/core/store.go:123-146 (add) @ 910c353`; `internal/core/protect_windows.go:22-43 (crypt) @ 910c353`.
- **Load-bearing:** run decrypts/rebuilds config, writes private runtime/config.json, performs bounded preflight then starts Xray. `internal/core/service.go:343-405 (run) @ 910c353`; `internal/core/process.go:24-46 (Test) @ 910c353`.
- **Load-bearing:** exclusive store lock precedes orphan cleanup; current-user and SYSTEM directory ACLs are inherited; session directory removed before StopCompleted. `internal/core/store.go:32-62 (newStore) @ 910c353`; `internal/core/permissions_windows.go:15-48 (secureDirectory) @ 910c353`; `internal/core/service.go:343-354 (run) @ 910c353`.

Rejects >2MiB JSON, duplicates ignoring case, >64 nesting, trailing data, missing/ambiguous VLESS, unsupported field names/network/security and external dependencies/local resource paths. `internal/core/config.go:50-123 (selectOutbound) @ 910c353`; `internal/core/config.go:152-180 (unsafeDependency) @ 910c353`; `internal/core/config.go:185-240 (rejectDuplicateKeys) @ 910c353`.

List decrypts every record but returns metadata only; one unreadable record fails listing. Rename preserves ID/config/creation time, reencrypting through a synced temporary file. `internal/core/store.go:91-110 (list) @ 910c353`; `internal/core/store.go:170-204 (rename) @ 910c353`.

Names are limited to 100 bytes in store and 80 characters in UI. No file reader, subscription parser, profile-content getter or update-config operation matched scoped search; import is paste-based. Protocol validity is deferred to Xray preflight. Static reads do not establish native DPAPI/ACL or successful execution.
