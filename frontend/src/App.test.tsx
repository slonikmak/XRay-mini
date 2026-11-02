// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';
import { desktop, initialStatus, type DesktopAPI } from './bridge';
vi.mock('./bridge', async (importOriginal) => ({ ...await importOriginal<typeof import('./bridge')>(), desktop: vi.fn() }));
const profile = { id: 'local-1', name: 'My profile', createdAt: '2026-10-30T12:00:00Z' };
let api: DesktopAPI;
beforeEach(() => {
  localStorage.clear();
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute('open', ''); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute('open'); };
  api = {
    PreviewSubscription: vi.fn().mockResolvedValue([{ id: 'opaque-1', name: 'Candidate one', transport: 'XHTTP/TLS' }, { id: 'opaque-2', name: 'Candidate two', transport: 'XHTTP/TLS' }]),
    ImportSubscriptionCandidate: vi.fn().mockResolvedValue(profile), ClearSubscriptionPreview: vi.fn().mockResolvedValue(undefined),
    PreviewImport: vi.fn().mockResolvedValue({ droppedSections: [], warning: '' }),
    RenameProfile: vi.fn().mockResolvedValue(profile), SubscribeStatus: vi.fn().mockReturnValue(vi.fn()),
    ListProfiles: vi.fn().mockResolvedValue([profile]), ImportProfile: vi.fn().mockResolvedValue(profile),
    DeleteProfile: vi.fn().mockResolvedValue(undefined), Connect: vi.fn().mockResolvedValue(initialStatus),
    Disconnect: vi.fn().mockResolvedValue(initialStatus), GetStatus: vi.fn().mockResolvedValue(initialStatus), GetRecentLogs: vi.fn().mockResolvedValue([]),
  };
  vi.mocked(desktop).mockReturnValue(api);
});
afterEach(() => { cleanup(); vi.mocked(desktop).mockReturnValue(undefined); vi.restoreAllMocks(); });
function openDetails() {
  const button = screen.queryByRole('button', { name: 'Open details' });
  if (button) fireEvent.click(button);
}
function openImport() {
  openDetails();
  fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
}
describe('desktop UI', () => {
  it('uses one keyboard-operable power circle and disables it without a profile', async () => {
    const user = userEvent.setup();
    const view = render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect(screen.getByRole('heading', { name: 'Proxy is off' })).toBeTruthy();
    expect(document.querySelector('.connection-action')).toBeNull();
    expect(screen.getByText('SOCKS5 127.0.0.1:10808 · HTTP 127.0.0.1:10809')).toBeTruthy();
    const power = screen.getByRole('button', { name: 'Connect' });
    expect(document.querySelectorAll('.connection-symbol')).toHaveLength(1);
    power.focus();
    expect(document.activeElement).toBe(power);
    await user.keyboard('{Enter}');
    expect(api.Connect).toHaveBeenCalledWith(profile.id);
    view.unmount();
    vi.mocked(api.ListProfiles).mockResolvedValue([]);
    render(<App/>);
    await screen.findByRole('button', { name: /No profile selected/ });
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('updates the process and network indicators independently from status events', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    const running = { ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy', sessionId: 'owned', profileId: profile.id };
    act(() => notify(running));
    expect(screen.getByRole('heading', { name: 'Proxy is ready' })).toBeTruthy();
    expect(document.querySelector('.connection-action')).toBeNull();
    expect(screen.getByRole('button', { name: 'Disconnect' }).classList.contains('active')).toBe(true);
    expect(screen.getByText('Running').querySelector('.good-dot')).toBeTruthy();
    expect(screen.getByText('Reachable').querySelector('.good-dot')).toBeTruthy();
    expect(document.querySelector('.status-pill.good')).toBeTruthy();
    act(() => notify({ ...running, revision: 2, networkHealth: 'unhealthy' }));
    expect(screen.getByText('Running').querySelector('.good-dot')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Connection check failed' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Check failed');
    expect(within(document.querySelector('.state-grid') as HTMLElement).getByText('Check failed').querySelector('.good-dot')).toBeNull();
    expect(screen.getByRole('button', { name: 'Disconnect' }).classList.contains('active')).toBe(false);
    expect(document.querySelector('.connection-action')?.textContent).toBe('Disconnect');
    expect(document.querySelector('.status-pill.good')).toBeNull();
    act(() => notify({ ...running, revision: 3, state: 'Disconnecting', processState: 'stopped' }));
    expect((screen.getByRole('button', { name: 'Finishing cleanup…' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText('Not running').querySelector('.good-dot')).toBeNull();
    expect(screen.getByText('Not verified').querySelector('.good-dot')).toBeNull();
    expect(screen.queryByText('Reachable')).toBeNull();
  });

  it.each([
    ['starting', { state: 'Connecting', processState: 'preparing', networkHealth: 'unknown' }, 'Starting proxy…', 'Connecting', 'Cancel startup', false],
    ['checking', { state: 'Connecting', processState: 'running', networkHealth: 'checking' }, 'Checking connection…', 'Connecting', 'Disconnect', false],
    ['unverified', { state: 'Connecting', processState: 'running', networkHealth: 'unknown' }, 'Connection not verified', 'Connecting', 'Disconnect', false],
    ['ready', { state: 'Connected', processState: 'running', networkHealth: 'healthy' }, 'Proxy is ready', 'Connected', 'Disconnect', false],
    ['first failed check', { state: 'Connected', processState: 'running', networkHealth: 'unhealthy' }, 'Connection check failed', 'Check failed', 'Disconnect', false],
    ['degraded', { state: 'Degraded', processState: 'running', networkHealth: 'unhealthy' }, 'Connection check failed', 'Degraded', 'Disconnect', false],
    ['stopping', { state: 'Disconnecting', processState: 'stopping', networkHealth: 'unknown' }, 'Disconnecting…', 'Disconnecting', 'Stopping…', true],
    ['cleanup', { state: 'Disconnecting', processState: 'stopped', networkHealth: 'unknown' }, 'Disconnecting…', 'Disconnecting', 'Finishing cleanup…', true],
    ['retained error', { state: 'Error', processState: 'stopping', networkHealth: 'unknown' }, 'Connection error', 'Error', 'Retry stop', false],
  ])('shows the agreed heading and action for %s', async (_label, state, heading, pill, action, disabled) => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, ...state, revision: 1, sessionId: 'owned', profileId: profile.id });
    render(<App/>);
    expect(await screen.findByRole('heading', { name: heading })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe(pill);
    expect((screen.getByRole('button', { name: action }) as HTMLButtonElement).disabled).toBe(disabled);
  });

  it('waits for a first backend snapshot before claiming the proxy is stopped', async () => {
    let finishStatus!: (status: typeof initialStatus) => void;
    vi.mocked(api.GetStatus).mockReturnValue(new Promise(resolve => { finishStatus = resolve; }));
    render(<App/>);
    expect(screen.getByRole('heading', { name: 'Checking status…' })).toBeTruthy();
    expect(screen.getByText('Waiting for desktop status.')).toBeTruthy();
    expect(screen.getAllByText('Checking…')).toHaveLength(2);
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByText('Disconnected')).toBeNull();
    expect(screen.queryByText('Not running')).toBeNull();
    expect(screen.queryByText('Local proxy is stopped')).toBeNull();
    expect(document.querySelector('.good-dot')).toBeNull();
    await act(async () => finishStatus({ ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy', sessionId: 'owned', profileId: profile.id }));
    expect(await screen.findByRole('heading', { name: 'Proxy is ready' })).toBeTruthy();
    expect((screen.getByRole('button', { name: 'Disconnect' }) as HTMLButtonElement).disabled).toBe(false);
  });

  it('shows a failed first read honestly and accepts a later retained-session recovery event', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.GetStatus).mockRejectedValue(new Error('status unavailable'));
    render(<App/>);
    expect(await screen.findByRole('heading', { name: 'Status unavailable' })).toBeTruthy();
    expect(screen.getByText('Desktop status is unavailable.')).toBeTruthy();
    expect(screen.queryByText('Disconnected')).toBeNull();
    expect(screen.queryByText('Not running')).toBeNull();
    expect(screen.queryByText('Local proxy is stopped')).toBeNull();
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(true);
    act(() => notify({ ...initialStatus, revision: 1, state: 'Error', processState: 'stopping', sessionId: 'retained', profileId: profile.id }));
    const retry = screen.getByRole('button', { name: 'Retry stop' }) as HTMLButtonElement;
    expect(retry.disabled).toBe(false);
    fireEvent.click(retry);
    await waitFor(() => expect(api.Disconnect).toHaveBeenCalledOnce());
  });

  it('shows the power action during cancellation, a pending stop, retained error and cleanup', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    let finishStop!: (status: typeof initialStatus) => void;
    let finishRetry!: (status: typeof initialStatus) => void;
    const starting = { ...initialStatus, revision: 1, state: 'Connecting', processState: 'preparing', sessionId: 'owned', profileId: profile.id };
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.GetStatus).mockResolvedValue(starting);
    vi.mocked(api.Disconnect).mockReturnValueOnce(new Promise(resolve => { finishStop = resolve; })).mockReturnValueOnce(new Promise(resolve => { finishRetry = resolve; }));
    render(<App/>);
    const cancel = await screen.findByRole('button', { name: 'Cancel startup' });
    expect(document.querySelector('.connection-action')?.textContent).toBe('Cancel startup');
    fireEvent.click(cancel);
    expect(document.querySelector('.connection-action')?.textContent).toBe('Stopping…');
    expect(screen.getByRole('heading', { name: 'Disconnecting…' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Disconnecting');
    expect((screen.getByRole('button', { name: 'Stopping…' }) as HTMLButtonElement).disabled).toBe(true);
    await act(async () => finishStop({ ...starting, revision: 2, state: 'Error', processState: 'stopping' }));
    expect(screen.getByRole('button', { name: 'Retry stop' })).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Connection error' })).toBeTruthy();
    expect(document.querySelector('.connection-action')?.textContent).toBe('Retry stop');
    fireEvent.click(screen.getByRole('button', { name: 'Retry stop' }));
    expect(screen.getByRole('heading', { name: 'Connection error' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Error');
    await act(async () => finishRetry({ ...starting, revision: 2, state: 'Error', processState: 'stopping' }));
    act(() => notify({ ...starting, revision: 3, state: 'Disconnecting', processState: 'stopped' }));
    expect(screen.getByRole('heading', { name: 'Disconnecting…' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Disconnecting');
    expect((screen.getByRole('button', { name: 'Finishing cleanup…' }) as HTMLButtonElement).disabled).toBe(true);
    expect(document.querySelector('.connection-action')?.textContent).toBe('Finishing cleanup…');
    expect(document.querySelectorAll('.connection-symbol')).toHaveLength(1);
  });

  it('shows the Details footer version and colors its dot only for healthy running status', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    const footer = screen.getByText(/Local-first · Yours only · Xray Mini 0\.1/);
    const dot = footer.querySelector('.small-dot')!;
    expect(dot.classList.contains('good-dot')).toBe(false);
    const running = { ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy', sessionId: 'owned', profileId: profile.id };
    act(() => notify(running));
    expect(dot.classList.contains('good-dot')).toBe(true);
    act(() => notify({ ...running, revision: 2, networkHealth: 'unhealthy' }));
    expect(dot.classList.contains('good-dot')).toBe(false);
    act(() => notify({ ...running, revision: 3, state: 'Error' }));
    expect(dot.classList.contains('good-dot')).toBe(false);
  });

  it('marks only the actual passed check as a successful diagnostic entry', async () => {
    const stamp = '2026-10-30T12:34:56Z';
    const passed = 'HTTP proxy check passed; other destinations may differ';
    vi.mocked(api.GetRecentLogs).mockResolvedValue([
      { time: stamp, level: 'info', message: 'Preparing local proxy' },
      { time: stamp, level: 'info', message: passed },
      { time: stamp, level: 'error', message: 'Proxy check failed' },
    ]);
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    const prepared = (await screen.findByText('Preparing local proxy')).closest('.log-entry')!;
    const successful = screen.getByText(passed).closest('.log-entry')!;
    const failed = screen.getByText('Proxy check failed').closest('.log-entry')!;
    expect(prepared.querySelector('.good-dot')).toBeNull();
    expect(successful.querySelector('.good-dot')).toBeTruthy();
    expect(failed.querySelector('.good-dot')).toBeNull();
    expect(within(failed as HTMLElement).getByLabelText('error')).toBeTruthy();
    expect(failed.textContent).toContain('error');
    expect(successful.querySelector('time')?.getAttribute('datetime')).toBe(stamp);
  });

  it('keeps the active profile until cleanup and connects the newly selected profile afterwards', async () => {
    const other = { ...profile, id: 'local-2', name: 'Other profile' };
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    const running = { ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy', sessionId: 'owned', profileId: profile.id };
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.ListProfiles).mockResolvedValue([profile, other]);
    vi.mocked(api.GetStatus).mockResolvedValue(running);
    vi.mocked(api.Disconnect).mockResolvedValue({ ...running, revision: 2, state: 'Disconnecting', processState: 'stopped' });
    render(<App/>);
    fireEvent.click(await screen.findByRole('button', { name: /My profile Active profile/ }));
    fireEvent.click(screen.getByRole('button', { name: /Other profile Xray configuration/ }));
    expect(api.Connect).not.toHaveBeenCalled();
    expect(api.Disconnect).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByText('Selected:', { exact: false }).textContent).toContain('Other profile');
    expect(screen.getByText('Selected:', { exact: false }).textContent).toContain('My profile');
    fireEvent.click(screen.getByRole('button', { name: 'Disconnect' }));
    const cleanup = await screen.findByRole('button', { name: 'Finishing cleanup…' });
    expect((cleanup as HTMLButtonElement).disabled).toBe(true);
    expect(api.Connect).not.toHaveBeenCalled();
    act(() => notify({ ...initialStatus, revision: 3 }));
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    expect(api.Connect).toHaveBeenCalledWith(other.id);
  });

  it('navigates details, addresses, diagnostics, profiles and import with a way back', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, xrayVersion: 'pinned' });
    vi.mocked(api.GetRecentLogs).mockResolvedValue([{ time: '12:00', level: 'info', message: 'Safe event' }]);
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    expect(screen.getByRole('heading', { name: 'Details' })).toBeTruthy();
    expect(screen.getByText('pinned')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Copy SOCKS5 address' }));
    fireEvent.click(screen.getByRole('button', { name: 'Copy HTTP address' }));
    await waitFor(() => expect(writeText.mock.calls).toEqual([['127.0.0.1:10808'], ['127.0.0.1:10809']]));
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(screen.getByRole('heading', { name: 'Diagnostics' })).toBeTruthy();
    expect(await screen.findByText('Safe event')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: /My profile Xray configuration/ }));
    expect(screen.getByRole('heading', { name: 'Profiles' })).toBeTruthy();
    openImport();
    expect(screen.getByRole('heading', { name: 'Import a profile' })).toBeTruthy();
    expect(screen.queryByRole('dialog')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Profiles' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Details' })).toBeTruthy();
    openImport();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Details' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('button', { name: 'Open details' })).toBeTruthy();
  });

  it('shows bounded diagnostic logs with readable times and preserves their original timestamps', async () => {
    const stamp = '2026-10-30T12:34:56.123456789Z';
    const entries = Array.from({ length: 101 }, (_, index) => ({ time: stamp, level: 'info', message: `Entry ${index}` }));
    entries.push({ time: 'unrecognized timestamp', level: 'warning', message: 'Fallback time' });
    vi.mocked(api.GetRecentLogs).mockResolvedValue(entries);
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    await screen.findByText('Entry 100');
    expect(api.GetRecentLogs).toHaveBeenCalledWith(100);
    expect(screen.queryByText('Entry 0')).toBeNull();
    expect(screen.getAllByText(new Date(stamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })).length).toBe(99);
    const time = screen.getByText('Entry 100').parentElement!.querySelector('time')!;
    expect(time.getAttribute('datetime')).toBe(stamp);
    expect(time.getAttribute('title')).toBe(stamp);
    expect(screen.getByText('unrecognized timestamp').getAttribute('title')).toBe('unrecognized timestamp');
  });

  it('shows unavailable runtime and empty diagnostics honestly', async () => {
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    expect(screen.getByText('Version unavailable')).toBeTruthy();
    expect(screen.getByText(/Last proxy check: Not checked yet/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(await screen.findByText('No activity yet. Runtime events will appear here.')).toBeTruthy();
    expect(screen.getByText('Last 100 runtime events since app start.')).toBeTruthy();
  });

  it('keeps connection controls usable when a diagnostic read fails', async () => {
    vi.mocked(api.GetRecentLogs).mockRejectedValue(new Error('logs unavailable'));
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(await screen.findByText('Unable to load diagnostics.')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
    expect(screen.queryByText(/Unable to refresh desktop state/)).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false);
  });

  it('reports diagnostics as unavailable when the desktop bridge is missing', async () => {
    vi.mocked(desktop).mockReturnValue(undefined);
    render(<App/>);
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(await screen.findByText('Unable to load diagnostics.')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
  });

  it('does not call an empty log result activity when the backend failed to initialize', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Error', processState: 'error', message: 'Backend startup failed' });
    render(<App/>);
    await screen.findByText('Backend startup failed');
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(await screen.findByText('Unable to load diagnostics.')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
  });

  it('waits for the startup status before presenting an empty log result', async () => {
    let finishStatus!: (value: typeof initialStatus) => void;
    vi.mocked(api.GetStatus).mockReturnValue(new Promise(resolve => { finishStatus = resolve; }));
    render(<App/>);
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    await waitFor(() => expect(api.GetRecentLogs).toHaveBeenCalledTimes(2));
    expect(screen.getByText('Loading diagnostics…')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
    await act(async () => finishStatus({ ...initialStatus, revision: 1, state: 'Error', processState: 'error', message: 'Backend startup failed' }));
    expect(await screen.findByText('Unable to load diagnostics.')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
  });

  it('reports diagnostics unavailable when the initial status cannot be read', async () => {
    vi.mocked(api.GetStatus).mockRejectedValue(new Error('status unavailable'));
    render(<App/>);
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(await screen.findByText('Unable to load diagnostics.')).toBeTruthy();
    expect(screen.queryByText(/No activity yet/)).toBeNull();
  });

  it.each(['fails', 'stalls'])('retains successful logs when profile refresh %s', async outcome => {
    const entry = { time: '12:00', level: 'info', message: 'Independent log' };
    vi.mocked(api.GetRecentLogs).mockResolvedValue([entry]);
    if (outcome === 'fails') vi.mocked(api.ListProfiles).mockRejectedValue(new Error('profiles unavailable'));
    else vi.mocked(api.ListProfiles).mockReturnValue(new Promise(() => {}));
    render(<App/>);
    openDetails();
    const diagnostics = await screen.findByRole('button', { name: /Diagnostics 1/ });
    fireEvent.click(diagnostics);
    expect(await screen.findByText('Independent log')).toBeTruthy();
  });

  it.each(['succeeds', 'fails'])('keeps a newer diagnostic snapshot when an older read %s later', async outcome => {
    let finishOld!: () => void;
    let finishNew!: () => void;
    const old = { time: '11:00', level: 'info', message: 'Old log' };
    const recent = { time: '12:00', level: 'info', message: 'Fresh log' };
    vi.mocked(api.GetRecentLogs)
      .mockReturnValueOnce(new Promise((resolve, reject) => { finishOld = () => outcome === 'fails' ? reject(new Error('old failure')) : resolve([old]); }))
      .mockReturnValueOnce(new Promise(resolve => { finishNew = () => resolve([recent]); }));
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: /Diagnostics/ }));
    expect(screen.getByText('Loading diagnostics…')).toBeTruthy();
    expect(api.GetRecentLogs).toHaveBeenCalledTimes(2);
    await act(async () => finishNew());
    expect(await screen.findByText('Fresh log')).toBeTruthy();
    await act(async () => finishOld());
    expect(screen.getByText('Fresh log')).toBeTruthy();
    expect(screen.queryByText('Old log')).toBeNull();
    expect(screen.queryByText('Unable to load diagnostics.')).toBeNull();
  });

  it.each(['rename', 'import', 'delete', 'subscription'])('can stop while pending %s storage completes', async kind => {
    let finish!: () => void;
    const held = new Promise<typeof profile>(resolve => { finish = () => resolve(profile); });
    const activeProfileId = kind === 'delete' ? 'another-active-profile' : profile.id;
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Connected', processState: 'running', sessionId: 'owned', profileId: activeProfileId });
    const stopMessage = 'Xray stop is not confirmed; process and storage ownership are retained';
    vi.mocked(api.Disconnect).mockResolvedValueOnce({ ...initialStatus, revision: 2, state: 'Error', processState: 'stopping', sessionId: 'owned', profileId: activeProfileId, message: stopMessage }).mockReturnValue(new Promise(() => {}));
    if (kind === 'rename') vi.mocked(api.RenameProfile).mockReturnValue(held);
    else if (kind === 'import') vi.mocked(api.ImportProfile).mockReturnValue(held);
    else if (kind === 'subscription') vi.mocked(api.ImportSubscriptionCandidate).mockReturnValue(held);
    else vi.mocked(api.DeleteProfile).mockReturnValue(held.then(() => undefined));
    render(<App/>); await screen.findByRole('button', { name: /My profile (Active profile|Xray configuration)/ });
    openDetails(); fireEvent.click(screen.getByRole('button', { name: kind === 'rename' ? 'Rename profile' : kind === 'delete' ? 'Delete profile' : 'Import profile' }));
    if (kind === 'delete') fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete profile' }));
    else {
      if (kind === 'subscription') {
        fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
        fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'vless://synthetic-secret' } });
        fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
        await screen.findByLabelText('Choose a profile');
        fireEvent.change(screen.getByLabelText('Choose a profile'), { target: { value: 'opaque-1' } });
      }
      fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Saved name' } });
      if (kind === 'import') fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: '{"outbounds":[]}' } });
      fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    }
    const operation = kind === 'rename' ? api.RenameProfile : kind === 'import' ? api.ImportProfile : kind === 'subscription' ? api.ImportSubscriptionCandidate : api.DeleteProfile;
    await waitFor(() => expect(operation).toHaveBeenCalledOnce());
    const importScreen = kind === 'import' || kind === 'subscription';
    const area = within(importScreen ? document.querySelector('main')! : screen.getByRole('dialog'));
    if (importScreen) {
      expect((screen.getByRole('button', { name: 'Back' }) as HTMLButtonElement).disabled).toBe(true);
      fireEvent.click(screen.getByRole('button', { name: 'Back' }));
      expect(screen.getByRole('heading', { name: 'Import a profile' })).toBeTruthy();
    }
    const stop = area.getByRole('button', { name: /Disconnect/ });
    expect((stop as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(stop); fireEvent.click(stop);
    expect(api.Disconnect).toHaveBeenCalledOnce();
    expect(importScreen ? screen.getByRole('heading', { name: 'Import a profile' }) : screen.getByRole('dialog')).toBeTruthy();
    const retry = await area.findByRole('button', { name: 'Retry stop' });
    expect(importScreen ? area.getByText(stopMessage).textContent : area.getByRole('status').textContent).toBe(stopMessage);
    expect((retry as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(retry); fireEvent.click(retry);
    expect(api.Disconnect).toHaveBeenCalledTimes(2);
    await act(async () => finish());
  });
  it.each(['stopping', 'stopped'])('offers explicit recovery for an owned Error/%s session', async processState => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Error', processState, sessionId: 'retained', profileId: profile.id });
    render(<App/>);
    const retry = await screen.findByRole('button', { name: 'Retry stop' });
    expect((retry as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(retry);
    await waitFor(() => expect(api.Disconnect).toHaveBeenCalledOnce());
    expect(api.Connect).not.toHaveBeenCalled();
  });
  it.each(['error', 'notice'])('preserves profile %s feedback while a concurrent stop fails and is retried', async outcome => {
    let finishProfile!: () => void;
    let failStop!: (error: Error) => void;
    const profileFeedback = outcome === 'error' ? 'Profile save failed' : 'Profile renamed.';
    vi.mocked(api.RenameProfile).mockReturnValue(new Promise((resolve, reject) => {
      finishProfile = () => outcome === 'error' ? reject(new Error(profileFeedback)) : resolve(profile);
    }));
    vi.mocked(api.Disconnect).mockReturnValueOnce(new Promise((_, reject) => { failStop = reject; })).mockReturnValue(new Promise(() => {}));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Connected', processState: 'running', sessionId: 'owned', profileId: profile.id });
    render(<App/>); await screen.findByRole('button', { name: /My profile Active profile/ });
    openDetails(); fireEvent.click(screen.getByRole('button', { name: 'Rename profile' }));
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.RenameProfile).toHaveBeenCalledOnce());
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Disconnect' }));
    await act(async () => finishProfile());
    await screen.findByText(profileFeedback);
    await act(async () => failStop(new Error('Stop request failed')));
    await screen.findByText('Stop request failed');
    if (outcome === 'error') {
      expect(within(screen.getByRole('dialog')).getByText('Stop request failed').getAttribute('role')).toBe('alert');
      fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Disconnect' }));
    } else fireEvent.click(screen.getByRole('button', { name: 'Disconnect' }));
    expect(api.Disconnect).toHaveBeenCalledTimes(2);
    expect(screen.getByText(profileFeedback)).toBeTruthy();
  });
  it('retains disabled cleanup ownership until a newer released-session snapshot', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 2, state: 'Disconnecting', processState: 'stopped', sessionId: 'owned', profileId: profile.id });
    render(<App/>); await screen.findByRole('button', { name: /My profile Active profile/ });
    expect((screen.getByRole('button', { name: 'Finishing cleanup…' }) as HTMLButtonElement).disabled).toBe(true);
    act(() => notify({ ...initialStatus, revision: 1 }));
    expect((screen.getByRole('button', { name: 'Finishing cleanup…' }) as HTMLButtonElement).disabled).toBe(true);
    act(() => notify({ ...initialStatus, revision: 3 }));
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false);
    expect(api.Connect).not.toHaveBeenCalled();
  });
  it('permits profile repair and another connection after a cleaned failure', async () => {
    let finishConnect!: (status: typeof initialStatus) => void;
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Error', processState: 'error', profileId: profile.id });
    vi.mocked(api.Connect).mockReturnValue(new Promise(resolve => { finishConnect = resolve; }));
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails();
    expect((screen.getByRole('button', { name: 'Import profile' }) as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    expect(api.Connect).toHaveBeenCalledWith(profile.id);
    expect(screen.getByRole('heading', { name: 'Starting proxy…' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Connecting');
    await act(async () => finishConnect({ ...initialStatus, revision: 2, state: 'Connecting', processState: 'preparing', sessionId: 'new', profileId: profile.id }));
  });
  it('subscribes once before reading a late bridge snapshot and receives future events', async () => {
    const unsubscribe = vi.fn();
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return unsubscribe; });
    vi.mocked(desktop).mockReturnValue(undefined);
    const view = render(<App/>);
    vi.mocked(desktop).mockReturnValue(api);
    fireEvent.focus(window);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect(api.SubscribeStatus).toHaveBeenCalledOnce();
    expect(vi.mocked(api.SubscribeStatus).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(api.GetStatus).mock.invocationCallOrder[0]);
    act(() => notify({ ...initialStatus, revision: 4, state: 'Connected', processState: 'running', networkHealth: 'healthy' }));
    expect(screen.getByText('Proxy is ready')).toBeTruthy();
    fireEvent.focus(window);
    expect(api.SubscribeStatus).toHaveBeenCalledOnce();
    view.unmount();
    expect(unsubscribe).toHaveBeenCalledOnce();
    act(() => notify({ ...initialStatus, revision: 5 }));
  });
  it('keeps a synchronous subscription event when the following snapshot is older', async () => {
    vi.mocked(api.SubscribeStatus).mockImplementation(notify => {
      notify({ ...initialStatus, revision: 4, state: 'Connected', processState: 'running', networkHealth: 'healthy' });
      return vi.fn();
    });
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect(api.SubscribeStatus).toHaveBeenCalledOnce();
    expect(screen.getByText('Proxy is ready')).toBeTruthy();
  });
  it('loads metadata, requires a selection and imports only the chosen candidate', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
    const source = screen.getByLabelText('Subscription URL or VLESS link') as HTMLInputElement;
    expect(source.type).toBe('password');
    fireEvent.change(source, { target: { value: 'https://example.invalid/private-token' } });
    fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
    await screen.findByLabelText('Choose a profile');
    expect(api.ImportSubscriptionCandidate).not.toHaveBeenCalled();
    expect(api.Connect).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Choose a profile'), { target: { value: 'opaque-2' } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.ImportSubscriptionCandidate).toHaveBeenCalledWith('opaque-2', 'Candidate two'));
    await screen.findByRole('heading', { name: 'Details' });
    openImport();
    expect((screen.getByLabelText('Subscription URL or VLESS link') as HTMLInputElement).value).toBe('');
    expect(localStorage.length).toBe(0);
  });
  it.each(['stopped', 'running', 'preparing'])('offers a reachable %s connection action and cancels loading without late results', async processState => {
    let finish!: (value: Awaited<ReturnType<DesktopAPI['PreviewSubscription']>>) => void;
    vi.mocked(api.PreviewSubscription).mockReturnValue(new Promise(resolve => { finish = resolve; }));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, processState, profileId: processState === 'stopped' ? '' : profile.id });
    render(<App/>); await screen.findByRole('button', { name: /My profile (Xray configuration|Active profile)/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
    fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'vless://synthetic-secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
    await waitFor(() => expect(api.PreviewSubscription).toHaveBeenCalledOnce());
    const connectionAction = screen.getByRole('button', { name: /during loading/ }) as HTMLButtonElement;
    expect(connectionAction.disabled).toBe(false);
    fireEvent.click(connectionAction);
    await waitFor(() => processState === 'stopped' ? expect(api.Connect).toHaveBeenCalledWith(profile.id) : expect(api.Disconnect).toHaveBeenCalledOnce());
    await waitFor(() => expect((screen.getByRole('button', { name: 'Back' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel loading' }));
    await act(async () => finish([{ id: 'late', name: 'Late candidate', transport: 'XHTTP/TLS' }]));
    expect(screen.queryByLabelText('Choose a profile')).toBeNull();
    expect(api.ClearSubscriptionPreview).toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Details' })).toBeTruthy();
  });
  it('awaits all serialized clears before starting a new subscription preview', async () => {
    const finishClears: (() => void)[] = [];
    vi.mocked(api.ClearSubscriptionPreview).mockImplementation(() => new Promise(resolve => { finishClears.push(resolve); }));
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
    fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'https://example.invalid/synthetic' } });
    fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
    for (let count = 1; count <= 3; count++) {
      await waitFor(() => expect(api.ClearSubscriptionPreview).toHaveBeenCalledTimes(count));
      expect(api.PreviewSubscription).not.toHaveBeenCalled();
      await act(async () => finishClears[count - 1]());
    }
    await screen.findByLabelText('Choose a profile');
    expect(api.PreviewSubscription).toHaveBeenCalledOnce();
  });
  it('clears a pending subscription when returning to Profiles and ignores its late result', async () => {
    let finish!: (value: Awaited<ReturnType<DesktopAPI['PreviewSubscription']>>) => void;
    vi.mocked(api.PreviewSubscription).mockReturnValue(new Promise(resolve => { finish = resolve; }));
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: /My profile Xray configuration/ }));
    openImport();
    fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
    fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'vless://synthetic-secret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
    await waitFor(() => expect(api.PreviewSubscription).toHaveBeenCalledOnce());
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Profiles' })).toBeTruthy();
    await act(async () => finish([{ id: 'late', name: 'Late candidate', transport: 'XHTTP/TLS' }]));
    openImport();
    expect((screen.getByLabelText('Subscription URL or VLESS link') as HTMLInputElement).value).toBe('');
    expect(screen.queryByLabelText('Choose a profile')).toBeNull();
  });
  it('returns to Profiles after a successful JSON import without connecting', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: /My profile Xray configuration/ }));
    openImport();
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Imported' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: '{"outbounds":[]}' } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await screen.findByRole('heading', { name: 'Profiles' });
    expect(api.ImportProfile).toHaveBeenCalledOnce();
    expect(api.Connect).not.toHaveBeenCalled();
  });
  it('keeps the selected connection available while an import is pending', async () => {
    let finish!: (value: typeof profile) => void;
    vi.mocked(api.ImportProfile).mockReturnValue(new Promise(resolve => { finish = resolve; }));
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Imported' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: '{"outbounds":[]}' } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.ImportProfile).toHaveBeenCalledOnce());
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    expect(api.Connect).toHaveBeenCalledWith(profile.id);
    await act(async () => finish(profile));
  });
  it('invalidates candidates on source edits and clears the backend preview on unmount', async () => {
    const view = render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Import from'), { target: { value: 'link' } });
    fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'https://example.invalid/a' } });
    fireEvent.click(screen.getByRole('button', { name: 'Load profiles' }));
    await screen.findByLabelText('Choose a profile');
    const beforeEdit = vi.mocked(api.ClearSubscriptionPreview).mock.calls.length;
    fireEvent.change(screen.getByLabelText('Subscription URL or VLESS link'), { target: { value: 'https://example.invalid/b' } });
    expect(screen.queryByLabelText('Choose a profile')).toBeNull();
    await waitFor(() => expect(api.ClearSubscriptionPreview).toHaveBeenCalledTimes(beforeEdit + 1));
    const calls = vi.mocked(api.ClearSubscriptionPreview).mock.calls.length;
    view.unmount();
    await waitFor(() => expect(api.ClearSubscriptionPreview).toHaveBeenCalledTimes(calls + 1));
  });
  it('shows safe startup status even when profiles fail to refresh', async () => {
    const message = 'another Xray Mini instance owns this profile directory';
    vi.mocked(api.ListProfiles).mockRejectedValue(new Error('profile read failed'));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Error', processState: 'error', message });
    render(<App/>);
    await screen.findByText(message);
    expect(screen.getByRole('heading', { name: 'Connection error' })).toBeTruthy();
    openDetails();
    expect((screen.getByRole('button', { name: 'Import profile' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByText('Desktop bridge unavailable')).toBeNull();
  });
  it('keeps a new retained-session error visible above older notices', async () => {
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.ListProfiles).mockRejectedValue(new Error('profile read failed'));
    const writeText = vi.fn().mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('clipboard failed'));
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    render(<App/>);
    await screen.findByText(/Unable to refresh desktop state/);
    openDetails();
    fireEvent.click(screen.getByRole('button', { name: 'Copy SOCKS5 address' }));
    await screen.findByText(/Copied 127\.0\.0\.1:10808/);
    fireEvent.click(screen.getByRole('button', { name: 'Copy HTTP address' }));
    await screen.findByText(/Clipboard unavailable/);
    const messages = document.querySelector('.status-messages') as HTMLElement;
    messages.scrollTop = 100;
    const message = 'Xray stop is not confirmed; process and storage ownership are retained';
    act(() => notify({ ...initialStatus, revision: 1, state: 'Error', processState: 'stopping', sessionId: 'owned', profileId: profile.id, message }));
    expect(messages.firstElementChild?.textContent).toBe(message);
    expect(messages.scrollTop).toBe(0);
    expect(screen.getByRole('button', { name: 'Retry stop' })).toBeTruthy();
  });
  it('applies status without waiting for a pending profile read', async () => {
    let resolveProfiles!: (profiles: typeof profile[]) => void;
    vi.mocked(api.ListProfiles).mockReturnValue(new Promise(resolve => { resolveProfiles = resolve; }));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy' });
    render(<App/>);
    await screen.findByText('Proxy is ready');
    await act(async () => resolveProfiles([profile]));
  });
  it('recognizes a late native bridge even when backend startup failed', async () => {
    vi.mocked(desktop).mockReturnValue(undefined);
    render(<App/>);
    expect(screen.getByText('Desktop bridge unavailable')).toBeTruthy();
    vi.mocked(desktop).mockReturnValue(api);
    vi.mocked(api.ListProfiles).mockRejectedValue(new Error('unavailable'));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Error', processState: 'error', message: 'cannot protect profile directory' });
    fireEvent.focus(window);
    await screen.findByText('cannot protect profile directory');
    await waitFor(() => expect(screen.queryByText('Desktop bridge unavailable')).toBeNull());
  });
  it('shows an honest empty browser preview without simulated profiles or network success', () => {
    vi.mocked(desktop).mockReturnValue(undefined); render(<App/>);
    expect(screen.getByText('Desktop bridge unavailable')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Status unavailable' })).toBeTruthy();
    expect(screen.queryByText('Disconnected')).toBeNull();
    expect(screen.queryByText('Not running')).toBeNull();
    expect(screen.queryByText('Local proxy is stopped')).toBeNull();
    expect(screen.queryByText('My profile')).toBeNull();
    expect(screen.queryByText('Reachable')).toBeNull();
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(true);
  });
  it('renders a real backend profile without claiming process or network success', async () => {
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect(screen.getAllByText('Not running').length).toBeGreaterThan(0);
    expect(screen.getByText('Not verified')).toBeTruthy();
    expect(screen.queryByText('Reachable')).toBeNull();
  });
  it('shows the first failed probe honestly even before the two-failure degraded threshold', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Connected', processState: 'running', networkHealth: 'unhealthy', profileId: profile.id });
    render(<App/>);
    await screen.findByRole('heading', { name: 'Connection check failed' });
    expect(document.querySelector('.status-pill')?.textContent).toBe('Check failed');
    expect(within(document.querySelector('.state-grid') as HTMLElement).getByText('Check failed')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Disconnect' }).classList.contains('active')).toBe(false);
    openDetails();
    expect((screen.getByRole('button', { name: 'Delete profile' }) as HTMLButtonElement).disabled).toBe(true);
  });
  it('rejects a share URL locally without forwarding it or echoing it in an error', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: 'vless://secret-value' } });
    const form = screen.getByLabelText('Profile name').closest('form')!;
    form.scrollTop = 100;
    fireEvent.submit(form);
    expect(form.firstElementChild?.getAttribute('role')).toBe('alert');
    expect(form.scrollTop).toBe(0);
    expect(screen.getByRole('alert').textContent).not.toContain('secret-value');
    expect(api.ImportProfile).not.toHaveBeenCalled();
  });
  it('passes complete JSON unchanged, clears it after save, and never stores it in localStorage', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    const json = '{ "outbounds": [{ "protocol": "vless", "settings": { "id": "private-value" } }] }';
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Imported' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: json } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.ImportProfile).toHaveBeenCalledWith('Imported', json));
    await screen.findByRole('heading', { name: 'Details' });
    await waitFor(() => expect((screen.getByRole('button', { name: 'Import profile' }) as HTMLButtonElement).disabled).toBe(false));
    openImport();
    expect((screen.getByLabelText('Xray configuration') as HTMLTextAreaElement).value).toBe('');
    expect(localStorage.length).toBe(0);
  });
  it('clears secrets after cancel and prevents delete without confirmation', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: 'private-text' } });
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    openImport();
    expect((screen.getByLabelText('Xray configuration') as HTMLTextAreaElement).value).toBe('');
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete profile' }));
    expect(api.DeleteProfile).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Keep profile' }));
    expect(api.DeleteProfile).not.toHaveBeenCalled();
  });
  it('does not submit duplicate connect requests while one is pending', async () => {
    let done!: (value: typeof initialStatus) => void;
    vi.mocked(api.Connect).mockReturnValue(new Promise(resolve => { done = resolve; }));
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    const connect = screen.getByRole('button', { name: 'Connect' });
    fireEvent.click(connect); fireEvent.click(connect);
    expect(api.Connect).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('heading', { name: 'Starting proxy…' })).toBeTruthy();
    expect(document.querySelector('.status-pill')?.textContent).toBe('Connecting');
    expect((connect as HTMLButtonElement).disabled).toBe(true);
    done(initialStatus);
    await screen.findByRole('button', { name: 'Connect' });
  });
  it('rejects older events, recovers snapshot on focus and unsubscribes', async () => {
    const unsubscribe = vi.fn();
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return unsubscribe; });
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 2, state: 'Connected', processState: 'running', networkHealth: 'healthy', profileId: profile.id });
    const mounted = render(<App/>);
    await screen.findByText('Proxy is ready');
    act(() => notify({ ...initialStatus, revision: 1, state: 'Error', processState: 'error', networkHealth: 'unhealthy' }));
    expect(screen.getByText('Proxy is ready')).toBeTruthy();
    fireEvent.focus(window);
    await waitFor(() => expect(api.GetStatus).toHaveBeenCalledTimes(3));
    mounted.unmount();
    expect(unsubscribe).toHaveBeenCalledOnce();
  });
  it('allows cancellation when the backend is preparing', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, processState: 'preparing', profileId: profile.id });
    render(<App/>);
    const cancel = await screen.findByRole('button', { name: 'Cancel startup' });
    expect((cancel as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(cancel);
    await waitFor(() => expect(api.Disconnect).toHaveBeenCalledTimes(1));
  });
  it('uses only the configured bundled runtime without an executable path input', async () => {
    localStorage.clear();
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false);
    expect(screen.queryByLabelText(/executable path/i)).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    expect(api.Connect).toHaveBeenCalledWith(profile.id);
  });
  it('shows backend action failures and re-enables controls', async () => {
    vi.mocked(api.Connect).mockRejectedValue('Xray executable was not found.');
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    await screen.findByText('Xray executable was not found.');
    await waitFor(() => expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false));
  });
  it('requires explicit acknowledgment of discarded sections and invalidates preview when edited', async () => {
    vi.mocked(api.PreviewImport).mockResolvedValue({ droppedSections: ['routing', 'inbounds'], warning: 'Original routes will not be copied.' });
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openImport();
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Imported' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: '{"outbounds":[]}' } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await screen.findByText(/Discarded sections: routing, inbounds/);
    expect(api.ImportProfile).not.toHaveBeenCalled();
    await waitFor(() => expect((screen.getByRole('checkbox') as HTMLInputElement).disabled).toBe(false));
    expect((screen.getByRole('button', { name: 'Confirm import' }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: '{"outbounds":[], "routing":{}}' } });
    expect(screen.queryByRole('checkbox')).toBeNull();
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await screen.findByRole('checkbox');
    await waitFor(() => expect((screen.getByRole('checkbox') as HTMLInputElement).disabled).toBe(false));
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.ImportProfile).toHaveBeenCalledOnce());
  });
  it('renames a profile through the backend', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    openDetails(); fireEvent.click(screen.getByRole('button', { name: 'Rename profile' }));
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'New name' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save name' }));
    await waitFor(() => expect(api.RenameProfile).toHaveBeenCalledWith(profile.id, 'New name'));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });
  it('shows degraded state, pinned version and last-check timestamp', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Degraded', processState: 'running', networkHealth: 'unhealthy', xrayVersion: 'test-pinned', lastCheckedAt: '2026-10-30T12:00:00Z' });
    render(<App/>);
    await screen.findByRole('heading', { name: 'Connection check failed' });
    openDetails();
    expect(screen.getByText('test-pinned')).toBeTruthy();
    expect(screen.getByText(/Last proxy check: 2026-10-30T12:00:00Z/)).toBeTruthy();
  });

  it('keeps a newer event when the initial snapshot resolves late', async () => {
    let resolve!: (status: typeof initialStatus) => void;
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    vi.mocked(api.GetStatus).mockReturnValue(new Promise(done => { resolve = done; }));
    render(<App/>);
    act(() => notify({ ...initialStatus, revision: 9, state: 'Connected', processState: 'running', networkHealth: 'healthy' }));
    expect(screen.getByText('Proxy is ready')).toBeTruthy();
    await act(async () => resolve({ ...initialStatus, revision: 8 }));
    expect(screen.getByText('Proxy is ready')).toBeTruthy();
  });

  it('allows cancellation after Connect acceptance while profile refresh is still blocked', async () => {
    let finishProfiles!: (profiles: typeof profile[]) => void;
    let notify!: Parameters<DesktopAPI['SubscribeStatus']>[0];
    vi.mocked(api.SubscribeStatus).mockImplementation(callback => { notify = callback; return vi.fn(); });
    const starting = { ...initialStatus, revision: 1, state: 'Connecting', processState: 'preparing', profileId: profile.id };
    vi.mocked(api.Connect).mockResolvedValue(starting);
    render(<App/>);
    await screen.findByRole('button', { name: /My profile Xray configuration/ });
    vi.mocked(api.ListProfiles).mockReturnValue(new Promise(resolve => { finishProfiles = resolve; }));
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    await waitFor(() => expect(api.ListProfiles).toHaveBeenCalledTimes(2));
    act(() => notify(starting));
    const cancel = await screen.findByRole('button', { name: 'Cancel startup' });
    expect((cancel as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(cancel);
    await waitFor(() => expect(api.Disconnect).toHaveBeenCalledOnce());
    await act(async () => finishProfiles([profile]));
  });

});
