// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
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
    PreviewImport: vi.fn().mockResolvedValue({ droppedSections: [], warning: '' }),
    RenameProfile: vi.fn().mockResolvedValue(profile), SubscribeStatus: vi.fn().mockReturnValue(vi.fn()),
    ListProfiles: vi.fn().mockResolvedValue([profile]), ImportProfile: vi.fn().mockResolvedValue(profile),
    DeleteProfile: vi.fn().mockResolvedValue(undefined), Connect: vi.fn().mockResolvedValue(initialStatus),
    Disconnect: vi.fn().mockResolvedValue(initialStatus), GetStatus: vi.fn().mockResolvedValue(initialStatus), GetRecentLogs: vi.fn().mockResolvedValue([]),
  };
  vi.mocked(desktop).mockReturnValue(api);
});
afterEach(() => { cleanup(); vi.mocked(desktop).mockReturnValue(undefined); vi.restoreAllMocks(); });
describe('desktop UI', () => {
  it('shows safe startup status even when profiles fail to refresh', async () => {
    const message = 'another Xray Mini instance owns this profile directory';
    vi.mocked(api.ListProfiles).mockRejectedValue(new Error('profile read failed'));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Error', processState: 'error', message });
    render(<App/>);
    await screen.findByText(message);
    expect((screen.getByRole('button', { name: 'Import profile' }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.queryByText('Desktop bridge unavailable')).toBeNull();
  });
  it('applies status without waiting for a pending profile read', async () => {
    let resolveProfiles!: (profiles: typeof profile[]) => void;
    vi.mocked(api.ListProfiles).mockReturnValue(new Promise(resolve => { resolveProfiles = resolve; }));
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, revision: 1, state: 'Connected', processState: 'running', networkHealth: 'healthy' });
    render(<App/>);
    await screen.findByText('Proxy is ready.');
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
    await screen.findByText('Xray is running.');
    expect(screen.getByText('Check failed')).toBeTruthy();
    expect(screen.queryByText('Proxy is ready.')).toBeNull();
    expect((screen.getByRole('button', { name: 'Delete profile' }) as HTMLButtonElement).disabled).toBe(true);
  });
  it('rejects a share URL locally without forwarding it or echoing it in an error', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Test' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: 'vless://secret-value' } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    expect(screen.getByRole('alert').textContent).not.toContain('secret-value');
    expect(api.ImportProfile).not.toHaveBeenCalled();
  });
  it('passes complete JSON unchanged, clears it after save, and never stores it in localStorage', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
    const json = '{ "outbounds": [{ "protocol": "vless", "settings": { "id": "private-value" } }] }';
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'Imported' } });
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: json } });
    fireEvent.submit(screen.getByLabelText('Profile name').closest('form')!);
    await waitFor(() => expect(api.ImportProfile).toHaveBeenCalledWith('Imported', json));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    await waitFor(() => expect((screen.getByRole('button', { name: 'Import profile' }) as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
    expect((screen.getByLabelText('Xray configuration') as HTMLTextAreaElement).value).toBe('');
    expect(localStorage.length).toBe(0);
  });
  it('clears secrets after cancel and prevents delete without confirmation', async () => {
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
    fireEvent.change(screen.getByLabelText('Xray configuration'), { target: { value: 'private-text' } });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
    expect((screen.getByLabelText('Xray configuration') as HTMLTextAreaElement).value).toBe('');
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
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
    await screen.findByText('Proxy is ready.');
    act(() => notify({ ...initialStatus, revision: 1, state: 'Error', processState: 'error', networkHealth: 'unhealthy' }));
    expect(screen.getByText('Proxy is ready.')).toBeTruthy();
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
    fireEvent.click(screen.getByRole('button', { name: 'Import profile' }));
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
    fireEvent.click(screen.getByRole('button', { name: 'Rename profile' }));
    fireEvent.change(screen.getByLabelText('Profile name'), { target: { value: 'New name' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save name' }));
    await waitFor(() => expect(api.RenameProfile).toHaveBeenCalledWith(profile.id, 'New name'));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });
  it('shows degraded state, pinned version and last-check timestamp', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, state: 'Degraded', processState: 'running', networkHealth: 'unhealthy', xrayVersion: 'test-pinned', lastCheckedAt: '2026-10-30T12:00:00Z' });
    render(<App/>);
    await screen.findByText('Degraded');
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
    expect(screen.getByText('Proxy is ready.')).toBeTruthy();
    await act(async () => resolve({ ...initialStatus, revision: 8 }));
    expect(screen.getByText('Proxy is ready.')).toBeTruthy();
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
