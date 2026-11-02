import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react';
import { desktop, initialStatus, type ImportPreview, type LogEntry, type Profile, type Status, type SubscriptionCandidate } from './bridge';

type IconName = 'power' | 'plus' | 'copy' | 'shield' | 'arrow' | 'back' | 'info' | 'check' | 'close';
function Icon({ name }: { name: IconName }) {
  const paths: Record<IconName, ReactNode> = {
    power: <><path d="M12 3v9"/><path d="M6 5.5a9 9 0 1 0 12 0"/></>,
    plus: <path d="M12 5v14M5 12h14"/>, copy: <><rect x="8" y="8" width="12" height="13" rx="2"/><path d="M16 8V3H3v13h5"/></>,
    shield: <><path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z"/><path d="m8 12 3 3 5-6"/></>,
    arrow: <path d="m9 5 7 7-7 7"/>, back: <path d="m15 18-6-6 6-6M9 12h11"/>,
    info: <><circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/></>,
    check: <path d="m5 12 4 4L19 6"/>,
    close: <path d="m6 6 12 12M6 18 18 6"/>,
  };
  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">{paths[name]}</svg>;
}
function Modal({ title, children, onClose, busy, recovery }: { title: string; children: ReactNode; onClose: () => void; busy: boolean; recovery?: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const dialog = ref.current!; dialog.showModal(); return () => dialog.close(); }, []);
  return <dialog ref={ref} aria-labelledby="dialog-title" onCancel={e => { e.preventDefault(); if (!busy) onClose(); }}>
    <header className="modal-header"><h2 id="dialog-title">{title}</h2><button className="icon-button" aria-label="Close dialog" disabled={busy} onClick={onClose}><Icon name="close"/></button></header>{recovery}{children}
  </dialog>;
}
const processLabels: Record<Status['processState'], string> = { stopped: 'Not running', preparing: 'Starting…', running: 'Running', stopping: 'Stopping…', error: 'Error' };
const healthLabels: Record<Status['networkHealth'], string> = { unknown: 'Not verified', checking: 'Checking…', healthy: 'Reachable', unhealthy: 'Check failed' };
function logTime(value: string) {
  const time = new Date(value);
  return Number.isNaN(time.getTime()) ? value : time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
}

