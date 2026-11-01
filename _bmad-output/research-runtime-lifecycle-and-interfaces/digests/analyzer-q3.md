# Q3 — processes and delivery

15 source/config/test files read at 910c353; no execution/history; file budget exhausted.

- **Load-bearing:** both preflight and working invocation pass startVerifiedContained: executable lock, validate, create child, unlock. Only adjacent Windows amd64 runtime/xray.exe with pinned digest accepted. `internal/core/process.go:35-46 (startVerifiedContained) @ 910c353`; `internal/core/runtime.go:13-58 (XrayVersion) @ 910c353`; `internal/core/process_windows.go:226-235 (lockExecutable) @ 910c353`.
- **Load-bearing:** Windows start creates kill-on-close Job Object, injects it via JOB_LIST during CreateProcessW, permits only explicit stdin/stdout/stderr inheritance; job handle not inherited. No assign-after-create fallback. `internal/core/process_windows.go:99-208 (startContained) @ 910c353`.
- **Load-bearing:** windowsChild.Wait sync.Once waits OS process, closes job/process handles and joins drains; direct-process Kill errors ignored by caller, wait is INFINITE. `internal/core/process_windows.go:56-95 (Wait) @ 910c353`.
- **Load-bearing:** package task builds GUI then downloads pinned official Windows Xray archive, checks archive/exe hashes, replaces executable and copies notices without executing download. `Taskfile.yml:8-46 (bindings) @ 910c353`; `scripts/package_runtime.py:14-36 (VERSION) @ 910c353`.
- Application startup selects UserConfigDir/XrayMini; shutdown closes core. Main registers Wails service and embeds frontend. Manifest requests asInvoker; metadata says unsigned prototype. `app.go:21-45 (ServiceStartup) @ 910c353`; `main.go:10-27 (main) @ 910c353`; `build/windows/wails.exe.manifest:15-20 (requestedPrivileges) @ 910c353`.
- Installer/WebView runtime delivery/signing tasks not found in assigned paths; no TerminateJobObject call. Existing Windows test creates helper binary children directly (bypassing production hash gate), then tests parent exit and executable file lock. `internal/core/process_windows_test.go:18-102 (TestWindowsJobKillsChildOnParentExit) @ 910c353`.

Boundary: RemoveAll failure still emits StopCompleted; Close has no deadline. Packaging writes exe before notices, so later copy failure can leave partial package. Non-Windows adapter is development exec.CommandContext and no-op executable lock, not production security evidence.
