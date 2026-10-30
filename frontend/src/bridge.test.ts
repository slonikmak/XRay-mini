// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Call, System } from '@wailsio/runtime';
import { desktop, initialStatus } from './bridge';
vi.mock('@wailsio/runtime', () => ({
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
    await expect(desktop()!.Connect('profile-1', 'C:\\Tools\\xray.exe')).resolves.toEqual(initialStatus);
    expect(Call.ByID).toHaveBeenCalledWith(164877153, 'profile-1', 'C:\\Tools\\xray.exe');
  });
  it('normalizes nil Go slices and propagates backend rejection', async () => {
    vi.mocked(System.IsWindows).mockReturnValue(true);
    vi.mocked(Call.ByID).mockResolvedValue(null);
    await expect(desktop()!.ListProfiles()).resolves.toEqual([]);
    await expect(desktop()!.GetLogs()).resolves.toEqual([]);
    vi.mocked(Call.ByID).mockRejectedValue(new Error('Backend unavailable'));
    await expect(desktop()!.Disconnect()).rejects.toThrow('Backend unavailable');
  });
});
