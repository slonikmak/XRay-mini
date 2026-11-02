---
title: 'Simplify audit findings without changing product behavior'
type: 'refactor'
ticket: ''
created: '2026-11-02'
status: 'built'
baseline_revision: 'd12e3d528681c5b65c0355ded5cd9f35d8073f5a'
route: 'oneshot'
route_source: 'auto'
risk: 'medium'
review: 'quick'
review_source: 'pinned'
lenses_ran: ['quick']
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A few duplicate checks, types and unused paths make the current implementation harder to follow. The earlier audit also proposed removing synchronization and tests without enough benefit.

**Approach:** Apply the selected mechanical reductions as one loose project task in the current branch; preserve the existing untracked `design/` directory. Keep both store mutexes, Wails ID tests and CSS unchanged.

- In `internal/core/subscription.go`, retain `PrepareConfig` validation and remove the immediately preceding duplicate preview validation.
- In `internal/core/service.go`, remove unused `Service.PreviewImport` and test-only `GetLogs`; retain the diagnostic allowlist but remove its three categories with no runtime producer. Update service tests to use `GetRecentLogs(200)` and real stdout/stderr categories; ensure the allowlist test checks a secret after the log buffer is full.
- Remove unused Go archive/release constants in `internal/core/runtime.go`; retain the executable pin and version and update `docs/ARCHITECTURE.md` to point at the package script and notices for archive provenance.
- Reuse generated Wails model types in `frontend/src/bridge.ts`, preserving its normalized `ImportPreview` and `DesktopAPI`; remove only redundant array fallbacks in `frontend/src/App.tsx`. Preserve null normalization at the bridge, ordering guards, independent diagnostics refresh and generated bindings.

</frozen-after-approval>

## Implementation Notes

Oneshot: small, mechanical changes across a few existing modules; the security boundary and diagnostics test require focused verification.

- Removed the redundant subscription preview call, unused service wrappers and dead Go metadata; the package script and notice retain archive provenance.
- Restricted diagnostic categories to actual producers. Strengthened the existing bounded-buffer test by sending rejected messages after the buffer is filled and waiting for a subsequent allowed marker.
- Reused generated frontend models with an explicitly normalized preview type and kept bridge normalization and UI ordering safeguards.
- Linked the current FR-02, FR-07 and ADR-008 cross-check from `docs/BROWNFIELD.md`, preserving the historical snapshot.
- FR-02: `parseVLESSLink` still calls `PrepareConfig`, which runs `selectOutbound`, duplicate-key rejection and dependency checks. `TestSubscriptionFormatsAndMapping` and `TestSubscriptionRejectsAmbiguityAndUnsupportedInput` passed.
- FR-07: `outputDrain.Diagnostics` emits only the two retained categories; `Service.loop` still filters them and bounds the log. `TestDiagnosticAllowlistAndBound` checks that a secret and removed categories are rejected after the buffer fills.
- ADR-008: `App` still subscribes before reading the status and rejects older revisions. Its tests for synchronous events, late snapshots and older events passed; independent diagnostics refresh remains in place.

## Plan Change Log

## Review Triage Log

Quick review: no code findings. Follow-up: low, patched — the Brownfield link initially claimed a requirement cross-check without enough traceable detail; the three rule-specific notes above now identify the relevant code and tests.

## Verification

- `go test -tags production -count=1 -timeout=2m ./internal/core` — Windows core tests pass.
- `npm run typecheck` and `npm test` in `frontend` — generated model compatibility and UI/bridge regressions pass.
- `git diff --check` and compare hashes of the two pre-existing `design/` files — no whitespace errors or changes to user files.

Observed on Windows amd64 with Go 1.26.10, on the working tree based on the recorded revision: `go test -tags production -count=1 -timeout=2m ./...` and `go vet -tags production ./...` passed; frontend typecheck, production build and all 55 frontend tests passed; 10 package-runtime tests passed. The diagnostic test passed 10 repetitions and, using a temporary Go overlay, failed as expected when the allowlist was bypassed. Both pre-existing `design/` files retained their SHA-256 hashes; `git diff --check` passed. These checks do not establish complete Windows GUI or transport acceptance.
