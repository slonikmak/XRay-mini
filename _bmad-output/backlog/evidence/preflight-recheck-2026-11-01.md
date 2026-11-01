# Saved-profile preflight recheck — 2026-11-01

User reported the generic Trusted Xray preflight failure after re-import. The original failure was not reproduced; no cause or code fix is claimed.

Current committed source: `95b13846a33064b98d09c1a0d866196b7d676d55`, pre-existing unrelated dirty bindings preserved. Production GUI SHA256 `8dd510de3d9c6f451fdde08caf7623ac89b2ce45dc5b39af1563ea22fedc1ad7`; pinned Xray 26.10.29 SHA256 `07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10`. Windows 11, same user. Package verifier passed.

Read the single saved profile with same-user DPAPI in memory, without replacing the store or logging config/name/credentials. It is XHTTP/TLS. Pinned Xray standalone `run -test -config stdin:` returned 0 / Configuration OK. An opt-in test overlay outside the repository then reused unprotect, PrepareConfig, secureDirectory and startVerifiedContained with the same saved profile; native job creation and the single Wait passed. Its private temp file was removed by test cleanup; verification executable moved outside distributable bin. Standalone CLI is supplementary diagnosis, not the product containment proof.

Launched the existing production GUI normally with actual user APPDATA, selected the already saved profile and pressed Connect. Preflight passed and Xray ran. First network probe failed; subsequent production probe reached Connected/Reachable. Separate HTTP and SOCKS5 HTTPS 204 checks passed and both egress IPs differed from direct baseline: [proxy record](preflight-recheck-proxy-2026-11-01.json). No actual IP/source URL/config is stored in evidence.

Application and connection left running for the user, intentionally handed off as Connected; the active session runtime is not deleted underneath it. No profile edit/re-import, OS proxy/CA/routes change, product source modification or full bug-resolution claim. If preflight fails again, the exact native creation/exit category is still needed; the existing generic UI message does not establish an integrity or profile defect by itself.
