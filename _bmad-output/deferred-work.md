
- source_plan: `plan-concurrency-stop-recovery.md`
  summary: Add explicit recovery for a failed native preflight-child termination.
  evidence: Pre-existing execRunner.Test waits synchronously; startContained cancellation makes one initial Kill attempt and ignores its error. Current working-child retry does not control that preflight child. A retained/failed preflight requires a separately supervised completion/retry contract and native failure injection; Close remains bounded and does not release unconfirmed ownership.

- source_plan: `C:/Users/kp/Documents/projects/XRay-mini/_bmad-output/plan-tray-connection-icon.md`
  summary: Verify Wails tray recovery when Explorer restarts during a status icon update.
  evidence: Wails may release the previous HICON before a failed shell update. An isolated Windows 11 probe successfully re-added and updated the icon after simulating shell removal; a real Explorer restart was not exercised.

- source_plan: `C:/Users/kp/Documents/projects/XRay-mini/_bmad-output/plan-tray-connection-icon.md`
  summary: Verify an icon state change while Wails tray initialization is deliberately delayed.
  evidence: Wails schedules window and tray startup concurrently; the initial icon uses the pre-tray service snapshot, and normal isolated launch registered the tray, but the later-update scheduling interleaving was not reproduced.
