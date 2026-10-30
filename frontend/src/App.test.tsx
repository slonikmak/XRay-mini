// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { desktop, initialStatus, type DesktopAPI } from './bridge';
vi.mock('./bridge', async (importOriginal) => ({ ...await importOriginal<typeof import('./bridge')>(), desktop: vi.fn() }));
const profile = { id: 'local-1', name: 'My profile', createdAt: '2026-10-30T12:00:00Z' };
let api: DesktopAPI;
beforeEach(() => {
  localStorage.clear();
  localStorage.setItem('xray-mini.executable', 'C:\\Tools\\xray.exe');
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute('open', ''); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute('open'); };
  api = {
    ListProfiles: vi.fn().mockResolvedValue([profile]), ImportProfile: vi.fn().mockResolvedValue(profile),
    DeleteProfile: vi.fn().mockResolvedValue(undefined), Connect: vi.fn().mockResolvedValue(initialStatus),
    Disconnect: vi.fn().mockResolvedValue(initialStatus), GetStatus: vi.fn().mockResolvedValue(initialStatus), GetLogs: vi.fn().mockResolvedValue([]),
  };
  vi.mocked(desktop).mockReturnValue(api);
});
afterEach(() => { cleanup(); vi.mocked(desktop).mockReturnValue(undefined); vi.restoreAllMocks(); });
describe('desktop UI', () => {
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
  it('keeps running process and failed network check distinct', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, processState: 'running', networkHealth: 'unhealthy', profileId: profile.id });
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
    expect(localStorage.length).toBe(1);
    expect(localStorage.getItem('xray-mini.executable')).not.toContain('private-value');
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
  it('rejects older backend status revisions and removes polling on unmount', async () => {
    const clearTimer = vi.spyOn(window, 'clearInterval');
    vi.mocked(api.GetStatus).mockResolvedValueOnce({ ...initialStatus, revision: 2, processState: 'running', networkHealth: 'healthy', profileId: profile.id })
      .mockResolvedValue({ ...initialStatus, revision: 1, processState: 'error', networkHealth: 'unhealthy' });
    const mounted = render(<App/>);
    await screen.findByText('Proxy is ready.');
    await waitFor(() => expect(api.GetStatus).toHaveBeenCalledTimes(2), { timeout: 2500 });
    expect(screen.getByText('Proxy is ready.')).toBeTruthy();
    expect(screen.queryByText('Check failed')).toBeNull();
    mounted.unmount();
    expect(clearTimer).toHaveBeenCalled();
  });
  it('allows cancellation when the backend is preparing', async () => {
    vi.mocked(api.GetStatus).mockResolvedValue({ ...initialStatus, processState: 'preparing', profileId: profile.id });
    render(<App/>);
    const cancel = await screen.findByRole('button', { name: 'Cancel startup' });
    expect((cancel as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(cancel);
    await waitFor(() => expect(api.Disconnect).toHaveBeenCalledTimes(1));
  });
  it('requires an executable path before connecting', async () => {
    localStorage.clear();
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(true);
  });
  it('shows backend action failures and re-enables controls', async () => {
    vi.mocked(api.Connect).mockRejectedValue('Xray executable was not found.');
    render(<App/>); await screen.findByRole('button', { name: /My profile Xray configuration/ });
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    await screen.findByText('Xray executable was not found.');
    await waitFor(() => expect((screen.getByRole('button', { name: 'Connect' }) as HTMLButtonElement).disabled).toBe(false));
  });
});
