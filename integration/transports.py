#!/usr/bin/env python3
"""Real Linux Xray loopback E2E, never a Windows product-runtime test.
Requires Python 3, curl, openssl, Go 1.26.10, and official release ZIP + .dgst.
No public test endpoints, credentials, privilege, or OS proxy/trust changes.
"""
import argparse
import hashlib
import http.server
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import tempfile
import threading
import time
import uuid
import zipfile

VERSION = '26.10.29'
ARCHIVE_SHA256 = '8a5f3e6a1057e4bd0f8a7c57c9d91204b4be0b74222a6e210d070603101163c2'
ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from run_evidence import (add_artifact, error_category, finish_record, new_record,
                          utc_now, write_record)

def run(args, **kwargs):
    return subprocess.run([str(x) for x in args], check=True, timeout=90,
                          capture_output=True, **kwargs)

def port():
    with socket.socket() as s:
        s.bind(('127.0.0.1', 0))
        return s.getsockname()[1]

def wait_port(n, proc):
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline:
        if proc.poll() is not None:
            raise RuntimeError('fixture exited before readiness')
        try:
            with socket.create_connection(('127.0.0.1', n), timeout=.2):
                return
        except OSError:
            time.sleep(.05)
    raise TimeoutError('fixture readiness timeout')

def stop(p):
    if p.poll() is None:
        p.terminate()
        try:
            p.wait(timeout=5)
        except subprocess.TimeoutExpired:
            p.kill()
            p.wait(timeout=5)

def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--archive', required=True, type=Path)
    ap.add_argument('--digest', required=True, type=Path)
    ap.add_argument('--go', default='go')
    ap.add_argument('--results', type=Path)
    args = ap.parse_args()
    # Every attempted run gets a fresh record, including failures before setup.
    output = args.results or ROOT / 'integration' / (
        'results-linux-amd64-' + utc_now().replace(':', '-') + '-' + uuid.uuid4().hex + '.json')
    if output.exists():
        ap.error('results path already exists; select a new evidence path')
    results = {}
    cleanup = {'fixture_processes_reaped': True, 'log_files_closed': True,
               'http_server_closed': True, 'temporary_directory_removed': True}
    failure = None
    try:
        results = new_record(ROOT, command=['python', 'integration/transports.py'],
            scope='linux-loopback-fixture', record=results,
            artifacts={'archive': args.archive, 'digest_file': args.digest},
            tools={'go': args.go, 'curl': 'curl', 'openssl': 'openssl'})
        results.update(engine=VERSION, platform='Linux amd64 fixture only')
        for name in ('engine_executable', 'prepareconfig_adapter', 'tls_target'):
            results['artifacts'][name] = {'status': 'unavailable', 'sha256': None}
        run_fixture(args, results, cleanup)
        if not all(cleanup.values()):
            raise RuntimeError('fixture cleanup failed')
    except BaseException as error:
        failure = error
        results['tests'].append({'scenario': results.get('_active_scenario', 'fixture-setup'),
                                 'outcome': 'FAIL', 'failure_category': error_category(error)})
        raise
    finally:
        results.pop('_active_scenario', None)
        finish_record(results, outcome='PASS' if failure is None else 'FAIL',
                      cleanup=cleanup, error=failure)
        write_record(results, output)
        print(json.dumps(results, indent=2), flush=True)


