---
title: "Reflect connection health in the tray icon"
type: feature
ticket: ""
created: "2026-11-03"
status: built
baseline_revision: "72e4aed7a02e130d2c72f689f6dbb1691e7b1d4e"
route: oneshot
route_source: auto
risk: medium
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="user intent from conversation">

## Intent

**Problem:** The tray icon looks the same whether the proxy is connected, disconnected, or experiencing an error.

**Approach:** Derive a recognizable tray icon variant from the latest Go status snapshot. Use mint only after a successful connection check, a muted icon while off or transitioning, and a warning variant for failed checks or errors. Keep the tooltip accurate without exposing profile data.

</frozen-after-approval>

## Implementation Notes

Oneshot: reuse the existing Wails tray and status notification flow; preserve session ownership, window behavior, and the original application icon.

- Mint is shown only for Connected with a running process and a healthy last check. A failed check or Error/Degraded uses amber, and off/transitioning uses gray.
- The first failed check can retain the Connected state, so the tray status says Check failed in that case. Existing action selection still uses the service state and session.
- Variants derive once from the embedded application icon and change only when their visual state changes. Windows created and released native tray-sized icon handles for all variants.
- On Windows 11 build 26300, the production package, Go tests and vet passed. A separate isolated application instance registered its tray with Windows; automation could not inspect the notification area itself.

## Plan Change Log

## Review Triage Log

- medium / patch: Wails can initialize the window and tray in separate goroutines. The initial icon is now derived from the startup snapshot during ServiceStartup, before Wails initializes either native object.
- medium unverified / defer: Wails can release the previous icon before a failed shell update during an Explorer restart. On this Windows 11 host, an isolated probe reproduced the shell removal/update sequence and successfully re-added and updated the icon; an actual Explorer restart was not performed.
- medium / patch: The Windows test now extracts each actual native-sized HICON and verifies its color and recognizable dark mark; the tested host used 28x28 icons.
- medium unverified / defer: A later status change could theoretically arrive while Wails has assigned its tray implementation but not yet initialized its native window. The normal isolated application launch registered its tray, but this scheduling interleaving was not reproduced.


## Verification

- Verify icon selection across disconnected, connecting, healthy, first failed check, degraded, recovery, disconnecting, and error states.
- Verify the icon variants decode, remain distinguishable at tray size, and retain the existing mark.
- Run native Windows Go tests and the production package build without overlapping npm operations.
- Inspect the Windows tray when feasible; report any unverified interaction honestly.
