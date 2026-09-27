# Configuration

Tidefetch keeps one JSON config file, creates it with sensible defaults on first run, and layers command-line flags and environment variables on top. This page is the reference for all three.

## Commands

| Command | Purpose |
| --- | --- |
| `tidefetch [flags] [URL ...]` | Open the terminal UI; any URLs are queued on start |
| `tidefetch serve [flags]` | Run the web UI server |
| `tidefetch doctor` | Check the config, the aria2 binary, RPC connectivity and writable paths |
| `tidefetch version` | Print the version |

## TUI flags

| Flag | Meaning |
| --- | --- |
| `-url ws://host:6800/jsonrpc` | aria2 WebSocket RPC endpoint; overrides `rpc_url` |
| `-secret VALUE` | aria2 RPC secret; overrides `secret` |
| `-dir PATH` | Default download directory; overrides `download_dir` |
| `-no-spawn` | Never start a local `aria2c`; fail if the endpoint is unreachable |
| `-version` | Print the version and exit |

## Web server flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-host` | `web_host` (`127.0.0.1`) | Listen address |
| `-port` | `web_port` (`8210`) | Listen port |
| `-password` | none | Set or replace the web password; stored as a bcrypt hash |
| `-no-auth` | off | Disable authentication explicitly |
| `-url`, `-secret`, `-dir`, `-no-spawn` | | Same as the TUI flags |

```sh
# This machine only; no password needed
tidefetch serve

# On the network; a password is mandatory
tidefetch serve -host 0.0.0.0 -password 'a-long-unique-password'

# Behind a proxy that authenticates users itself
tidefetch serve -host 127.0.0.1 -no-auth

# Against an aria2 you already run elsewhere
tidefetch serve -host 0.0.0.0 -password 'web-password' \
  -url ws://10.0.0.20:6800/jsonrpc -secret 'aria2-rpc-secret' -no-spawn
```

## Environment variables

| Variable | Purpose |
| --- | --- |
| `TIDEFETCH_PASSWORD` | Web password, used when `-password` is not given |
| `TIDEFETCH_PASSWORD_FILE` | Path to a file holding the web password, for Docker and Kubernetes secrets; surrounding whitespace is trimmed |
| `HOME` | Base of the config and data directories; the container image sets it to `/config` |
| `XDG_CONFIG_HOME` | Overrides the config root on Linux |

`TIDEFETCH_PASSWORD` wins when both password variables are set. Both apply to `tidefetch serve` only.

## Files

