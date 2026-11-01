# Native Windows real-core loopback evidence, 2026-11-01

PASS for this deliberately limited native core fixture. This is not complete
Windows 10/11, GUI, package, external server or external-egress acceptance.

## Executed checkout and host

- App HEAD: `ee4832faa679f632b0bda850fe6c6f288dedcd24`, with existing uncommitted
  work, including production lifecycle changes. The test ran that dirty checkout;
  it does not claim that clean HEAD alone has this result.
- Host: `Microsoft Windows 11 Home`, amd64, `10.0.26300`, UBR `9711`, display
  version `26H2` (CIM caption used; legacy ProductName registry value says Windows 10).
- Toolchain: `go version go1.26.10 windows/amd64`.
- Revised fixture passing execution ended about `2026-11-01T14:04:18+01:00`.
- Pinned Xray: `26.10.29`, existing adjacent `bin/runtime/xray.exe`, SHA-256
  `07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10`.
  No download was needed. Production validation ran before fixture commands and
  lock/digest/contained creation applied to every real preflight and working child.
- Native test executable SHA-256:
  `a7cd172460fc5e5a88bf6393d550c72abb96d169225bcf1e543247469e8ad063`.
- Test source SHA-256:
  `2f6672dd14d692dc993f2576043e2528c37d6e637965c644235ba49fcac00a70`.
- Executed dirty `internal/core/service.go` SHA-256:
  `17b61122147b04bd8f47c3b3dd94eaa348971e67c49fdd34dc084afd185a586d`.
- Executed dirty `internal/core/process_windows.go` SHA-256:
  `137a580f7c4cbbfbba9488f8190d5c6c23e6148d8f2f14f9e9cf24ad70e9482e`.

## Exact successful PowerShell commands

```powershell
go test -c ./internal/core -o bin/core-native.test.exe
$env:XRAY_MINI_NATIVE_LOOPBACK='1'
try {
    & .\bin\core-native.test.exe '-test.v' '-test.run=^TestNativeWindowsLoopback$' '-test.timeout=180s'
    $nativeExit=$LASTEXITCODE
} finally {
    Remove-Item Env:XRAY_MINI_NATIVE_LOOPBACK -ErrorAction SilentlyContinue
}
exit $nativeExit
```

The environment variable was scoped to the command process, not persisted in the
user/system environment. Initial unquoted dotted PowerShell flags failed at flag
parsing before any fixture started; the quoted invocation above is the actual run.

## Results

Revised native execution: **PASS, 18.14 seconds** (RAW/REALITY 6.56s,
XHTTP/REALITY 4.73s, XHTTP/TLS 6.56s). Only the native matrix was rerun after
adding duplicate-Connect assertions; the ordinary-suite and opt-in SKIP results
below are from the earlier fixture revision.

| Transport | HTTP response | SOCKS5 response | Peer stopped, both fail without new target hits | Degraded then recovered Connected without client restart | Terminal cleanup |
|---|---|---|---|---|---|
| RAW/REALITY | PASS | PASS | PASS | PASS; client launches = 1 | Real owned working child killed; Error after cleanup |
| XHTTP/REALITY | PASS | PASS | PASS | PASS; client launches = 1 | Disconnect -> Disconnected |
| XHTTP/TLS | PASS | PASS | PASS | PASS; client launches = 1 | Disconnect -> Disconnected |

- All three transport cases remained `Connecting/running`, with no health result,
  while the initial real probe was gated. Releasing it allowed an actual HTTP proxy
  request to the local HTTP 204 target to establish `Connected`.
- **3/3** duplicate `Connect(profile.ID)` calls while the real client was
  `Connected` returned an error, with exactly one observed working client launch.
- **6/6** transport positive proxy requests returned the generated target token,
  plus **2/2** correct-pin requests after the wrong-pin phase.
- **6/6** stopped-peer requests failed; the target remained directly healthy and
  its application request count did not increase while the peer was absent.
- **3/3** peer-loss cases reached `Degraded`; restoring the same peer config
  returned them to `Connected`, with exactly one working client launch per case.
- TLS wrong-pin control: **2/2** proxy requests rejected, `Degraded`, no new
  application target hits, directly healthy target. The restored peer's single-Wait
  completion channel stayed open throughout the phase. Reconnecting the original
  correct-pin profile then returned `Connected` and both HTTP/SOCKS5 token requests
  succeeded through the same peer. Disconnect cleaned up normally.
- Five actual Service working children total: three transport profiles, one
  wrong-pin profile and one correct-pin confirmation. Each received exactly one
  observed `Wait`.
- Six actual peer working children total: initial and restored peer per transport;
  each had one `Wait`, and its helper was joined. Owned fixture listeners were
  rebound after exit to check release. All Service runtime directories were absent
  after cleanup; owned temporary fixture material was removed.
- Peer stop is final only after its Wait/helper completion is confirmed; a timed-out
  explicit stop leaves registered cleanup able to retry. Bounded termination falls
  back to closing the owned kill-on-close job. If exit still cannot be confirmed,
  teardown reports failure and retains the owned directory/configuration instead
  of letting `t.TempDir` remove it. The failure/fallback branches were inspected;
  this run did not inject OS termination failure.
- Post-run OS inspection: **0** listeners on 10808/10809, **0** `xray` processes.
- Ordinary `go test ./internal/core -count=1`: **PASS**, `8.317s`.
- Executing the native case without the opt-in variable: **SKIP**, `0.00s`, PASS
  process result. Runtime/ports are not inspected by this case unless opted in.
- `git diff --check -- internal/core/native_transport_windows_test.go`: exit 0.

## Scope and limits

The fixture uses real `NewService`, `ImportProfile`, `Connect`, the session worker,
the real `execRunner`, native Job Objects and fixed product proxy ports. It uses
synthetic UUIDs, X25519 keys and one-hour self-signed localhost leaf certificates,
generated by Go's standard library. Every listener is loopback. Peer freedom
outbounds permit only the local target address/port. Imported input deliberately
contains an untrusted direct outbound/routing rule; normal product import rebuilds
the client configuration. Requests are bounded; no application fallback is enabled.

Only existing test hooks were changed on these Service instances: the health probe
waits on a cancellable test gate and then requests a local HTTP 204 through the real
HTTP proxy; health interval is 200 ms. The production health endpoint, HTTPS trust,
five-second budget and thirty-second cadence are not validated by this local probe.

XHTTP/TLS uses `serverName: localhost` and exact `pinnedPeerCertSha256` verification
of the generated leaf, with no `allowInsecure` or OS CA change. This proves leaf pin
enforcement and transport traffic, not ordinary OS trust-chain acceptance. See
[Xray TLS documentation](https://xtls.github.io/en/config/transports/tls.html#tlsobject).

No production Go, existing tests, frontend, scripts or tickets were modified by this
fixture implementation. No admin operation, proxy/route setting, external endpoint
or CA installation was used. Test artifacts are opt-in and the test binary is a
verification artifact, not part of the application package.

Still unavailable: Windows 10, another account, missing WebView2, GUI behavior,
forced GUI-parent exit, external authorized peers/egress, and full DACL/DPAPI/Job
Object acceptance matrix. Production protection/containment code was exercised here;
that does not close the unavailable system scenarios.
