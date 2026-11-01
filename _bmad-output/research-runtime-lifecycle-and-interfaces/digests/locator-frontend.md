# Locator: frontend/

Snapshot: 910c353. Search only.

- `frontend/src/bridge.ts:18 (DesktopAPI) @ 910c353`, interface and native adapter; generated Wails methods, models and event payloads under frontend/bindings/.
- `frontend/src/App.tsx:48 (refresh) @ 910c353`, concurrent profiles/status/logs refresh and revisions; effect subscribes before snapshot at line 62.
- `frontend/src/App.tsx:90 (importProfile) @ 910c353`, preview/acknowledgment; toggleConnection107; diagnostics162; rename175/delete176.
- `frontend/src/App.test.tsx:23 (describe) @ 910c353`, 17 UI cases; bridge.test.ts has four bridge cases. Mocked jsdom coverage, no native window evidence.
- No Playwright/Cypress/Puppeteer, periodic polling, browser session storage, fetch/XHR/WebSocket matched in frontend source search.
