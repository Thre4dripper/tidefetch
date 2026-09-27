# Homelab operations

Running Tidefetch as a long-lived service on a NAS, mini PC, VM or small cluster. The baseline is one instance, persistent config and download storage, a strong password, and TLS or a VPN for anything beyond the LAN.

## Topology

```text
LAN / VPN clients
       │
       ▼
TLS reverse proxy :443
       │ private network
       ▼
tidefetch serve :8210 ──loopback RPC──> aria2
       │
       ├── /config     config, session, history, password hash
       └── /downloads  finished and partial files
```

The container image keeps aria2's RPC on loopback; never publish port 6800. Ports 6881 TCP and UDP are optional and only improve inbound BitTorrent connectivity.

Pick a platform guide: [Docker and Podman](deployment/docker.md), [Docker Swarm](deployment/swarm.md), [Kubernetes and k3s](deployment/kubernetes.md), [Unraid](deployment/unraid.md) or [bare metal with systemd](deployment/systemd.md).

## Storage

Choose paths covered by your backup policy and hand them to UID/GID 1000, which the image runs as:

```sh
sudo mkdir -p /srv/tidefetch/config /srv/downloads
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

- On NFS, confirm that root squashing and UID mapping still let UID 1000 create, rename and delete files; aria2 renames files on completion.
- Avoid SMB/CIFS for in-progress downloads when a local filesystem or NFS is available. If CIFS is unavoidable, mount it on the host with `uid=1000,gid=1000` before starting the container.
- Put in-progress downloads on SSD when you expect many concurrent writes, and move finished files afterwards if needed.
- Keep `file-allocation=none`, the default, on copy-on-write or thin-provisioned storage.

## Networking

| Port | Protocol | Required | Purpose |
| --- | --- | --- | --- |
| `8210` | TCP | Yes | Web UI and HTTP API |
| `6881` | TCP | Optional | Incoming BitTorrent peers |
| `6881` | UDP | Optional | DHT and UDP trackers |
| `6800` | TCP | Never publish | Internal aria2 RPC |

Forward 6881 TCP and UDP from your router only if you use BitTorrent and want inbound peers. Never forward 8210 to the internet without TLS and a password.

For remote access, a VPN such as Tailscale or WireGuard is simpler and safer than a public port. For a public hostname, follow [Reverse proxy and TLS](reverse-proxy.md).

## Secrets

Prefer a password file over an environment variable, so the password never shows in `docker inspect` or `ps`:

```sh
sudo mkdir -p /opt/tidefetch/secrets
openssl rand -base64 36 | sudo tee /opt/tidefetch/secrets/web_password >/dev/null
sudo chmod 600 /opt/tidefetch/secrets/web_password
```

Mount it and set `TIDEFETCH_PASSWORD_FILE`, as in the [Compose secrets example](deployment/docker.md#keep-the-password-out-of-the-file).

## Backups

`/config` is small and irreplaceable: it holds the settings, the bcrypt password hash, the RPC secret, the aria2 session and the download history. Stop the service first so aria2 flushes its session, then archive it:

```sh
docker compose stop tidefetch
sudo tar -C /srv/tidefetch -czf "tidefetch-config-$(date +%F).tar.gz" config
docker compose start tidefetch
```

For a named volume:

```sh
docker run --rm \
  -v tidefetch-config:/source:ro \
  -v "$PWD:/backup" \
  alpine tar -C /source -czf /backup/tidefetch-config.tar.gz .
```

Back up `/downloads` according to what the files are worth; partial downloads can be recreated. [Data and persistence](data-and-persistence.md) lists every file.

### Restore

```sh
docker compose down
sudo rm -rf /srv/tidefetch/config/*
sudo tar -C /srv/tidefetch/config -xzf tidefetch-config-2026-01-31.tar.gz --strip-components=1
sudo chown -R 1000:1000 /srv/tidefetch/config
docker compose up -d
docker compose logs tidefetch       # aria2 should report the restored session
```

## Upgrades and rollback

Pin production deployments to a release tag rather than `latest`, using any tag from the [releases page](https://github.com/Thre4dripper/tidefetch/releases):

```yaml
image: ghcr.io/thre4dripper/tidefetch:0.1.0
```

```sh
docker compose pull
docker compose up -d
docker image prune -f
```

Roll back by restoring the previous tag and running `docker compose up -d` again. The config format is forward-compatible, but take a config backup before crossing a major version.

## Health and monitoring

The image's health check requests `/` every 30 seconds:

```sh
docker inspect --format '{{json .State.Health}}' tidefetch
docker stats tidefetch
```

An external monitor can check `https://tidefetch.example.com/`. A `200` proves the server is up; the signed-in UI separately shows whether aria2 is connected. Do not poll aggressively: Tidefetch already runs one efficient poll loop against aria2 and fans updates out to every browser.

## Tuning

- Limit concurrent downloads and per-download connections on low-power CPUs (Settings → Transfer).
- Keep the browser and the Tidefetch server on the same LAN when the aria2 daemon is remote.
- Raise the container memory limit above 512 MB only for very large queues or heavy torrent metadata.

## Platform notes

**Unraid.** Use the template and follow [Unraid](deployment/unraid.md). Map `/config` to an appdata share and `/downloads` to the download share.

**TrueNAS SCALE.** Create a custom app with the GHCR image, one replica, port 8210 and two dataset mounts owned by UID/GID 1000. Keep the app on a fixed image tag and store the password in a Kubernetes Secret.

**Synology Container Manager.** Create a project from the Compose example. Map `/config` to `/volume1/docker/tidefetch` and `/downloads` to a download shared folder. Set ownership from an SSH shell if the UI cannot assign UID 1000.

**Proxmox.** Run Tidefetch in a small VM or an unprivileged LXC with Docker or Podman. For LXC bind mounts, map container UID 1000 to a writable host UID rather than making the container privileged.

**k3s and Kubernetes.** One replica on ReadWriteOnce storage; see [Kubernetes and k3s](deployment/kubernetes.md). Horizontal replicas are neither needed nor safe against one aria2 session volume.
