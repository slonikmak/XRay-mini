"""Offline provenance checks; these do not constitute transport/native acceptance."""
import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

import run_evidence as evidence

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('transports', ROOT / 'integration' / 'transports.py')
transports = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transports)


class EvidenceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix='run-evidence-test-')
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.git('init', '-q')
        self.git('config', 'user.name', 'Evidence test')
        self.git('config', 'user.email', 'evidence@example.invalid')
        (self.root / 'source.txt').write_text('fixture source\n')
        self.git('add', 'source.txt')
        self.git('commit', '-qm', 'Synthetic provenance fixture')

    def git(self, *args):
        return subprocess.run(['git', *args], cwd=self.root, check=True,
                              capture_output=True, text=True).stdout.strip()

    def record(self):
        return evidence.new_record(self.root, command=['python', 'native_acceptance.py'],
                                   scope='synthetic-offline')

    def test_exact_canonical_app_commit_and_clean_state(self):
        record = self.record()
        self.assertEqual(record['app'], {'status': 'available',
                         'revision': self.git('rev-parse', 'HEAD^{commit}'), 'dirty': False})
        self.assertRegex(record['app']['revision'], r'^[0-9a-f]{40}$')
        record['engine'] = {'revision': 'not-the-app'}
        self.assertNotEqual(record['app']['revision'], record['engine']['revision'])

    def test_tracked_and_untracked_changes_are_dirty_without_contents(self):
        for tracked in (True, False):
            with self.subTest(tracked=tracked):
                path = self.root / ('source.txt' if tracked else 'secret-name.txt')
                path.write_text('SECRET-CONFIG-BODY')
                record = self.record()
                self.assertTrue(record['app']['dirty'])
                self.assertNotIn('SECRET', json.dumps(record))
                self.assertNotIn('secret-name', json.dumps(record))
                self.git('checkout', '--', 'source.txt')
                if not tracked:
                    path.unlink()

    def test_unavailable_git_and_unborn_head_are_explicit(self):
        with patch.object(evidence, '_output', return_value=None):
            self.assertEqual(self.record()['app'],
                             {'status': 'unavailable', 'revision': None, 'dirty': None})
        outside = self.root / 'unborn'
        outside.mkdir()
        subprocess.run(['git', 'init', '-q', str(outside)], check=True, capture_output=True)
        self.assertIsNone(evidence.app_identity(outside)['revision'])

    def test_artifact_hashes_and_absent_tools_do_not_save_paths(self):
        source = self.root / 'source.txt'
        record = evidence.new_record(self.root, command=['python', 'check.py'],
            scope='synthetic-offline', artifacts={'source': source, 'missing': self.root / 'absent'},
            tools={'go': str(self.root / 'missing-go')})
        self.assertEqual(record['artifacts']['source']['sha256'],
                         hashlib.sha256(source.read_bytes()).hexdigest())
        self.assertEqual(record['artifacts']['missing']['status'], 'unavailable')
        self.assertEqual(record['tools']['go']['status'], 'unavailable')
        self.assertEqual(record['tools']['wails']['status'], 'unavailable')
        expected = 'unavailable' if sys.platform == 'win32' else 'not_applicable'
        self.assertEqual(record['tools']['webview2']['status'], expected)
        self.assertNotIn(str(self.root), json.dumps(record))

    def test_version_parsing_saves_only_numeric_version(self):
        for name, output, expected in (
            ('go', 'go version go1.26.10 windows/amd64 SECRET', '1.26.10'),
            ('curl', 'curl 8.10.1 SECRET', '8.10.1'),
            ('openssl', 'OpenSSL 3.5.0 SECRET', '3.5.0'),
        ):
            with self.subTest(name=name), patch.object(evidence, '_output', return_value=output):
                self.assertEqual(evidence.tool_version(name, 'unused'),
                                 {'status': 'available', 'version': expected})
        with patch.object(evidence, '_output', return_value='SECRET-BROKEN-TOOL'):
            self.assertEqual(evidence.tool_version('go', 'unused')['status'], 'unavailable')
        record = self.record()
        evidence.set_version(record, 'webview2', '142.0.1.2')
        self.assertEqual(record['tools']['webview2']['version'], '142.0.1.2')
        with self.assertRaises(ValueError):
            evidence.set_version(record, 'webview2', 'SECRET version')

    def test_go_version_probe_uses_build_root(self):
        def selected_version(command, cwd=None):
            return 'go version go1.26.10 windows/amd64' if cwd == self.root else \
                   'go version go1.25.0 windows/amd64'
        with patch.object(evidence, 'app_identity', return_value={}), \
             patch.object(evidence, '_output', side_effect=selected_version) as output:
            record = evidence.new_record(self.root, command=['python', 'check.py'],
                scope='synthetic-offline', tools={'go': 'selected-go'})
        output.assert_called_once_with(['selected-go', 'version'], cwd=self.root)
        self.assertEqual(record['tools']['go']['version'], '1.26.10')

    def test_environment_records_os_build_beyond_release_label(self):
        with patch.object(evidence.platform, 'release', return_value='11'), \
             patch.object(evidence.platform, 'version', return_value='10.0.26100'):
            record = self.record()
        self.assertEqual(record['environment']['release'], '11')
        self.assertEqual(record['environment']['version'], '10.0.26100')

    def test_failure_categories_exclude_untrusted_messages_and_argv(self):
        errors = (
            (KeyboardInterrupt('SECRET'), 'interrupted'),
            (InterruptedError('SECRET'), 'interrupted'),
            (TimeoutError('SECRET'), 'timeout'),
            (subprocess.TimeoutExpired(['SECRET'], 1), 'timeout'),
            (subprocess.CalledProcessError(1, ['SECRET'], stderr='SECRET'), 'subprocess_failed'),
            (AssertionError('SECRET'), 'assertion_failed'),
            (OSError('SECRET'), 'io_failed'),
            (RuntimeError('SECRET'), 'fixture_failed'),
        )
        for error, category in errors:
            with self.subTest(category=category):
                record = self.record()
                evidence.finish_record(record, outcome='PASS', cleanup={'reaped': False}, error=error)
                self.assertEqual(record['result'], 'FAIL')
                self.assertEqual(record['failure_category'], category)
                self.assertEqual(record['cleanup']['status'], 'FAIL')
                self.assertNotIn('SECRET', json.dumps(record))

    def test_skip_is_never_pass_and_failed_cleanup_forces_failure(self):
        record = self.record()
        evidence.finish_record(record, outcome='SKIP', cleanup={'reaped': True})
        self.assertEqual(record['result'], 'SKIP')
        evidence.finish_record(record, outcome='PASS', cleanup={'reaped': False})
        self.assertEqual(record['result'], 'FAIL')
        self.assertEqual(record['failure_category'], 'cleanup_failed')
        evidence.finish_record(record, outcome='PASS', cleanup={'reaped': True})
        self.assertEqual(record['result'], 'PASS')
        self.assertIsNotNone(record['finished_at_utc'])
        record['tests'].append({'scenario': 'native-only', 'outcome': 'SKIP'})
        evidence.finish_record(record, outcome='PASS', cleanup={'reaped': True})
        self.assertEqual(record['result'], 'SKIP')
        record['tests'].append({'scenario': 'native-only', 'outcome': 'FAIL'})
        evidence.finish_record(record, outcome='PASS', cleanup={'reaped': True})
        self.assertEqual(record['result'], 'FAIL')

    def test_prior_evidence_is_never_overwritten(self):
        path = self.root / 'prior.json'
        before = b'Historical fixture evidence\r\n'
        path.write_bytes(before)
        with self.assertRaises(FileExistsError):
            evidence.write_record(self.record(), path)
        self.assertEqual(path.read_bytes(), before)
        fresh = self.root / 'fresh.json'
        record = self.record()
        evidence.write_record(record, fresh)
        self.assertEqual(json.loads(fresh.read_text()), record)

    def test_scenario_failure_dominates_requested_skip(self):
        record = self.record()
        record['tests'] = [{'scenario': 'failed', 'outcome': 'FAIL'},
                           {'scenario': 'skipped', 'outcome': 'SKIP'}]
        evidence.finish_record(record, outcome='SKIP', cleanup={'reaped': True})
        self.assertEqual(record['result'], 'FAIL')
        self.assertEqual(record['failure_category'], 'scenario_failed')

    def test_harness_failure_and_interrupt_are_saved_offline(self):
        for index, error in enumerate((AssertionError('SECRET-STDERR'), KeyboardInterrupt('SECRET'))):
            with self.subTest(error=type(error).__name__):
                output = self.root / f'failed-{index}.json'
                def fail(args, record, cleanup):
                    record['_active_scenario'] = 'raw-reality-http-request'
                    cleanup['fixture_processes_reaped'] = False
                    raise error
                argv = ['transports.py', '--archive', 'unused', '--digest', 'unused',
                        '--results', str(output)]
                with patch.object(sys, 'argv', argv), \
                     patch.object(transports, 'new_record', return_value=self.record()), \
                     patch.object(transports, 'run_fixture', side_effect=fail), \
                     contextlib.redirect_stdout(io.StringIO()):
                    with self.assertRaises(type(error)):
                        transports.main()
                saved = json.loads(output.read_text())
                self.assertEqual(saved['result'], 'FAIL')
                self.assertEqual(saved['cleanup']['status'], 'FAIL')
                self.assertEqual(saved['tests'][-1]['scenario'], 'raw-reality-http-request')
                self.assertEqual(saved['tests'][-1]['outcome'], 'FAIL')
                self.assertNotIn('SECRET', output.read_text())

    def test_archive_pin_failure_is_recorded_without_running_fixture(self):
        archive = self.root / 'bad.zip'
        archive.write_bytes(b'synthetic-invalid-archive')
        output = self.root / 'bad-archive.json'
        argv = ['transports.py', '--archive', str(archive), '--digest', 'unused',
                '--results', str(output)]
        with patch.object(sys, 'argv', argv), \
             patch.object(transports, 'new_record', return_value=self.record()), \
             contextlib.redirect_stdout(io.StringIO()), \
             patch.object(transports.subprocess, 'Popen', side_effect=AssertionError('must not launch')):
            with self.assertRaisesRegex(AssertionError, 'archive pin mismatch'):
                transports.main()
        saved = json.loads(output.read_text())
        self.assertEqual(saved['failure_category'], 'assertion_failed')
        self.assertEqual(saved['tests'][-1]['scenario'], 'fixture-setup')
        self.assertEqual(saved['cleanup']['status'], 'PASS')

    def test_interruption_during_identity_collection_saves_partial_record(self):
        output = self.root / 'interrupted-identity.json'
        argv = ['transports.py', '--archive', 'unused', '--digest', 'unused',
                '--results', str(output)]
        with patch.object(sys, 'argv', argv), \
             patch.object(evidence, 'app_identity', side_effect=KeyboardInterrupt('SECRET')), \
             contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(KeyboardInterrupt):
                transports.main()
        saved = json.loads(output.read_text())
        self.assertEqual(saved['result'], 'FAIL')
        self.assertEqual(saved['failure_category'], 'interrupted')
        self.assertEqual(saved['app']['status'], 'unavailable')
        self.assertNotIn('SECRET', output.read_text())

    def test_cleanup_continues_after_one_failure(self):
        bad, good = Mock(), Mock()
        good.poll.return_value = None
        # stop() waits on a process; emulate its exit after terminate.
        good.wait.side_effect = lambda **_: setattr(good.poll, 'return_value', 0)
        bad.poll.return_value = None
        bad.terminate.side_effect = OSError('SECRET cleanup error')
        stream, target = Mock(), Mock()
        target.shutdown.side_effect = OSError('SECRET shutdown error')
        cleanup = {'fixture_processes_reaped': True, 'log_files_closed': True,
                   'http_server_closed': True}
        transports.cleanup_fixtures([good, bad], [('log', stream)], target, cleanup)
        self.assertFalse(cleanup['fixture_processes_reaped'])
        self.assertFalse(cleanup['http_server_closed'])
        good.terminate.assert_called_once()
        stream.close.assert_called_once()
        target.server_close.assert_called_once()
        self.assertNotIn('SECRET', json.dumps(cleanup))


if __name__ == '__main__':
    unittest.main()
