# Build, checks and demonstration package

## Prerequisites

Use Windows 11 x64 with WebView2 installed for the actual demo. The repository pins Go 1.26.10 and Wails 3.0.0. Use Node.js 22+ and Python 3. CI selects Node 22 and Python 3.12. The Windows hosted build runner is a build/test environment, not proof of Windows 11 desktop acceptance.

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor
wails3 task package
python scripts/package_runtime.py --verify bin
```

`task package` runs bindings generation, npm ci, frontend build, Windows resources, the production Go build, pinned Xray installation and complete-package verification. Do not run another npm install/test concurrently in that working directory. Runtime installation alone is not GUI/package verification.

Run `.\bin\XrayMini.exe`. Development uses `wails3 task dev`, which keeps the required runtime beside the launched executable. Plain `go run .` creates an executable elsewhere and is not a supported connected-app launch.

## Tests

```powershell
go test -tags production -timeout 3m ./...
go vet -tags production ./...
python -m unittest discover -s scripts -p "test_*.py" -v
cd frontend
npm ci
npm run typecheck
npm test
npm run build
cd ..
python scripts/check_npm_audit.py
```

Run the root Go suite after frontend build because the application embeds frontend assets. On Linux, `go test -race -count=2 -timeout=3m ./internal/core` and `go vet ./internal/core` exercise portable core code, not Windows adapters. Windows race instrumentation needs a working C toolchain. Native real-transport fixtures remain explicitly opt-in; follow their existing evidence instructions rather than inventing peers or using personal secrets in CI.

## Create the distributable

```powershell
python scripts/package_demo.py --bin bin --output dist-demo
```

The packager requires a clean Git checkout, verifies the existing complete package, and copies only an explicit allowlist: GUI, pinned Xray, runtime notices/license and the quick-start guide. It emits an unsigned ZIP, an internal file-hash manifest with the source commit, and SHA256SUMS. Extra files from bin, saved profiles, temporary configs and verification executables are not archived. Existing archives are not overwritten.

Source revision and hashes provide traceability, not an Authenticode signature, reproducible-build proof or complete security audit. Package verification checks the GUI's PE structure; it does not prove the GUI was built from a particular source. In CI, a clean checkout and the build steps establish the recorded build process.

## GitHub Actions

`CI` runs on main pushes, pull requests and manual requests. It runs portable Go race/vet checks, frontend checks, Python tests, dependency audit and a Windows production build/test/package job. All actions are commit-pinned. Credentials are not persisted in checkout. Test/audit reports and unsigned packages are Actions artifacts; they are not published releases.

Dependency audit fails on high/critical findings in either complete or production-only trees, and fails if the registry result is unavailable. Other checks can still run, but **CI is not green and no draft release is authorized by that run**. See [DEPENDENCY-REVIEW.md](DEPENDENCY-REVIEW.md).

To create a release candidate, run `Draft demo release` on main. It reruns the same CI workflow, downloads that run's package, verifies SHA256SUMS and creates a draft prerelease targeting the exact source SHA. It does not publish it. Review the artifact, audit, notices and Windows demonstration before manually publishing or sharing.

The repository remains private. GitHub may require approval or app workflow-write permissions before newly added workflows can run. A committed workflow file and a successful local script test are not evidence of a successful hosted build.
