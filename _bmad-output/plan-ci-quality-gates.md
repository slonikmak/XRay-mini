---
title: "Require native Xray loopback and an aggregate CI gate"
type: chore
ticket: ''
created: '2026-11-03'
status: built
baseline_revision: 'a24ad27ab66f77ca9dbe7c36148a042a20262630'
route: full
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user approved the proposed first iteration in chat">

## Intent

**Problem:** CI skips the existing real Xray native loopback and has no single required merge check. A successful Windows package currently does not establish that the pinned runtime passed the supported local transport scenarios.

**Approach:** Run the existing opt-in Windows loopback with the packaged pinned runtime, require evidence that all three transports actually ran and passed, and aggregate all five existing CI jobs into one stable Quality gate. Configure main to require that check when repository permissions allow it.

## Boundaries & Constraints

**Always:** Use the existing fixture and pinned runtime validation; retain the strict clean-checkout package guard and allowlisted archive. Save machine-readable loopback output on failure and success. Treat missing, skipped, failed, cancelled, or incomplete required scenarios as unsuccessful. Preserve existing checks and draft-release checks. Use ordinary GitHub runners and existing PR/manual triggers. Distinguish hosted Windows Server evidence from Windows 11 product acceptance.

**Never:** Add schedules, dedicated runners, external peers, credentials, a GUI acceptance claim, new test dependencies, or product/runtime changes. Do not weaken existing gates. No commit or push is requested. Remote merge enforcement may be configured using its stable future check name, but this does not deploy local workflow edits.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| Complete native loopback | go test2json stream containing run/pass for root and raw-reality, xhttp-reality, xhttp-tls plus package pass | Verifier succeeds | Preserve report |
| False green | Empty/truncated report, omitted root or transport, skip/fail, invalid JSON, no package completion | Verifier fails | Fixed diagnostic, nonzero exit |
| Process failure | Compile or native command fails/times out | Windows job fails | Preserve available report/logs |
| Aggregate pass | Every required dependency is success | Quality gate passes | None |
| Aggregate failure | Any dependency failed/cancelled/skipped | Quality gate fails even after dependency failure | always() at job scope |
| Archive layout | Native test binary was compiled into bin | Only existing allowlisted deliverables shipped | Keep fixture binary out of archive |

</frozen-after-approval>

## Code Map

- `.github/workflows/ci.yml`: five existing jobs; Windows builds package before tests, then archives allowlisted files. Add fixture execution sequentially and aggregate final job with all five needs.
- `internal/core/native_transport_windows_test.go`: `TestNativeWindowsLoopback`, opt-in via XRAY_MINI_NATIVE_LOOPBACK=1; executable must be adjacent to runtime/xray.exe. Existing fixture validates digest, DPAPI/ACL/process path, both proxy protocols, stopped-peer no-bypass, recovery, and cleanup. Do not change it.
- `scripts/package_demo.py`: clean checkout and explicit FILES prevent distribution of test binaries. Keep unchanged.
- `scripts/test_*.py`: stdlib unittest discovered by existing Python job; add verifier negative cases here.
- `docs/VERIFICATION.md`: current evidence and platform limits; document repeatable gate commands without claiming hosted execution.
- GitHub main currently has no branch protection (404 via gh) and no rulesets ([]). Connector cannot read administration endpoints; gh can read them. Preserve unrelated settings if state changes before a write.

## Tasks & Acceptance

**Execution:**
- [x] `scripts/check_native_loopback.py`, `scripts/test_check_native_loopback.py` -- validate actual test2json execution/completion and reject false green cases using only stdlib.
- [x] `.github/workflows/ci.yml` -- compile production fixture beside bundled runtime, run via go tool test2json, enforce report, upload report with Windows logs, add stable aggregate Quality gate with always() and five dependencies.
- [x] `docs/VERIFICATION.md` -- describe gate scope, repeatable commands and remaining Windows 11/manual acceptance.
- [ ] GitHub main protection -- pending workflow publication: require Quality gate after its first successful hosted run, then verify persisted settings. Administration access is available; enabling a missing future check now would block PRs until publication.

**Acceptance Criteria:**
- Given a freshly built Windows package, when the native gate runs with its pinned runtime, then all three transport subtests and the root complete successfully through the existing fixture and the report verifier accepts them.
- Given an incomplete or skipped loopback report, when the verifier runs, then it returns nonzero even if the test process returned zero.
- Given any required job result other than success, when the aggregate job runs, then it fails instead of being skipped.
- Given the gate configuration, when inspected, then it has no schedule or dedicated-machine requirement and the fixture executable is not in the package file allowlist.

## Implementation Notes

- Added a strict stdlib event verifier: requires package start, ordered run/pass for root and three transport subtests, and final package pass; rejects malformed, duplicate, unexpected, skipped, failed and incomplete streams with a fixed diagnostic.
- Windows workflow compiles with production tags, runs the pinned-runtime fixture sequentially after existing tests/vet, retains JSONL and logs, and checks both process and verifier exit codes. The fixture path must be absent before compilation and the generated executable is removed in `finally`, preserving full-bin distribution hygiene and the unchanged archive allowlist/clean-checkout guard.
- Stable `Quality gate` uses job-level `always()` with all five existing jobs as dependencies and explicitly accepts only their success results. Existing jobs, triggers and reusable draft-release checks are unchanged.
- No product/runtime/test-fixture changes, project dependencies, commit or push. Remote protection is left to the coordinating agent; local edits are not deployed by a settings change.

## Plan Change Log

- 2026-11-03: no intent/scope changes. Implementation additionally cleans up its own verification executable and refuses an occupied fixture path, as required for safe full-bin distribution.

## Review Triage Log

- 2026-11-03: independent quick review of the complete diff and surrounding code returned no findings (high/medium/low/false/maybe-false: 0/0/0/0/0). Local acceptance checks and matrix cases passed; remote activation remains explicitly pending publication.
## Verification

- PASS: `python -m unittest discover -s scripts -p 'test_*.py' -v`: 46 existing and new tests.
- PASS: `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/ci.yml`; 22 executions of the actual embedded aggregate script: all-success; failure/cancelled/skipped/empty result for each dependency; missing dependency.
- PASS: exact new workflow PowerShell block on Windows 11 Home x64 build 26300, Go 1.26.10; source `a24ad27ab66f77ca9dbe7c36148a042a20262630` plus local CI/verifier/docs changes; production core test executable and existing verified Xray 26.10.31 (`d5b558191cccb1e92a5772b68b8cab67717b26f634b0f15153215f65ae982a7e`). Root 12.22s; three subtests and package pass. Report/logs: `artifacts/native-loopback*`. Generated test executable absent after completion.
- PASS: deliberately disabled opt-in produces process exit 0 and SKIP; verifier returns 1. Existing fixture sentinel is refused without overwrite/deletion. Initial local command trial exposed PowerShell splitting unquoted dotted test arguments; quoting them fixed the command, and the failed report was rejected.
- PASS: `python scripts/package_runtime.py --verify bin` verifies the existing complete package and pinned bytes. Existing Python archive allowlist test confirms extra test executables are excluded; packaging code is unchanged. A new clean-checkout archive was not created from this intentionally dirty working tree.
- Main protection and rulesets were inspected via gh: none configured, administration access available. Do not require a check absent from the published workflow. Hosted execution and activation follow publication; no hosted Windows Server CI result or complete Windows 11 product acceptance is claimed.
- Final local handoff: the build workflow records the reviewed implementation in a scoped local commit. No push or remote settings changes are included.
