<script lang="ts">
  import {
    ArrowRight,
    BookOpen,
    Check,
    Clipboard,
    CloudCog,
    Container,
    Cpu,
    Gauge,
    Globe2,
    HardDrive,
    Keyboard,
    Layers3,
    Network,
    Package,
    Server,
    ServerCog,
    ShieldCheck,
    SquareTerminal,
    Workflow,
    Zap
  } from '@lucide/svelte';
  import MediaFrame from './MediaFrame.svelte';
  import { media } from './media';
  import { href } from './router';

  const repo = 'https://github.com/Thre4dripper/tidefetch';
  let installTab = $state<'script' | 'brew' | 'docker' | 'go'>('script');
  let copied = $state(false);
  let surface = $state<'terminal' | 'web'>('terminal');

  const commands = {
    script: 'curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh',
    brew: 'brew install thre4dripper/tap/tidefetch',
    docker: 'docker run -d -p 8210:8210 -v tidefetch:/config ghcr.io/thre4dripper/tidefetch',
    go: 'go install github.com/Thre4dripper/tidefetch/cmd/tidefetch@latest'
  };

  const installNotes = {
    script: 'macOS, Linux and FreeBSD. Detects your platform, verifies checksums, no runtime needed. Windows: irm .../install.ps1 | iex',
    brew: 'macOS and Linuxbrew, with aria2 pulled in as a dependency.',
    docker: 'All-in-one image with the aria2 engine baked in. Helm chart available.',
    go: 'Straight from source. Install aria2 separately, then run tidefetch doctor.'
  };

  const features = [
    {
      icon: Keyboard,
      title: 'Keyboard-first, mouse-friendly',
      body: 'Vim-style motions, single-key actions and a full mouse hit-test layer. Nothing is buried three menus deep.',
      tone: 'lime'
    },
    {
      icon: Gauge,
      title: 'Telemetry you can read',
      body: 'Braille speed graphs, per-task lifetime history, piece maps, disk gauges and live session stats — btop energy.',
      tone: 'cyan'
    },
    {
      icon: Cpu,
      title: 'Tiny footprint',
      body: 'A single static Go binary, no runtime, no Electron, ~30 MB resident. It belongs on a Pi as much as a workstation.',
      tone: 'violet'
    },
    {
      icon: Network,
      title: 'Every aria2 protocol',
      body: 'HTTP(S), FTP, SFTP, BitTorrent, Metalink, magnets, multi-mirror, custom headers, proxies and remote daemons.',
      tone: 'cyan'
    },
    {
      icon: ShieldCheck,
      title: 'Safe to expose',
      body: 'bcrypt auth, HttpOnly sessions, same-origin guards, rate limits and strict CSP. The RPC secret never leaves the server.',
      tone: 'violet'
    },
    {
      icon: HardDrive,
      title: 'Respects your storage',
      body: 'Browse the host filesystem, watch free space, choose per-task destinations and resume cleanly across restarts.',
      tone: 'lime'
    }
  ];

  const registries = [
    { name: 'Install script', cmd: 'curl -fsSL …/install.sh | sh' },
    { name: 'PowerShell', cmd: 'irm …/install.ps1 | iex' },
    { name: 'Homebrew', cmd: 'brew install tidefetch' },
    { name: 'Docker', cmd: 'docker pull ghcr.io/…/tidefetch' },
    { name: 'GHCR', cmd: 'docker pull ghcr.io/…/tidefetch' },
    { name: 'Helm', cmd: 'helm install oci://…/tidefetch' },
    { name: 'Go', cmd: 'go install …/tidefetch' }
  ];

  const deployments = [
    { icon: Container, name: 'Docker', copy: 'One container, two volumes', href: href('docs/deployment/docker') },
    { icon: Layers3, name: 'Compose', copy: 'Copy, configure, launch', href: href('docs/deployment/docker') },
    { icon: Workflow, name: 'Swarm', copy: 'Secret-backed stack file', href: href('docs/deployment/swarm') },
    { icon: CloudCog, name: 'Kubernetes', copy: 'Helm chart and manifests', href: href('docs/deployment/kubernetes') },
    { icon: ServerCog, name: 'Unraid', copy: 'Community Apps template', href: href('docs/deployment/unraid') },
    { icon: Cpu, name: 'Bare metal', copy: 'systemd unit, no Docker', href: href('docs/homelab') },
    { icon: Globe2, name: 'Reverse proxy', copy: 'Caddy, Nginx, Traefik', href: href('docs/reverse-proxy') }
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
      <a class="hero-badge" href={href('docs/getting-started')}>
        <span class="dot"></span> Open source · single binary · powered by aria2 <ArrowRight size={12} />
      </a>
      <h1>The download manager<br />that lives in your <em>terminal</em>.</h1>
      <p>
        Tidefetch is a keyboard-first terminal UI for the aria2 download engine, built
        for people who live in a shell. The same binary also serves a self-hosted web
        UI, so headless servers and homelabs get a browser dashboard on demand.
      </p>
      <div class="hero-actions">
        <a class="btn-primary" href={href('#install')}><Package size={16} /> Install Tidefetch</a>
        <a class="btn-ghost" href={href('docs/getting-started')}><BookOpen size={15} /> Read the docs</a>
      </div>
      <div class="hero-proof">
        <span><Check size={13} /> One static binary, no runtime</span>
        <span><Check size={13} /> Works over SSH</span>
        <span><Check size={13} /> MIT licensed</span>
      </div>
    </div>

    <div class="hero-stage">
      <div class="stage-frame">
        {#if media.hero.enabled}
          <video autoplay muted loop playsinline poster={media.hero.poster} aria-label="Tidefetch overview">
            <source src={media.hero.webm} type="video/webm" />
            <source src={media.hero.mp4} type="video/mp4" />
          </video>
        {:else if media.terminal.enabled}
          <img class="stage-shot" src={media.terminal.src} alt={media.terminal.alt} />
        {:else}
          <MediaFrame item={media.terminal} />
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
        aria2 is phenomenal and completely invisible. Tidefetch keeps every bit of that
        power and makes it legible — without leaving the keyboard.
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
      <span class="kicker">Two interfaces, one engine</span>
      <h2>A TUI for your laptop.<br />A web UI for your server.</h2>
      <p>
        Tidefetch serves two jobs from one binary. Run it as a terminal download
        manager on your own machine, or run <code>tidefetch serve</code> on a NAS, VPS or
        Raspberry Pi and manage the same queue from any browser on your network.
      </p>
      <div class="surface-toggle" role="tablist" aria-label="Interface preview">
        <button
          role="tab"
          aria-selected={surface === 'terminal'}
          class:active={surface === 'terminal'}
          onclick={() => (surface = 'terminal')}>
          <SquareTerminal size={14} /> Terminal UI
        </button>
        <button
          role="tab"
          aria-selected={surface === 'web'}
          class:active={surface === 'web'}
          onclick={() => (surface = 'web')}>
          <Globe2 size={14} /> Web UI
        </button>
      </div>
    </div>

    <div class="surface-stage">
      {#if surface === 'terminal'}
        <MediaFrame item={media.terminalAlt.enabled ? media.terminalAlt : media.terminal} />
        <div class="surface-side">
          <span class="side-tag primary">For individuals</span>
          <h3>A CLI tool, not a service</h3>
          <ul class="surface-points">
            <li>btop-style braille charts and disk gauges</li>
            <li>Full mouse support inside the terminal</li>
            <li>Queue, files, peers, piece map and options</li>
            <li>Perfect over SSH, tmux and serial consoles</li>
          </ul>
        </div>
      {:else}
        <MediaFrame item={media.web} />
        <div class="surface-side">
          <span class="side-tag">For homelabs</span>
          <h3>Self-hosted on headless boxes</h3>
          <ul class="surface-points">
            <li>WebSocket push — no per-tab polling storms</li>
            <li>Virtualized lists and canvas piece maps</li>
            <li>Password login, rate limits, strict CSP</li>
            <li>Responsive from ultrawide down to phone</li>
          </ul>
        </div>
      {/if}
    </div>
  </section>

  <!-- ── Install ──────────────────────────────────────────────────── -->
  <section class="section" id="install">
    <div class="install-panel">
      <div class="install-copy">
        <span class="kicker">Install it your way</span>
        <h2>One line on macOS,<br />Linux and Windows.</h2>
        <p>
          A single static binary with no runtime to install. Take the one-liner, or use
          Homebrew, Docker and Helm if you would rather something else owned the upgrade.
        </p>
        <a class="link-arrow" href={href('docs/installation')}>Every install method <ArrowRight size={14} /></a>
      </div>
      <div class="command-card">
        <div class="command-tabs" role="tablist" aria-label="Install methods">
          <button role="tab" aria-selected={installTab === 'script'} class:active={installTab === 'script'} onclick={() => (installTab = 'script')}>Script</button>
          <button role="tab" aria-selected={installTab === 'brew'} class:active={installTab === 'brew'} onclick={() => (installTab = 'brew')}>Homebrew</button>
          <button role="tab" aria-selected={installTab === 'docker'} class:active={installTab === 'docker'} onclick={() => (installTab = 'docker')}>Docker</button>
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

    <div class="registry-grid">
      {#each registries as reg (reg.name)}
        <div class="registry">
          <strong>{reg.name}</strong>
          <code>{reg.cmd}</code>
        </div>
      {/each}
    </div>
  </section>

  <!-- ── Deploy ───────────────────────────────────────────────────── -->
  <section class="section" id="deploy">
    <div class="section-head">
      <span class="kicker">Only if you want the web UI</span>
      <h2>Headless servers.<br />Homelabs. Clusters.</h2>
      <p>
        Skip this entire section if you just want the terminal UI — a single binary is
        all you need. These guides are for running <code>tidefetch serve</code> as a
        long-lived service, with documented storage, health checks, secrets and backups.
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
      <div><span>YOUR TERMINAL</span><b>tidefetch TUI</b></div>
      <ArrowRight size={15} />
      <div><span>OR PORT 8210</span><b>Web UI</b></div>
      <ArrowRight size={15} />
      <div><span>PRIVATE RPC</span><b>aria2 engine</b></div>
      <ArrowRight size={15} />
      <div><span>PERSISTENT</span><b>Your storage</b></div>
    </div>
  </section>

  <!-- ── Closing ──────────────────────────────────────────────────── -->
  <section class="closing">
    <div class="closing-glow" aria-hidden="true"></div>
    <span class="closing-mark"><Server size={26} /></span>
    <h2>Give aria2 the interface<br />it deserves.</h2>
    <p>For TUI lovers, headless servers and self-hosted homelabs.</p>
    <div class="closing-actions">
      <a class="btn-primary" href={href('#install')}><Package size={16} /> Install Tidefetch</a>
      <a class="btn-ghost" href={repo} target="_blank" rel="noreferrer"><Zap size={15} /> View source</a>
    </div>
  </section>
</main>
