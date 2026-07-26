<script lang="ts">
  import { onMount } from 'svelte';
  import { store } from './lib/store.svelte';
  import Login from './lib/components/Login.svelte';
  import Sidebar from './lib/components/Sidebar.svelte';
  import Topbar from './lib/components/Topbar.svelte';
  import TaskList from './lib/components/TaskList.svelte';
  import TaskDetail from './lib/components/TaskDetail.svelte';
  import HistoryView from './lib/components/HistoryView.svelte';
  import AddModal from './lib/components/AddModal.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import Onboarding from './lib/components/Onboarding.svelte';

  onMount(() => {
    store.boot();
  });
</script>

{#if store.authed === null}
  <div class="boot">
    <div class="pulse">⬡</div>
  </div>
{:else if store.authed === false}
  <Login />
{:else}
  <div class="layout">
    <Sidebar />
    <main>
      <Topbar />
      {#if store.view === 'downloads'}
        <TaskList />
      {:else}
        <HistoryView />
      {/if}
    </main>
    {#if store.selectedGid}
      <TaskDetail gid={store.selectedGid} onclose={() => (store.selectedGid = null)} />
    {/if}
  </div>
  {#if store.addOpen}
    <AddModal onclose={() => (store.addOpen = false)} />
  {/if}
  {#if store.settingsOpen}
    <SettingsModal onclose={() => (store.settingsOpen = false)} />
  {/if}
  {#if store.onboarding}
    <Onboarding />
  {/if}
{/if}

<div class="toasts">
  {#each store.toasts as t (t.id)}
    <div class="toast {t.kind}">{t.text}</div>
  {/each}
</div>

{#if store.authed && !store.connected}
  <div class="connbar">Reconnecting to aria2…</div>
{/if}

<style>
  .layout {
    display: flex;
    height: 100%;
    overflow: hidden;
  }
  main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .boot {
    height: 100%;
    display: grid;
    place-items: center;
  }
  .pulse {
    font-size: 42px;
    background: var(--grad);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    animation: fadeIn 0.8s ease infinite alternate;
  }
  .toasts {
    position: fixed;
    bottom: 18px;
    right: 18px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    z-index: 100;
  }
  .toast {
    padding: 10px 16px;
    border-radius: var(--radius-sm);
    background: var(--panel-solid);
    border: 1px solid var(--border-strong);
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.45);
    animation: fadeUp 0.18s ease;
    max-width: 380px;
    font-size: 13px;
  }
  .toast.ok { border-left: 3px solid var(--ok); }
  .toast.err { border-left: 3px solid var(--err); }
  .connbar {
    position: fixed;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(251, 191, 36, 0.12);
    color: var(--warn);
    border: 1px solid rgba(251, 191, 36, 0.3);
    border-top: none;
    border-radius: 0 0 8px 8px;
    padding: 4px 14px;
    font-size: 12px;
    z-index: 90;
  }
</style>
