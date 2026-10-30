#!/usr/bin/env python3
"""Fetch only pinned official Xray bytes, verify archive + executable, package minimal runtime.
Usage: python3 scripts/package_runtime.py [output-directory, default bin]
Never executes downloaded files. Requires Python 3 standard library only.
"""
import hashlib
import io
import pathlib
import shutil
import sys
import urllib.request
import zipfile

VERSION = "26.10.29"
ARCHIVE_SHA256 = "fce33e81cccf50e7aab572b175519a50b5e0f9d2f5c8d1ed73012ea5c9ad4d58"
EXE_SHA256 = "07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10"
URL = f"https://github.com/XTLS/Xray-core/releases/download/v{VERSION}/Xray-windows-64.zip"

def checked(data, digest):
    if hashlib.sha256(data).hexdigest() != digest:
        raise SystemExit("Integrity check failed; no runtime was installed")
    return data

def main():
    destination = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "bin")
    with urllib.request.urlopen(URL, timeout=120) as response:
        archive = checked(response.read(64 * 1024 * 1024), ARCHIVE_SHA256)
    with zipfile.ZipFile(io.BytesIO(archive)) as z:
        executable = checked(z.read("xray.exe"), EXE_SHA256)
        license_text = z.read("LICENSE")
    runtime = destination / "runtime"
    runtime.mkdir(parents=True, exist_ok=True)
    (runtime / "xray.exe.tmp").write_bytes(executable)
    (runtime / "xray.exe.tmp").replace(runtime / "xray.exe")
    (runtime / "Xray-core-LICENSE").write_bytes(license_text)
    shutil.copyfile(pathlib.Path(__file__).resolve().parent.parent / "licenses" / "NOTICE.md", runtime / "NOTICE.md")
    print(f"Verified official Xray {VERSION}: {runtime / 'xray.exe'}")

if __name__ == "__main__":
    main()
