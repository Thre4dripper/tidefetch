<script lang="ts">
  import {
    ArrowRight,
    BookOpen,
    Check,
    Clipboard,
    CloudCog,
    Container,
    Download,
    Gauge,
    Globe2,
    HardDrive,
    Layers3,
    Network,
    Play,
    ServerCog,
    ShieldCheck,
    SquareTerminal,
    Workflow,
    Zap
  } from '@lucide/svelte';
  import MediaFrame from './MediaFrame.svelte';
  import ProductMock from './ProductMock.svelte';
  import { media } from './media';

  const repo = 'https://github.com/Thre4dripper/tidefetch';
  let installTab = $state<'docker' | 'source' | 'go'>('docker');
  let copied = $state(false);
  let surface = $state<'web' | 'terminal'>('web');

  const commands = {
    docker:
      'git clone https://github.com/Thre4dripper/tidefetch.git && cd tidefetch && docker compose -f packaging/docker/docker-compose.yml up -d --build',
    source: 'git clone https://github.com/Thre4dripper/tidefetch.git && cd tidefetch && make install',
    go: 'go install github.com/Thre4dripper/tidefetch/cmd/tidefetch@latest'
  };

  const installNotes = {
    docker: 'All-in-one image: web UI + aria2 engine. Copy .env.example and set a password first.',
    source: 'Requires aria2, Go 1.26.1, Node 20+ and Make. Installs the tidefetch binary.',
    go: 'Binary only — install aria2 separately, then run tidefetch doctor.'
  };

  const features = [
    {
      icon: Zap,
      title: 'Fast where it matters',
      body: 'WebSocket deltas, virtualized queues and server-downsampled piece maps stay smooth where older aria2 frontends choke.',
      tone: 'lime'
    },
    {
      icon: Gauge,
      title: 'Transfer intelligence',
      body: 'Lifetime speed charts, peers, servers, files, checksums, mirrors and every aria2 option — in context, per download.',
      tone: 'cyan'
    },
    {
      icon: ShieldCheck,
      title: 'Secure by default',
      body: 'bcrypt passwords, HttpOnly sessions, same-origin guards and rate limits. The RPC secret never reaches the browser.',
      tone: 'violet'
    },
    {
      icon: HardDrive,
      title: 'Built for real storage',
      body: 'Browse the host filesystem, watch free space, pick per-task destinations and survive restarts with session persistence.',
      tone: 'cyan'
    },
    {
      icon: Workflow,
      title: 'Queue control without friction',
      body: 'Pause, reorder, retry, throttle, remove, delete files — and inspect links IDM-style before committing bandwidth.',
      tone: 'violet'
    },
    {
      icon: Network,
      title: 'Every aria2 protocol',
      body: 'HTTP(S), FTP, SFTP, BitTorrent, Metalink, magnets, multi-mirror, custom headers, proxies and remote daemons.',
      tone: 'lime'
    }
  ];

  const deployments = [
    { icon: Container, name: 'Docker', copy: 'Single container, two volumes', href: '#/docs/deployment/docker' },
    { icon: Layers3, name: 'Compose', copy: 'Copy, configure, launch', href: '#/docs/deployment/docker' },
    { icon: Workflow, name: 'Swarm', copy: 'Secret-backed stack file', href: '#/docs/deployment/swarm' },
    { icon: CloudCog, name: 'Kubernetes', copy: 'Manifests, PVCs and probes', href: '#/docs/deployment/kubernetes' },
    { icon: ServerCog, name: 'Unraid', copy: 'Community Apps template', href: '#/docs/deployment/unraid' },
    { icon: Globe2, name: 'Reverse proxy', copy: 'Caddy, Nginx, Traefik', href: '#/docs/reverse-proxy' }
  ];

  async function copyInstall() {
    await navigator.clipboard.writeText(commands[installTab]);
    copied = true;
    window.setTimeout(() => (copied = false), 1600);
  }
</script>

