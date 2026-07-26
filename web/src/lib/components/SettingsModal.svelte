<script lang="ts">
  import { onMount } from 'svelte';
  import { store } from '../store.svelte';
  import { api } from '../api';
  import Modal from './Modal.svelte';
  import PasswordInput from './PasswordInput.svelte';

  let { onclose }: { onclose: () => void } = $props();

  type Tab = 'transfer' | 'behaviour' | 'bittorrent' | 'network' | 'interface' | 'advanced' | 'security';
  let tab = $state<Tab>('transfer');

  let opts = $state<Record<string, string>>({});
  let edits = $state<Record<string, string>>({});
  let loaded = $state(false);
  let filter = $state('');

  // Password form
  let pwCurrent = $state('');
  let pwNew = $state('');
  let pwConfirm = $state('');
  let pwBusy = $state(false);

  interface Field {
    key: string;
    label: string;
    kind?: 'select' | 'text';
    choices?: string[];
    hint?: string;
  }

  const TABS: { id: Tab; label: string }[] = [
    { id: 'transfer', label: 'Transfer' },
    { id: 'behaviour', label: 'Behaviour' },
    { id: 'bittorrent', label: 'BitTorrent' },
    { id: 'network', label: 'Network' },
    { id: 'interface', label: 'Interface' },
    { id: 'advanced', label: 'Advanced' },
    { id: 'security', label: 'Security' }
  ];

  const FIELDS: Record<string, Field[]> = {
    transfer: [
      { key: 'max-overall-download-limit', label: 'Global download limit', kind: 'select', choices: ['0', '1M', '2M', '5M', '10M', '20M', '50M'], hint: '0 = unlimited' },
      { key: 'max-overall-upload-limit', label: 'Global upload limit', kind: 'select', choices: ['0', '256K', '512K', '1M', '2M', '5M'], hint: '0 = unlimited' },
      { key: 'max-concurrent-downloads', label: 'Concurrent downloads', kind: 'select', choices: ['1', '2', '3', '5', '8', '10', '16'] },
      { key: 'split', label: 'Connections per download', kind: 'select', choices: ['1', '4', '8', '16', '32', '64'] },
      { key: 'max-connection-per-server', label: 'Connections per server', kind: 'select', choices: ['1', '2', '4', '8', '16'] },
      { key: 'min-split-size', label: 'Min split size', kind: 'select', choices: ['1M', '5M', '10M', '20M', '1G'], hint: 'smaller = more segments' }
    ],
    behaviour: [
      { key: 'continue', label: 'Resume partial downloads', kind: 'select', choices: ['true', 'false'] },
      { key: 'file-allocation', label: 'File allocation', kind: 'select', choices: ['none', 'prealloc', 'trunc', 'falloc'] },
      { key: 'max-tries', label: 'Max retries', kind: 'select', choices: ['0', '3', '5', '10', '20'], hint: '0 = retry forever' },
      { key: 'retry-wait', label: 'Retry wait (seconds)', kind: 'select', choices: ['0', '5', '10', '30', '60'] },
      { key: 'auto-file-renaming', label: 'Auto-rename conflicts', kind: 'select', choices: ['true', 'false'] },
      { key: 'allow-overwrite', label: 'Allow overwrite', kind: 'select', choices: ['true', 'false'] }
    ],
    bittorrent: [
      { key: 'seed-ratio', label: 'Seed ratio', kind: 'select', choices: ['0.0', '0.5', '1.0', '2.0'], hint: '0.0 = seed forever' },
      { key: 'bt-max-peers', label: 'Max peers', kind: 'select', choices: ['0', '20', '55', '100', '200'] },
      { key: 'bt-request-peer-speed-limit', label: 'Peer speed target', kind: 'select', choices: ['50K', '256K', '1M', '5M'] },
      { key: 'listen-port', label: 'Listen port', kind: 'text' },
      { key: 'dht-listen-port', label: 'DHT port', kind: 'text' }
    ],
    network: [
      { key: 'all-proxy', label: 'Proxy', kind: 'text', hint: 'http://host:port — empty = none' },
      { key: 'user-agent', label: 'User agent', kind: 'text' },
      { key: 'connect-timeout', label: 'Connect timeout (seconds)', kind: 'select', choices: ['10', '30', '60', '120'] },
      { key: 'timeout', label: 'I/O timeout (seconds)', kind: 'select', choices: ['10', '30', '60', '120'] }
    ]
  };

  onMount(async () => {
    try {
      opts = await api.globalOptions();
    } catch (e) {
      store.toast('err', (e as Error).message);
    } finally {
      loaded = true;
    }
  });

  function val(key: string): string {
    return edits[key] ?? opts[key] ?? '';
  }

  function set(key: string, v: string) {
    edits = { ...edits, [key]: v };
  }

  /** Format raw byte counts as aria2 shorthand for display (20971520 → 20M). */
  function human(v: string): string {
    if (!/^\d{4,}$/.test(v)) return v;
    const n = Number(v);
    for (const [div, suffix] of [[1 << 30, 'G'], [1 << 20, 'M'], [1 << 10, 'K']] as const) {
      if (n >= div && n % div === 0) return `${n / div}${suffix}`;
    }
    return v;
  }

  const dirty = $derived(Object.entries(edits).some(([k, v]) => v !== (opts[k] ?? '')));

  async function save() {
    const changed: Record<string, string> = {};
    for (const [k, v] of Object.entries(edits)) {
      if (v !== (opts[k] ?? '')) changed[k] = v;
    }
    try {
      await api.setGlobalOptions(changed);
      store.toast('ok', 'Settings saved');
      opts = { ...opts, ...changed };
      edits = {};
    } catch (e) {
      store.toast('err', `Save failed: ${(e as Error).message}`);
    }
  }

  async function changePassword(e: Event) {
    e.preventDefault();
    if (pwNew !== pwConfirm) {
      store.toast('err', 'New passwords do not match');
      return;
    }
    pwBusy = true;
    try {
      await api.changePassword(pwCurrent, pwNew);
      store.toast('ok', store.authEnabled ? 'Password changed' : 'Password set — sign-in now required');
      store.authEnabled = true;
      pwCurrent = pwNew = pwConfirm = '';
    } catch (err) {
      store.toast('err', (err as Error).message);
    } finally {
      pwBusy = false;
    }
  }

  const allEntries = $derived(
    Object.entries(opts)
      .filter(([k]) => k.toLowerCase().includes(filter.toLowerCase()))
      .sort(([a], [b]) => a.localeCompare(b))
  );
