"""Safe identity for a single verification run; this module does not run tests.

Native callers can use new_record(...), append their scenario outcomes to
record['tests'], then finish_record(...) and write_record(...). Command and scope
must be fixed public labels, never reconstructed argv or configuration values.
Artifacts are identified by public labels and hashes; their paths are not saved.
Tool output and exception messages are never copied into evidence.
"""
from datetime import datetime, timezone
import hashlib
import json
import platform
from pathlib import Path
import re
import subprocess
import sys


def utc_now():
    return datetime.now(timezone.utc).isoformat()


def _output(command, cwd=None):
    try:
        return subprocess.run(command, cwd=cwd, check=True, timeout=10,
                              capture_output=True, text=True).stdout.strip()
    except (InterruptedError, TimeoutError):
        raise
    except (OSError, subprocess.SubprocessError, UnicodeError):
        return None


def app_identity(root):
    revision = _output(['git', 'rev-parse', '--verify', 'HEAD^{commit}'], cwd=root)
    if revision is None or not re.fullmatch(r'(?:[0-9a-f]{40}|[0-9a-f]{64})', revision):
        return {'status': 'unavailable', 'revision': None, 'dirty': None}
    status = _output(['git', 'status', '--porcelain=v1', '-z',
                      '--untracked-files=all'], cwd=root)
    return {'status': 'available' if status is not None else 'unavailable',
            'revision': revision, 'dirty': bool(status) if status is not None else None}


def artifact_identity(path):
    try:
        digest = hashlib.sha256()
        with Path(path).open('rb') as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                digest.update(chunk)
        return {'status': 'available', 'sha256': digest.hexdigest()}
    except (InterruptedError, TimeoutError):
        raise
    except OSError:
        return {'status': 'unavailable', 'sha256': None}


def tool_version(name, executable, *, cwd=None):
    # Parse only a numeric version from the expected header, not arbitrary output.
    commands = {
        'go': (['version'], r'go version go(\d+(?:\.\d+){1,3})(?:\s|$)'),
        'curl': (['--version'], r'curl (\d+(?:\.\d+){1,3})(?:\s|$)'),
        'openssl': (['version'], r'OpenSSL (\d+(?:\.\d+){1,3}[a-z]?)(?:\s|$)'),
    }
    arguments, pattern = commands[name]
    output = _output([str(executable), *arguments], cwd=cwd)
    match = re.match(pattern, output or '')
    return {'status': 'available' if match else 'unavailable',
            'version': match[1] if match else None}


def new_record(root, *, command, scope, artifacts=None, tools=None, record=None):
    """Collect current app/platform identity using fixed, non-secret labels.

    tools optionally maps go/curl/openssl to executables; probes run from root
    with the inherited environment, matching fixture builds. Missing native tool or
    WebView2 versions remain explicit unavailable; callers may add an observed
    numeric version with set_version after a separate native inspection.
    Supply a mutable record to retain partial metadata if collection is interrupted.
    """
    if not command or any(not re.fullmatch(r'[A-Za-z0-9_./-]{1,100}', x) for x in command):
        raise ValueError('command must contain fixed public labels without arguments')
    if not re.fullmatch(r'[A-Za-z0-9_-]{1,100}', scope):
        raise ValueError('scope must be a fixed public label')
    record = {} if record is None else record
    record.update({
        'schema_version': 1,
        'started_at_utc': utc_now(),
        'finished_at_utc': None,
        'command': list(command),
        'scope': scope,
        'app': {'status': 'unavailable', 'revision': None, 'dirty': None},
        'environment': {'system': platform.system(), 'release': platform.release(),
                        'version': platform.version(), 'machine': platform.machine()},
        'tools': {
            'python': {'status': 'available', 'version': platform.python_version()},
            'go': {'status': 'unavailable', 'version': None},
            'wails': {'status': 'unavailable', 'version': None},
            'webview2': {'status': 'unavailable' if sys.platform == 'win32'
                        else 'not_applicable', 'version': None},
        },
        'artifacts': {},
        'tests': [],
        'result': 'FAIL',
        'failure_category': 'incomplete',
        'cleanup': {'status': 'unavailable', 'checks': {}},
    })
    record['app'] = app_identity(root)
    for name, executable in (tools or {}).items():
        record['tools'][name] = tool_version(name, executable, cwd=root)
    for name, path in (artifacts or {}).items():
        add_artifact(record, name, path)
    return record


def add_artifact(record, name, path):
    if not re.fullmatch(r'[A-Za-z0-9_-]{1,100}', name):
        raise ValueError('artifact name must be a fixed public label')
    record['artifacts'][name] = artifact_identity(path)


def set_version(record, name, version):
    """Attach a separately observed numeric version, without saving raw output."""
    if name not in record['tools'] or not re.fullmatch(r'\d+(?:\.\d+){1,3}[a-z]?', version):
        raise ValueError('expected a known tool and numeric version')
    record['tools'][name] = {'status': 'available', 'version': version}


def error_category(error):
    if isinstance(error, (KeyboardInterrupt, InterruptedError)):
        return 'interrupted'
    if isinstance(error, (TimeoutError, subprocess.TimeoutExpired)):
        return 'timeout'
    if isinstance(error, subprocess.CalledProcessError):
        return 'subprocess_failed'
    if isinstance(error, AssertionError):
        return 'assertion_failed'
    if isinstance(error, OSError):
        return 'io_failed'
    return 'fixture_failed'


def finish_record(record, *, outcome, cleanup, error=None):
    """Finalize PASS/FAIL/SKIP; cleanup failure or any exception forces FAIL."""
    if outcome not in ('PASS', 'FAIL', 'SKIP') or not cleanup:
        raise ValueError('expected an outcome and explicit cleanup checks')
    scenario_outcomes = {test.get('outcome') for test in record['tests']}
    if 'FAIL' in scenario_outcomes:
        outcome = 'FAIL'
    elif outcome == 'PASS' and 'SKIP' in scenario_outcomes:
        outcome = 'SKIP'
    cleaned = all(value is True for value in cleanup.values())
    record['cleanup'] = {'status': 'PASS' if cleaned else 'FAIL', 'checks': cleanup}
    record['result'] = 'FAIL' if error is not None or not cleaned else outcome
    record['failure_category'] = (error_category(error) if error is not None else
                                  'cleanup_failed' if not cleaned else
                                  'scenario_failed' if outcome == 'FAIL' else None)
    record['finished_at_utc'] = utc_now()


def write_record(record, path):
    """Save to a new path only, preserving all prior evidence byte for byte."""
    with Path(path).open('x', encoding='utf-8', newline='\n') as stream:
        json.dump(record, stream, indent=2)
        stream.write('\n')
