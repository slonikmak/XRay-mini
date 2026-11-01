# Startup and diagnostic refresh decision — 2026-11-01

Source baseline: ee4832faa679f632b0bda850fe6c6f288dedcd24 plus dirty implementation changes. Windows 11 Home 10.0.26300.9711, normal user, Wails 3.0.0.

F3 confirmed. Pinned Wails propagates ServiceStartup failure before opening pending windows; main exits on application.Run error. The isolated old packaged GUI exited 1 without a window; safe cause existed only on stderr ([before record](startup-before.json)). This is a native observation, not inferred from a rejected mock API.

Ticket 11 now retains safe initialization failure, opens the window, exposes Error through GetStatus and rejects mutations. Native observation of the final GUI showed the lock-ownership cause, Error, disabled Connect and no browser-preview banner. See [final process record](gui-startup-error-process-v2.json). Unit tests independently cover failed and pending ancillary calls and late native host injection. The first candidate's false browser-preview label was found in the native run, fixed, regression-tested and rebuilt.

F4 decision under the user's explicit delegation of autonomous execution: retain existing refresh behavior. Diagnostic-only presence categories are buffered without a status event; UI refreshes logs on status/focus/visibility/actions. Normally the next health status updates the view on the existing approximately 30-second cadence; this is not a maximum delivery guarantee and adds no realtime SLA. Code-path inspection established this limitation; no controlled wall-clock latency experiment was run. Existing bounded secret-free diagnostics remain in scope; no additional polling is needed for this finding. This is a delegated implementation decision, not a claim that the user separately approved a numeric delay.

Required follow-up was split into ticket 11 and added to system acceptance dependencies. Spike completion does not constitute full Windows acceptance.
