# Dependency review and release gate

## Known signal, not a completed security verdict

The retained build log for the 1 November 2026 recovery change reported `3 vulnerabilities (1 moderate, 2 critical)`. It did not identify the advisory paths or establish whether the shipped renderer was affected. The log is [here](../_bmad-output/backlog/evidence/concurrency-recovery-2026-11-01-build.log).

Do not infer either "two exploitable vulnerabilities in the EXE" or "only harmless development packages" from that line. No dependency upgrade or waiver is included in this portfolio-packaging change. Direct registry access was unavailable in the authoring environment; the first hosted CI run subsequently obtained the concrete reports below.

## First hosted result — 1 November 2026

CI run `41357353610`, source commit `5014da4937eba936e20277778c4a798efc5c37d2`, obtained a full-tree result of **two critical and one moderate affected dependency nodes**. The separate production-only report has zero findings. These counts are package-level entries, not three independently exploitable flaws in the application.

The affected path is the development test toolchain: `vitest` inherits critical findings from `tinypool`; `@vitest/mocker` contributes moderate findings. The report references Tinypool [GHSA-5gmw-xhrv-c9v3](https://github.com/advisories/GHSA-5gmw-xhrv-c9v3) and [GHSA-85c8-ppgw-ccpr](https://github.com/advisories/GHSA-85c8-ppgw-ccpr), plus mocker [GHSA-82fw-gwwq-j7x9](https://github.com/advisories/GHSA-82fw-gwwq-j7x9) and [GHSA-qxg6-89mg-rqv8](https://github.com/advisories/GHSA-qxg6-89mg-rqv8). No exploit or reachability test was performed here.

npm suggests a **SemVer-major Vitest upgrade**, not a verified drop-in change. Production-only success narrows the dependency scope; it does not justify disabling the full-tree gate or declaring the build environment safe. The draft-release gate remains blocked until a reviewed upgrade passes tests, build and both audits.

The exact full/prod reports and lockfile identity are retained in [dependency-audit-5014da4.json](evidence/dependency-audit-5014da4.json). Hosted Go race/vet, frontend and Python checks have passed for that source; Windows packaging is a separate job and its status must be read from Actions.

## Repeatable audit

```powershell
python scripts/check_npm_audit.py
```

The script reads the committed frontend lockfile and runs separate full-tree and production-only npm audits. It preserves JSON reports, stderr, command exit codes, the lockfile hash and a summary under `artifacts/dependency-audit/`. It does not install packages, run lifecycle scripts or modify the lockfile. Audit submits dependency metadata to the configured npm registry, not application profiles or subscription sources.

Exit 0 means neither report contains high/critical findings. Exit 1 means the release gate has findings. Exit 2 means an audit could not be established (missing tool, timeout, malformed response, registry error or incomplete data). An unavailable audit is not a zero-finding audit. Moderate/low findings remain visible for review.

CI uploads the reports even on failure. The draft-release workflow requires every CI job, including this gate, to succeed. A Windows package artifact can exist while the security job is red; artifact existence is not release approval.

## Resolve a finding

Read the actual advisory, affected range and dependency path. Distinguish build/test-tool exposure from code embedded in the shipped renderer. Update the narrowest compatible dependency and regenerate package-lock with npm; then rerun tests, build and both audits. Do not edit integrity hashes manually or use `npm audit fix --force` without reviewing breaking changes.

This policy has no automatic allowlist. Any future risk acceptance must explicitly name the advisory, affected version, exposure reasoning, owner and expiry rather than silently ignoring the check. Windows/API/Go/runtime security needs separate review; npm audit is not a comprehensive security guarantee.
