import * as AppService from "../bindings/github.com/slonikmak/XRay-mini/app";
import { System, Events } from "@wailsio/runtime";

export interface Profile { id: string; name: string; createdAt: string }
export interface Status {
  revision: number;
  state: string;
  sessionId: string;
  lastCheckedAt: string;
  xrayVersion: string;
  processState: string;
  networkHealth: string;
  profileId: string;
  message: string;
}
export interface ImportPreview { droppedSections: string[]; warning: string }
export interface SubscriptionCandidate { id: string; name: string; transport: string }
export interface LogEntry { time: string; level: string; message: string }
export interface DesktopAPI {
  ListProfiles(): Promise<Profile[]>;
  ImportProfile(name: string, configJSON: string): Promise<Profile>;
  PreviewImport(configJSON: string): Promise<ImportPreview>;
  PreviewSubscription(source: string): Promise<SubscriptionCandidate[]>;
  ImportSubscriptionCandidate(id: string, name: string): Promise<Profile>;
  ClearSubscriptionPreview(): Promise<void>;
  RenameProfile(id: string, name: string): Promise<Profile>;
  SubscribeStatus(callback: (status: Status) => void): () => void;
  DeleteProfile(id: string): Promise<void>;
  Connect(id: string): Promise<Status>;
  Disconnect(): Promise<Status>;
  GetStatus(): Promise<Status>;
  GetRecentLogs(limit: number): Promise<LogEntry[]>;
}
const api: DesktopAPI = {
  ...AppService,
  SubscribeStatus: callback => Events.On('connection:state-changed', event => callback(event.data as Status)),
  PreviewImport: async (configJSON) => {
    const preview = await AppService.PreviewImport(configJSON);
    return { ...preview, droppedSections: preview.droppedSections ?? [] };
  },
  ListProfiles: async () => (await AppService.ListProfiles()) ?? [],
  PreviewSubscription: async source => (await AppService.PreviewSubscription(source)) ?? [],
  GetRecentLogs: async (limit) => (await AppService.GetRecentLogs(limit)) ?? [],
};

// Runtime import alone also works in a normal browser; only the native host
// injects its environment. Never treat the npm runtime itself as a live backend.
export const desktop = (): DesktopAPI | undefined =>
  System.IsWindows() || System.IsMac() || System.IsLinux() ? api : undefined;
export const initialStatus: Status = { revision: 0, state: 'Disconnected', sessionId: '', lastCheckedAt: '', xrayVersion: '', processState: 'stopped', networkHealth: 'unknown', profileId: '', message: '' };
