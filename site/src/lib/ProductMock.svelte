<script lang="ts">
  import {
    Activity,
    ArrowDownToLine,
    Check,
    Clock3,
    Download,
    FileArchive,
    FolderOpen,
    Gauge,
    History,
    Pause,
    Plus,
    Search,
    Settings2
  } from '@lucide/svelte';

  let { compact = false }: { compact?: boolean } = $props();

  const tasks = [
    { name: 'debian-13.1.0-amd64-netinst.iso', meta: '658 MB of 710 MB', progress: 92, speed: '18.4 MB/s', eta: '3s', state: 'active' },
    { name: 'archive-footage-04.tar.zst', meta: '12.8 GB of 31.2 GB', progress: 41, speed: '8.7 MB/s', eta: '36m', state: 'active' },
    { name: 'fedora-workstation-42.iso', meta: '2.4 GB of 2.4 GB', progress: 100, speed: 'Complete', eta: '', state: 'done' },
    { name: 'research-dataset.meta4', meta: '4 mirrors · queued', progress: 18, speed: 'Paused', eta: '', state: 'paused' }
  ];
</script>

<div class:compact class="product-mock" aria-label="Illustrative Tidefetch dashboard preview">
  <aside class="mock-sidebar">
    <div class="mock-brand"><span>⬡</span><b>Tidefetch</b></div>
    <nav aria-label="Product preview navigation">
      <span class="active"><Download size={15} /> Downloads <small>4</small></span>
      <span><Activity size={15} /> Active <small>2</small></span>
      <span><Clock3 size={15} /> Queued <small>1</small></span>
      <span><Check size={15} /> Finished <small>1</small></span>
      <span><History size={15} /> History</span>
    </nav>
    <div class="mock-storage">
      <div><span>Storage</span><b>68%</b></div>
      <i><em></em></i>
      <small>354 GB free</small>
    </div>
    <span class="mock-setting"><Settings2 size={15} /> Settings</span>
  </aside>

  <section class="mock-main">
    <header class="mock-topbar">
      <div>
        <span class="eyebrow">DOWNLOADS</span>
        <strong>Good morning</strong>
      </div>
      <label><Search size={15} /><span>Search downloads</span></label>
      <button type="button"><Plus size={15} /> Add download</button>
    </header>

    <div class="mock-metrics">
      <div><span><ArrowDownToLine size={14} /> DOWN</span><strong>27.1 <small>MB/s</small></strong></div>
      <div><span><Gauge size={14} /> SESSION</span><strong>84.6 <small>GB</small></strong></div>
      <div class="mock-bars" aria-hidden="true">
        {#each [22, 28, 24, 36, 46, 42, 58, 74, 61, 78, 64, 82, 70, 91, 76, 68, 55, 62] as height}
          <i style={`height:${height}%`}></i>
        {/each}
      </div>
    </div>

    <div class="mock-list-head"><span>ALL DOWNLOADS</span><span>SIZE</span><span>STATUS</span></div>
    <div class="mock-tasks">
      {#each tasks as task, index}
        <article class:focus={index === 0}>
          <div class="file-icon"><FileArchive size={17} /></div>
          <div class="task-copy">
            <strong>{task.name}</strong>
            <span>{task.meta}</span>
            <i><em class={task.state} style={`width:${task.progress}%`}></em></i>
          </div>
          <div class="task-speed"><b>{task.speed}</b><span>{task.eta ? `ETA ${task.eta}` : task.state}</span></div>
          <button class="task-action" type="button" aria-label={`Pause ${task.name}`}><Pause size={14} /></button>
        </article>
      {/each}
    </div>
  </section>

  <aside class="mock-detail">
    <div class="detail-head"><span>TRANSFER DETAIL</span><button type="button" aria-label="Open folder"><FolderOpen size={14} /></button></div>
    <div class="detail-file"><FileArchive size={20} /><div><strong>debian-13.1.0</strong><span>ISO disk image</span></div></div>
    <div class="detail-ring"><strong>92<small>%</small></strong><span>658 MB / 710 MB</span></div>
    <div class="detail-chart" aria-label="Illustrative transfer speed chart">
      {#each [15, 25, 22, 39, 31, 45, 52, 48, 61, 72, 59, 79, 67, 85, 76, 88] as value}
        <i style={`height:${value}%`}></i>
      {/each}
    </div>
    <dl>
      <div><dt>Connections</dt><dd>16</dd></div>
      <div><dt>Pieces</dt><dd>2,714 / 2,948</dd></div>
      <div><dt>Source</dt><dd>4 mirrors</dd></div>
    </dl>
  </aside>
</div>