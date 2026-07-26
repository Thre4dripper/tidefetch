<script lang="ts">
  import { store } from '../store.svelte';
  import { api, type ProbeResult } from '../api';
  import { fmtBytes } from '../format';
  import Modal from './Modal.svelte';
  import DirBrowser from './DirBrowser.svelte';

  let { onclose }: { onclose: () => void } = $props();

  let kind = $state<'uri' | 'torrent' | 'metalink'>('uri');
  let uris = $state('');
  let fileName = $state('');
  let filePayload = $state('');
  let dir = $state(store.downloadDir);
  let advanced = $state(false);
  let busy = $state(false);
  let browsing = $state(false);

  // Advanced options
  let split = $state('16');
  let maxConn = $state('16');
  let downLimit = $state('');
  let upLimit = $state('');
  let out = $state('');
  let cont = $state(true);
  let allocation = $state('');
  let checksum = $state('');
  let referer = $state('');
  let userAgent = $state('');
  let proxy = $state('');
  let seedRatio = $state('');
  let paused = $state(false);

  // Structured custom headers
  interface HeaderRow {
    id: number;
    name: string;
    custom: string;
    value: string;
  }
  const HEADER_NAMES = ['Authorization', 'Cookie', 'Accept', 'Accept-Language', 'X-API-Key', 'Custom…'];
  let headerSeq = 0;
  let headerRows = $state<HeaderRow[]>([]);

  function addHeader() {
    headerRows = [...headerRows, { id: ++headerSeq, name: 'Authorization', custom: '', value: '' }];
  }
  function removeHeader(id: number) {
    headerRows = headerRows.filter((h) => h.id !== id);
  }

  // Probe
  let probing = $state(false);
  let probe = $state<ProbeResult | null>(null);
  let probeErr = $state('');

  const firstURL = $derived(uris.split('\n').map((s) => s.trim()).find((s) => s.startsWith('http')) ?? '');

  async function runProbe() {
    if (!firstURL) return;
    probing = true;
    probe = null;
    probeErr = '';
    try {
      probe = await api.probe(firstURL);
    } catch (e) {
      probeErr = (e as Error).message;
    } finally {
      probing = false;
    }
  }

  function pickFile(accept: string) {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = accept;
    input.onchange = () => {
      const f = input.files?.[0];
      if (!f) return;
      fileName = f.name;
      const reader = new FileReader();
      reader.onload = () => {
        const url = reader.result as string;
        filePayload = url.slice(url.indexOf(',') + 1); // strip data: prefix
      };
      reader.readAsDataURL(f);
    };
    input.click();
  }

  function buildOptions(): Record<string, string> {
    const o: Record<string, string> = { dir };
    if (split) o['split'] = split;
    if (maxConn) o['max-connection-per-server'] = maxConn;
    if (downLimit) o['max-download-limit'] = downLimit;
    if (upLimit) o['max-upload-limit'] = upLimit;
    if (out) o['out'] = out;
    o['continue'] = cont ? 'true' : 'false';
    if (allocation) o['file-allocation'] = allocation;
    if (checksum) o['checksum'] = checksum;
    if (referer) o['referer'] = referer;
    if (userAgent) o['user-agent'] = userAgent;
    if (proxy) o['all-proxy'] = proxy;
    if (seedRatio) o['seed-ratio'] = seedRatio;
    if (paused) o['pause'] = 'true';
    const headerLines = headerRows
      .map((h) => {
        const name = (h.name === 'Custom…' ? h.custom : h.name).trim();
        return name && h.value.trim() ? `${name}: ${h.value.trim()}` : '';
      })
      .filter(Boolean);
    if (headerLines.length) o['header'] = headerLines.join('\n');
    return o;
  }

  async function submit() {
    busy = true;
    try {
      const payload =
        kind === 'uri'
          ? { kind, uris: uris.split('\n').map((s) => s.trim()).filter(Boolean), options: buildOptions() }
          : { kind, payload: filePayload, options: buildOptions() };
      const res = await api.add(payload);
      store.toast('ok', `Added ${res.gids.length} download${res.gids.length === 1 ? '' : 's'}`);
      onclose();
    } catch (e) {
      store.toast('err', `Add failed: ${(e as Error).message}`);
    } finally {
      busy = false;
    }
  }

  const canSubmit = $derived(
    kind === 'uri' ? uris.trim().length > 0 : filePayload.length > 0
  );
</script>

