#!/usr/bin/env python3
"""Package verified Windows bytes from a clean checkout; never execute binaries."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import zipfile

import package_runtime

ROOT = Path(__file__).resolve().parent.parent
FILES = ("XrayMini.exe", "runtime/xray.exe", "runtime/Xray-core-LICENSE", "runtime/NOTICE.md")


def clean_revision(root: Path) -> str:
    def git(*args: str) -> str:
        return subprocess.check_output(["git", "-C", str(root), *args], text=True, stderr=subprocess.PIPE).strip()
    revision = git("rev-parse", "HEAD")
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("Expected a full Git commit SHA")
    if git("status", "--porcelain", "--untracked-files=all"):
        raise ValueError("Demo packaging requires a clean checkout; commit or remove pending files first")
    return revision


def create_archive(source: Path, output: Path, revision: str, guide: Path) -> Path:
    if not re.fullmatch(r"[0-9a-f]{40}", revision):
        raise ValueError("Expected a full Git commit SHA")
    # Do not follow links to files outside the intended package layout.
    if source.is_symlink() or (source / "runtime").is_symlink():
        raise ValueError("Package directories must not be symlinks")
    for name in FILES:
        if (source / name).is_symlink():
            raise ValueError("Package files must not be symlinks")
    verified = package_runtime.verify_package(source)
    payload = {name: (source / name).read_bytes() for name in FILES}
    # Verify the bytes actually archived, not merely an earlier read of each file.
    for name, data in payload.items():
        if hashlib.sha256(data).hexdigest() != verified.get(source / name):
            raise ValueError("Package changed after verification; build in an isolated directory")
    payload["START-HERE.txt"] = guide.read_bytes()
    manifest = {
        "schema_version": 1,
        "source_commit": revision,
        "platform": "windows-amd64",
        "runtime_version": package_runtime.VERSION,
        "unsigned": True,
        "acceptance": "Demonstration package; complete Windows 11 product acceptance is not established",
        "provenance": "Source revision and byte hashes; not a signature or reproducible-build proof",
        "sha256": {name: hashlib.sha256(data).hexdigest() for name, data in sorted(payload.items())},
    }
    payload["manifest.json"] = (json.dumps(manifest, indent=2, sort_keys=True) + "\n").encode()
    output.mkdir(parents=True, exist_ok=True)
    archive = output / f"XrayMini-windows-x64-{revision[:12]}.zip"
    sums = output / "SHA256SUMS"
    if archive.exists() or sums.exists():
        raise ValueError("Output already exists; use a new empty output directory")
    created = False
    created_sums = False
    try:
        with archive.open("xb") as raw:
            created = True
            with zipfile.ZipFile(raw, "w", compression=zipfile.ZIP_DEFLATED) as z:
                for name, data in sorted(payload.items()):
                    info = zipfile.ZipInfo(f"XrayMini/{name}", date_time=(2026, 1, 1, 0, 0, 0))
                    info.compress_type = zipfile.ZIP_DEFLATED
                    info.external_attr = 0o100644 << 16
                    z.writestr(info, data)
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        with sums.open("x", encoding="utf-8", newline="\n") as f:
            created_sums = True
            f.write(f"{digest}  {archive.name}\n")
    except Exception:
        if created:
            archive.unlink(missing_ok=True)
        if created_sums:
            sums.unlink(missing_ok=True)
        raise
    return archive


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bin", default="bin", type=Path)
    parser.add_argument("--output", default="dist-demo", type=Path)
    args = parser.parse_args(argv)
    try:
        revision = clean_revision(ROOT)
        archive = create_archive(args.bin, args.output, revision, ROOT / "docs/DEMO-QUICKSTART.txt")
        print(f"Unsigned demo package: {archive}\nSource commit: {revision}")
        print("No new execution, transport or GUI acceptance is claimed by packaging.")
        return 0
    except (OSError, ValueError, subprocess.SubprocessError, package_runtime.PackageError) as error:
        print(f"[DEMO_PACKAGE_FAILED] {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
