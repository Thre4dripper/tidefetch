# Tidefetch documentation

Tidefetch is a terminal UI and self-hosted web UI for the aria2 download
engine. It can manage a local aria2 process automatically or connect to an
existing daemon over JSON-RPC.

## Start here

| Goal | Guide |
| --- | --- |
| Install on macOS, Linux, or Windows | [Installation](installation.md) |
| Understand flags, files, and authentication | [Configuration](configuration.md) |
| Run on a NAS or home server | [Homelab operations](homelab.md) |
| Put Tidefetch behind HTTPS | [Reverse proxies](reverse-proxy.md) |
| Diagnose a failed deployment | [Troubleshooting](troubleshooting.md) |
| Run with Docker or Podman | [Docker](deployment/docker.md) |
| Deploy a Docker Swarm stack | [Docker Swarm](deployment/swarm.md) |
| Deploy to Kubernetes or k3s | [Kubernetes](deployment/kubernetes.md) |
| Install on Unraid | [Unraid](deployment/unraid.md) |

## Fastest current installation

The repository does not have a tagged binary or registry release yet. The
commands below build the current checkout and are usable now.

### Container

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
TIDEFETCH_PASSWORD='replace-this-password' \
  docker compose -f packaging/docker/docker-compose.yml up -d --build
```

Open `http://<server-ip>:8210`.

### Native

Install aria2, Go 1.26.1, and Node.js 20 or newer, then:

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
make build
./tidefetch doctor
./tidefetch
```

## Architecture

```text
Browser ──HTTP/WebSocket──> Tidefetch broker ──private JSON-RPC──> aria2
                                  │                         │
                                  ├── auth + API            ├── transfers
                                  ├── history               └── session state
                                  └── embedded web app
```

The browser never receives the aria2 RPC secret. In the all-in-one container,
RPC stays on loopback and only port 8210 is required for the dashboard. Expose
the optional BitTorrent TCP/UDP ports when accepting inbound peers matters.

## Support baseline

- Linux: primary server and container platform
- macOS: native TUI/web server and source builds
- Windows: native builds are supported by Go; WSL2 or Docker Desktop is the
  most predictable self-hosted path
- Browsers: current Chrome, Firefox, Safari, and Edge
- aria2: the system package is used natively; Alpine provides it in the image

Tidefetch is independent of and not affiliated with the aria2 project.
