# Tidefetch

Tidefetch is a download manager for the [aria2](https://aria2.github.io) engine. It ships as one static binary with two interfaces: a keyboard-first terminal UI for the machine in front of you, and a self-hosted web UI for a NAS, VPS or Raspberry Pi that has no screen.

## Who it is for

| You are | Use |
| --- | --- |
| Working in a terminal on your own machine | `tidefetch`, the terminal UI |
| Running downloads on a headless server or homelab | `tidefetch serve`, the web UI |
| Scripting downloads or building a dashboard | The [HTTP API](api.md) behind the web UI |

Both interfaces drive the same aria2 daemon and share one configuration, queue and history, so you can switch between them at any time.

## What it does

- **Every aria2 protocol.** HTTP(S), FTP, SFTP, BitTorrent, magnet links, Metalink and multi-mirror downloads.
- **Full queue control.** Add, pause, resume, retry, reorder and remove downloads; pick the files inside a torrent; edit per-download and global aria2 options.
- **Live telemetry.** Speed graphs, per-download speed history, piece maps, connection and peer lists, disk usage.
- **Persistent state.** aria2 saves the queue to disk, so downloads survive restarts. Tidefetch keeps a searchable, categorised history of finished downloads.
- **Safe to expose.** The web UI has bcrypt passwords, HttpOnly sessions, login rate limiting, a same-origin guard and a strict Content Security Policy. The aria2 RPC secret never leaves the server.
- **Small.** One Go binary and no runtime. Multi-arch container images, a Helm chart, and Compose, Swarm, Unraid and systemd assets.

## How it fits together

```text
Terminal UI ─┐
             ├──> tidefetch ──private JSON-RPC──> aria2 ──> your storage
Browser ─────┘    (auth, history, HTTP API)
```

Tidefetch starts a local `aria2c` for you, or attaches to one you already run. In the container image aria2 is bundled and its RPC port stays on loopback; only port 8210 is exposed.

## Start here

1. [Installation](installation.md): the one-line script, Homebrew, the container image, Helm or Go.
2. [Quick start](quick-start.md): your first download in the terminal UI and the web UI.
3. [Terminal UI](terminal-ui.md) and [Web UI](web-ui.md): every screen and shortcut.
4. [Deployment](deployment/docker.md): run `tidefetch serve` as a long-lived service.

## Platform support

| Platform | Terminal UI | Web UI server |
| --- | --- | --- |
| Linux (amd64, arm64, armv7) | Yes | Yes |
| macOS (Intel, Apple silicon) | Yes | Yes |
| Windows (amd64, arm64) | Yes, Windows Terminal recommended | Yes |
| Docker, Kubernetes, Unraid | No | Yes, aria2 bundled |

Tidefetch needs aria2 1.36 or newer on the host, except in the container image where it is bundled. The web UI supports current Chrome, Firefox, Safari and Edge.

## Licensing

Tidefetch is MIT licensed. aria2 is a separate GPL-2.0-or-later project; Tidefetch talks to it over JSON-RPC and does not link against it. Tidefetch is an independent project and is not affiliated with aria2.
