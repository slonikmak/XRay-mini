"""Offline package checks: synthetic bytes only, no runtime execution or download."""
import contextlib
import hashlib
import io
import pathlib
import struct
import tempfile
import unittest
from unittest import mock
import zipfile

import package_runtime as package


class PackageRuntimeTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.destination = pathlib.Path(temporary.name) / "package"
        self.executable = b"synthetic runtime, never executed"
        # Minimal amd64 PE32+ GUI headers, one section, no executable code.
        self.gui = bytearray(512)
        self.gui[:2] = b"MZ"
        struct.pack_into("<I", self.gui, 0x3C, 64)
        self.gui[64:68] = b"PE\0\0"
        struct.pack_into("<HH", self.gui, 68, 0x8664, 1)
        struct.pack_into("<HH", self.gui, 84, 240, 0x0002)
        struct.pack_into("<H", self.gui, 88, 0x20B)
        struct.pack_into("<H", self.gui, 156, 2)
        struct.pack_into("<II", self.gui, 344, 144, 368)
        self.license = (package.LICENSES / "Xray-core-LICENSE").read_bytes()
        self.notice = (package.LICENSES / "NOTICE.md").read_bytes()
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, "w") as output:
            output.writestr("xray.exe", self.executable)
            output.writestr("LICENSE", self.license)
        self.archive = archive.getvalue()
        self.addCleanup(mock.patch.stopall)
        mock.patch.object(package, "ARCHIVE_SHA256", hashlib.sha256(self.archive).hexdigest()).start()
        mock.patch.object(package, "EXE_SHA256", hashlib.sha256(self.executable).hexdigest()).start()
        self.download = mock.patch.object(
            package.urllib.request, "urlopen", side_effect=lambda *args, **kwargs: io.BytesIO(self.archive)
        ).start()

    def complete_package(self):
        runtime = self.destination / "runtime"
        runtime.mkdir(parents=True, exist_ok=True)
        files = {
            self.destination / "XrayMini.exe": self.gui,
            runtime / "xray.exe": self.executable,
            runtime / "Xray-core-LICENSE": self.license,
            runtime / "NOTICE.md": self.notice,
        }
        for path, data in files.items():
            path.write_bytes(data)
        return files

    def run_cli(self, *flags):
        stdout, stderr = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            result = package.main([str(self.destination), *flags])
        return result, stdout.getvalue(), stderr.getvalue()

    def assert_failed(self, result, category):
        code, stdout, stderr = result
        self.assertEqual(code, 1)
        self.assertNotIn("Package ready", stdout)
        self.assertIn(category, stderr)

    def test_complete_package_offline_reports_every_file_and_hash(self):
        files = self.complete_package()
        code, stdout, stderr = self.run_cli("--verify")
        self.assertEqual((code, stderr), (0, ""))
        self.assertIn("Package ready", stdout)
        self.assertIn("GUI hash is informational", stdout)
        self.assertIn("signature/source integrity is not established", stdout)
        for path, data in files.items():
            self.assertIn(f"SHA-256 {hashlib.sha256(data).hexdigest()}  {path}", stdout)
        self.download.assert_not_called()

    def test_malformed_truncated_and_wrong_target_gui_headers_fail_offline(self):
        wrong_architecture = bytearray(self.gui)
        struct.pack_into("<H", wrong_architecture, 68, 0x014C)
        console = bytearray(self.gui)
        struct.pack_into("<H", console, 156, 3)
        for data in (b"arbitrary text", self.gui[:100], self.gui[:-1], wrong_architecture, console):
            with self.subTest(gui=data[:100]):
                self.complete_package()
                (self.destination / "XrayMini.exe").write_bytes(data)
                self.assert_failed(self.run_cli("--verify"), "PACKAGE_INVALID")
        self.download.assert_not_called()

    def test_every_required_file_missing_or_empty_fails_offline(self):
        for path in self.complete_package():
            for replacement in (None, b""):
                with self.subTest(file=path.name, replacement=replacement):
                    self.complete_package()
                    if replacement is None:
                        path.unlink()
                    else:
                        path.write_bytes(replacement)
                    self.assert_failed(self.run_cli("--verify"), "PACKAGE_MISSING" if replacement is None else "PACKAGE_INVALID")
        self.download.assert_not_called()

    def test_tampered_runtime_and_each_legal_file_fail_offline(self):
        self.complete_package()
        for name, category in (
            ("xray.exe", "PACKAGE_DIGEST_MISMATCH"),
            ("Xray-core-LICENSE", "PACKAGE_CONTENT_MISMATCH"),
            ("NOTICE.md", "PACKAGE_CONTENT_MISMATCH"),
        ):
            with self.subTest(file=name):
                self.complete_package()
                (self.destination / "runtime" / name).write_bytes(b"corrupted")
                self.assert_failed(self.run_cli("--verify"), category)
        self.download.assert_not_called()

    def test_legal_text_accepts_only_line_ending_changes(self):
        files = self.complete_package()
        for path, data in files.items():
            if path.name in ("NOTICE.md", "Xray-core-LICENSE"):
                path.write_bytes(data.replace(b"\r\n", b"\n"))
        self.assertEqual(self.run_cli("--verify")[0], 0)

    def test_unreadable_file_has_actionable_category(self):
        self.complete_package()
        with mock.patch.object(pathlib.Path, "read_bytes", side_effect=PermissionError("access denied")):
            self.assert_failed(self.run_cli("--verify"), "PACKAGE_UNREADABLE")
        self.download.assert_not_called()

    def test_install_cli_remains_runtime_only_and_full_check_requires_gui(self):
        code, stdout, stderr = self.run_cli()
        self.assertEqual((code, stderr), (0, ""))
        self.assertIn("Runtime installed and verified", stdout)
        self.assertIn("--verify", stdout)
        self.assertNotIn("Package ready", stdout)
        self.download.assert_called_once_with(package.URL, timeout=120)
        self.assert_failed(self.run_cli("--verify"), "PACKAGE_MISSING")

    def test_each_ancillary_write_failure_fails_and_retry_recovers(self):
        original_write = pathlib.Path.write_bytes
        for name in ("Xray-core-LICENSE", "NOTICE.md"):
            with self.subTest(file=name):
                self.complete_package()
                target = self.destination / "runtime" / name
                target.unlink()
                (self.destination / "runtime" / "xray.exe").write_bytes(b"old executable")

                def denied_write(path, data):
                    if path == target:
                        raise PermissionError("synthetic write denial")
                    return original_write(path, data)

                obstacle = (
                    mock.patch.object(pathlib.Path, "write_bytes", denied_write)
                    if name == "Xray-core-LICENSE"
                    else mock.patch.object(package.shutil, "copyfile", side_effect=PermissionError("synthetic write denial"))
                )
                with obstacle:
                    self.assert_failed(self.run_cli(), "INSTALL_FAILED")
                self.assertEqual((self.destination / "runtime" / "xray.exe").read_bytes(), self.executable)
                self.assert_failed(self.run_cli("--verify"), "PACKAGE_MISSING")
                self.assertEqual(self.run_cli()[0], 0)
                self.assertEqual(self.run_cli("--verify")[0], 0)

    def test_archive_and_executable_digest_mismatch_preserve_existing_files(self):
        for constant in ("ARCHIVE_SHA256", "EXE_SHA256"):
            with self.subTest(digest=constant):
                self.complete_package()
                runtime = self.destination / "runtime" / "xray.exe"
                runtime.write_bytes(b"previous runtime")
                with mock.patch.object(package, constant, "0" * 64):
                    self.assert_failed(self.run_cli(), "INTEGRITY_FAILED")
                self.assertEqual(runtime.read_bytes(), b"previous runtime")

    def test_corrupt_archive_license_cannot_report_install_success(self):
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, "w") as output:
            output.writestr("xray.exe", self.executable)
            output.writestr("LICENSE", b"wrong legal content")
        self.archive = archive.getvalue()
        with mock.patch.object(package, "ARCHIVE_SHA256", hashlib.sha256(self.archive).hexdigest()):
            self.assert_failed(self.run_cli(), "PACKAGE_CONTENT_MISMATCH")
        self.assert_failed(self.run_cli("--verify"), "PACKAGE_MISSING")


if __name__ == "__main__":
    unittest.main()
