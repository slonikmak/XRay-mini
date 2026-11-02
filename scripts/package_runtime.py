#!/usr/bin/env python3
"""Fetch only pinned official Xray bytes, verify archive + executable, package minimal runtime.
Usage: python3 scripts/package_runtime.py [output-directory, default bin]
       python3 scripts/package_runtime.py --verify [output-directory, default bin]
Installation verifies runtime files only. Use --verify to check full package readiness
(GUI, pinned runtime and legal files), offline, without executing any binary.
GUI PE structure validation does not establish signature or source integrity.
Never executes downloaded files. Requires Python 3 standard library only.
"""
import argparse
import hashlib
import io
import pathlib
import shutil
import struct
import sys
import urllib.request
import zipfile

VERSION = "26.10.31"
ARCHIVE_SHA256 = "2e04a9995a6f2ad1b28c0bb25c708ecb2695e43a1c9c7889c4b6ac2b48cf7a6a"
EXE_SHA256 = "d5b558191cccb1e92a5772b68b8cab67717b26f634b0f15153215f65ae982a7e"
URL = f"https://github.com/XTLS/Xray-core/releases/download/v{VERSION}/Xray-windows-64.zip"
LICENSES = pathlib.Path(__file__).resolve().parent.parent / "licenses"


class PackageError(Exception):
    pass


def required_bytes(path):
    try:
        data = path.read_bytes()
    except FileNotFoundError as error:
        raise PackageError(f"[PACKAGE_MISSING] Required file missing: {path}") from error
    except OSError as error:
        raise PackageError(f"[PACKAGE_UNREADABLE] Cannot read {path}: {error}") from error
    if not data:
        raise PackageError(f"[PACKAGE_INVALID] Required file is empty: {path}")
    return data


def verify_runtime(destination):
    runtime = destination / "runtime"
    executable = required_bytes(runtime / "xray.exe")
    if hashlib.sha256(executable).hexdigest() != EXE_SHA256:
        raise PackageError(f"[PACKAGE_DIGEST_MISMATCH] Replace {runtime / 'xray.exe'} with pinned Xray {VERSION}")
    files = {runtime / "xray.exe": hashlib.sha256(executable).hexdigest()}
    for name in ("Xray-core-LICENSE", "NOTICE.md"):
        path = runtime / name
        data = required_bytes(path)
        # Git checkouts may use CRLF while the official archive uses LF.
        if data.replace(b"\r\n", b"\n") != required_bytes(LICENSES / name).replace(b"\r\n", b"\n"):
            raise PackageError(f"[PACKAGE_CONTENT_MISMATCH] Restore required legal text: {path}")
        files[path] = hashlib.sha256(data).hexdigest()
    return files


def verify_package(destination):
    gui = destination / "XrayMini.exe"
    data = required_bytes(gui)
    try:
        pe = struct.unpack_from("<I", data, 0x3C)[0]
        if data[:2] != b"MZ" or pe < 64 or data[pe:pe + 4] != b"PE\0\0":
            raise ValueError
        machine, sections = struct.unpack_from("<HH", data, pe + 4)
        optional_size, flags = struct.unpack_from("<HH", data, pe + 20)
        optional = pe + 24
        table = optional + optional_size
        if (machine != 0x8664 or not sections or optional_size < 112
                or table + sections * 40 > len(data)
                or not flags & 0x0002 or flags & 0x2000
                or struct.unpack_from("<H", data, optional)[0] != 0x20B
                or struct.unpack_from("<H", data, optional + 68)[0] != 2):
            raise ValueError
        # Check section extents so a truncated executable cannot pass on headers alone.
        for section in range(sections):
            size, offset = struct.unpack_from("<II", data, table + section * 40 + 16)
            if size and offset + size > len(data):
                raise ValueError
    except (ValueError, struct.error) as error:
        raise PackageError(f"[PACKAGE_INVALID] Expected a complete Windows amd64 GUI PE executable: {gui}") from error
    files = {gui: hashlib.sha256(data).hexdigest()}
    files.update(verify_runtime(destination))
    return files

def checked(data, digest):
    if hashlib.sha256(data).hexdigest() != digest:
        raise PackageError("[INTEGRITY_FAILED] Integrity check failed; no runtime was installed")
    return data

def install_runtime(destination):
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
    shutil.copyfile(LICENSES / "NOTICE.md", runtime / "NOTICE.md")
    verify_runtime(destination)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("destination", nargs="?", default="bin", type=pathlib.Path)
    parser.add_argument("--verify", action="store_true", help="check the complete package offline; never download or execute files")
    args = parser.parse_args(argv)
    try:
        if args.verify:
            files = verify_package(args.destination)
            print(f"Package ready: {args.destination} (file completeness and pinned Xray {VERSION}; GUI hash is informational)")
            print("GUI PE structure checked; signature/source integrity is not established.")
            for path, digest in files.items():
                print(f"SHA-256 {digest}  {path}")
        else:
            install_runtime(args.destination)
            print(f"Runtime installed and verified: Xray {VERSION}. Full package readiness requires: python scripts/package_runtime.py --verify \"{args.destination}\"")
    except PackageError as error:
        print(str(error), file=sys.stderr)
        return 1
    except (OSError, zipfile.BadZipFile, KeyError) as error:
        print(f"[INSTALL_FAILED] Runtime installation incomplete: {error}. Fix the file/download obstacle and rerun installation, then --verify.", file=sys.stderr)
        return 1
    return 0

if __name__ == "__main__":
    sys.exit(main())
