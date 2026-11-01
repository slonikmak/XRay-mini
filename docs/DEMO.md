# Guided portfolio demonstration

Target: Windows 11 x64 with WebView2 installed. This guide does not claim a new demonstration has already been recorded. Use an authorized test peer and a fresh demo profile, not an employer's or personal production credentials.

## Before sharing

Use one identified build from a clean commit. Run CI, inspect the dependency audit and verify the complete package. Keep the source commit and package SHA-256 next to the recording. Rehearse the exact packaged EXE; earlier GUI evidence is not acceptance of a new binary.

The repository is private. Give the reviewer authorized repository access or share a separately reviewed package/recording. Do not make the repository public before checking files and history for secrets and rights to redistribute bundled material.

## Product demonstration

**Explain the scope first.** "This is a Windows client around Xray-core. It exposes local proxies; it does not capture all Windows traffic."

**Show import and connection.** Use an already sanitized profile or keep credential entry off camera. Show the import preview, the selected profile, Connecting and the separate process/network indicators. A running Xray is not yet a successful connection.

**Prove the data path.** With the owned test peer available, run:

```powershell
curl.exe --fail --show-error --proxy http://127.0.0.1:10809 https://www.gstatic.com/generate_204
curl.exe --fail --show-error --socks5-hostname 127.0.0.1:10808 https://www.gstatic.com/generate_204
```

Check the exit code after each request. This checks those requests, not every route or DNS policy. A control site can be unavailable independently of the peer. Do not show `-v` output, subscription URLs or credentials in the recording.

**Show one failure.** Cancel during Connecting, or use an intentionally unavailable authorized test peer. Explain what the status means. Rare failures such as Kill refusal are best demonstrated by deterministic tests; label them as simulated failures, not reproduced Windows errors.

**Disconnect and inspect cleanup.** Confirm the UI is stopped and no child/listeners owned by this demo remain. Do not kill unrelated `xray.exe` processes. If there is retained cleanup, show Retry stop rather than claiming a completed disconnect.

## Technical discussion

Open [CODEMAP.md](CODEMAP.md) and the two key files: `internal/core/service.go` and its tests. Trace Connect -> preparation -> process ownership -> health -> Disconnect -> Wait -> cleanup. Then explain the regression `ProcessExited -> ProxyReady` and why session IDs alone did not solve it.

A useful statement of personal contribution is: "I defined the client scope and expected behavior, directed AI-assisted implementation, integrated the components and used specifications, regression tests and review to drive corrections. Xray-core implements the network protocols."

## Recording checklist

Capture real application output, not a generated mockup. Use a neutral profile name. Never record a pasted JSON, VLESS link, QR code, subscription token, raw diagnostic log or personal storage path. Redact consistently without altering success/failure indicators.

Two useful stills are the real connected overview and an honest error/recovery state. Record a short product walkthrough and keep the architecture discussion separate. Note which transport, source commit, build hash and environment were actually shown. A browser-only preview must be labeled as such and must not be presented as a working Windows connection.

Store only reviewed media under `docs/media/` and then link it from README. No media is linked until it exists. A new interactive Windows recording is still a manual evidence item; CI cannot certify it.

## Stop conditions

Do not distribute a build with an unexplained high/critical dependency finding, a failed package-integrity check, or an unverified source/binary association. Do not disable Windows protection for the demo. Missing signing and outstanding acceptance should be stated, not disguised.