<main class="landing">
  <!-- ── Hero ─────────────────────────────────────────────────────── -->
  <section class="hero">
    <div class="hero-glow" aria-hidden="true"></div>
    <div class="hero-grid" aria-hidden="true"></div>

    <div class="hero-copy">
      <a class="hero-badge" href="#/docs/getting-started">
        <span class="dot"></span> Open source · TUI + Web UI for aria2 <ArrowRight size={12} />
      </a>
      <h1>
        Downloads, <em>beautifully</em><br />under control.
      </h1>
      <p>
        Tidefetch is a fast terminal UI and self-hosted web UI for the aria2 download engine.
        One binary, full queue control and live telemetry — built for TUI lovers, headless
        servers and homelabs.
      </p>
      <div class="hero-actions">
        <a class="btn-primary" href="#/#install"><Download size={16} /> Install Tidefetch</a>
        <a class="btn-ghost" href="#/docs/getting-started"><BookOpen size={15} /> Read the docs</a>
      </div>
      <div class="hero-proof">
        <span><Check size={13} /> One binary</span>
        <span><Check size={13} /> MIT licensed</span>
        <span><Check size={13} /> No cloud account</span>
        <span><Check size={13} /> ~30 MB RAM</span>
      </div>
    </div>

    <div class="hero-stage" id="demo">
      <div class="stage-frame">
        {#if media.hero.enabled}
          <video autoplay muted loop playsinline poster={media.hero.poster} aria-label="Tidefetch product overview">
            <source src={media.hero.webm} type="video/webm" />
            <source src={media.hero.mp4} type="video/mp4" />
          </video>
        {:else}
          <ProductMock />
          <div class="video-slot"><Play size={12} /> hero-demo.webm slot</div>
        {/if}
      </div>
    </div>
  </section>

  <!-- ── Protocol strip ───────────────────────────────────────────── -->
  <section class="strip" aria-label="Supported protocols">
    <span>HTTP(S)</span><i></i><span>BitTorrent</span><i></i><span>Metalink</span><i></i>
    <span>FTP / SFTP</span><i></i><span>Magnets</span><i></i><span>Multi-mirror</span><i></i>
    <span>Remote RPC</span><i></i><span>100+ aria2 options</span>
  </section>

  <!-- ── Features ─────────────────────────────────────────────────── -->
  <section class="section" id="features">
    <div class="section-head">
      <span class="kicker">Why Tidefetch</span>
      <h2>Everything the engine knows.<br />Nothing in your way.</h2>
      <p>
        aria2 is phenomenal and invisible. Tidefetch keeps its power intact and makes it
        legible — inspect every transfer, shape every queue, understand what your server is doing.
      </p>
    </div>
    <div class="feature-grid">
      {#each features as feature (feature.title)}
        <article class="feature-card {feature.tone}">
          <div class="feature-icon"><feature.icon size={19} strokeWidth={1.8} /></div>
          <h3>{feature.title}</h3>
          <p>{feature.body}</p>
        </article>
      {/each}
    </div>
  </section>

  <!-- ── Interfaces ───────────────────────────────────────────────── -->
  <section class="section" id="interfaces">
    <div class="section-head">
      <span class="kicker">Two native interfaces</span>
      <h2>Stay in the terminal.<br />Or open it to the network.</h2>
      <p>
        Both share the same engine, queue semantics, history and advanced options.
        Pick the surface that fits the moment.
      </p>
      <div class="surface-toggle" role="tablist" aria-label="Interface preview">
        <button role="tab" aria-selected={surface === 'web'} class:active={surface === 'web'} onclick={() => (surface = 'web')}>
          <Globe2 size={14} /> Web UI
        </button>
        <button
          role="tab"
          aria-selected={surface === 'terminal'}
          class:active={surface === 'terminal'}
          onclick={() => (surface = 'terminal')}>
          <SquareTerminal size={14} /> Terminal
        </button>
      </div>
    </div>

    <div class="surface-stage">
      {#if surface === 'web'}
        <MediaFrame item={media.web} />
        <ul class="surface-points">
          <li>WebSocket push — no per-tab polling storms</li>
          <li>Virtualized lists and canvas piece maps</li>
          <li>Password login, rate limits, strict CSP</li>
          <li>Responsive from ultrawide to phone</li>
        </ul>
      {:else}
        <MediaFrame item={media.terminal} />
        <ul class="surface-points">
          <li>btop-style charts and full mouse support</li>
          <li>Keyboard-first queue and file workflow</li>
          <li>Disk gauge, telemetry sidebar, file browser</li>
          <li>Complete aria2 settings editor built in</li>
        </ul>
      {/if}
    </div>
  </section>

  <!-- ── Install ──────────────────────────────────────────────────── -->
  <section class="section" id="install">
    <div class="install-panel">
      <div class="install-copy">
        <span class="kicker">Start here</span>
        <h2>Running in under a minute.</h2>
        <p>
          Use the all-in-one container for a server, or install the binary next to an existing
          aria2 daemon. Every platform is covered in the docs.
        </p>
        <a class="link-arrow" href="#/docs/installation">Every install method <ArrowRight size={14} /></a>
      </div>
      <div class="command-card">
        <div class="command-tabs" role="tablist" aria-label="Install methods">
          <button role="tab" aria-selected={installTab === 'docker'} class:active={installTab === 'docker'} onclick={() => (installTab = 'docker')}>Docker</button>
          <button role="tab" aria-selected={installTab === 'source'} class:active={installTab === 'source'} onclick={() => (installTab = 'source')}>Source</button>
          <button role="tab" aria-selected={installTab === 'go'} class:active={installTab === 'go'} onclick={() => (installTab = 'go')}>Go</button>
        </div>
        <div class="command-body">
          <span aria-hidden="true">$</span>
          <code>{commands[installTab]}</code>
          <button type="button" class="copy-btn" aria-label="Copy install command" onclick={copyInstall}>
            {#if copied}<Check size={15} />{:else}<Clipboard size={15} />{/if}
          </button>
        </div>
        <p class="command-note">{installNotes[installTab]}</p>
      </div>
    </div>
  </section>

  <!-- ── Deploy ───────────────────────────────────────────────────── -->
  <section class="section" id="deploy">
    <div class="section-head">
      <span class="kicker">Homelab ready</span>
      <h2>From one NAS to a cluster.</h2>
      <p>
        Documented storage, health checks, secrets, backups and upgrades —
        with configs designed to be copied, reviewed and owned.
      </p>
    </div>
    <div class="deploy-grid">
      {#each deployments as deployment (deployment.name)}
        <a class="deploy-card" href={deployment.href}>
          <deployment.icon size={20} strokeWidth={1.7} />
          <div>
            <strong>{deployment.name}</strong>
            <span>{deployment.copy}</span>
          </div>
          <ArrowRight size={15} />
        </a>
      {/each}
    </div>
    <div class="arch-strip" aria-label="Architecture">
      <div><span>YOUR BROWSER</span><b>Dashboard</b></div>
      <ArrowRight size={15} />
      <div><span>PORT 8210</span><b>Tidefetch broker</b></div>
      <ArrowRight size={15} />
      <div><span>PRIVATE RPC</span><b>aria2 engine</b></div>
      <ArrowRight size={15} />
      <div><span>PERSISTENT</span><b>Your storage</b></div>
    </div>
  </section>

  <!-- ── Closing ──────────────────────────────────────────────────── -->
  <section class="closing">
    <div class="closing-glow" aria-hidden="true"></div>
    <span class="closing-mark">⬡</span>
    <h2>Give aria2 the interface<br />it deserves.</h2>
    <p>Open source, self-hosted and built for the machines you already own.</p>
    <div class="closing-actions">
      <a class="btn-primary" href="#/#install"><Download size={16} /> Install Tidefetch</a>
      <a class="btn-ghost" href={repo} target="_blank" rel="noreferrer">View source</a>
    </div>
  </section>
</main>
