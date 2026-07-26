# ⬡ tidefetch

**A fast terminal UI + self-hosted web UI for [aria2](https://aria2.github.io)** —
the download engine that speaks HTTP(S), FTP, SFTP, BitTorrent and Metalink.

[Product site](https://thre4dripper.github.io/tidefetch/) ·
[Documentation](docs/index.md) ·
[Installation](docs/installation.md) ·
[Homelab guide](docs/homelab.md)

One binary, three faces:

- `tidefetch` — a polished, mouse-driven **TUI** with live charts, queue
  control, history and an IDM-style workflow
- `tidefetch serve` — a modern, resource-friendly **web UI** for homelabs
  and remote servers (WebSocket deltas, virtualized lists, canvas piece
  maps — built to stay smooth where older aria2 frontends choke)
- `packaging/docker` — an **all-in-one container**: web UI + aria2 engine,
  one port, two volumes, done

```
 ⬡ tidefetch   ▼ 12.4 MB/s  ▲ 1.2 MB/s  ▁▂▃▅▇█▆▅▃▂  active 3 · queued 2 · stopped 8      ● aria2 1.37.0
  1 All   2 Active   3 Queued   4 Finished  │  ＋ Add   ⟲ History   ⚙ Settings
┃ ⇣ ubuntu-24.04.2-desktop-arm64.iso                                              ⧉ ACTIVE
  ██████████████████████▌───────────────────────  46.8%  2.5 GB / 5.3 GB  12.4 MB/s  eta 3m52s  seeds 74
  ⏸ linux-6.9.tar.xz                                                                PAUSED
  ████████▏──────────────────────────────────────  17.2%  24 MB / 142 MB  paused
```

tidefetch is an independent project powered by aria2 and is not affiliated
with the aria2 project.

## Features

**Both UIs**

- Add anything aria2 supports — HTTP(S), FTP, SFTP, magnets, `.torrent`,
  `.metalink`/`.meta4`, multi-mirror
- Full queue control: pause/resume (single or all), reorder, remove,
  remove + delete files, retry failed
- Link inspector — IDM-style pre-download check: resumability, real size,
  server filename, content type
- Advanced per-download options: rename, split/connections, speed limits,
  file allocation (`none`/`prealloc`/`trunc`/`falloc`), checksum, referer,
  custom headers, proxy, user agent, seed ratio, add-paused
- Live per-download details: files (with selection), peers, servers,
  piece map, speeds
- Global aria2 settings editor (curated groups + all 100-plus raw options)
- Persistent download history with automatic IDM-style categories,
  search, filter, one-click re-download
- Daemon management: attaches to a running aria2, or spawns one with
  session persistence — downloads keep running when the UI is closed

**Terminal UI** — btop-style mouse support, sparkline charts, disk gauge,
built-in file browser, keyboard-first workflow.

**Web UI** — dark glass design, WebSocket push (no per-tab polling storms),
virtualized task list, canvas piece map (a million pieces render as ≤240
buckets), server-side link probe, password login with bcrypt + rate
limiting, strict CSP, same-origin guards.

## Install

The repository does not have a tagged binary or registry release yet. The
current source and local container builds work now. See the
[installation guide](docs/installation.md) for release archives, Go,
Homebrew, Winget, Scoop, Chocolatey, AUR, Nix, DEB, and RPM status.

Native builds require [aria2](https://aria2.github.io) (`brew install aria2` /
`apt install aria2`), Go 1.26.1, and Node 20+.

```sh
# from a checkout
make build && ./tidefetch          # TUI
./tidefetch serve                  # web UI on http://127.0.0.1:8210

# or install into $GOPATH/bin
make install
```

### Docker (all-in-one, self-hosted)

```sh
cd packaging/docker
cp .env.example .env               # replace the password in .env
docker compose up -d --build
# → http://<host>:8210
```

The image bundles the aria2 engine (installed from the Alpine package;
aria2 is GPL-2.0-or-later — see its [source](https://github.com/aria2/aria2)).
Volumes: `/config` (settings, session, history) and `/downloads`.

Production examples: [Docker](docs/deployment/docker.md) ·
[Swarm](docs/deployment/swarm.md) ·
[Kubernetes/k3s](docs/deployment/kubernetes.md) ·
[Unraid](docs/deployment/unraid.md) ·
[reverse proxies](docs/reverse-proxy.md)

## Usage

```sh
tidefetch                                  # open the TUI
tidefetch https://example.com/file.iso     # open and start downloading
tidefetch "magnet:?xt=urn:btih:..."        # magnets work too
tidefetch -url ws://nas:6800/jsonrpc -secret s3cret   # remote daemon
tidefetch serve -host 0.0.0.0 -password s3cret        # LAN web UI
tidefetch doctor                           # diagnose your setup
```

On first run a config file is created at `~/.config/tidefetch/config.json`
with a random RPC secret. History and the aria2 session live in
`~/.local/share/tidefetch/`. (Configs from the old *aria2tui* name migrate
automatically.)

### Web UI security

- Loopback binds work without a password; any other bind **requires**
  `-password` (stored as a bcrypt hash) or an explicit `-no-auth` for
  reverse-proxy setups.
- The aria2 RPC secret never reaches the browser — the UI talks only to
  the authenticated tidefetch backend.

## TUI keys

| Key | Action |
| --- | --- |
| `j/k` `↑/↓` | move · `g/G` top/bottom · `ctrl+d/u` half page |
| `1–4` | filter: All / Active / Queued / Finished |
| `space` / `p` | pause · resume selected |
| `P` / `R` | pause all · resume all |
| `enter` / `i` | details (info · files · peers · servers) |
| `a` | add downloads (URLs, magnets, torrent files) |
| `x` / `D` | remove · remove **and delete files** |
| `r` | retry a failed download |
| `J/K` | move in queue |
| `S` | cycle sort |
| `/` | search; `esc` clears |
| `o` / `y` | open folder · copy URL/magnet |
| `t` / `f` | side panel · file browser |
| `[` `]` / `{` `}` | global ↓ / ↑ speed limit −/+ |
| `h` / `s` / `?` | history · settings · help |
| `q` / `Q` | quit · quit **and** shut the daemon down |

## Standalone RPC client

The aria2 JSON-RPC client is an independent package with no UI
dependencies — use it in your own tools:

```go
import "github.com/Thre4dripper/tidefetch/pkg/aria2"

client, _ := aria2.Dial(ctx, "ws://127.0.0.1:6800/jsonrpc", "secret")
gid, _ := client.AddURI(ctx, []string{"https://example.org/file.iso"},
    aria2.Options{"dir": "/downloads", "split": "16"})

for n := range client.Notifications() {
    fmt.Println(n.Method, n.GID) // aria2.onDownloadComplete abc123…
}
```

Covers the full method surface: add (URI/torrent/metalink), tell*, pause,
remove, queue position, per-download and global options, global stats,
`system.multicall`, session save/shutdown, and event notifications over
WebSocket.

## Project layout

```
cmd/tidefetch/     entry point (tui · serve · doctor)
pkg/aria2/         standalone aria2 JSON-RPC client
internal/tui/      Bubble Tea terminal UI
internal/server/   web backend: broker, REST API, WebSocket hub, auth
internal/daemon/   aria2c discovery / spawn / attach
internal/config/   settings (+ legacy migration)
internal/history/  persistent categorized history
web/               Svelte 5 + Vite frontend (embedded via go:embed)
site/              standalone product/showcase site + media slots
docs/              install, configuration and homelab operations
packaging/         Docker, Swarm, Kubernetes and Unraid artifacts
```

## Development

```sh
make build      # web UI + binary (embedded assets)
make backend    # Go only, reuses last web build
make web        # frontend only
make site       # type-check + build the standalone product site
make site-dev   # product site development server
make test       # unit + integration tests (integration skips without aria2c)
make docker     # build the container image
```

Frontend dev loop: `cd web && npm run dev` proxies `/api` to a running
`tidefetch serve` on :8210.

## License

MIT for everything in this repository. The aria2 engine is a separate
GPL-2.0-or-later project; tidefetch talks to it over JSON-RPC and does not
link against it. Container images that bundle aria2 include its license
and point to its source.
