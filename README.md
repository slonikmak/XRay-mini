# Xray Mini · Windows local proxy

Go **1.26.10**, Wails **3.0.0**, React/TypeScript. Windows 10/11 **x64** only. This is a local SOCKS5/HTTP proxy, **not a system VPN**: no TUN, system proxy changes, kill switch, auto-update or administrator privileges.

## Build and package

Install Go, Node.js 22+, Python 3, Wails prerequisites and Microsoft WebView2 Runtime. From this directory:

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor
wails3 build
python scripts/package_runtime.py bin
python scripts/package_runtime.py --verify bin
```

Ship **all of `bin/`**, including `XrayMini.exe`, `runtime/xray.exe`, its license and notice. The packaging script downloads the official pinned **Xray 26.10.29** archive, verifies both archive and executable SHA-256, and installs only the executable and notices. No unknown binary is executed to determine its version. Runtime checks its exact executable digest before **each** preflight and working-process launch. No arbitrary executable path is accepted. The application is unsigned; native Windows acceptance remains pending (see verification).

The offline `--verify` command checks the complete package and reports file hashes;
it does not execute the GUI or runtime. Runtime installation alone reports only
runtime readiness. `wails3 task package` finishes with the complete-package check.

For development, the same `runtime/` directory must be beside the executable Wails actually launches. Do not replace the pin with an arbitrary local binary. The displayed version describes the pinned package, not a claim that a missing executable is installed.

## Use

1. Import client Xray JSON containing exactly **one VLESS outbound with streamSettings**, or choose **Subscription or VLESS link**. For a subscription, explicitly load an HTTPS URL returning a plain or Base64 VLESS list; alternatively paste one VLESS link. Choose one candidate and import it, then use Connect separately. Examples contain placeholders, not working server credentials.
2. Review and acknowledge the import preview: source inbounds, routing, other outbounds and other top-level sections are discarded. Only the standalone selected outbound is retained. Ambiguous selections, duplicate JSON keys and unsupported external dependencies/local resource references are rejected.
3. Connect. Configure the desired application manually for **SOCKS5 `127.0.0.1:10808`** (TCP only) or **HTTP `127.0.0.1:10809`**. Both are fixed loopback listeners. Runtime routing sends both through the single selected VLESS outbound; no direct outbound is included.
4. Disconnect before changing the active profile. Profiles can be renamed and deleted.

RAW/REALITY, XHTTP/REALITY and XHTTP/TLS fields are preserved in the supported standalone subset. Import and successful `run -test` do **not** prove transport connectivity.

Manual subscription loading uses normal TLS verification, a 15-second total limit,
a 2 MiB response limit, at most 100 profiles and 64 KiB per link. Redirects are
rejected. Supported links use `encryption=none`; XHTTP path, host, mode and the
self-contained `extra.xmux` fields are preserved. Unsupported or duplicate
parameters are rejected, including split-download dependencies. Other XHTTP extra
fields can still be imported through the existing complete JSON flow when supported.
Candidate previews contain only names, opaque IDs and transport labels and expire
after five minutes. Editing the source, canceling, replacing the preview, importing
or closing the app discards it. Source URLs are never saved and subscriptions do not
refresh automatically. Loading contacts only the entered HTTPS source and does not
save profiles or start Xray.

## Status and safety

`Connecting` → `Connected` only after a real HTTPS request through the local HTTP proxy. A probe has a **5 second timeout**, runs serially every **30 seconds**, and uses `https://www.gstatic.com/generate_204`. Two consecutive failures show `Degraded`; success resets the count. One destination cannot prove every destination works. Xray is not automatically restarted. `lastCheckedAt` records the last completed probe.

Profiles use Windows DPAPI. Runtime JSON is briefly plaintext in a private DACL-protected directory; normal cleanup removes it. An exclusive per-data-directory OS lock prevents another instance from deleting an active session; startup removes stale `runtime-*` after acquiring that lock. This is not secure disk erasure, nor protection against a compromised user/admin. Non-Windows storage is development-only, not DPAPI-equivalent.

If runtime cleanup is blocked, another working Xray is not admitted. Remove the
external file block and press Connect once to request cleanup, then connect after
cleanup finishes; API Disconnect or Close can also retry. Close reports unconfirmed
session cleanup after five seconds and bounds notification drain to 250ms. A blocked
notification callback may remain until it returns, but does not cause more callback
workers to be spawned or turn unconfirmed process cleanup into success.

Preflight and working Xray are created **inside** a kill-on-close Windows Job Object using `PROC_THREAD_ATTRIBUTE_JOB_LIST`; there is no create-then-assign crash window. Native crash/DPAPI/ACL/WebView2 verification still must run on Windows.

Diagnostics retain only bounded application messages and fixed Xray stdout/stderr-presence categories. Untrusted Xray output is drained but never displayed or retained. Server addresses, UUIDs, URLs and raw configuration are not copied into diagnostics.

## Tests

```sh
go test -race ./internal/core
go vet ./internal/core
cd frontend
npm ci
npm run typecheck
npm test
npm run build
```

Bindings are generated by `wails3 task bindings` (also part of build). Browser mode without Wails reports backend unavailable, not a fake connection.

See [architecture](docs/ARCHITECTURE.md), [verification and pending acceptance](docs/VERIFICATION.md), and [audit traceability](docs/AUDIT-FIXES.md). The authoritative [specification](docs/xray-mini-spec-v0.2.ru.md) includes manual import; the [baseline audit](docs/xray-mini-audit-v0.2.ru.md) describes the old commit, not the fixed tree.

For the code-to-spec assessment at `910c353`, source research and the next acceptance work, start with the [brownfield guide](docs/BROWNFIELD.md).
