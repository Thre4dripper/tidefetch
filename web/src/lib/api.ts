// REST + WebSocket client for the tidefetch backend.

export interface Task {
  gid: string;
  name: string;
  status: string;
  total: number;
  done: number;
  uploaded: number;
  downSpeed: number;
  upSpeed: number;
  conns: number;
  seeders: number;
  seeding: boolean;
  torrent: boolean;
  dir: string;
  uri?: string;
  errorCode?: string;
  errorMsg?: string;
  numFiles: number;
  progress: number;
  speeds?: number[];
}

export interface Stat {
  downSpeed: number;
  upSpeed: number;
  numActive: number;
  numWaiting: number;
  numStopped: number;
  sessionDown: number;
  sessionUp: number;
  diskFree: number;
  diskTotal: number;
}

export interface StatePayload {
  version: string;
  aria2: string;
  connected: boolean;
  downloadDir: string;
  authEnabled: boolean;
  tasks: Task[];
  stat: Stat;
}

export interface FileInfo {
  index: number;
  path: string;
  length: number;
  done: number;
  selected: boolean;
  uri?: string;
}

export interface PeerInfo {
  ip: string;
  port: number;
  downSpeed: number;
  upSpeed: number;
  seeder: boolean;
  progress: number;
}

export interface TaskDetail {
  task: Task;
  files: FileInfo[];
  peers: PeerInfo[] | null;
  servers: string[] | null;
  pieces: number[] | null;
  speedHistory: number[] | null;
  bt: { infoHash: string; pieceLen: number; numPieces: number };
}

export interface HistoryEntry {
  gid: string;
  name: string;
  url?: string;
  dir: string;
  size: number;
  status: string;
  category: string;
  added: string;
  finished: string;
}

export interface BrowseResult {
  path: string;
  parent: string;
  dirs: { name: string; path: string }[];
  free: number;
  total: number;
  home: string;
  downloadDir: string;
}

export interface ProbeResult {
  filename: string;
  size: number;
  contentType: string;
  resumable: boolean;
  via: string;
  finalUrl: string;
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function req<T>(method: string, url: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined
  });
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const j = await res.json();
      if (j.error) msg = j.error;
    } catch { /* not json */ }
    throw new ApiError(res.status, msg);
  }
  return res.json() as Promise<T>;
}

export const api = {
  login: (password: string) => req<{ ok: boolean }>('POST', '/api/login', { password }),
  logout: () => req<{ ok: boolean }>('POST', '/api/logout'),
  changePassword: (current: string, newPassword: string) =>
    req<{ ok: boolean }>('POST', '/api/password', { current, new: newPassword }),
  state: () => req<StatePayload>('GET', '/api/state'),
  add: (payload: { kind: string; uris?: string[]; payload?: string; options?: Record<string, string> }) =>
    req<{ gids: string[] }>('POST', '/api/add', payload),
  taskAction: (gid: string, action: string, deleteFiles = false) =>
    req<{ ok: boolean }>('POST', `/api/tasks/${gid}/action`, { action, deleteFiles }),
  bulkAction: (action: string) => req<{ ok: boolean }>('POST', '/api/tasks/actions', { action }),
  taskDetail: (gid: string) => req<TaskDetail>('GET', `/api/tasks/${gid}`),
  selectFiles: (gid: string, indices: number[]) =>
    req<{ ok: boolean }>('POST', `/api/tasks/${gid}/files`, { indices }),
  taskOptions: (gid: string) => req<Record<string, string>>('GET', `/api/tasks/${gid}/options`),
  setTaskOptions: (gid: string, opts: Record<string, string>) =>
    req<{ ok: boolean }>('PUT', `/api/tasks/${gid}/options`, opts),
  position: (gid: string, pos: number, how = 'POS_CUR') =>
    req<{ pos: number }>('POST', `/api/tasks/${gid}/position`, { pos, how }),
  globalOptions: () => req<Record<string, string>>('GET', '/api/options'),
  setGlobalOptions: (opts: Record<string, string>) => req<{ ok: boolean }>('PUT', '/api/options', opts),
  history: (q = '', category = '') =>
    req<{ entries: HistoryEntry[]; categories: string[] }>(
      'GET',
      `/api/history?q=${encodeURIComponent(q)}&category=${encodeURIComponent(category)}`
    ),
  deleteHistory: (gid: string) => req<{ ok: boolean }>('DELETE', `/api/history/${gid}`),
  clearHistory: () => req<{ ok: boolean }>('DELETE', '/api/history'),
  browse: (path = '') => req<BrowseResult>('GET', `/api/browse?path=${encodeURIComponent(path)}`),
  mkdir: (path: string, name: string) => req<{ path: string }>('POST', '/api/browse/mkdir', { path, name }),
  probe: (url: string) => req<ProbeResult>('GET', `/api/probe?url=${encodeURIComponent(url)}`)
};

export interface WsDelta {
  type: 'snapshot' | 'delta' | 'conn';
  tasks?: Task[];
  updated?: Task[];
  removed?: string[];
  stat?: Stat;
  connected?: boolean;
}

export function openSocket(onMessage: (m: WsDelta) => void, onDown: () => void): () => void {
  let ws: WebSocket | null = null;
  let closed = false;
  let retry = 800;

  const connect = () => {
    if (closed) return;
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${proto}//${location.host}/api/ws`);
    ws.onmessage = (ev) => {
      retry = 800;
      try {
        onMessage(JSON.parse(ev.data));
      } catch { /* ignore malformed frame */ }
    };
    ws.onclose = () => {
      if (closed) return;
      onDown();
      setTimeout(connect, retry);
      retry = Math.min(retry * 2, 10000);
    };
    ws.onerror = () => ws?.close();
  };
  connect();

  return () => {
    closed = true;
    ws?.close();
  };
}
