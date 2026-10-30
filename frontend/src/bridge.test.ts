// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Call, System, Events } from '@wailsio/runtime';
import { desktop, initialStatus } from './bridge';
vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn() },
  Call: { ByID: vi.fn() },
  System: { IsWindows: vi.fn(), IsMac: vi.fn(), IsLinux: vi.fn() },
}));
beforeEach(() => { vi.resetAllMocks(); });
describe('Wails v3 service bridge', () => {
  it('does not mistake an imported runtime for a native host', () => {
    expect(desktop()).toBeUndefined();
    expect(Call.ByID).not.toHaveBeenCalled();
  });
  it('uses generated v3 method IDs and preserves arguments', async () => {
    vi.mocked(System.IsWindows).mockReturnValue(true);
    vi.mocked(Call.ByID).mockResolvedValue(initialStatus);
    await expect(desktop()!.Connect('profile-1')).resolves.toEqual(initialStatus);
    expect(Call.ByID).toHaveBeenCalledWith(164877153, 'profile-1');
  });
  it('normalizes nil Go slices and propagates backend rejection', async () => {
    vi.mocked(System.IsWindows).mockReturnValue(true);
    vi.mocked(Call.ByID).mockResolvedValue(null);
    await expect(desktop()!.ListProfiles()).resolves.toEqual([]);
    await expect(desktop()!.GetRecentLogs(100)).resolves.toEqual([]);
    expect(Call.ByID).toHaveBeenLastCalledWith(3603263191, 100);
    vi.mocked(Call.ByID).mockResolvedValue({ droppedSections: null, warning: '' });
    await expect(desktop()!.PreviewImport('{}')).resolves.toEqual({ droppedSections: [], warning: '' });
    expect(Call.ByID).toHaveBeenLastCalledWith(624461070, '{}');
    await desktop()!.RenameProfile('profile-1', 'Renamed');
    expect(Call.ByID).toHaveBeenLastCalledWith(1067637764, 'profile-1', 'Renamed');
    vi.mocked(Call.ByID).mockRejectedValue(new Error('Backend unavailable'));
    await expect(desktop()!.Disconnect()).rejects.toThrow('Backend unavailable');
  });
  it('subscribes to the exact state event and unwraps its data', () => {
    vi.mocked(System.IsWindows).mockReturnValue(true);
    const callback = vi.fn(), unsubscribe = vi.fn();
    vi.mocked(Events.On).mockReturnValue(unsubscribe);
    expect(desktop()!.SubscribeStatus(callback)).toBe(unsubscribe);
    expect(Events.On).toHaveBeenCalledWith('connection:state-changed', expect.any(Function));
    vi.mocked(Events.On).mock.calls[0][1]({ name: 'connection:state-changed', data: initialStatus });
    expect(callback).toHaveBeenCalledWith(initialStatus);
  });

});
