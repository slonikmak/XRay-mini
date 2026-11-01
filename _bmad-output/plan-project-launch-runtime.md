---
title: 'Launch the project with its bundled runtime'
type: bugfix
created: '2026-11-01'
status: built
baseline_revision: 'eb31b6c1dba589038c39afbc82f5994cbc87d587'
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="Fixing the reported Connect failure under the user's continuing autonomous authorization">

## Intent

The user's actual window runs from a Go temporary build directory under go run, with no adjacent runtime/xray.exe. Native inspection confirms the running executable path and absent adjacent runtime; the packaged bin GUI/runtime are present. Earlier checks launched bin instead and missed the user's entry point. Fix the project launch path in Zed so the Go main runnable uses the existing Wails development task and bin output. Classify a missing bundled runtime separately, with a fixed actionable message and visible error heading; do not imply the selected profile is invalid.

Always preserve adjacent-only runtime resolution, exact digest validation, native Job containment, single writer and secret-free errors. Never add cwd/PATH fallback, arbitrary binary override, copying runtime into Go caches, startup network downloads or global editor changes. Keep existing Taskfile/dev configuration as the source of launch behavior. Existing unrelated dirty files are preserved.

Given the project Go main runnable, its workspace task must select wails3 task dev, whose app executable remains in bin beside the pinned runtime. Given a missing runtime, Connect must finish safely with a missing-runtime explanation including supported launch commands, without raw path or extra child. Given another preflight failure, the existing secret-safe category remains. Given Error, the headline must show failure rather than Ready. Verify actual old go-run reproduction and the corrected native launch, including Connect/proxy traffic, not just a different binary launched without explaining the change.

</frozen-after-approval>

## Implementation Notes

Small direct change: runtime.go/process_windows.go/service.go classification, focused native/service tests, App.tsx error heading and existing regression, workspace .zed/tasks.json and README. No new public API/dependency. Prior autonomous scope covers routine dirty-tree/checkpoint decisions. Zed workspace runnable tags override language defaults per https://zed.dev/docs/tasks ; Go main uses go-main per upstream zed Go language tasks. No UI automation of terminal commands: verify the task's command using CLI and the GUI using native UI tools.

## Review Triage Log

## Verification

Native missing-runtime classification; service cleanup/no-child and unknown-error sanitization; frontend Error heading; core/production tests and vet, frontend typecheck/tests, production Wails build and package verifier. Run configured Wails dev command through CLI, inspect actual exe path and adjacent runtime, then Connect the existing profile and check real proxy requests. No secrets or profile-body copies in repository artifacts.

## Completion

Quick review: no findings. Production Go/vet, frontend 30 tests, production build and preserved production-package verification passed. Actual go run negative case showed the precise runtime error; exact configured Wails task launched bin with adjacent runtime. After user interaction, read-only inspection and real HTTP/SOCKS requests confirmed Connected and 204 responses. User app/connection intentionally remain active. See [evidence](backlog/evidence/project-launch-runtime-2026-11-01.md). The editor gutter itself was not automated; its configured CLI task was tested. Transient first health-check readiness is a separate observation, not changed here.
