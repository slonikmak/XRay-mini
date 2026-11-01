
- source_plan: `plan-concurrency-stop-recovery.md`
  summary: Add explicit recovery for a failed native preflight-child termination.
  evidence: Pre-existing execRunner.Test waits synchronously; startContained cancellation makes one initial Kill attempt and ignores its error. Current working-child retry does not control that preflight child. A retained/failed preflight requires a separately supervised completion/retry contract and native failure injection; Close remains bounded and does not release unconfirmed ownership.
