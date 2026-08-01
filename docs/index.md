# Tidefetch documentation

Tidefetch is a terminal UI for the [aria2](https://aria2.github.io) download
engine, with an optional self-hosted web UI. One static binary, no runtime,
no cloud account.

## Start here

| Goal | Guide |
| --- | --- |
| Install on macOS, Linux, Windows or Docker | [Installation](installation.md) |
| Understand flags, files and authentication | [Configuration](configuration.md) |
| Know what to put on a volume | [Data & persistence](data-and-persistence.md) |
| Run it on a NAS or home server | [Homelab operations](homelab.md) |
| Put the web UI behind HTTPS | [Reverse proxies](reverse-proxy.md) |
| Diagnose a problem | [Troubleshooting](troubleshooting.md) |
| Deploy with Docker or Podman | [Docker](deployment/docker.md) |
| Deploy a Swarm stack | [Docker Swarm](deployment/swarm.md) |
| Deploy to Kubernetes or k3s | [Kubernetes](deployment/kubernetes.md) |
| Install on Unraid | [Unraid](deployment/unraid.md) |
| Cut and publish a release | [Publishing](publishing.md) |

## Quick start

```sh
brew install thre4dripper/tap/tidefetch   # macOS / Linuxbrew
sudo apt install tidefetch                # Debian / Ubuntu
docker run -d -p 8210:8210 ghcr.io/thre4dripper/tidefetch
```

Then:

```sh
tidefetch            # open the terminal UI
tidefetch doctor     # verify aria2, config and paths
tidefetch serve      # optional web UI on :8210
```

## The two interfaces

**Terminal UI** — the primary interface. Keyboard-first with full mouse
support, braille throughput graphs, disk gauges, a piece map, file browser and
the complete aria2 settings surface. It runs anywhere a shell does: over SSH,
inside tmux, on a Raspberry Pi.

**Web UI** — `tidefetch serve`. The same queue, history and settings rendered
for a browser, for when downloads live on a headless machine. WebSocket push,
virtualized lists, bcrypt auth and a strict CSP.

Both talk to the same aria2 daemon and share one config and history file.

## Architecture

```text
Terminal UI ─┐
             ├──> Tidefetch ──private JSON-RPC──> aria2 ──> your storage
Browser ─────┘     (auth, history, API)
```

The aria2 RPC secret never reaches the browser. In the container image RPC
stays on loopback and only port 8210 is exposed.

## Core concepts

- **Download directory** — where finished files land. Set globally in config or
  per-task when adding.
- **Session** — aria2 persists the queue to disk, so downloads survive restarts
  of both the UI and the daemon.
- **History** — completed and failed transfers are recorded with IDM-style
  categories, searchable and re-downloadable.
- **Daemon mode** — Tidefetch spawns a local `aria2c` automatically, or attaches
  to an existing one with `-url` and `-secret`.

## Support baseline

- **Linux** — primary target for servers and containers
- **macOS** — native TUI and web server
- **Windows** — native binary; Windows Terminal recommended
- **aria2** — 1.36 or newer
- **Browsers** — current Chrome, Firefox, Safari and Edge

Tidefetch is an independent project and is not affiliated with aria2. Tidefetch
is MIT licensed; aria2 is GPL-2.0-or-later and is used over JSON-RPC as a
separate process.
