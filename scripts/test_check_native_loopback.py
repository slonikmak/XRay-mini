"""Offline report checks; synthetic events are not native execution evidence."""
import contextlib
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest

import check_native_loopback as check


def event(action, test=None, **fields):
    result = {"Action": action, "Package": check.PACKAGE, **fields}
    if test is not None:
        result["Test"] = test
    return result


def passing_report():
    events = [event("start"), event("run", check.ROOT)]
    for test in check.TESTS[1:]:
        events.extend([event("run", test), event("output", test, Output="fixture log\n"), event("pass", test)])
    return events + [event("pass", check.ROOT), event("output", Output="PASS\n"), event("pass")]


class NativeLoopbackReportTests(unittest.TestCase):
    def verify(self, events):
        check.verify(json.dumps(item) for item in events)

    def test_complete_report_passes(self):
        self.verify(passing_report())

    def test_every_missing_start_or_completion_is_rejected(self):
        events = passing_report()
        for index, item in enumerate(events):
            if item["Action"] != "output":
                with self.subTest(missing=item), self.assertRaises(ValueError):
                    self.verify(events[:index] + events[index + 1:])

    def test_each_skipped_or_failed_test_and_package_is_rejected(self):
        for test in (*check.TESTS, None):
            for action in ("skip", "fail"):
                events = passing_report()
                for item in events:
                    if item["Action"] == "pass" and item.get("Test") == test:
                        item["Action"] = action
                with self.subTest(test=test, action=action), self.assertRaises(ValueError):
                    self.verify(events)

    def test_output_text_cannot_substitute_for_execution(self):
        with self.assertRaises(ValueError):
            self.verify([event("start"), event("output", Output="--- PASS: TestNativeWindowsLoopback\nPASS\n"), event("pass")])

    def test_malformed_empty_truncated_and_wrong_package_reports_fail(self):
        for report in ([], [""], ["not json"], ['{"Action":'], ["null"], ["[]"], ["1"], ['"pass"']):
            with self.subTest(report=report), self.assertRaises(ValueError):
                check.verify(report)
        events = passing_report()
        for item in events:
            item["Package"] = "other/package"
        with self.assertRaises(ValueError):
            self.verify(events)
        for end in range(len(passing_report())):
            with self.subTest(truncated=end), self.assertRaises(ValueError):
                self.verify(passing_report()[:end])

    def test_duplicate_out_of_order_and_malformed_events_fail(self):
        valid = passing_report()
        reports = [
            valid + [event("output", Output="after completion")],
            valid + [event("pass")],
            [valid[0], valid[0], *valid[1:]],
            [valid[0], valid[2], valid[1], *valid[3:]],
            [*valid[:2], event("pass", check.ROOT), *valid[2:]],
            [*valid[:2], event("run", check.ROOT), *valid[2:]],
        ]
        for action, test in (("run", []), ("pass", {}), ("run", "Unknown"), ("pause", check.ROOT)):
            reports.append([*valid[:2], event(action, test), *valid[2:]])
        malformed_output = copy.deepcopy(valid)
        malformed_output[3]["Output"] = None
        reports.append(malformed_output)
        for index, events in enumerate(reports):
            with self.subTest(case=index), self.assertRaises(ValueError):
                self.verify(events)

    def test_cli_preserves_input_and_uses_fixed_diagnostic(self):
        with tempfile.TemporaryDirectory() as temporary:
            report = Path(temporary) / "report.jsonl"
            for contents, expected in (("\n".join(map(json.dumps, passing_report())), 0),
                                       ('{"secret":"do not echo"}', 1)):
                report.write_text(contents, encoding="utf-8-sig")
                before = report.read_bytes()
                stdout, stderr = io.StringIO(), io.StringIO()
                with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                    self.assertEqual(check.main([str(report)]), expected)
                self.assertEqual(report.read_bytes(), before)
                self.assertEqual(stderr.getvalue(), check.FAILURE + "\n" if expected else "")
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(check.main([str(report.with_name("missing"))]), 1)


if __name__ == "__main__":
    unittest.main()