def run_fixture(args, results, cleanup):
    archive = args.archive.read_bytes()
    digest = hashlib.sha256(archive).hexdigest()
    assert digest == ARCHIVE_SHA256, 'official release archive pin mismatch'
    assert f'SHA2-256= {digest}' in args.digest.read_text(), 'official .dgst mismatch'
    results['archive_sha256'] = digest
    processes, logs = [], []
    target = None
    temporary = tempfile.TemporaryDirectory(prefix='xray-loopback-')
    work = Path(temporary.name)
    try:
        binary = work / 'xray'
        with zipfile.ZipFile(args.archive) as z:
            binary.write_bytes(z.read('xray'))
        binary.chmod(0o700)
        results['executable_sha256'] = hashlib.sha256(binary.read_bytes()).hexdigest()
        version_output = run([binary, 'version']).stdout.decode().splitlines()[0]
        assert 'Xray '+VERSION in version_output
        results['engine_version'] = VERSION
        add_artifact(results, 'engine_executable', binary)
        adapter = work / 'prepareconfig'
        run([args.go, 'build', '-o', adapter, './integration/prepareconfig'], cwd=ROOT)
        add_artifact(results, 'prepareconfig_adapter', adapter)
        cert, key = work/'cert.pem', work/'key.pem'
        run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
             '-keyout', key, '-out', cert, '-days', '1', '-subj', '/CN=localhost',
             '-addext', 'subjectAltName=DNS:localhost'])
        keys = run([binary, 'x25519']).stdout.decode().splitlines()
        private = next(x.split(': ',1)[1] for x in keys if x.startswith('PrivateKey:'))
        public = next(x.split(': ',1)[1] for x in keys if x.startswith('Password (PublicKey):'))
        token = uuid.uuid4().hex.encode()
        class Handler(http.server.BaseHTTPRequestHandler):
            hits = 0
            def do_GET(self):
                Handler.hits += 1
                self.send_response(200)
                self.send_header('Content-Length', str(len(token)))
                self.end_headers()
                self.wfile.write(token)
            def log_message(self, *_): pass
        target = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        threading.Thread(target=target.serve_forever, daemon=True).start()
        url = f'http://127.0.0.1:{target.server_port}/probe'
        def launch(label, command, env=None):
            f = open(work/(label+'.log'), 'wb')
            logs.append((label, f))
            p = subprocess.Popen([str(x) for x in command], stdout=f, stderr=f, env=env)
            processes.append(p)
            return p
        def engine(label, cfg, env=None):
            path = work/(label+'.json')
            path.write_text(json.dumps(cfg))
            run([binary, 'run', '-test', '-c', path], env=env)
            return launch(label, [binary, 'run', '-c', path], env)
        tls_port = port()
        tls_binary = work / 'tls-target'
        run([args.go, 'build', '-o', tls_binary, './integration/tlstarget'], cwd=ROOT)
        add_artifact(results, 'tls_target', tls_binary)
        tls = launch('reality-target', [tls_binary, str(tls_port), cert, key])
        wait_port(tls_port, tls)
        for network, security in [('raw','reality'), ('xhttp','reality'), ('xhttp','tls')]:
            label = network+'-'+security
            results['_active_scenario'] = label+'-setup'
            server_port = port()
            uid = str(uuid.uuid4())
            server_stream = {'network':network, 'security':security}
            client_stream = {'network':network, 'security':security}
            if network == 'xhttp':
                for st in (server_stream, client_stream):
                    st['xhttpSettings'] = {'path':'/fixture', 'mode':'auto'}
            if security == 'reality':
                server_stream['realitySettings'] = {
                    'target':f'127.0.0.1:{tls_port}', 'serverNames':['localhost'],
                    'privateKey':private, 'shortIds':['0123456789abcdef']}
                client_stream['realitySettings'] = {
                    'serverName':'localhost', 'fingerprint':'chrome',
                    'password':public, 'shortId':'0123456789abcdef'}
            else:
                server_stream['tlsSettings'] = {'certificates':[
                    {'certificateFile':str(cert), 'keyFile':str(key)}], 'alpn':['h2']}
                client_stream['tlsSettings'] = {'serverName':'localhost', 'alpn':['h2'], 'useSystemCA':True}
            user = {'id':uid, 'encryption':'none'}
            peer = {'id':uid}
            if network == 'raw':
                user['flow'] = peer['flow'] = 'xtls-rprx-vision'
            server = engine(label+'-server', {'log':{'loglevel':'warning'},
                'inbounds':[{'listen':'127.0.0.1','port':server_port,'protocol':'vless',
                    'settings':{'clients':[peer], 'decryption':'none'}, 'streamSettings':server_stream}],
                'outbounds':[{'protocol':'freedom', 'settings':{'finalRules':[
                    {'action':'allow', 'ip':['127.0.0.1'], 'port':str(target.server_port)}]}}]})
            wait_port(server_port, server)
            imported = {'outbounds':[{'protocol':'vless', 'settings':{'vnext':[
                {'address':'127.0.0.1','port':server_port,'users':[user]}]},
                'streamSettings':client_stream}, {'protocol':'freedom','tag':'untrusted-direct'}],
                'routing':{'rules':[{'type':'field','network':'tcp','outboundTag':'untrusted-direct'}]}}
            prepared = json.loads(run([adapter], input=json.dumps(imported).encode()).stdout)
            assert len(prepared['outbounds']) == 1
            assert prepared['outbounds'][0]['streamSettings'] == client_stream
            # Only fixture adaptation: avoid claiming the product's fixed ports.
            proxy_ports = {}
            for inbound in prepared['inbounds']:
                inbound['port'] = port()
                proxy_ports[inbound['protocol']] = inbound['port']
            env = os.environ.copy()
            # Process-local test CA trust; verification stays enabled, no OS trust edit.
            env['SSL_CERT_FILE'] = str(cert)
            client = engine(label+'-client', prepared, env)
            for n in proxy_ports.values(): wait_port(n, client)
            def request(protocol):
                proxy = ('socks5h' if protocol == 'socks' else 'http')+'://127.0.0.1:'+str(proxy_ports[protocol])
                return subprocess.run(['curl','--silent','--show-error','--fail', '--noproxy','',
                    '--max-time','6','--proxy',proxy,url], capture_output=True, timeout=8)
            for protocol in ('http','socks'):
                results['_active_scenario'] = label+'-'+protocol+'-request'
                response = request(protocol)
                assert response.returncode == 0 and response.stdout == token, (label, protocol, 'request failed')
                results['tests'].append({'transport':label, 'proxy':protocol, 'request':'PASS', 'outcome':'PASS'})
                print(f'PASS {label} {protocol}: real target response', flush=True)
            stop(server)
            results['_active_scenario'] = label+'-target-still-alive'
            # Target remains reachable, so failure cannot be credited to target shutdown.
            assert run(['curl','--silent','--noproxy','*','--max-time','3',url]).stdout == token
            hits = Handler.hits
            for protocol in ('http','socks'):
                results['_active_scenario'] = label+'-'+protocol+'-no-bypass'
                response = request(protocol)
                assert response.returncode != 0 and response.stdout != token, (label,protocol,'direct bypass')
                assert Handler.hits == hits, 'target reached after VLESS server stopped'
                results['tests'].append({'transport':label, 'proxy':protocol, 'server_stopped_no_bypass':'PASS', 'outcome':'PASS'})
                print(f'PASS {label} {protocol}: stopped VLESS server blocks traffic; target alive', flush=True)
            stop(client)
        results['_active_scenario'] = 'fixture-cleanup'
    finally:
        cleanup_fixtures(processes, logs, target, cleanup)
        try:
            temporary.cleanup()
        except BaseException:
            cleanup['temporary_directory_removed'] = False
        else:
            cleanup['temporary_directory_removed'] = not work.exists()
        results['all_fixture_processes_reaped'] = cleanup['fixture_processes_reaped']


def cleanup_fixtures(processes, logs, target, cleanup):
    # A single failing teardown must not prevent the remaining cleanup attempts.
    for process in reversed(processes):
        try:
            stop(process)
            if process.poll() is None:
                cleanup['fixture_processes_reaped'] = False
        except BaseException:
            cleanup['fixture_processes_reaped'] = False
    for _, stream in logs:
        try:
            stream.close()
        except BaseException:
            cleanup['log_files_closed'] = False
    if target:
        for close in (target.shutdown, target.server_close):
            try:
                close()
            except BaseException:
                cleanup['http_server_closed'] = False

if __name__ == '__main__':
    # Whole-suite ceiling in addition to individual subprocess/socket timeouts.
    def expired(*_): raise TimeoutError('whole-suite 240 second deadline')
    signal.signal(signal.SIGALRM, expired)
    def terminated(*_): raise InterruptedError('suite terminated; cleaning up fixtures')
    signal.signal(signal.SIGTERM, terminated)
    signal.alarm(240)
    main()