export default function App() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [status, setStatus] = useState<Status>(initialStatus);
  const [statusReadState, setStatusReadState] = useState<'loading' | 'unavailable' | 'ready'>('loading');
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [logsState, setLogsState] = useState<'loading' | 'unavailable' | 'ready'>('loading');
  const [available, setAvailable] = useState(Boolean(desktop()));
  const [busy, setBusy] = useState('');
  const [connectionBusy, setConnectionBusy] = useState('');
  const [error, setError] = useState('');
  const [connectionError, setConnectionError] = useState('');
  const [syncError, setSyncError] = useState('');
  const [notice, setNotice] = useState('');
  const [modal, setModal] = useState<'delete' | 'rename' | null>(null);
  const [screen, setScreen] = useState<'home' | 'details' | 'profiles' | 'diagnostics' | 'import'>('home');
  const [profilesBack, setProfilesBack] = useState<'home' | 'details'>('home');
  const [importBack, setImportBack] = useState<'details' | 'profiles'>('details');
  const [name, setName] = useState('');
  const [config, setConfig] = useState('');
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [acknowledged, setAcknowledged] = useState(false);
  const [importKind, setImportKind] = useState<'json' | 'link'>('json');
  const [source, setSource] = useState('');
  const [candidates, setCandidates] = useState<SubscriptionCandidate[]>([]);
  const [candidateId, setCandidateId] = useState('');
  const [loadingSubscription, setLoadingSubscription] = useState(false);
  const subscriptionGeneration = useRef(0);
  const clearingSubscription = useRef<Promise<void>>(Promise.resolve());
  const alive = useRef(false);
  const lock = useRef(false);
  const connectionLock = useRef(false);
  const revision = useRef(0);
  const logsRevision = useRef(0);
  const pollBusy = useRef(false);
  const messagesRef = useRef<HTMLDivElement>(null);
  const importFormRef = useRef<HTMLFormElement>(null);
  const selected = profiles.find(p => p.id === selectedId);
  const statusKnown = available && statusReadState === 'ready';
  const statusWaiting = available && statusReadState === 'loading';
  const active = !!status.sessionId || ['preparing', 'running', 'stopping'].includes(status.processState);
  const backendUnavailable = status.state === 'Error' && !status.sessionId && !status.profileId;
  const transition = status.processState === 'preparing' || status.processState === 'stopping';
  const recovery = active && status.state === 'Error';
  const connectionDisabled = !statusKnown || !!connectionBusy || (!recovery && (status.processState === 'stopping' || (active && status.processState === 'stopped'))) || (!active && (!selected || !!syncError || backendUnavailable));
  const connectionLabel = connectionBusy ? `${connectionBusy}…` : recovery ? 'Retry stop' : status.processState === 'stopping' ? 'Stopping…' : status.processState === 'preparing' ? 'Cancel startup' : active && status.processState === 'stopped' ? 'Finishing cleanup…' : active ? 'Disconnect' : 'Connect';

  useEffect(() => { document.documentElement.scrollTop = 0; document.body.scrollTop = 0; }, [screen]);
  useEffect(() => {
    if (screen !== 'home' && status.state === 'Error' && status.message && messagesRef.current) messagesRef.current.scrollTop = 0;
  }, [screen, status.state, status.message]);
  useEffect(() => {
    if (screen === 'home' && connectionError && messagesRef.current) messagesRef.current.scrollTop = 0;
  }, [screen, connectionError]);
  useEffect(() => {
    if (screen === 'import' && error && importFormRef.current) importFormRef.current.scrollTop = 0;
  }, [screen, error]);

  const refreshLogs = useCallback(async () => {
    const version = ++logsRevision.current;
    if (alive.current) setLogsState('loading');
    try {
      const api = desktop();
      if (!api) throw new Error('Desktop bridge unavailable');
      const nextLogs = await api.GetRecentLogs(100);
      if (alive.current && version === logsRevision.current) { setLogs(nextLogs); setLogsState('ready'); }
    } catch {
      if (alive.current && version === logsRevision.current) setLogsState('unavailable');
    }
  }, []);

  const refresh = useCallback(async (force = false) => {
    const api = desktop();
    if (!api) { if (alive.current) setAvailable(false); void refreshLogs(); return; }
    if (alive.current) setAvailable(true);
    if (pollBusy.current && !force) return;
    void refreshLogs();
    const version = ++revision.current;
    pollBusy.current = true;
    if (alive.current) setStatusReadState(previous => previous === 'ready' ? previous : 'loading');
    try {
      // A failed or slow profile/log read must not hide the backend's safe status.
      const statusRequest = api.GetStatus().then(nextStatus => {
        if (alive.current && version === revision.current) {
          setStatus(previous => nextStatus.revision < previous.revision ? previous : nextStatus);
          setStatusReadState('ready');
        }
      }, error => {
        if (alive.current && version === revision.current) setStatusReadState(previous => previous === 'ready' ? previous : 'unavailable');
        throw error;
      });
      const [nextProfiles] = await Promise.all([api.ListProfiles(), statusRequest]);
      if (!alive.current || version !== revision.current) return;
      setAvailable(true); setSyncError(''); setProfiles(nextProfiles);
      setSelectedId(id => nextProfiles.some(p => p.id === id) ? id : nextProfiles[0]?.id || '');
    } catch { if (alive.current && version === revision.current) setSyncError('Unable to refresh desktop state. Try again; the process state shown may be out of date.'); }
    finally { if (version === revision.current) pollBusy.current = false; }
  }, [refreshLogs]);
  useEffect(() => {
    alive.current = true;
    let listening = true;
    let unsubscribe: (() => void) | undefined;
    const recover = () => {
      // A host discovered after mount needs the same listen-before-read order.
      if (!unsubscribe) unsubscribe = desktop()?.SubscribeStatus(next => {
        if (!listening || !alive.current) return;
        setStatus(previous => next.revision < previous.revision ? previous : next);
        setStatusReadState('ready');
        void refresh();
      });
      void refresh(true);
    };
    recover();
    window.addEventListener('focus', recover);
    document.addEventListener('visibilitychange', recover);
    return () => { listening = false; alive.current = false; revision.current++; logsRevision.current++; subscriptionGeneration.current++; const api = desktop(); clearingSubscription.current = clearingSubscription.current.then(() => api?.ClearSubscriptionPreview()).catch(() => {}); pollBusy.current = false; unsubscribe?.(); window.removeEventListener('focus', recover); document.removeEventListener('visibilitychange', recover); };
  }, [refresh]);

  async function perform(label: string, action: () => Promise<void>, connection = false) {
    const commandLock = connection ? connectionLock : lock;
    const updateBusy = connection ? setConnectionBusy : setBusy;
    if (commandLock.current) return;
    commandLock.current = true; updateBusy(label);
    if (connection) setConnectionError('');
    else { setError(''); setNotice(''); }
    try { await action(); }
    catch (e) { if (alive.current) (connection ? setConnectionError : setError)(typeof e === 'string' ? e : e instanceof Error ? e.message : 'The desktop action failed. Please try again.'); }
    finally {
      // Command acceptance must release the controls before unrelated profile I/O.
      // refresh has its own generation/status-revision guards and may finish later.
      commandLock.current = false;
      if (alive.current) { updateBusy(''); void refresh(true); }
    }
  }
  function clearSubscription() {
    subscriptionGeneration.current++; setCandidates([]); setCandidateId(''); setLoadingSubscription(false);
    const api = desktop();
    clearingSubscription.current = clearingSubscription.current.then(() => api?.ClearSubscriptionPreview()).catch(() => {});
  }
  function clearImport() { clearSubscription(); setSource(''); setConfig(''); setName(''); setPreview(null); setAcknowledged(false); setError(''); }
  function closeModal() { if (lock.current) return; clearImport(); setModal(null); }
  function closeImport() { if (lock.current) return; clearImport(); setScreen(importBack); }
  async function loadSubscription() {
    const api = desktop(); if (!api || !source.trim()) return;
    clearSubscription();
    const generation = subscriptionGeneration.current;
    setLoadingSubscription(true); setError('');
    try {
      await clearingSubscription.current;
      if (generation !== subscriptionGeneration.current || !alive.current) return;
      const result = await api.PreviewSubscription(source.trim());
      if (generation !== subscriptionGeneration.current || !alive.current) return;
      setCandidates(result); setCandidateId(''); setName('');
    } catch { if (generation === subscriptionGeneration.current && alive.current) setError('Unable to load profiles. Check the HTTPS source, network, and supported VLESS parameters.'); }
    finally { if (generation === subscriptionGeneration.current && alive.current) setLoadingSubscription(false); }
  }
  async function importProfile(event: FormEvent) {
    event.preventDefault();
    const api = desktop(); if (!api) return;
    if (importKind === 'link') {
      if (!candidateId || !name.trim() || loadingSubscription) return;
      await perform('Importing', async () => {
        const profile = await api.ImportSubscriptionCandidate(candidateId, name.trim());
        if (!alive.current) return;
        clearImport(); setScreen(importBack); setSelectedId(profile.id); setNotice('Profile imported. Select Connect when ready.');
      });
      return;
    }
    try { const parsed: unknown = JSON.parse(config); if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error(); }
    catch { setError('Enter a complete, valid Xray JSON object, or choose Subscription or VLESS link.'); return; }
    await perform('Importing', async () => {
      if (!preview) {
        const next = await api.PreviewImport(config);
        if (!alive.current) return;
        setPreview(next);
        if (next.droppedSections.length) return;
      } else if (preview.droppedSections.length && !acknowledged) return;
      const profile = await api.ImportProfile(name.trim(), config);
      if (!alive.current) return;
      clearImport(); setScreen(importBack); setSelectedId(profile.id); setNotice('Profile imported. Configuration stays on this computer.');
    });
  }
  async function toggleConnection() {
    const api = desktop(); if (!api || (!active && !selected)) return;
    await perform(active ? 'Stopping' : 'Starting', async () => {
      const next = active ? await api.Disconnect() : await api.Connect(selected!.id);
      if (alive.current) setStatus(previous => next.revision < previous.revision ? previous : next);
    }, true);
  }
  async function copyAddress(address: string) {
    try { await navigator.clipboard.writeText(address); if (alive.current) setNotice(`Copied ${address}`); }
    catch { if (alive.current) setError(`Clipboard unavailable. Copy manually: ${address}`); }
  }

  const disconnecting = status.state === 'Disconnecting' || status.processState === 'stopping' || (active && status.processState === 'stopped') || (active && connectionBusy === 'Stopping');
  const starting = !active && connectionBusy === 'Starting';
  const healthy = statusKnown && status.state !== 'Error' && status.processState === 'running' && status.networkHealth === 'healthy';
  const ready = healthy && !disconnecting;
  const checkFailed = statusKnown && status.state !== 'Error' && !disconnecting && status.processState === 'running' && status.networkHealth === 'unhealthy';
  const heading = !statusKnown ? statusWaiting ? 'Checking status…' : 'Status unavailable' : starting ? 'Starting proxy…' : status.state === 'Error' ? 'Connection error' : disconnecting ? 'Disconnecting…' : status.processState === 'preparing' ? 'Starting proxy…' : checkFailed ? 'Connection check failed' : ready ? 'Proxy is ready' : status.processState === 'running' ? status.networkHealth === 'checking' ? 'Checking connection…' : 'Connection not verified' : 'Proxy is off';
  const statusLabel = !statusKnown ? statusWaiting ? 'Checking status…' : 'Status unavailable' : starting ? 'Connecting' : status.state === 'Error' ? status.state : disconnecting ? 'Disconnecting' : checkFailed && status.state !== 'Degraded' ? 'Check failed' : status.state;
  const modalStop = <>{active && <div><button type="button" className="secondary" aria-label={loadingSubscription ? `${connectionLabel} during loading` : undefined} disabled={connectionDisabled} onClick={() => { void toggleConnection(); }}>{connectionLabel}</button>{status.message && <p role="status">{status.message}</p>}</div>}{connectionError && <div className="alert error" role="alert">{connectionError}</div>}</>;
  const activeProfile = profiles.find(profile => profile.id === status.profileId);
  const statusDetail = !statusKnown ? statusWaiting ? 'Waiting for desktop status.' : 'Desktop status is unavailable.' : status.message || (status.processState === 'stopped' ? 'Local proxy is stopped' : 'Process status and network reachability are checked separately.');
  const logsUnavailable = logsState === 'unavailable' || (logsState === 'ready' && !logs.length && (backendUnavailable || statusReadState === 'unavailable'));
  const logsLoading = logsState === 'loading' || (logsState === 'ready' && !logs.length && statusReadState === 'loading');
  const logCount = !logsUnavailable && !logsLoading ? logs.length : '–';
  const openProfiles = (from: 'home' | 'details') => { setProfilesBack(from); setScreen('profiles'); };
  const openImport = () => { setError(''); setImportBack(screen === 'profiles' ? 'profiles' : 'details'); setScreen('import'); };
  const importButton = <button className="add-profile" disabled={!available || !!busy || backendUnavailable} onClick={openImport}><Icon name="plus"/>Import profile</button>;
  const profileCard = (profile: Profile) => <><span className="profile-icon"><Icon name="shield"/></span><span className="profile-text"><strong title={profile.name}>{profile.name}</strong><small>{active && status.profileId === profile.id ? 'Active profile' : 'Xray configuration'}</small></span></>;

  return <div className="app-shell">
    <main>
      {screen === 'home' ? <header className="home-header"><span className={`status-pill ${ready ? 'good' : ''}`}><span className="small-dot"/>{statusLabel}</span><button className="icon-button header-button" aria-label="Open details" onClick={() => setScreen('details')}><Icon name="info"/></button></header> :
      <header className="screen-header"><button className="icon-button header-button" aria-label="Back" disabled={screen === 'import' && !!busy} onClick={() => screen === 'import' ? closeImport() : setScreen(screen === 'profiles' ? profilesBack : screen === 'diagnostics' ? 'details' : 'home')}><Icon name="back"/></button><h1>{screen === 'details' ? 'Details' : screen === 'profiles' ? 'Profiles' : screen === 'import' ? 'Import a profile' : 'Diagnostics'}</h1>{screen === 'profiles' && <span className="header-count">{profiles.length}</span>}{screen === 'diagnostics' && <span className="header-count">{logCount}</span>}</header>}
      <div className={`status-messages ${screen === 'home' ? 'home-messages' : ''}`} ref={messagesRef}>
        {screen !== 'home' && status.state === 'Error' && status.message && <div className="alert error" role="alert">{status.message}</div>}
        {!available && <div className="bridge-banner" role="status"><strong>Desktop bridge unavailable</strong><span>This is a browser preview. Open the Wails desktop app to import profiles and run Xray.</span></div>}
        {connectionError && !modal && <div className="alert error" role="alert">{connectionError}</div>}
        {syncError && <div className="alert error" role="alert">{syncError}<button onClick={() => { void refresh(true); }}>Retry</button></div>}
        {error && !modal && screen !== 'import' && <div className="alert error" role="alert">{error}</div>}
        {notice && <div className="alert notice" role="status">{notice}</div>}
      </div>
      {screen !== 'home' && active && <div className="session-action"><span>Active: {activeProfile?.name || 'Profile unavailable'}</span><button className="secondary" aria-label={screen === 'import' && loadingSubscription ? `${connectionLabel} during loading` : undefined} disabled={connectionDisabled} onClick={() => { void toggleConnection(); }}>{connectionLabel}</button></div>}
      {screen === 'import' && !active && !!busy && <div className="session-action"><span>Selected: {selected?.name || 'No profile selected'}</span><button className="secondary" disabled={connectionDisabled} onClick={() => { void toggleConnection(); }}>{connectionLabel}</button></div>}
      {screen === 'home' && <div className="home-screen"><div className="home-content">
        {selected ? <button className="profile selected" onClick={() => openProfiles('home')}>{profileCard(selected)}<Icon name="arrow"/></button> :
          <button className="profile empty-profile" onClick={() => openProfiles('home')}><span className="profile-icon"><Icon name="plus"/></span><span className="profile-text"><strong>No profile selected</strong><small>Import a profile to get started</small></span><Icon name="arrow"/></button>}
        {active && status.profileId !== selectedId && <p className="active-profile"><span>Selected: <strong title={selected?.name || 'None'}>{selected?.name || 'None'}</strong></span> · <span>Active: <strong title={activeProfile?.name || 'Unavailable'}>{activeProfile?.name || 'Unavailable'}</strong></span></p>}
        <section className="connection-card">
        <button type="button" className={`connection-symbol ${ready ? 'active' : ''}`} aria-label={connectionLabel} title={connectionLabel} disabled={connectionDisabled} onClick={() => { void toggleConnection(); }}><Icon name="power"/></button>
        {statusKnown && (transition || recovery || !!connectionBusy || (active && status.processState === 'stopped') || (status.processState === 'running' && !healthy)) && <span className="connection-action" aria-hidden="true">{connectionLabel}</span>}
        <h1>{heading}</h1>
        <p className="hero-subtitle">SOCKS5 127.0.0.1:10808 · HTTP 127.0.0.1:10809</p>
        </section>
        <div className="state-grid"><div><span className="state-label">Xray process</span><strong><span className={`small-dot ${statusKnown && status.processState === 'running' ? 'good-dot' : ''}`}/>{statusKnown ? processLabels[status.processState] : statusWaiting ? 'Checking…' : 'Unavailable'}</strong></div><div><span className="state-label">Network check</span><strong><span className={`small-dot ${healthy ? 'good-dot' : ''}`}/>{!statusKnown ? statusWaiting ? 'Checking…' : 'Unavailable' : status.processState === 'running' ? healthLabels[status.networkHealth] : 'Not verified'}</strong></div></div>
        {statusKnown && status.state === 'Error' ? <button type="button" className="status-detail status-detail-button" title={statusDetail} aria-label={`${statusDetail} Open details`} onClick={() => setScreen('details')}>{statusDetail}</button> : <p className="status-detail" title={statusDetail}>{statusDetail}</p>}
      </div></div>}
      {screen === 'details' && <div className="detail-screen">
      <span className="eyebrow">Local endpoints</span>
      <section className="endpoints" aria-label="Local proxy addresses">
        {[['SOCKS5', '127.0.0.1:10808'], ['HTTP', '127.0.0.1:10809']].map(([protocol, address]) => <div className="endpoint" key={protocol}><div><span className="eyebrow">{protocol} proxy</span><code>{address}</code></div><button className="icon-button" aria-label={`Copy ${protocol} address`} onClick={() => { void copyAddress(address); }}><Icon name="copy"/></button></div>)}
      </section>
      <p className="routing-note"><Icon name="shield"/>Only apps configured to use these proxies are routed.</p>
      <span className="eyebrow">Profile</span>
      <section className="profile-panel">
        {selected ? <button className="profile selected" onClick={() => openProfiles('details')}>{profileCard(selected)}<Icon name="arrow"/></button> : <button className="profile empty-profile" onClick={() => openProfiles('details')}><span className="profile-text"><strong>No profile selected</strong><small>Choose or import a profile</small></span><Icon name="arrow"/></button>}
        <div className="profile-actions"><span>{profiles.length} {profiles.length === 1 ? 'profile' : 'profiles'}</span>{selected && <><button className="secondary" aria-label="Rename profile" disabled={!available || !!busy} onClick={() => { setName(selected.name); setError(''); setModal('rename'); }}>Rename</button><button className="text-danger" aria-label="Delete profile" disabled={!available || !!busy || (active && status.profileId === selected.id)} onClick={() => { setError(''); setModal('delete'); }}>Delete</button></>}</div>
      </section>
      {importButton}
      <span className="eyebrow runtime-label">Runtime</span>
      <section className="settings-card">
        <p>Official Xray Core <code>{status.xrayVersion || 'Version unavailable'}</code></p>
        <p className="hint">Bundled runtime. Pinned SHA-256 verified before launch. See THIRD-PARTY-NOTICES and licenses.</p>
        <p className="hint">Last proxy check: {status.lastCheckedAt || 'Not checked yet'}. Other destinations may differ.</p>
      </section>
      <button className="diagnostics-link" onClick={() => { setScreen('diagnostics'); void refreshLogs(); }}><span>Diagnostics <span className="header-count">{logCount}</span></span><span className="muted">Recent activity</span><Icon name="arrow"/></button>
      <p className="footer-note"><span className={`small-dot ${healthy ? 'good-dot' : ''}`}/> Local-first · Yours only · Xray Mini 0.1</p>
      </div>}
      {screen === 'profiles' && <div className="profiles-screen">
        <nav className="profile-list" aria-label="Profiles">
          {profiles.map(profile => <button key={profile.id} className={`profile ${profile.id === selectedId ? 'selected' : ''}`} onClick={() => setSelectedId(profile.id)} aria-pressed={profile.id === selectedId}>{profileCard(profile)}{profile.id === selectedId && <Icon name="check"/>}</button>)}
          {!profiles.length && <p className="empty-message">No profiles yet. Import your first Xray profile.</p>}
        </nav>
        {importButton}
        <p className="footer-note">Select a profile to connect.</p>
      </div>}
      {screen === 'diagnostics' && <div className="diagnostics-screen">
        <div className="log-list" aria-label="Recent diagnostic logs">{logsUnavailable ? <p role="status">Unable to load diagnostics.</p> : logsLoading ? <p role="status">Loading diagnostics…</p> : logs.length ? logs.slice(-100).map((entry, index) => <div className="log-entry" key={`${entry.time}-${index}`}><time dateTime={entry.time} title={entry.time}>{logTime(entry.time)}</time><span role="img" aria-label={entry.level} className={`small-dot log-dot ${entry.level === 'info' && entry.message === 'HTTP proxy check passed; other destinations may differ' ? 'good-dot' : ''}`}/><span className="log-message">{entry.level !== 'info' && <span className={`log-level ${entry.level}`}>{entry.level} · </span>}{entry.message}</span></div>) : <p>No activity yet. Runtime events will appear here.</p>}</div>
        <p className="footer-note">Last 100 runtime events since app start.</p>
      </div>}
      {screen === 'import' && <form className="import-screen" ref={importFormRef} onSubmit={event => { void importProfile(event); }}>
      {error && <div className="alert error" role="alert">{error}</div>}
      <div className="import-field"><label htmlFor="import-kind">Import from</label><select id="import-kind" value={importKind} disabled={!!busy} onChange={e => { clearSubscription(); setSource(''); setConfig(''); setName(''); setPreview(null); setAcknowledged(false); setError(''); setImportKind(e.target.value as 'json' | 'link'); }}><option value="json">Xray JSON</option><option value="link">Subscription or VLESS link</option></select></div>
      {importKind === 'json' ? <><p className="modal-description">Paste a complete Xray JSON configuration. Reality and xHTTP settings are preserved; local proxy inbounds are managed by the app.</p>
      <div className="import-field"><label htmlFor="profile-name">Profile name</label><input id="profile-name" autoFocus value={name} onChange={e => setName(e.target.value)} maxLength={80} required disabled={!!busy} placeholder="My connection" autoComplete="off"/></div>
      <div className="import-field import-config"><label htmlFor="profile-json">Xray configuration</label><textarea id="profile-json" value={config} onChange={e => { setConfig(e.target.value); setPreview(null); setAcknowledged(false); }} required disabled={!!busy} placeholder={'{\n  "outbounds": [...]\n}'} spellCheck={false} autoComplete="off" autoCapitalize="off"/></div>
      {preview && preview.droppedSections.length > 0 && <div className="alert notice" role="status"><strong>Only the selected VLESS outbound will be imported.</strong><p>{preview.warning}</p><p>Discarded sections: {preview.droppedSections.join(', ')}</p><label><input type="checkbox" checked={acknowledged} onChange={e => setAcknowledged(e.target.checked)} disabled={!!busy}/>I understand these sections will not be used.</label></div>}
      </> : <><p className="modal-description">Load an HTTPS subscription or one VLESS link, then choose one profile to save. Loading does not connect. The preview expires after five minutes.</p>
      <div className="import-field"><label htmlFor="subscription-source">Subscription URL or VLESS link</label><input id="subscription-source" type="password" autoFocus value={source} onChange={e => { clearSubscription(); setSource(e.target.value); setName(''); setError(''); }} maxLength={65536} disabled={!!busy} autoComplete="off" spellCheck={false} autoCapitalize="off"/></div>
      <div className="subscription-actions"><button type="button" className="secondary" disabled={!!busy || !source.trim()} onClick={() => { void loadSubscription(); }}>{loadingSubscription ? 'Reload profiles' : 'Load profiles'}</button>{loadingSubscription && <button type="button" className="secondary" onClick={clearSubscription}>Cancel loading</button>}</div>
      {loadingSubscription && <><p role="status">Loading profiles…</p>{!active && <button type="button" className="secondary" aria-label="Connect selected profile during loading" disabled={connectionDisabled} onClick={() => { void toggleConnection(); }}>{connectionLabel}</button>}</>}
      {candidates.length > 0 && <><div className="import-field"><label htmlFor="subscription-candidate">Choose a profile</label><select id="subscription-candidate" value={candidateId} disabled={!!busy} onChange={e => { setCandidateId(e.target.value); setName(candidates.find(candidate => candidate.id === e.target.value)?.name || ''); }}><option value="">Select one of {candidates.length} profiles</option>{candidates.map(candidate => <option key={candidate.id} value={candidate.id}>{candidate.name} · {candidate.transport}</option>)}</select></div><div className="import-field"><label htmlFor="profile-name">Profile name</label><input id="profile-name" value={name} onChange={e => setName(e.target.value)} maxLength={80} required disabled={!!busy} autoComplete="off"/></div></>}
      </>}
      <p className="privacy-note"><Icon name="shield"/><span>Contains credentials. {importKind === 'link' && 'HTTPS sources are requested only when you load. '}Only the chosen profile is stored locally; pasted text is cleared on save or when you go back.</span></p>
      <button className="primary import-submit" type="submit" disabled={!!busy || !name.trim() || (importKind === 'link' ? !candidateId || loadingSubscription : !config.trim() || (!!preview?.droppedSections.length && !acknowledged))}>{busy ? 'Checking…' : importKind === 'json' && preview?.droppedSections.length ? 'Confirm import' : 'Import profile'}</button>
      </form>}
    </main>

    {modal === 'rename' && selected && <Modal title="Rename profile" onClose={closeModal} busy={!!busy} recovery={modalStop}><form onSubmit={e => { e.preventDefault(); const api = desktop(); if (api) void perform('Renaming', async () => { await api.RenameProfile(selected.id, name.trim()); if (alive.current) { setModal(null); setName(''); setNotice('Profile renamed.'); } }); }}><label htmlFor="rename-name">Profile name</label><input id="rename-name" autoFocus value={name} onChange={e => setName(e.target.value)} maxLength={80} disabled={!!busy} required/>{error && <div className="alert error" role="alert">{error}</div>}<div className="modal-actions"><button type="button" className="secondary" onClick={closeModal} disabled={!!busy}>Cancel</button><button className="primary" disabled={!!busy || !name.trim()}>Save name</button></div></form></Modal>}
    {modal === 'delete' && selected && <Modal title="Delete this profile?" onClose={closeModal} busy={!!busy} recovery={modalStop}><p className="modal-description">Remove <strong>{selected.name}</strong> and its saved configuration from this computer? This cannot be undone.</p>{error && <div className="alert error" role="alert">{error}</div>}<div className="modal-actions"><button className="secondary" onClick={closeModal} disabled={!!busy} autoFocus>Keep profile</button><button className="danger" disabled={!!busy} onClick={() => { const api = desktop(); if (api) void perform('Deleting', async () => { await api.DeleteProfile(selected.id); if (alive.current) { setModal(null); setNotice('Profile deleted.'); } }); }}>{busy ? 'Deleting…' : 'Delete profile'}</button></div></Modal>}
  </div>;
}
