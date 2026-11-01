---
title: 'Verify a complete deliverable package'
type: feature
ticket: 4
created: '2026-11-01'
status: built
baseline_revision: 'ee4832faa679f632b0bda850fe6c6f288dedcd24'
route: full
route_source: auto
risk: medium
review: quick
review_source: auto
lenses_ran: [quick]
review_loop_iteration: 0
context: ['C:/Users/kp/Documents/projects/XRay-mini/AGENTS.md']
---

<frozen-after-approval reason="User approved autonomous implementation of ticket4">

## Intent

Provide a repeatable check that a deliverable contains GUI, exactly pinned runtime,
and the required license/notice. A failed/partial runtime installation must not
receive a ready-package result. Preserve official pins and offline testability.

## Boundaries & Constraints

Use Python stdlib and existing script. Never execute downloaded binaries. Keep
existing download/hash behavior and CLI compatibility. A verification mode should
check an already prepared destination without downloading anything. No installer,
signing, updater, generalized packaging framework or new dependency.

## I/O & Edge-Case Matrix

| Scenario | Input | Expected | Failure handling |
|---|---|---|---|
| Complete package | GUI, pinned exe, required notices | Check succeeds and identifies files/hashes | No network |
| Missing/tampered | Any required component absent/corrupt | Check fails, never prints ready | Fixed actionable category |
| Ancillary failure | License or NOTICE write denied after exe replacement | Install fails; incomplete output fails verification | Explicit failure and repeatable recovery |
| Retry | Write obstacle removed | Repeat creates complete package | No unknown binary execution |
| Digest mismatch | Synthetic archive/exe wrong digest | No successful installation | Existing trust boundary preserved |

</frozen-after-approval>

## Code Map

- `scripts/package_runtime.py`: downloads official pinned zip; replaces exe before
  writing license/notice. Current exception exits nonzero but there is no independent
  complete-package check. Reuse `checked`, constants and argparse/stdlib as needed.
- `licenses/NOTICE.md`, `licenses/Xray-core-LICENSE`: expected notice/legal text.
- `Taskfile.yml`: package builds GUI then runtime; parent handles any integration edit.

## Tasks & Acceptance

Worker owns only `scripts/package_runtime.py` and focused new Python packaging tests
(prefer `scripts/test_package_runtime.py`). You are not alone; preserve unrelated
edits and adjust to other agents. Do not edit Taskfile, Go, frontend, provenance
helper or integration harness. Do not commit.

- [x] Implement simple offline verification of a destination and use it before
  reporting full package success. If install-only success must be distinguished,
  make its wording unambiguous and document how full readiness is checked.
- [x] Tests use synthetic zip/files and mocked download, no real network or runtime.
- [x] Cover every matrix row, including each ancillary write failure and retry.

Acceptance: a non-builder can run the verifier on bin and see a truthful result;
synthetic failures cannot obtain readiness; official pins and legal content preserved.

## Implementation Notes

Autonomy covers plan checkpoints. Other agents own core and provenance; this plan's
tests may run independently, but never run npm/Wails builds.

## Plan Change Log

## Review Triage Log

## Verification

- `python -m unittest discover -s scripts -p test_package_runtime.py -v` — offline tests pass.
- Existing bin may be checked read-only after implementation; do not download if avoidable.

## Completion, 2026-11-01

Review finding patched: nonempty arbitrary GUI bytes could pass. Added bounded Windows amd64 GUI PE structural validation, with corrupt/truncated/wrong-kind cases. All 10 packaging tests pass; final offline bin verification passes. This does not establish GUI signature or source authenticity.
