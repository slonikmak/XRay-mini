---
title: "Restore CI and pinned runtime packaging"
type: bugfix
ticket: ""
created: "2026-11-02"
status: built
baseline_revision: "e983e4739bd722375fa3ea9f3697685cc3bb48f5"
route: oneshot
route_source: auto
risk: high
review: quick
review_source: pinned
lenses_ran: [quick]
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** CI cannot complete because the pinned official Xray release was withdrawn and the test toolchain fails the dependency release gate.

**Approach:** Upgrade to an available official Xray release with verified archive and executable pins, update the compatible test toolchain, and verify the real Windows package and supported transports without relaxing security gates.

</frozen-after-approval>

## Implementation Notes

Oneshot: the changes are localized version, hash, dependency, and evidence updates; preserve the existing security checks and prior test evidence.

- Xray 26.10.29 was withdrawn upstream; the official 26.10.31 Windows archive matched the release API and companion SHA-256, the extracted executable was hashed, and its license matched the existing notice.
- The application and package pins now agree and are covered by an offline consistency check.
- Vitest 4.1.12 removes the vulnerable Tinypool dependency; clean npm install, 77 frontend tests, typecheck, build and full/production audits passed on Node 24.20.0 / npm 11.19.0.
- Windows 11 build 26300: all three opt-in native Xray 26.10.31 loopback transports passed through HTTP and SOCKS5; historical 26.10.29 results remain clearly identified.
- The complete Wails package, runtime verification, native production Go suite/vet, and 39 Python packaging/evidence tests passed locally; the hosted CI run remains the final check.

## Plan Change Log

## Review Triage Log

- medium / patch: Audit evidence lacked the source baseline; recorded the full base commit and local dependency changes.
- medium / patch: The local lockfile hash depends on CRLF checkout; recorded both the observed Windows hash and LF-normalized Git hash.
- medium / patch: Build prerequisites claimed Node 23 support; documented the Node versions supported by Vitest 4.

## Verification

- Compare the official Xray archive against release metadata and checksum; verify its license and executable hash.
- Run frontend install, typecheck, tests, build and both dependency audits.
- Run Python package tests, native Windows Go tests and vet, Wails production package, and opt-in Windows real-core loopback for all supported transports.
- Commit scoped changes, push and inspect the hosted CI results and package artifact.
