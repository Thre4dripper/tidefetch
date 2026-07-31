# Configuration

Tidefetch creates a config on first run, starts or attaches to aria2, and
persists history plus the aria2 session independently of the UI process.

## Commands

```text
tidefetch [flags] [URL ...]   open the TUI and optionally queue URLs
tidefetch serve [flags]       run the web interface
tidefetch doctor              inspect dependencies and writable paths
tidefetch version             print the build version
```

## TUI flags

| Flag | Meaning |
| --- | --- |
| `-url ws://host:6800/jsonrpc` | Override the aria2 WebSocket RPC endpoint |
| `-secret VALUE` | Override the aria2 RPC secret |
| `-dir PATH` | Override the default download directory |
| `-no-spawn` | Require an existing daemon instead of starting aria2c |
| `-version` | Print the version and exit |

Arguments left after flags are queued as downloads when the TUI starts.

## Web server flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-host` | Config value, initially `127.0.0.1` | HTTP listen address |
| `-port` | Config value, initially `8210` | HTTP listen port |
| `-password` | None | Set or replace the bcrypt-backed web password |
| `-no-auth` | `false` | Disable Tidefetch auth explicitly |
| `-url` | Config value | Override the aria2 RPC endpoint |
| `-secret` | Config value | Override the aria2 RPC secret |
| `-dir` | Config value | Override the download directory |
| `-no-spawn` | `false` | Require an existing aria2 daemon |

Examples:

```sh
# Local browser only; no password required
tidefetch serve

# LAN service; authentication is mandatory
tidefetch serve -host 0.0.0.0 -password 'a-long-unique-password'

# Behind a TLS proxy that provides its own authentication
tidefetch serve -host 127.0.0.1 -no-auth

# Attach to a remote aria2 daemon
tidefetch serve -host 0.0.0.0 -password 'web-password' \
  -url ws://10.0.0.20:6800/jsonrpc -secret 'aria2-rpc-secret' -no-spawn
```

## Environment variables

| Variable | Purpose |
| --- | --- |
| `TIDEFETCH_PASSWORD` | Web password; used when `-password` is absent |
| `TIDEFETCH_PASSWORD_FILE` | Read the web password from a mounted file |
| `HOME` | Controls config discovery in containers |
| `XDG_CONFIG_HOME` | Overrides the config root on Unix systems |

`TIDEFETCH_PASSWORD` takes precedence over `TIDEFETCH_PASSWORD_FILE`. Do not
set both. The file value has surrounding whitespace removed.

## Files and directories

The config file is created with mode `0600`.

| Platform | Config file |
| --- | --- |
| Linux | `${XDG_CONFIG_HOME:-~/.config}/tidefetch/config.json` |
| macOS | `~/Library/Application Support/tidefetch/config.json` |
| Windows | `%AppData%\tidefetch\config.json` |
| Container | Below `/config` because the image sets `HOME=/config` |

Runtime data currently lives at `~/.local/share/tidefetch` on every native
platform. It contains:

- `history.json`: categorized completed/failed task history
- `session.aria2`: aria2 session persistence

The all-in-one image stores both config and runtime data beneath the mounted
`/config` volume. Back up the whole volume instead of selecting subpaths.

For a full inventory of every file, the container directory tree and backup
procedures, see [Data & persistence](data-and-persistence.md).

## Config schema

Example `config.json`:

```json
{
  "rpc_url": "ws://127.0.0.1:6800/jsonrpc",
  "secret": "replace-with-a-random-rpc-secret",
  "auto_spawn": true,
  "download_dir": "/srv/downloads",
  "poll_ms": 700,
  "history_limit": 2000,
  "sidebar": true,
  "compact_rows": false,
  "confirm_remove": true,
  "extra_spawn_args": ["--bt-max-peers=80"],
  "default_split": "16",
  "default_max_conn": "16",
  "web_host": "127.0.0.1",
  "web_port": 8210
}
```

Do not copy another installation's `secret` or `web_password_hash`. Let
Tidefetch generate the RPC secret and set the web password through the CLI or
Security settings. Password hashes are written automatically.

## Local daemon lifecycle

With `auto_spawn: true`, Tidefetch:

1. Tries the configured RPC endpoint.
2. Finds `aria2c` in `PATH` when the endpoint is unavailable.
3. Starts aria2 with RPC bound to loopback and a secret.
4. Enables continuation, session persistence, BitTorrent metadata, and
   periodic session saves.
5. Connects the TUI or web broker to the new daemon.

The daemon can outlive the TUI. Pressing normal quit leaves downloads running;
the explicit shutdown action stops the daemon.

Use `-no-spawn` or set `auto_spawn` to `false` when another service manager
owns aria2.

## Existing aria2 daemon

Start aria2 with RPC and a secret. Keep the RPC port private whenever possible.

```sh
aria2c \
  --enable-rpc=true \
  --rpc-listen-all=false \
  --rpc-listen-port=6800 \
  --rpc-secret='replace-this-rpc-secret' \
  --continue=true \
  --save-session="$HOME/.local/share/aria2/session.txt" \
  --input-file="$HOME/.local/share/aria2/session.txt"
```

Configure Tidefetch:

```sh
tidefetch -url ws://127.0.0.1:6800/jsonrpc \
  -secret 'replace-this-rpc-secret' -no-spawn
```

For a remote daemon, firewall port 6800 to the Tidefetch host and use a private
LAN, WireGuard, or Tailscale network. aria2 RPC is not a substitute for TLS.

## Authentication model

- Loopback HTTP binds may run without a web password.
- Non-loopback binds require `-password`, an existing password hash, or an
  explicit `-no-auth`.
- Passwords are stored as bcrypt hashes.
- Browser sessions use HttpOnly SameSite cookies.
- State-changing cross-origin requests are rejected.
- The aria2 RPC secret remains server-side.

Use `-no-auth` only when the listener is loopback-only or a trusted reverse
proxy enforces authentication. See [Reverse proxies](reverse-proxy.md).

## Container permissions

The image runs as UID/GID `1000:1000`. Bind-mounted config and download
directories must be writable by that identity:

```sh
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

Do not run the image privileged. Tidefetch needs ordinary file access and the
declared network ports only.