# Real transport integration tests

## Executed result — 2026-10-30

**PASS: 6 real proxied requests and 6 stopped-server no-bypass checks.**

| VLESS transport | HTTP proxy | SOCKS5 proxy | Both paths fail with VLESS server stopped |
|---|---|---|---|
| RAW + REALITY + Vision | PASS | PASS | PASS |
| XHTTP + REALITY (`mode: auto`) | PASS | PASS | PASS |
| XHTTP + TLS (`mode: auto`, certificate verified) | PASS | PASS | PASS |

Machine-readable evidence: [`integration/results-linux-amd64.json`](../integration/results-linux-amd64.json).
The harness asserts that every fixture child was terminated and reaped. It also
checks the target remains healthy and its request counter does not increase during
negative tests. This is not merely `xray run -test` configuration validation.

**Scope:** Linux amd64 upstream engine compatibility and real application
`core.PrepareConfig` output. This does **not** prove Windows application launch,
Windows process/job supervision, DPAPI, Wails UI, Windows packaged executable
integrity enforcement, installer, or Windows fixed-port behavior. Those need
separate Windows validation. It does not prove every optional XHTTP mode or profile.

## Official engine provenance

Only the unmodified official [XTLS/Xray-core v26.10.29 release](https://github.com/XTLS/Xray-core/releases/tag/v26.10.29)
was executed. The archive digest was independently matched to both the release
asset metadata and the official companion checksum file:

- Archive: [`Xray-linux-64.zip`](https://github.com/XTLS/Xray-core/releases/download/v26.10.29/Xray-linux-64.zip)
- Checksum: [`Xray-linux-64.zip.dgst`](https://github.com/XTLS/Xray-core/releases/download/v26.10.29/Xray-linux-64.zip.dgst)
- Archive SHA-256: `8a5f3e6a1057e4bd0f8a7c57c9d91204b4be0b74222a6e210d070603101163c2`
- Extracted Linux executable SHA-256: `e783b9d2b46ed1ab3617c59b04dfbce38c999322d018c6e060f18d88bf6158aa`
- Version output: `Xray 26.10.29 (Xray, Penetrates Everything.) 0afec66 (go1.27.3 linux/amd64)`

These Linux pins are separate from the shipped Windows runtime pins. No Linux
binary or generated secrets are committed.

## Historical commands for Linux amd64

Upstream withdrew release 26.10.29, so the historical download URLs below no
longer work. The retained results describe that release only; they do not
establish compatibility of the currently bundled version.

Requires Python 3, curl, OpenSSL, Go 1.26.10, and available loopback sockets.
Download only from the official release URLs above; do not substitute a mirror.

```sh
mkdir -p /tmp/xray-linux-26.10.29
base=https://github.com/XTLS/Xray-core/releases/download/v26.10.29
curl -fL --max-time 120 "$base/Xray-linux-64.zip" \
  -o /tmp/xray-linux-26.10.29/Xray-linux-64.zip
curl -fL --max-time 60 "$base/Xray-linux-64.zip.dgst" \
  -o /tmp/xray-linux-26.10.29/Xray-linux-64.zip.dgst
python3 integration/transports.py \
  --archive /tmp/xray-linux-26.10.29/Xray-linux-64.zip \
  --digest /tmp/xray-linux-26.10.29/Xray-linux-64.zip.dgst \
  --go go \
  --results integration/results-linux-amd64.json
```

The executed command used `--go /tmp/xray-toolchain/go/bin/go` with
`GOCACHE=/tmp/xray-go-cache GOPATH=/tmp/xray-go-path`.

## Fixture isolation and semantics

- `integration/prepareconfig` invokes the **real exported builder**, not a copied
  implementation. The input includes a malicious/unwanted direct outbound and
  routing rule; the builder discards them. The harness asserts one outbound remains
  and stream settings are unchanged. The only post-build modification is ephemeral
  loopback inbound ports instead of product ports 10808/10809.
- All listeners, VLESS peers, REALITY camouflage target, and HTTP target are on
  `127.0.0.1`. Requests carry a randomly generated body token. There are no user's
  credentials, public proxy servers, remote probes, DNS targets, or OS proxy edits.
- A concurrent TLS 1.3 local Go fixture supplies the REALITY handshake target.
  Fresh X25519 keys, UUIDs, and one-day self-signed certificates are generated per
  execution and removed with the temporary directory.
- TLS verification is **not disabled**. Only the client child receives
  `SSL_CERT_FILE` pointing to the temporary test CA and `useSystemCA: true` selects
  that trust source. No OS trust store is changed. Xray defaults to its bundled CA
  roots, so setting the environment variable alone is insufficient.
- Upstream v26.10.29 Freedom blocks private destinations by default. The test
  **server**, not client, has an explicit `finalRules` allow entry for only the
  target's loopback IP and ephemeral port. This permits the authorized loopback
  fixture without introducing a client direct outbound.
- Every request has a six-second curl timeout and an eight-second subprocess
  deadline. Readiness is bounded; the entire suite has a 240-second deadline.
  `finally` terminates, waits, and if necessary kills child processes, then closes
  the target. SIGTERM also enters cleanup. As with any process, SIGKILL cannot run
  Python cleanup.
