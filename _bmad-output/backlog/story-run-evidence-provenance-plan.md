---
title: 'Identify newly executed verification evidence'
type: feature
ticket: 5
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

<frozen-after-approval reason="User approved autonomous implementation of ticket5">

## Intent

New integration and native results must identify actual application revision,
dirty state, platform/tool versions, relevant artifact digests, command/scope,
scenario outcomes and cleanup. Preserve historical Linux JSON unchanged. Record
failed and interrupted runs as failures with their available cleanup outcome.

## Boundaries & Constraints

Use stdlib, one small reusable evidence helper if worthwhile. Do not build a generic
test runner. No credentials, config bodies, unrestricted environment/diff dumps or
untrusted raw error text in evidence. An engine commit is never the app commit.
Missing versions are explicit unavailable/not applicable. Never emit PASS for SKIP.
No platform or network acceptance execution in this task.

## I/O & Edge-Case Matrix

| Scenario | Input | Expected | Failure handling |
|---|---|---|---|
| Clean git checkout | Real full HEAD | Exact app revision and clean flag | Separate from engine metadata |
| Dirty checkout | Tracked/untracked source change | Dirty recorded; no clean-snapshot claim | Do not expose diff contents |
| Tool/artifact absent | Missing tool/WebView or package | Explicit unavailable/NA | Scope remains honest |
| Failed/interrupted harness | Nonzero request/fixture failure | FAIL and cleanup evidence saved | Fixed error category, no secrets |
| Prior result | Existing historical JSON | Bytes unchanged | New result goes to another path |

</frozen-after-approval>

## Code Map

- `integration/transports.py`: Linux-only fixture with strong pin, PrepareConfig
  adapter, positive/no-bypass checks and cleanup; current result write occurs only
  after successful body and misses failures. Keep protocol checks/pins unchanged.
- `integration/results-linux-amd64.json`: historical evidence; never rewrite.
- `scripts/run_evidence.py` (new if useful): narrowly collect safe run identity and
  artifact hashes; suitable for parent native evidence, not another test framework.

## Tasks & Acceptance

Worker owns `integration/transports.py`, optional `scripts/run_evidence.py`, and focused
new Python provenance tests (prefer `scripts/test_run_evidence.py`). You are not alone;
preserve others' changes. Do not edit packaging, Go/frontend or historical result.
Do not commit, launch network tests, download core or run npm/Wails.

- [x] Implement safe identity/schema with exact full canonical git revision.
- [x] Integrate new metadata into future Linux fixture runs and save failure/cleanup
  results without changing historical output. Avoid secret-bearing command arguments.
- [x] Provide a small callable API for parent acceptance records and explain it.
- [x] Cover matrix rows in isolated temp repositories/synthetic failures.

Acceptance: future runs have usable provenance and failures survive; existing
historical JSON's hash remains unchanged; no new PASS is implied by helper tests.

## Implementation Notes

Autonomy covers plan checkpoints. Core and packaging are owned by other agents.
The parent may later extend the harness for a separate native setup task after this
worker finishes; do not add Windows transport behavior in this ticket.

## Plan Change Log

## Review Triage Log

## Verification

- `python -m unittest discover -s scripts -p test_run_evidence.py -v` — isolated tests pass.
- Compile-check changed Python files; compare historical JSON hash before/after.

## Completion, 2026-11-01

Review findings patched: Go version probes now run at the module root; scenario FAIL dominates requested SKIP. Parent added actual platform.version/build. All 15 provenance tests pass. Historical Linux result bytes unchanged; no transport rerun is claimed by these unit tests.
