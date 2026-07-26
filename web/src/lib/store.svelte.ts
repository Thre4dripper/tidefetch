// Global reactive store (Svelte 5 runes).
import { api, openSocket, type Task, type Stat, type WsDelta } from './api';

export type Filter = 'all' | 'active' | 'waiting' | 'stopped';

export interface Prefs {
  notify: boolean;
  compact: boolean;
  confirmRemove: boolean;
}

function loadPrefs(): Prefs {
  try {
    return { notify: false, compact: false, confirmRemove: true, ...JSON.parse(localStorage.getItem('tf-prefs') ?? '{}') };
  } catch {
    return { notify: false, compact: false, confirmRemove: true };
  }
}

const MAX_POINTS = 90;

class Store {
  authed = $state<boolean | null>(null); // null = unknown (checking)
  authEnabled = $state(true);
  connected = $state(true);
  version = $state('');
  aria2 = $state('');
  downloadDir = $state('');
  onboarding = $state(false);

  tasks = $state<Task[]>([]);
  stat = $state<Stat>({ downSpeed: 0, upSpeed: 0, numActive: 0, numWaiting: 0, numStopped: 0 });

  filter = $state<Filter>('all');
  search = $state('');
  selectedGid = $state<string | null>(null);
  view = $state<'downloads' | 'history'>('downloads');

  addOpen = $state(false);
  settingsOpen = $state(false);

  prefs = $state<Prefs>(loadPrefs());

  downHistory = $state<number[]>([]);
  upHistory = $state<number[]>([]);

  toasts = $state<{ id: number; kind: 'ok' | 'err'; text: string }[]>([]);

  #toastSeq = 0;
  #closeWs: (() => void) | null = null;

  filtered = $derived.by(() => {
    const q = this.search.trim().toLowerCase();
    return this.tasks.filter((t) => {
      switch (this.filter) {
        case 'active':
          if (t.status !== 'active') return false;
          break;
        case 'waiting':
          if (t.status !== 'waiting' && t.status !== 'paused') return false;
          break;
        case 'stopped':
          if (t.status !== 'complete' && t.status !== 'error' && t.status !== 'removed') return false;
          break;
      }
      if (q && !t.name.toLowerCase().includes(q)) return false;
      return true;
    });
  });

  counts = $derived.by(() => {
    let active = 0, waiting = 0, stopped = 0;
    for (const t of this.tasks) {
      if (t.status === 'active') active++;
      else if (t.status === 'waiting' || t.status === 'paused') waiting++;
      else stopped++;
    }
    return { all: this.tasks.length, active, waiting, stopped };
  });

  selected = $derived.by(() => this.tasks.find((t) => t.gid === this.selectedGid) ?? null);

  async boot() {
    try {
      const st = await api.state();
      this.applyState(st);
      this.authed = true;
      this.startWs();
      if (!localStorage.getItem('tf-onboarded')) {
        this.onboarding = true;
      }
    } catch (e: unknown) {
      if ((e as { status?: number }).status === 401) {
        this.authed = false;
      } else {
        this.authed = true; // reachable but errored; surface via toast
        this.toast('err', String((e as Error).message ?? e));
      }
    }
  }

  async login(password: string) {
    await api.login(password);
    await this.boot();
  }

  async logout() {
    this.#closeWs?.();
    await api.logout();
    this.authed = false;
  }

  savePrefs() {
    localStorage.setItem('tf-prefs', JSON.stringify(this.prefs));
  }

  async enableNotifications(): Promise<boolean> {
    if (!('Notification' in window)) {
      this.toast('err', 'This browser does not support notifications');
      return false;
    }
    const perm = await Notification.requestPermission();
    if (perm !== 'granted') {
      this.toast('err', 'Notification permission was denied');
      return false;
    }
    return true;
  }

  #notifyTransitions(updated: Task[], previous: Map<string, Task>) {
    if (!this.prefs.notify || !('Notification' in window) || Notification.permission !== 'granted') return;
    for (const t of updated) {
      const before = previous.get(t.gid);
      if (!before || before.status === t.status) continue;
      if (t.status === 'complete') {
        new Notification('Download complete', { body: t.name, tag: t.gid });
      } else if (t.status === 'error') {
        new Notification('Download failed', { body: `${t.name}\n${t.errorMsg ?? ''}`, tag: t.gid });
      }
    }
  }

  finishOnboarding() {
    localStorage.setItem('tf-onboarded', '1');
    this.onboarding = false;
  }

  applyState(st: { version: string; aria2: string; connected: boolean; downloadDir: string; authEnabled?: boolean; tasks: Task[]; stat: Stat }) {
    this.version = st.version;
    this.aria2 = st.aria2;
    this.connected = st.connected;
    this.downloadDir = st.downloadDir;
    this.authEnabled = st.authEnabled ?? true;
    this.tasks = st.tasks ?? [];
    this.stat = st.stat;
    this.pushPoint(st.stat);
  }

  startWs() {
    this.#closeWs?.();
    this.#closeWs = openSocket(
      (m: WsDelta) => {
        if (m.type === 'snapshot') {
          this.tasks = m.tasks ?? [];
          if (m.stat) this.stat = m.stat;
          if (m.connected !== undefined) this.connected = m.connected;
        } else if (m.type === 'delta') {
          if (m.updated?.length || m.removed?.length) {
            const byGid = new Map(this.tasks.map((t) => [t.gid, t]));
            this.#notifyTransitions(m.updated ?? [], byGid);
            for (const t of m.updated ?? []) byGid.set(t.gid, t);
            for (const gid of m.removed ?? []) byGid.delete(gid);
            this.tasks = sortTasks([...byGid.values()]);
          }
          if (m.stat) {
            this.stat = m.stat;
            this.pushPoint(m.stat);
          }
        } else if (m.type === 'conn' && m.connected !== undefined) {
          this.connected = m.connected;
        }
      },
      () => {
        this.connected = false;
      }
    );
  }

  pushPoint(s: Stat) {
    const d = [...this.downHistory, s.downSpeed];
    const u = [...this.upHistory, s.upSpeed];
    if (d.length > MAX_POINTS) d.shift();
    if (u.length > MAX_POINTS) u.shift();
    this.downHistory = d;
    this.upHistory = u;
  }

  toast(kind: 'ok' | 'err', text: string) {
    const id = ++this.#toastSeq;
    this.toasts = [...this.toasts, { id, kind, text }];
    setTimeout(() => {
      this.toasts = this.toasts.filter((t) => t.id !== id);
    }, 4200);
  }

  async run(label: string, fn: () => Promise<unknown>) {
    try {
      await fn();
      this.toast('ok', label);
    } catch (e) {
      this.toast('err', `${label} failed: ${(e as Error).message}`);
    }
  }
}

const rank: Record<string, number> = { active: 0, waiting: 1, paused: 2, error: 3, complete: 4, removed: 5 };

function sortTasks(ts: Task[]): Task[] {
  return ts.sort((a, b) => {
    const r = (rank[a.status] ?? 9) - (rank[b.status] ?? 9);
    return r !== 0 ? r : a.name.localeCompare(b.name);
  });
}

export const store = new Store();
