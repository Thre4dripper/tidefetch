# Homelab operations

This guide covers a long-running Tidefetch service on a NAS, mini PC, VM, or
small cluster. The recommended baseline is one Tidefetch instance, persistent
config and download storage, a strong password, and TLS or a private VPN for
remote access.

## Recommended topology

```text
LAN / VPN clients
       │
       ▼
TLS reverse proxy :443
       │ private container network
       ▼
Tidefetch :8210 ──loopback RPC──> aria2
       │
       ├── /config     session, history, password hash
       └── /downloads  completed and partial files
```

The all-in-one image keeps aria2 RPC private. Do not publish port 6800 from the
container. Ports 6881 TCP and UDP are optional but improve inbound BitTorrent
connectivity.

## Host preparation

Choose paths that are included in the NAS backup policy:

```sh
sudo mkdir -p /srv/tidefetch/config /srv/downloads
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

The image runs as UID/GID `1000:1000`. On NFS, confirm that root squashing and
UID mapping still permit UID 1000 to create, rename, and delete files.

Avoid SMB/CIFS mounts for active partial downloads when a local filesystem or
NFS is available. If CIFS is necessary, mount it on the host with `uid=1000`,
`gid=1000`, and appropriate file/directory modes before starting the container.

## Compose deployment

Create a deployment directory:

```sh
sudo mkdir -p /opt/tidefetch/secrets
cd /opt/tidefetch
openssl rand -base64 36 | sudo tee secrets/web_password >/dev/null
sudo chmod 600 secrets/web_password
```

Use the published image after a release, or replace `image` with the `build`
block from the repository's Compose file while developing:

```yaml
services:
  tidefetch:
    image: ghcr.io/thre4dripper/tidefetch:latest
    container_name: tidefetch
    restart: unless-stopped
    environment:
      TIDEFETCH_PASSWORD_FILE: /run/secrets/web_password
      TZ: Etc/UTC
    secrets:
      - web_password
    ports:
      - "8210:8210"
      - "6881:6881"
      - "6881:6881/udp"
    volumes:
      - /srv/tidefetch/config:/config
      - /srv/downloads:/downloads
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    deploy:
      resources:
        limits:
          memory: 512M

secrets:
  web_password:
    file: ./secrets/web_password
```

Launch and inspect:

```sh
docker compose up -d
docker compose ps
docker compose logs -f --tail=100 tidefetch
curl -I http://127.0.0.1:8210/
```

Compose ignores `deploy.replicas`, but modern Compose applies supported
resource limits. Tidefetch is stateful and should have one replica.

## Networking

| Port | Protocol | Required | Purpose |
| --- | --- | --- | --- |
| `8210` | TCP | Yes | Tidefetch HTTP/WebSocket UI |
| `6881` | TCP | Optional | Incoming BitTorrent peers |
| `6881` | UDP | Optional | DHT and UDP tracker traffic |
| `6800` | TCP | No | Internal aria2 RPC; keep private |

Forward 6881 TCP/UDP from the router to the Tidefetch host only if BitTorrent
is used and inbound connectivity is desired. Never forward 8210 directly to
the public internet without TLS and authentication.

For private remote access, Tailscale, WireGuard, or another VPN is simpler and
safer than a public port forward. For a public hostname, follow the
[reverse proxy guide](reverse-proxy.md).

## Backups

The `/config` volume is small and critical. It contains the config, bcrypt
password hash, RPC secret, aria2 session, and Tidefetch history.

Consistent filesystem backup:

```sh
cd /opt/tidefetch
docker compose stop tidefetch
sudo tar -C /srv/tidefetch -czf "tidefetch-config-$(date +%F).tar.gz" config
docker compose start tidefetch
```

Back up `/downloads` according to the value of the data. Partial downloads can
usually be recreated; completed archives may require normal 3-2-1 protection.

Named-volume backup:

```sh
docker run --rm \
  -v tidefetch_config:/source:ro \
  -v "$PWD:/backup" \
  alpine tar -C /source -czf /backup/tidefetch-config.tar.gz .
```

## Restore

Stop the service before replacing config data:

```sh
docker compose down
sudo rm -rf /srv/tidefetch/config/*
sudo tar -C /srv/tidefetch/config -xzf tidefetch-config-2026-07-26.tar.gz \
  --strip-components=1
sudo chown -R 1000:1000 /srv/tidefetch/config
docker compose up -d
```

Run `docker compose logs tidefetch` and confirm that aria2 restores the
expected session.

## Upgrades and rollback

Pin production deployments to a release instead of `latest`:

```yaml
image: ghcr.io/thre4dripper/tidefetch:0.2.0
```

Upgrade:

```sh
docker compose pull
docker compose up -d
docker image prune -f
```

Rollback by restoring the previous image tag and running `docker compose up
-d`. Config migration is designed to be forward-compatible, but take a config
backup before crossing major versions.

## Health and monitoring

The image health check requests `/` every 30 seconds. Inspect it with:

```sh
docker inspect --format '{{json .State.Health}}' tidefetch
docker stats tidefetch
```

An external uptime monitor can check `https://tidefetch.example.com/`. A 200
response proves the broker is serving; the authenticated UI separately reports
aria2 reconnection state.

Avoid aggressive external polling. Tidefetch already owns one efficient RPC
poll loop and distributes deltas to every browser.

## NAS and virtualization notes

### Unraid

Use the provided XML template and follow [Unraid deployment](deployment/unraid.md).
Map `/config` to an appdata share and `/downloads` to the final download share.

### TrueNAS SCALE

Use a custom app with the GHCR image, one replica, port 8210, and two host-path
or dataset mounts. Set dataset ownership to UID/GID 1000. Keep the app on a
fixed image tag and use a Kubernetes secret for the password.

### Synology Container Manager

Create a project from the Compose example. Map `/config` to
`/volume1/docker/tidefetch` and `/downloads` to a download shared folder. Set
ownership from an SSH shell before launch if the UI cannot assign UID 1000.

### Proxmox

Run Tidefetch in a small VM or an unprivileged LXC with Docker/Podman. For LXC
bind mounts, map container UID 1000 to a writable host UID. Do not solve a UID
mapping issue by enabling a privileged Tidefetch container.

### k3s and Kubernetes

Use one replica with ReadWriteOnce storage. Follow the
[Kubernetes deployment](deployment/kubernetes.md). Multiple browser clients
already share one broker; horizontal Tidefetch replicas are neither required
nor safe against one aria2 session volume.

## Performance tuning

- Put partial downloads on SSD when unpacking or many concurrent writes are
  expected; move completed files afterward if needed.
- Use `file-allocation=none` on copy-on-write or thin-provisioned storage.
- Limit concurrent downloads and per-task connections for low-power NAS CPUs.
- Keep the browser dashboard and aria2 broker on the same LAN when using a
  remote daemon.
- Increase container memory above 512 MB only for exceptionally large queues
  or heavy torrent metadata workloads.
