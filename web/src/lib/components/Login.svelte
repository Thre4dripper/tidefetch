<script lang="ts">
  import { store } from '../store.svelte';
  import PasswordInput from './PasswordInput.svelte';

  let password = $state('');
  let busy = $state(false);
  let error = $state('');
</script>

<div class="wrap">
  <form
    class="card panel"
    onsubmit={async (e) => {
      e.preventDefault();
      busy = true;
      error = '';
      try {
        await store.login(password);
      } catch (err) {
        error = (err as Error).message;
      } finally {
        busy = false;
      }
    }}
  >
    <div class="logo">⬡</div>
    <h1>tidefetch</h1>
    <p class="dim">Sign in to manage your downloads</p>
    <PasswordInput bind:value={password} placeholder="Password" autocomplete="current-password" />
    {#if error}<div class="err">{error}</div>{/if}
    <button class="btn primary" disabled={busy || !password}>
      {busy ? 'Signing in…' : 'Sign in'}
    </button>
  </form>
</div>

<style>
  .wrap {
    height: 100%;
    display: grid;
    place-items: center;
  }
  .panel {
    width: 340px;
    padding: 36px 32px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    text-align: center;
    animation: fadeUp 0.25s ease;
  }
  .logo {
    font-size: 40px;
    background: var(--grad);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  h1 {
    margin: 0;
    font-size: 22px;
    letter-spacing: 0.5px;
  }
  p { margin: 0 0 10px; }
  .err {
    color: var(--err);
    font-size: 13px;
  }
</style>
