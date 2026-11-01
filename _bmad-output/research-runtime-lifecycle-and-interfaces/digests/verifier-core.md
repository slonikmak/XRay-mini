# Independent verifier — refs 1–11

Fresh context, git objects at 910c353 only. All11 claims verified against source.

1. One VLESS/streamSettings and reconstruction: extend selectOutbound citation to45–146 (streamSettings is after82).
2. Current-user DPAPI: crypt22–43, add123–146.
3. Lock before stale cleanup: newStore32–62, Windows share0 lock8–17.
4. Loop ownership/atomic snapshot: loop187–228, GetStatus116; constructor initializes before loop.
5. Connect barrier and session-ID gating: loop234–255,286–318.
6. Failed RemoveAll still releases active through StopCompleted: run343–354,loop290–302.
7. Blocking callback can hold Close: publishLoop86–95 (not81),Close174–184.
8. Locked, pinned, adjacent runtime validation before creation: process24–46,runtime19–58,Windows226–235.
9. Atomic JOB_LIST and restricted handles: Windows102–208; both production entry points.
10. Once-only/infinite Wait, closes job/drains; ignored Kill errors: Windows74–95,service458–461.
11. Build/package ordering and pins confirmed; cite script main19–36 for installation, not VERSION alone.

Paths above refer to internal/core unless script/Taskfile is named. Exact claim/citation records are in claims.json and the report.