<Modal title="Add download" {onclose} width={620}>
  <div class="kinds">
    <button class="kind" class:on={kind === 'uri'} onclick={() => (kind = 'uri')}>🔗 URLs / magnet</button>
    <button class="kind" class:on={kind === 'torrent'} onclick={() => { kind = 'torrent'; filePayload = ''; fileName = ''; }}>⧉ Torrent</button>
    <button class="kind" class:on={kind === 'metalink'} onclick={() => { kind = 'metalink'; filePayload = ''; fileName = ''; }}>🜄 Metalink</button>
  </div>

  {#if kind === 'uri'}
    <textarea
      rows="4"
      placeholder="One URL per line — http(s), ftp, sftp, magnet:…"
      bind:value={uris}
    ></textarea>
    <div class="proberow">
      <button class="btn sm" disabled={!firstURL || probing} onclick={runProbe}>
        {probing ? 'Checking…' : '⚡ Check link'}
      </button>
      {#if probe}
        <span class="probe" class:good={probe.resumable} class:bad={!probe.resumable}>
          {probe.resumable ? '✓ resumable' : '✗ resume unsupported'}
          {#if probe.size > 0}· {fmtBytes(probe.size)}{/if}
          {#if probe.filename}· {probe.filename}{/if}
          {#if probe.contentType}· {probe.contentType}{/if}
        </span>
      {:else if probeErr}
        <span class="probe bad">✗ {probeErr}</span>
      {/if}
    </div>
  {:else}
    <button class="btn filebtn" onclick={() => pickFile(kind === 'torrent' ? '.torrent' : '.metalink,.meta4')}>
      {fileName ? `📄 ${fileName}` : `Choose a ${kind} file…`}
    </button>
  {/if}

  <label class="field">
    <span>Save to</span>
    <div class="dirrow">
      <input bind:value={dir} class="mono" />
      <button class="btn sm" onclick={() => (browsing = true)}>Browse…</button>
    </div>
  </label>

  <button class="advtoggle dim" onclick={() => (advanced = !advanced)}>
    {advanced ? '▾' : '▸'} Advanced options
  </button>

  {#if advanced}
    <div class="grid">
      <label><span>Rename (out)</span><input bind:value={out} placeholder="original name" /></label>
      <label><span>Split (connections)</span><input bind:value={split} /></label>
      <label><span>Max conn / server</span><input bind:value={maxConn} /></label>
      <label><span>Down limit</span><input bind:value={downLimit} placeholder="e.g. 2M" /></label>
      <label><span>Up limit</span><input bind:value={upLimit} placeholder="e.g. 500K" /></label>
      <label>
        <span>File allocation</span>
        <select bind:value={allocation}>
          <option value="">default</option>
          <option value="none">none</option>
          <option value="prealloc">prealloc</option>
          <option value="trunc">trunc</option>
          <option value="falloc">falloc</option>
        </select>
      </label>
      <label><span>Checksum</span><input bind:value={checksum} placeholder="sha-256=…" /></label>
      <label><span>Referer</span><input bind:value={referer} /></label>
      <label><span>User agent</span><input bind:value={userAgent} /></label>
      <label><span>Proxy</span><input bind:value={proxy} placeholder="http://host:port" /></label>
      <label><span>Seed ratio</span><input bind:value={seedRatio} placeholder="1.0" /></label>
      <div class="wide headers">
        <span class="hlabel">Custom headers</span>
        {#each headerRows as h (h.id)}
          <div class="hrow">
            <select bind:value={h.name} aria-label="Header name">
              {#each HEADER_NAMES as n (n)}
                <option value={n}>{n}</option>
              {/each}
            </select>
            {#if h.name === 'Custom…'}
              <input class="hname" placeholder="Header-Name" bind:value={h.custom} />
            {/if}
            <input class="hval" placeholder="Value" bind:value={h.value} />
            <button type="button" class="btn icon danger" onclick={() => removeHeader(h.id)} aria-label="Remove header">✕</button>
          </div>
        {/each}
        <button type="button" class="btn sm addhdr" onclick={addHeader}>＋ Add header</button>
      </div>
      <label class="check"><input type="checkbox" bind:checked={cont} /> Continue partial downloads</label>
      <label class="check"><input type="checkbox" bind:checked={paused} /> Add paused</label>
    </div>
  {/if}

  <div class="footer">
    <button class="btn" onclick={onclose}>Cancel</button>
    <button class="btn primary" disabled={!canSubmit || busy} onclick={submit}>
      {busy ? 'Adding…' : '＋ Add download'}
    </button>
  </div>
</Modal>

{#if browsing}
  <DirBrowser
    start={dir}
    onpick={(p) => { dir = p; browsing = false; }}
    onclose={() => (browsing = false)}
  />
{/if}

<style>
  .kinds { display: flex; gap: 8px; margin-bottom: 14px; }
  .kind {
    padding: 8px 14px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    color: var(--text-dim);
    font-weight: 550;
  }
  .kind.on {
    color: var(--accent);
    border-color: rgba(52, 216, 195, 0.4);
    background: rgba(52, 216, 195, 0.08);
  }
  textarea { width: 100%; resize: vertical; font-family: var(--mono); font-size: 13px; }
  .proberow { display: flex; align-items: center; gap: 10px; margin-top: 8px; min-height: 28px; flex-wrap: wrap; }
  .probe { font-size: 12.5px; }
  .probe.good { color: var(--ok); }
  .probe.bad { color: var(--warn); }
  .filebtn { width: 100%; justify-content: center; padding: 22px; border-style: dashed; }
  .field { display: flex; flex-direction: column; gap: 6px; margin-top: 14px; }
  .field > span { font-size: 12px; color: var(--text-dim); }
  .dirrow { display: flex; gap: 8px; }
  .dirrow input { flex: 1; font-size: 12.5px; }
  .advtoggle { margin-top: 14px; font-size: 13px; padding: 4px 0; }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px 14px;
    margin-top: 10px;
    animation: fadeUp 0.15s ease;
  }
  .grid label { display: flex; flex-direction: column; gap: 4px; }
  .grid label > span { font-size: 11.5px; color: var(--text-dim); }
  .grid .wide { grid-column: 1 / -1; }
  .grid .check { flex-direction: row; align-items: center; gap: 8px; font-size: 13px; color: var(--text-dim); cursor: pointer; }
  .headers { display: flex; flex-direction: column; gap: 8px; }
  .hlabel { font-size: 11.5px; color: var(--text-dim); }
  .hrow { display: flex; gap: 8px; align-items: center; }
  .hrow select { flex: 0 0 150px; }
  .hrow .hname { flex: 0 0 140px; }
  .hrow .hval { flex: 1; min-width: 0; }
  .addhdr { align-self: flex-start; }
  .footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 20px;
  }
</style>
