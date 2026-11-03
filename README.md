# Xray Mini

**A Windows desktop networking client — Go, Wails and TypeScript, built with AI-assisted development and specification-driven checks.**

Xray Mini manages a separate Xray-core process and provides local HTTP/SOCKS5 proxies. It supports standalone **VLESS + RAW/REALITY**, **XHTTP/REALITY** and **XHTTP/TLS** configurations, with JSON and explicit subscription/link import.

**Windows 11 x64 · Wails 3.0.0 · Go 1.26.10 · React/TypeScript**

> This is a **local proxy client, not a system-wide VPN**. Applications must use its loopback proxies explicitly. There is no TUN, system-proxy change, kill switch, automatic restart or automatic update. The demo is unsigned and full product acceptance is still open.

## Start here

- **Review the engineering:** [code map and design decisions](docs/CODEMAP.md).
- **See what to demonstrate:** [guided demo and recording checklist](docs/DEMO.md).
- **Build or run it:** [build, package and CI instructions](docs/BUILD.md).
- **Understand the evidence:** [verification](docs/VERIFICATION.md), [current acceptance scope](docs/ACCEPTANCE-SCOPE.md) and [dependency review](docs/DEPENDENCY-REVIEW.md).

The CI workflow produces a complete package as an Actions artifact when its Windows job succeeds. A separate manual workflow creates a **draft prerelease only after all CI jobs pass**. A recorded walkthrough is not published in this README yet; the demo guide identifies the real Windows scenarios to capture. Private-repository access is still required.

## What the application does

Import and preview a configuration, choose a profile, connect, and configure a browser or another application to use `127.0.0.1:10808` (SOCKS5, TCP) or `127.0.0.1:10809` (HTTP). The client verifies and starts its pinned Xray executable, checks an HTTPS request through the proxy and reports process state separately from network health. Disconnect owns the full stop/wait/cleanup sequence.

Closing the window hides Xray Mini in the system tray; an active connection keeps running. The tray icon is mint when the proxy check succeeds, gray while off or transitioning, and amber when a check fails or an error occurs. Hover over it for the current status. Click the tray icon or choose **Open** to return to the same window. The tray menu offers Cancel connection, Disconnect or Retry stop when applicable. Choose **Exit** to stop the owned Xray process through normal application shutdown and quit.

Profile import is deliberately restrictive: exactly one standalone VLESS outbound is selected; source inbounds, routing and unrelated sections are discarded with an explicit preview. Both local listeners route through the selected outbound, without a direct fallback. Subscription loading is manual, bounded and separate from saving or connecting; candidate previews expire and can be canceled.

## What makes this more than a Connect button

**Concurrency with explicit ownership.** A single event loop owns session state. Workers send session-tagged events; snapshots are published atomically. A slow UI subscriber does not block connection control. Queues and diagnostics are bounded, and one owner waits for the child process.

**Failure handling, not just the happy path.** Process exit is terminal within a session, so late health results cannot resurrect it. Failed working-child termination can be retried explicitly without a second Wait. Resources remain owned until cleanup is confirmed. Stop/Retry stop stays accessible inside profile dialogs during pending saves.

**Actual Windows integration.** The client uses Job Objects, restricted handle inheritance, DPAPI, directory ACLs and an exclusive storage lock. Preflight and working Xray are created inside a kill-on-close job. The bundled runtime is digest-checked before execution.

**UI backed by authoritative state.** React sends commands through Wails bindings and renders Go snapshots/events with revision ordering. It does not infer connection success from a button click or a running process.

**Reviewable AI-assisted delivery.** Requirements, architecture decisions, tests, findings and fixes are preserved. One useful example is the regression from `ProcessExited` to a late `Connected`: the fix required a state-machine invariant, not another mutex.

## Architecture at a glance

```text
React / TypeScript
  commands | snapshots + revisioned events
           v
Wails facade (app.go)
           v
Service.loop — single owner of session state
  | commands/events       | atomic snapshots
  |                       +--> bounded notification publisher --> UI
  +--> session supervisor --> one Wait / explicit stop retry / cleanup
  +--> serial health probe --> local proxy --> selected VLESS peer
  +--> profile repository --> DPAPI + private files + instance lock
                           |
                           v
                    external xray.exe
                    HTTP / SOCKS5 -> REALITY / XHTTP -> remote peer
```

User traffic belongs to Xray-core, not the React UI or the Go control plane. Detailed responsibilities and source links are in the [code map](docs/CODEMAP.md).

## Authorship and scope

Created by **Anton Mashkov** as a focused desktop/networking portfolio project. AI agents are a principal implementation tool; the project makes requirements, integration decisions, review findings and acceptance boundaries inspectable. It is not a claim that all code was handwritten or that the author implemented VLESS, REALITY, XHTTP or cryptography: those are provided by Xray-core.

The project demonstrates a concrete Go/Windows client and a quality-control process. It does not establish years of commercial Go experience, large-scale VPN operations or expertise in Windows routing/TUN drivers.

## Verification: what is and is not established

The repository preserves a **1 November 2026 Windows 11 verification report** for the termination/recovery changes: native Go tests and vet, 42 frontend tests, repeated service regressions, a Windows cleanup-refusal case, three real-core loopback transports and a production build. A separate earlier report covers one external XHTTP/TLS endpoint through the packaged GUI. These are recorded runs for identified source/artifact snapshots, not proof for every future commit.

- [Termination/recovery evidence](_bmad-output/backlog/evidence/concurrency-recovery-2026-11-01.md)
- [External XHTTP/TLS demonstration evidence](_bmad-output/backlog/evidence/subscription-acceptance-2026-11-01.md)
- [Full verification and remaining checks](docs/VERIFICATION.md)

Current CI results live in Actions. **A configured workflow is not a passing run**, and a Windows Server runner is not Windows 11 interactive acceptance. CI does not use personal subscription URLs, private peers or credentials.

Open boundaries include cross-user DPAPI, remaining Windows GUI/stress and external REALITY checks, and recovery after failed termination of the separate preflight child. The prior npm build log also reported advisories whose applicability must be established; the new dependency job does not suppress them. See [dependency review](docs/DEPENDENCY-REVIEW.md).

## Build and demo package

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor
wails3 task package
python scripts/package_demo.py
```

Use a clean checkout and the prerequisites in [BUILD.md](docs/BUILD.md). Run `.\bin\XrayMini.exe`; plain `go run .` does not place the pinned runtime beside the executable. The archive includes the complete application/runtime layout, notices, a source-revision manifest and a quick-start guide. It contains **no saved profiles or live server credentials**.

Do not disable Windows protection to run an unsigned demo. A hash identifies bytes; it is not a code-signing certificate or a security audit.
