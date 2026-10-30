import * as AppService from "../bindings/github.com/slonikmak/XRay-mini/app";
import { System } from "@wailsio/runtime";

export interface Profile { id: string; name: string; createdAt: string }
export interface Status {
  revision: number;
  processState: string;
  networkHealth: string;
  profileId: string;
  message: string;
}
export interface LogEntry { time: string; level: string; message: string }
export interface DesktopAPI {
  ListProfiles(): Promise<Profile[]>;
  ImportProfile(name: string, configJSON: string): Promise<Profile>;
  DeleteProfile(id: string): Promise<void>;
  Connect(id: string, xrayPath: string): Promise<Status>;
  Disconnect(): Promise<Status>;
  GetStatus(): Promise<Status>;
  GetLogs(): Promise<LogEntry[]>;
}
const api: DesktopAPI = {
  ...AppService,
  ListProfiles: async () => (await AppService.ListProfiles()) ?? [],
  GetLogs: async () => (await AppService.GetLogs()) ?? [],
};

// Runtime import alone also works in a normal browser; only the native host
// injects its environment. Never treat the npm runtime itself as a live backend.
export const desktop = (): DesktopAPI | undefined =>
  System.IsWindows() || System.IsMac() || System.IsLinux() ? api : undefined;
export const initialStatus: Status = { revision: 0, processState: 'stopped', networkHealth: 'unknown', profileId: '', message: '' };