</script>

<Modal title="Settings" {onclose} width={680}>
  <div class="tabs" role="tablist">
    {#each TABS as t (t.id)}
      <button class="tab" class:on={tab === t.id} role="tab" aria-selected={tab === t.id} onclick={() => (tab = t.id)}>
        {t.label}
      </button>
    {/each}
  </div>

  {#if !loaded}
    <div class="dim center">Loading…</div>
  {:else if tab === 'interface'}
    <div class="iface">
      <p class="dim note">These apply to this browser only and save automatically.</p>
      <label class="pref">
        <input
          type="checkbox"
          checked={store.prefs.notify}
          onchange={async (e) => {
            const on = (e.target as HTMLInputElement).checked;
            if (on && !(await store.enableNotifications())) {
              (e.target as HTMLInputElement).checked = false;
              return;
            }
            store.prefs.notify = on;
            store.savePrefs();
          }}
        />
        <span>
          <b>Desktop notifications</b>
          <em>Notify when a download completes or fails, even in another tab.</em>
        </span>
      </label>
      <label class="pref">
        <input
          type="checkbox"
          checked={store.prefs.compact}
          onchange={(e) => { store.prefs.compact = (e.target as HTMLInputElement).checked; store.savePrefs(); }}
        />
        <span>
          <b>Compact download list</b>
          <em>Tighter rows — more downloads on screen at once.</em>
        </span>
      </label>
      <label class="pref">
        <input
          type="checkbox"
          checked={store.prefs.confirmRemove}
          onchange={(e) => { store.prefs.confirmRemove = (e.target as HTMLInputElement).checked; store.savePrefs(); }}
        />
        <span>
          <b>Ask before removing</b>
          <em>Show a confirmation when removing an in-progress download.</em>
        </span>
      </label>
    </div>
  {:else if tab === 'advanced'}
    <p class="dim note">Every raw aria2 global option. Changes apply after Save.</p>
    <input class="filter" placeholder="Filter options…" bind:value={filter} />
    <div class="allopts">
      {#each allEntries as [k, v] (k)}
        <label class="optrow">
          <span class="mono okey">{k}</span>
          <input class="mono" value={val(k)} oninput={(e) => set(k, (e.target as HTMLInputElement).value)} />
        </label>
      {/each}
      {#if allEntries.length === 0}
        <div class="dim center">No options match “{filter}”</div>
      {/if}
    </div>
  {:else if tab === 'security'}
    <div class="sec">
      <div class="authstate" class:open={!store.authEnabled}>
        {#if store.authEnabled}
          🔒 Sign-in required — this UI is password-protected.
        {:else}
          🔓 No password set — anyone who can reach this address has full control.
        {/if}
      </div>

      <form class="pwform" onsubmit={changePassword}>
        <h4>{store.authEnabled ? 'Change password' : 'Set a password'}</h4>
        {#if store.authEnabled}
          <label>
            <span>Current password</span>
            <PasswordInput bind:value={pwCurrent} autocomplete="current-password" />
          </label>
        {/if}
        <label>
          <span>New password</span>
          <PasswordInput bind:value={pwNew} autocomplete="new-password" minlength={6} placeholder="At least 6 characters" />
        </label>
        <label>
          <span>Confirm new password</span>
          <PasswordInput bind:value={pwConfirm} autocomplete="new-password" />
        </label>
        <button class="btn primary" disabled={pwBusy || pwNew.length < 6 || pwNew !== pwConfirm || (store.authEnabled && !pwCurrent)}>
          {pwBusy ? 'Saving…' : store.authEnabled ? 'Change password' : 'Set password'}
        </button>
      </form>

      <div class="about dim">
        tidefetch {store.version} · aria2 {store.aria2} · downloads to <span class="mono">{store.downloadDir}</span>
      </div>
    </div>
  {:else}
    <div class="grid">
      {#each FIELDS[tab] as f (f.key)}
        <label>
          <span>{f.label}</span>
          {#if f.kind === 'text'}
            <input value={val(f.key)} oninput={(e) => set(f.key, (e.target as HTMLInputElement).value)} />
          {:else}
            <select value={val(f.key)} onchange={(e) => set(f.key, (e.target as HTMLSelectElement).value)}>
              {#if !f.choices?.includes(val(f.key)) && val(f.key) !== ''}
                <option value={val(f.key)}>{human(val(f.key))}</option>
              {/if}
              {#each f.choices ?? [] as c (c)}
                <option value={c}>{c}</option>
              {/each}
            </select>
          {/if}
          {#if f.hint}<em class="hint">{f.hint}</em>{/if}
        </label>
      {/each}
    </div>
  {/if}

  {#if tab !== 'security' && tab !== 'interface'}
    <div class="footer">
      <button class="btn" onclick={onclose}>Close</button>
      <button class="btn primary" disabled={!dirty} onclick={save}>Save changes</button>
    </div>
  {/if}
</Modal>

<style>
  .tabs {
    display: flex;
    gap: 2px;
    margin-bottom: 18px;
    border-bottom: 1px solid var(--border);
    flex-wrap: nowrap;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .tabs::-webkit-scrollbar { display: none; }
  .tab {
    display: inline-flex;
    align-items: center;
    padding: 8px 12px;
    color: var(--text-dim);
    font-weight: 550;
    font-size: 13px;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    transition: color 0.12s;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .tab:hover { color: var(--text); }
  .tab.on { color: var(--accent); border-bottom-color: var(--accent); }

  .center { text-align: center; padding: 40px 0; }
  .note { margin: 0 0 12px; font-size: 12.5px; }

  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px 18px;
    min-height: 220px;
    align-content: start;
  }
  .grid label { display: flex; flex-direction: column; gap: 5px; }
  .grid label > span { font-size: 12px; color: var(--text-dim); }
  .hint { font-size: 11px; color: var(--text-faint); font-style: normal; }

  .filter { width: 100%; margin-bottom: 10px; }
  .allopts {
    max-height: 340px;
    min-height: 220px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .optrow {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    align-items: center;
  }
  .okey { font-size: 11.5px; color: var(--text-dim); word-break: break-all; }
  .optrow input { font-size: 12px; padding: 5px 8px; }

  .sec { display: flex; flex-direction: column; gap: 20px; min-height: 260px; }
  .iface { display: flex; flex-direction: column; gap: 16px; min-height: 220px; }
  .pref {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    cursor: pointer;
    padding: 4px 2px;
  }
  .pref input { margin-top: 2px; }
  .pref span { display: flex; flex-direction: column; gap: 2px; font-size: 13.5px; }
  .pref em { font-style: normal; font-size: 12px; color: var(--text-dim); }
  .authstate {
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    font-size: 13px;
    background: rgba(74, 222, 128, 0.07);
    border: 1px solid rgba(74, 222, 128, 0.25);
    color: var(--ok);
  }
  .authstate.open {
    background: rgba(251, 191, 36, 0.07);
    border-color: rgba(251, 191, 36, 0.3);
    color: var(--warn);
  }
  h4 {
    margin: 0 0 4px;
    font-size: 11.5px;
    text-transform: uppercase;
    letter-spacing: 0.7px;
    color: var(--text-dim);
  }
  .pwform {
    display: flex;
    flex-direction: column;
    gap: 10px;
    max-width: 320px;
  }
  .pwform label { display: flex; flex-direction: column; gap: 4px; }
  .pwform label > span { font-size: 12px; color: var(--text-dim); }
  .pwform .btn { align-self: flex-start; margin-top: 4px; }
  .about { font-size: 12px; border-top: 1px solid var(--border); padding-top: 14px; }

  .footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 22px;
    padding-top: 14px;
    border-top: 1px solid var(--border);
  }
</style>
