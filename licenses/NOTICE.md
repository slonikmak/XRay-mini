# Bundled Xray-core notice

Xray Mini distributes the unmodified official **Xray-core v26.10.29**, Windows
x64, from <https://github.com/XTLS/Xray-core/releases/tag/v26.10.29>.
Xray-core is a separate executable, licensed under the Mozilla Public License
2.0. The complete license accompanies the package as `Xray-core-LICENSE`.
Source for this exact release, including its third-party source/dependency
notices, is available at <https://github.com/XTLS/Xray-core/tree/v26.10.29> and
<https://github.com/XTLS/Xray-core/archive/refs/tags/v26.10.29.zip>.
No Xray-core source modifications are included in this distribution.

Only `xray.exe` and the license are extracted from the official runtime archive.
Wintun, launch scripts, GeoIP and GeoSite data are not shipped or needed by the
restricted MVP configuration. This app does not enable TUN or require Wintun.

Official ZIP SHA-256:
`fce33e81cccf50e7aab572b175519a50b5e0f9d2f5c8d1ed73012ea5c9ad4d58`

Exact extracted `xray.exe` SHA-256:
`07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10`

Pins established 2026-10-30 by comparing the downloaded official archive to the
GitHub release API asset SHA-256, then hashing the extracted executable. Hash
verification is not a claim that transport compatibility or Windows runtime
acceptance has passed. Run the documented Windows/network acceptance tests.

Build/package with `python3 scripts/package_runtime.py bin`; place `runtime/`
next to `XrayMini.exe`. The app checks the exact executable digest before each
preflight or working invocation. Updating Xray requires a reviewed pin update,
license review, rebuild and repeated acceptance tests; no automatic updates.

The installation directory should not be writable by other users. Digest
checking detects substitution at verification time; it is not a sandbox against
malware already controlling the same Windows user and modifying files between
verification and process creation.