| Platform | Config file | Data directory |
| --- | --- | --- |
| Linux | `${XDG_CONFIG_HOME:-~/.config}/tidefetch/config.json` | `~/.local/share/tidefetch/` |
| macOS | `~/Library/Application Support/tidefetch/config.json` | `~/.local/share/tidefetch/` |
| Windows | `%AppData%\tidefetch\config.json` | `%UserProfile%\.local\share\tidefetch\` |
| Container | `/config/.config/tidefetch/config.json` | `/config/.local/share/tidefetch/` |

The data directory holds `history.json` and `session.aria2`, the file aria2 restores the queue from. The config file is written with mode `0600`. `tidefetch doctor` prints the resolved paths, and [Data and persistence](data-and-persistence.md) explains what to back up.

Installations from the project's previous name, aria2tui, are migrated automatically on first run.

## Config file

Every key with its default. Keys you leave out keep their default.

| Key | Default | Meaning |
| --- | --- | --- |
| `rpc_url` | `ws://127.0.0.1:6800/jsonrpc` | aria2 RPC endpoint |
| `secret` | random | aria2 `--rpc-secret`, generated on first run |
| `aria2c_path` | unset | Path to `aria2c` when it is not on `PATH` |
| `auto_spawn` | `true` | Start a local aria2 when the endpoint does not answer |
| `download_dir` | `~/Downloads` | Default destination |
| `poll_ms` | `700` | UI refresh interval in milliseconds; the Settings screen accepts 300 or more |
| `history_limit` | `2000` | Entries kept in `history.json` |
| `theme` | `surge` | Terminal colour theme; see [Themes](terminal-ui.md#themes) |
| `sidebar` | `true` | Show the terminal side panel on start |
| `compact_rows` | `false` | Two-line download cards in the terminal |
| `confirm_remove` | `true` | Ask before removing a download |
| `extra_spawn_args` | `[]` | Extra `aria2c` arguments when Tidefetch starts the daemon |
| `default_split` | `"16"` | Prefilled split count in the Add form |
| `default_max_conn` | `"16"` | Prefilled connections per server in the Add form |
| `web_host` | `127.0.0.1` | `tidefetch serve` listen address |
| `web_port` | `8210` | `tidefetch serve` listen port |
| `web_password_hash` | unset | bcrypt hash of the web password; managed by Tidefetch, never edit it by hand |

```json
{
  "rpc_url": "ws://127.0.0.1:6800/jsonrpc",
  "secret": "generated-on-first-run",
  "auto_spawn": true,
  "download_dir": "/srv/downloads",
  "poll_ms": 700,
  "history_limit": 2000,
  "theme": "surge",
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

Do not copy `secret` or `web_password_hash` between machines. Let Tidefetch generate the secret, and set the password through `-password`, the environment or the Security settings.

## The local aria2 daemon

With `auto_spawn` on, Tidefetch first tries `rpc_url` and attaches if an aria2 answers with the configured secret. Otherwise it finds `aria2c` (from `aria2c_path` or `PATH`) and starts it with these options:

| Option | Value |
| --- | --- |
| `--enable-rpc`, `--rpc-listen-all=false` | RPC on loopback only |
| `--rpc-listen-port` | The port from `rpc_url`, or a free port if that one is busy |
| `--rpc-secret` | `secret` |
| `--dir` | `download_dir` |
| `--continue=true` | Resume partial files |
| `--input-file`, `--save-session` | `session.aria2` in the data directory, saved every 20 seconds |
| `--auto-save-interval=20` | Control files flushed every 20 seconds |
| `--max-concurrent-downloads=5` | Starting concurrency; change it under Settings → Transfer |
| `--file-allocation=none` | No preallocation |
| `--bt-save-metadata=true`, `--follow-torrent=true` | Magnet metadata is saved and followed |
| `--quiet=true`, `--summary-interval=0` | No console output |

`extra_spawn_args` are appended last, so they override anything above.

The daemon outlives the UI. `q` in the terminal leaves it running with your downloads; `Q` saves the session and shuts it down. `tidefetch serve` reconnects automatically if the daemon goes away and respawns it when allowed.

Set `auto_spawn` to `false`, or pass `-no-spawn`, when something else manages aria2.

## Using an existing aria2

Start aria2 with RPC and a secret, then point Tidefetch at it:

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

```sh
tidefetch -url ws://127.0.0.1:6800/jsonrpc -secret 'replace-this-rpc-secret' -no-spawn
```

For a daemon on another host, reach it over a private network (LAN, WireGuard, Tailscale) and firewall port 6800 to the Tidefetch host. aria2 RPC has no TLS of its own.

## Authentication model

- Loopback binds may run without a password.
- Non-loopback binds need a stored password hash, `-password`, `TIDEFETCH_PASSWORD`, `TIDEFETCH_PASSWORD_FILE` or an explicit `-no-auth`.
- Passwords are bcrypt hashes. Sessions are HttpOnly, SameSite cookies held in memory for 30 days.
- Eight failed sign-ins block the source IP for five minutes.
- State-changing cross-origin requests are rejected, and every response carries a strict Content Security Policy.
- The aria2 RPC secret stays on the server.

Use `-no-auth` only on loopback or behind a proxy that authenticates users. See [Reverse proxy and TLS](reverse-proxy.md).

## Container permissions

The image runs as UID/GID `1000:1000`. Bind-mounted config and download directories must be writable by that identity:

```sh
sudo chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
sudo chmod 700 /srv/tidefetch/config
```

Do not run the image privileged; it needs ordinary file access and its declared ports only.
