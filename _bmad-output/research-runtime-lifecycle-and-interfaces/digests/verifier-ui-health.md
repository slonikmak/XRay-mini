# Independent verifier — refs 12–20

Fresh context, git objects at 910c353 only.

-12 verified: HTTPS204 via HTTP proxy, immediate serial/30s probes and5s deadline.
-13 verified: first failure preserves current state (including existing Degraded), marks unhealthy; second degrades, success resets.
-14 verified: child text discarded, presence flags only; newest200 logs.
-15 verified: subscription-before-snapshot, older-revision rejection, focus/visibility resync.
-16 **disputed then corrected:** scheduling worker precedes reply, so I/O can race ahead. Verified formulation: acceptance does not wait for preparation/process/network I/O. UI lock release before unawaited refresh verified.
-17 verified **conditional:** if bridge remains callable after startup failure, failed ListProfiles prevents Promise.all applying fallback status. No claim that Wails actually opens that window after startup error.
-18 verified:16 App +4 bridge test definitions; mocks/jsdom, no native pass inferred.
-19 verified: real PrepareConfig adapter followed by ephemeral port substitution, Linux direct fixture processes bypass product supervisor.
-20 verified: saved report12PASS and cleanup true; engine identity, no app source commit/Windows result.
- Diagnostic-only events update logs without emitting status; UI waits for later refresh trigger (verified supplemental limit).

Original source paths process.go/service.go are internal/core/. Corrections applied to report before finalization.
