<script lang="ts">
  import { store } from '../store.svelte';
  import { api } from '../api';
  import PasswordInput from './PasswordInput.svelte';

  let step = $state(0);
  let pw = $state('');
  let pwConfirm = $state('');
  let pwBusy = $state(false);

  const total = $derived(store.authEnabled ? 2 : 3);

  function next() {
    if (step < total - 1) step += 1;
    else store.finishOnboarding();
  }

  async function setPassword(e: Event) {
    e.preventDefault();
    pwBusy = true;
    try {
      await api.changePassword('', pw);
      store.authEnabled = true;
      store.toast('ok', 'Password set');
      store.finishOnboarding();
    } catch (err) {
      store.toast('err', (err as Error).message);
    } finally {
      pwBusy = false;
    }
  }
</script>

<div class="backdrop" role="presentation">
  <div class="card panel" role="dialog" aria-label="Welcome to tidefetch">
    {#if step === 0}
      <div class="mark">⬡</div>
      <h2>Welcome to tidefetch</h2>
      <p class="dim">
        Your self-hosted download manager, powered by the aria2 engine.
        HTTP, FTP, SFTP, torrents, magnets and Metalink — all in one place.
      </p>
    {:else if step === 1}
      <h2>The basics</h2>
      <ul class="tour">
        <li><span class="ic">＋</span><div><b>Add anything</b><span class="dim">Paste URLs or magnets, or drop in torrent and Metalink files — with per-download speed limits, renaming and more under Advanced options.</span></div></li>
        <li><span class="ic">⇣</span><div><b>Click a download</b><span class="dim">Opens the detail panel: live piece map, files, peers and per-task options.</span></div></li>
        <li><span class="ic">⚡</span><div><b>Everything is live</b><span class="dim">Speeds, charts and progress stream in real time — leave it open on any device.</span></div></li>
      </ul>
    {:else}
      <h2>Protect this UI</h2>
      <p class="dim">
        No password is set. Anyone who can reach this address can control your
        downloads — set one now (you can change it later in Settings → Security).
      </p>
      <form class="pwform" onsubmit={setPassword}>
        <PasswordInput bind:value={pw} placeholder="Password (6+ characters)" minlength={6} autocomplete="new-password" />
        <PasswordInput bind:value={pwConfirm} placeholder="Confirm password" autocomplete="new-password" />
        <button class="btn primary" disabled={pwBusy || pw.length < 6 || pw !== pwConfirm}>
          {pwBusy ? 'Saving…' : 'Set password & finish'}
        </button>
      </form>
    {/if}

    <div class="nav">
      <button class="btn sm skip" onclick={() => store.finishOnboarding()}>Skip</button>
      <div class="dots">
        {#each Array(total) as _, i (i)}
          <span class="dot" class:on={i === step}></span>
        {/each}
      </div>
      {#if step < total - 1}
        <button class="btn primary sm" onclick={next}>Next →</button>
      {:else if store.authEnabled || step < 2}
        <button class="btn primary sm" onclick={() => store.finishOnboarding()}>Get started</button>
      {:else}
        <button class="btn sm" onclick={() => store.finishOnboarding()}>Later</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(4, 8, 14, 0.72);
    backdrop-filter: blur(8px);
    display: grid;
    place-items: center;
    z-index: 60;
    animation: fadeIn 0.2s ease;
  }
  .panel {
    width: 430px;
    max-width: calc(100vw - 40px);
    padding: 34px 32px 22px;
    background: var(--panel-solid);
    box-shadow: 0 24px 80px rgba(0, 0, 0, 0.55);
    animation: fadeUp 0.22s ease;
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 330px;
  }
  .mark {
    font-size: 44px;
    background: var(--grad);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    text-align: center;
  }
  h2 { margin: 4px 0 2px; font-size: 20px; text-align: center; }
  p { margin: 0; text-align: center; line-height: 1.55; font-size: 13.5px; }
  .tour {
    list-style: none;
    margin: 8px 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .tour li { display: flex; gap: 13px; align-items: flex-start; }
  .tour .ic {
    width: 34px;
    height: 34px;
    border-radius: 9px;
    background: rgba(52, 216, 195, 0.1);
    border: 1px solid rgba(52, 216, 195, 0.3);
    color: var(--accent);
    display: grid;
    place-items: center;
    font-size: 15px;
    flex-shrink: 0;
  }
  .tour div { display: flex; flex-direction: column; gap: 2px; font-size: 13px; }
  .tour .dim { font-size: 12.5px; line-height: 1.45; }
  .pwform {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 10px;
  }
  .pwform .btn { align-self: center; margin-top: 4px; }
  .nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: auto;
    padding-top: 18px;
  }
  .skip { color: var(--text-faint); border-color: transparent; background: none; }
  .skip:hover { color: var(--text-dim); background: none; border-color: transparent; }
  .dots { display: flex; gap: 7px; }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.15);
    transition: background 0.15s, transform 0.15s;
  }
  .dot.on { background: var(--accent); transform: scale(1.25); }
</style>
