# Xray Mini — unsigned portfolio prerelease

Windows 11 x64 with WebView2 installed. Extract the whole ZIP and keep runtime/ beside XrayMini.exe. Read START-HERE.txt and manifest.json. No profiles, server credentials or personal configuration are included.

This release is a local HTTP/SOCKS5 proxy client, not a system-wide VPN. Xray-core supplies VLESS/REALITY/XHTTP; the application supplies UI, profiles, process supervision and connection state.

This is a draft, not product-acceptance sign-off. CI and package checks do not establish full Windows 11 GUI, cross-user DPAPI, external REALITY peers or every failure mode. The preflight-child stop-recovery limitation is documented in the repository. Do not disable Windows protection to run an unsigned demo.

The workflow appends the exact source commit and run to these notes. Review all checks and perform the documented Windows demo before publication. Package checksums identify bytes; they are not an Authenticode signature.
