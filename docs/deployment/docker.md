# Docker and Podman

The all-in-one image contains Tidefetch, its embedded web UI, and the Alpine
aria2 package. It runs as UID/GID 1000 and needs two writable volumes.

## Current source build

Until the first GHCR image is published, build from the checkout:

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch/packaging/docker
cp .env.example .env
chmod 600 .env
```

Edit `.env` and replace the password, then:

```sh
docker compose up -d --build
docker compose ps
docker compose logs -f --tail=100 tidefetch
```

Open `http://<server-ip>:8210`.

## Password file with Compose secrets

Avoid a password in `.env` by applying the secret overlay:

```sh
cd packaging/docker
mkdir -p secrets
umask 077
openssl rand -base64 36 > secrets/web_password
docker compose \
  -f docker-compose.yml \
  -f docker-compose.secrets.yml \
  up -d --build
```

The overlay mounts the file at `/run/secrets/web_password` and sets
`TIDEFETCH_PASSWORD_FILE`. Store the generated value in a password manager.

## Published image

After GHCR publication, replace the Compose `build` block with:

```yaml
image: ghcr.io/thre4dripper/tidefetch:0.2.0
```

Or run it directly:

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
  ghcr.io/thre4dripper/tidefetch:0.2.0
```

Pin a version in persistent environments. `latest` is convenient for testing,
not a rollback strategy.

## Storage permissions

For bind mounts:

```sh
sudo mkdir -p /srv/tidefetch/config /srv/downloads
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

On SELinux hosts, append `:Z` to both bind mounts. On rootless Podman, use
`podman unshare chown -R 1000:1000 <path>` if direct ownership does not map.

## Ports

- `8210/tcp`: web UI, required
- `6881/tcp`: inbound BitTorrent peers, optional
- `6881/udp`: DHT and UDP trackers, optional

The aria2 RPC listener is internal and must not be published.

## Reverse proxy network

When Caddy, Nginx, or Traefik runs in Docker, put both services on an external
network and remove the host mapping for 8210:

```sh
docker network create proxy
```

```yaml
services:
  tidefetch:
    networks: [proxy]

networks:
  proxy:
    external: true
```

Proxy to `http://tidefetch:8210`. See [Reverse proxy and TLS](../reverse-proxy.md).

## Lifecycle

```sh
# Status and logs
docker compose ps
docker compose logs -f tidefetch

# Graceful restart
docker compose restart tidefetch

# Stop while preserving volumes
docker compose down

# Pull a published update
docker compose pull
docker compose up -d
```

## Backup and restore

Stop the service for a consistent config/session snapshot:

```sh
docker compose stop tidefetch
sudo tar -C /srv/tidefetch -czf tidefetch-config.tar.gz config
docker compose start tidefetch
```

Restore:

```sh
docker compose down
sudo mv /srv/tidefetch/config /srv/tidefetch/config.old
sudo mkdir /srv/tidefetch/config
sudo tar -C /srv/tidefetch/config -xzf tidefetch-config.tar.gz --strip-components=1
sudo chown -R 1000:1000 /srv/tidefetch/config
docker compose up -d
```

## Podman

Podman uses the same image and ports:

```sh
podman run -d --name tidefetch --replace \
  -p 8210:8210 \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config:Z \
  -v "$HOME/Downloads:/downloads:Z" \
  ghcr.io/thre4dripper/tidefetch:0.2.0
```

Generate a user systemd unit with Quadlet or the Podman tooling supported by
the host distribution when automatic startup is required.
