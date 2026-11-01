"""Offline tests for portfolio packaging and fail-closed audit classification."""
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

import check_npm_audit
import package_demo


class AuditTests(unittest.TestCase):
    def report(self, **updates):
        counts = dict(info=0, low=0, moderate=0, high=0, critical=0)
        counts.update(updates)
        counts["total"] = sum(counts.values())
        return {"metadata": {"vulnerabilities": counts}}

    def test_no_findings(self):
        self.assertEqual(check_npm_audit.classify(self.report(), 0)[0], 0)

    def test_moderate_remains_visible(self):
        result, text = check_npm_audit.classify(self.report(moderate=2), 0)
        self.assertEqual(result, 0)
        self.assertIn("moderate=2", text)

    def test_high_and_critical_block(self):
        for severity in ("high", "critical"):
            with self.subTest(severity=severity):
                self.assertEqual(check_npm_audit.classify(self.report(**{severity: 1}), 1)[0], 1)

    def test_missing_or_errored_report_is_not_clean(self):
        for report in (None, {}, [], {"error": "registry down"}, {"metadata": {"vulnerabilities": {"high": 0}}}):
            with self.subTest(report=report):
                self.assertEqual(check_npm_audit.classify(report, 0)[0], 2)

    def test_inconsistent_codes_and_counts(self):
        self.assertEqual(check_npm_audit.classify(self.report(), 1)[0], 2)
        self.assertEqual(check_npm_audit.classify(self.report(high=1), 0)[0], 2)
        self.assertEqual(check_npm_audit.classify(self.report(), 42)[0], 2)
        report = self.report()
        report["metadata"]["vulnerabilities"]["total"] = 3
        self.assertEqual(check_npm_audit.classify(report, 0)[0], 2)

    def test_invalid_count_types(self):
        for value in (-1, "0", False):
            report = self.report()
            report["metadata"]["vulnerabilities"]["high"] = value
            self.assertEqual(check_npm_audit.classify(report, 0)[0], 2)


class PackageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / "bin"
        self.output = self.root / "dist"
        self.guide = self.root / "guide.txt"
        self.guide.write_text("Unsigned demo; no credentials", encoding="utf-8")
        for name in package_demo.FILES:
            path = self.source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(f"synthetic fixture: {name}".encode())
        self.hashes = {self.source / name: hashlib.sha256((self.source / name).read_bytes()).hexdigest() for name in package_demo.FILES}
        self.revision = "a" * 40

    def create(self):
        with patch.object(package_demo.package_runtime, "verify_package", return_value=self.hashes):
            return package_demo.create_archive(self.source, self.output, self.revision, self.guide)

    def test_allowlist_manifest_and_checksum(self):
        (self.source / "private.profile").write_text("SECRET")
        (self.source / "test.exe").write_text("NOT FOR SHIPPING")
        archive = self.create()
        with zipfile.ZipFile(archive) as z:
            expected = {f"XrayMini/{name}" for name in (*package_demo.FILES, "START-HERE.txt", "manifest.json")}
            self.assertEqual(set(z.namelist()), expected)
            manifest = json.loads(z.read("XrayMini/manifest.json"))
            self.assertEqual(manifest["source_commit"], self.revision)
            self.assertTrue(manifest["unsigned"])
            for name, digest in manifest["sha256"].items():
                self.assertEqual(hashlib.sha256(z.read("XrayMini/" + name)).hexdigest(), digest)
        self.assertEqual((self.output / "SHA256SUMS").read_text(), f"{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n")

    def test_changed_verified_bytes_rejected(self):
        (self.source / "XrayMini.exe").write_bytes(b"changed")
        with self.assertRaises(ValueError):
            self.create()
        self.assertFalse(self.output.exists())

    def test_existing_package_not_overwritten(self):
        archive = self.create()
        before = archive.read_bytes()
        with self.assertRaises(ValueError):
            self.create()
        self.assertEqual(archive.read_bytes(), before)

    def test_invalid_revision_rejected(self):
        self.revision = "main"
        with self.assertRaises(ValueError):
            self.create()

    def test_verification_failure_propagates(self):
        with patch.object(package_demo.package_runtime, "verify_package", side_effect=package_demo.package_runtime.PackageError("bad runtime")):
            with self.assertRaises(package_demo.package_runtime.PackageError):
                package_demo.create_archive(self.source, self.output, self.revision, self.guide)

    def test_dirty_checkout_rejected(self):
        with patch.object(package_demo.subprocess, "check_output", side_effect=[self.revision + "\n", " M README.md\n"]):
            with self.assertRaises(ValueError):
                package_demo.clean_revision(self.root)

    def test_clean_checkout_revision(self):
        with patch.object(package_demo.subprocess, "check_output", side_effect=[self.revision + "\n", ""]):
            self.assertEqual(package_demo.clean_revision(self.root), self.revision)


if __name__ == "__main__":
    unittest.main()
