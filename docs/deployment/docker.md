# Docker and Podman

The all-in-one image contains Tidefetch, the embedded web UI and Alpine's aria2 package. It runs as UID/GID 1000 and needs two writable volumes and a password.

```text
ghcr.io/thre4dripper/tidefetch:latest      # GitHub Container Registry
ijlalahmad/tidefetch:latest                # Docker Hub mirror
```

Images are built for `linux/amd64` and `linux/arm64`, and Docker picks the right one. Every release adds tags such as `0.1.0` and `0.1` next to `latest`. Use `latest` to try it out and a release tag for anything you rely on.

## docker run

```sh
docker run -d \
  --name tidefetch \
  --restart unless-stopped \
  --security-opt no-new-privileges \
  --cap-drop ALL \
  -p 8210:8210 \
  -p 6881:6881 \
  -p 6881:6881/udp \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -e TZ='Etc/UTC' \
  -v /srv/tidefetch/config:/config \
  -v /srv/downloads:/downloads \
  ghcr.io/thre4dripper/tidefetch:latest
```

Open `http://<server-ip>:8210` and sign in with the password. The container runs `tidefetch serve -host 0.0.0.0 -port 8210 -dir /downloads`, which refuses to start without a password because it listens on all interfaces.

## Docker Compose

A minimal standalone file:

```yaml
services:
  tidefetch:
    image: ghcr.io/thre4dripper/tidefetch:latest
    container_name: tidefetch
    restart: unless-stopped
    environment:
      TIDEFETCH_PASSWORD: replace-this-password
      TZ: Etc/UTC
    ports:
      - "8210:8210"        # web UI
      - "6881:6881"        # BitTorrent peers, optional
      - "6881:6881/udp"    # DHT and UDP trackers, optional
    volumes:
      - /srv/tidefetch/config:/config
      - /srv/downloads:/downloads
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
```

```sh
docker compose up -d
docker compose logs -f --tail=100 tidefetch
```

### Keep the password out of the file

Use a Compose secret and `TIDEFETCH_PASSWORD_FILE` instead of a plain environment value:

```yaml
services:
  tidefetch:
    environment:
      TIDEFETCH_PASSWORD_FILE: /run/secrets/web_password
    secrets:
      - web_password

secrets:
  web_password:
    file: ./secrets/web_password
```

```sh
mkdir -p secrets && umask 077
openssl rand -base64 36 > secrets/web_password
docker compose up -d
```

The repository's `packaging/docker/docker-compose.secrets.yml` is this overlay. Apply it with `-f docker-compose.yml -f docker-compose.secrets.yml`.

### Build from source instead

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch/packaging/docker
cp .env.example .env && chmod 600 .env    # set TIDEFETCH_PASSWORD in .env
docker compose up -d --build
```

## Volumes and permissions

| Mount | Holds |
| --- | --- |
| `/config` | `config.json` (settings, RPC secret, password hash), `history.json`, the aria2 session and DHT tables |
| `/downloads` | Finished files, plus `.aria2` control files for downloads in progress |

Both are required. Without `/config`, the queue, history and password reset on every restart. [Data and persistence](../data-and-persistence.md) lists every file.

For bind mounts, hand the directories to UID/GID 1000 first:

```sh
sudo mkdir -p /srv/tidefetch/config /srv/downloads
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

On SELinux hosts append `:Z` to both bind mounts. With rootless Podman, use `podman unshare chown -R 1000:1000 <path>` when direct ownership does not map.

## Ports

| Port | Required | Purpose |
| --- | --- | --- |
| `8210/tcp` | Yes | Web UI and HTTP API |
| `6881/tcp` | No | Incoming BitTorrent peers |
| `6881/udp` | No | DHT and UDP trackers |

aria2's RPC listener stays on loopback inside the container. Never publish port 6800.

## Behind a reverse proxy

When Caddy, Nginx or Traefik also runs in Docker, put both on a shared network and stop publishing 8210 on the host:

```sh
docker network create proxy
```

```yaml
services:
  tidefetch:
    networks: [proxy]
    # no "ports:" entry for 8210

networks:
  proxy:
    external: true
```

Proxy to `http://tidefetch:8210`. See [Reverse proxy and TLS](../reverse-proxy.md).

## Everyday operations

```sh
docker compose ps                              # status and health
docker compose logs -f tidefetch               # logs
docker compose restart tidefetch               # graceful restart
docker compose pull && docker compose up -d    # upgrade
docker compose down                            # stop; volumes stay
docker exec tidefetch tidefetch doctor         # diagnostics inside the container
```

The image has a health check that requests `/` every 30 seconds:

```sh
docker inspect --format '{{json .State.Health}}' tidefetch
```

Stop with `docker compose stop` rather than `docker kill`, so aria2 flushes its session first.

## Backup and restore

Stop the service for a consistent copy of `/config`:

```sh
docker compose stop tidefetch
sudo tar -C /srv/tidefetch -czf "tidefetch-config-$(date +%F).tar.gz" config
docker compose start tidefetch
```

Restore into a stopped service and fix ownership:

```sh
docker compose down
sudo mv /srv/tidefetch/config /srv/tidefetch/config.old
sudo mkdir /srv/tidefetch/config
sudo tar -C /srv/tidefetch/config -xzf tidefetch-config-2026-01-31.tar.gz --strip-components=1
sudo chown -R 1000:1000 /srv/tidefetch/config
docker compose up -d
```

## Podman

```sh
podman run -d --name tidefetch --replace \
  -p 8210:8210 \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config:Z \
  -v /srv/downloads:/downloads:Z \
  ghcr.io/thre4dripper/tidefetch:latest
```

Use Quadlet or `podman generate systemd` for a service that starts at boot.
