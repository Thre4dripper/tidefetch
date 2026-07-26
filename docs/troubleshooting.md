# Troubleshooting

Start with the diagnostic command and service logs. Avoid deleting config or
session files until a backup exists.

## Native diagnostic

```sh
tidefetch doctor
tidefetch version
aria2c --version
```

`doctor` checks config loading, aria2 discovery, RPC connectivity, the download
directory, and the Tidefetch data directory.

## Container diagnostic

```sh
docker compose ps
docker compose logs --tail=200 tidefetch
docker inspect --format '{{json .State.Health}}' tidefetch
docker exec tidefetch tidefetch doctor
```

## Web page does not open

1. Confirm port 8210 is listening.
2. Test from the host with `curl -I http://127.0.0.1:8210/`.
3. Confirm the firewall permits the LAN client.
4. If using a proxy, test the backend directly before debugging TLS.

Linux listeners:

```sh
ss -lntp | grep 8210
```

macOS listeners:

```sh
lsof -nP -iTCP:8210 -sTCP:LISTEN
```

## Server refuses a non-loopback bind

Tidefetch requires authentication when listening beyond loopback. Supply one
of these:

```sh
tidefetch serve -host 0.0.0.0 -password 'a-long-unique-password'
TIDEFETCH_PASSWORD_FILE=/run/secrets/web_password tidefetch serve -host 0.0.0.0
```

Use `-no-auth` only behind a trusted authenticating proxy.

## Login fails

- Passwords are case-sensitive.
- `TIDEFETCH_PASSWORD` overrides `TIDEFETCH_PASSWORD_FILE` when both are set.
- Confirm the secret file contains the intended value and is readable by UID
  1000.
- Restart the service after changing an environment variable or secret mount.
- Browser sessions are in memory and a restart requires a new login.

Do not paste a plaintext password into `web_password_hash` in `config.json`.
Set it through `-password`, the environment, onboarding, or Security settings.

## aria2 is missing

Native installs require `aria2c` in `PATH`:

```sh
command -v aria2c
aria2c --version
```

Install it with `brew install aria2`, `sudo apt install aria2`, `sudo dnf
install aria2`, or the platform's package manager. The all-in-one image already
contains aria2.

## RPC connection fails

Check endpoint and secret:

```sh
tidefetch -url ws://127.0.0.1:6800/jsonrpc \
  -secret 'your-rpc-secret' -no-spawn
```

For an existing daemon, confirm these aria2 options:

```text
enable-rpc=true
rpc-listen-port=6800
rpc-secret=<same value used by Tidefetch>
```

Keep `rpc-listen-all=false` for a same-host daemon. For a remote daemon, verify
firewall routing from the Tidefetch host and use a private network.

## Download directory is not writable

The container runs as `1000:1000`:

```sh
docker exec tidefetch id
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
docker exec tidefetch sh -c 'touch /downloads/.write-test && rm /downloads/.write-test'
```

On SELinux systems, add `:Z` to Podman/Docker bind mounts. On NFS or CIFS,
inspect host mount UID/GID options.

## Web UI reconnects repeatedly

The HTTP server is alive but the broker cannot maintain its aria2 connection.

```sh
docker compose logs -f tidefetch
docker exec tidefetch ps
```

Check available disk space and whether aria2 exited. Restart once, then inspect
the first error rather than repeatedly cycling the container.

## Reverse proxy loads HTML but stays disconnected

- Preserve the original `Host` header.
- Enable WebSocket upgrades for `/api/ws`.
- Increase proxy read timeout to at least one hour.
- Serve Tidefetch at `/`, not a URL subpath.
- Do not cache `/api/*` responses.

See [Reverse proxy and TLS](reverse-proxy.md).

## BitTorrent is slow or unreachable

- Publish and forward 6881 over both TCP and UDP.
- Check that another service is not using the same host port.
- Verify the tracker/DHT status in task details.
- Confirm the torrent has active peers.
- Avoid exposing aria2 RPC while opening peer ports.

Port check:

```sh
docker port tidefetch
```

## Session does not survive a restart

Confirm `/config` is persistent and writable. The image sets `HOME=/config`,
and the aria2 session is saved periodically under that volume.

```sh
docker inspect tidefetch --format '{{json .Mounts}}'
docker exec tidefetch find /config -maxdepth 4 -type f -ls
```

Allow at least 20 seconds after a queue change for the periodic aria2 session
save, or stop the container cleanly.

## Safe reset

Back up `/config`, stop Tidefetch, and move only the config file aside. Do not
delete downloads or the aria2 session while diagnosing a UI preference issue.

```sh
docker compose stop tidefetch
sudo cp -a /srv/tidefetch/config /srv/tidefetch/config.backup
sudo find /srv/tidefetch/config -name config.json -exec mv {} {}.old \;
docker compose start tidefetch
```

A fresh config creates a new RPC secret and default settings. Existing history
and session files remain available if their data directory is preserved.

## Reporting an issue

Include:

- `tidefetch version` and `aria2c --version`
- operating system or container image tag
- deployment type and reverse proxy
- relevant logs with passwords, RPC secrets, private URLs, and tracker tokens
  removed
- exact steps and whether the TUI and web UI behave differently

Open an issue at <https://github.com/Thre4dripper/tidefetch/issues>.