# Dependency review and release gate

## Known signal, not a completed security verdict

The retained build log for the 1 November 2026 recovery change reported `3 vulnerabilities (1 moderate, 2 critical)`. It did not identify the advisory paths or establish whether the shipped renderer was affected. The log is [here](../_bmad-output/backlog/evidence/concurrency-recovery-2026-11-01-build.log).

Do not infer either "two exploitable vulnerabilities in the EXE" or "only harmless development packages" from that line. No dependency upgrade or waiver is included in this portfolio-packaging change. Direct registry access was unavailable in the authoring environment, so there is no claimed fresh npm result.

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
