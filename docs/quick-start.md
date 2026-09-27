# Quick start

Five minutes from a fresh install to a running download, first in the terminal, then in a browser.

Before you start, [install Tidefetch](installation.md) and check that `tidefetch doctor` reports `aria2c ok`.

## 1. Open the terminal UI

```sh
tidefetch
```

On first run Tidefetch writes a config file with a random RPC secret, starts a local `aria2c` daemon bound to loopback, and opens the Downloads screen. The daemon keeps running when you quit, so downloads continue in the background.

You can also queue downloads straight from the command line:

```sh
tidefetch https://releases.ubuntu.com/24.04/ubuntu-24.04.2-desktop-amd64.iso
tidefetch "magnet:?xt=urn:btih:…"
```

## 2. Add a download

Press `a`. Paste one or more links, one per line: HTTP(S), FTP, SFTP or magnet links, or the path of a local `.torrent` or `.metalink` file. Mirrors of the same file go on one line separated by spaces.

| Key | Action |
| --- | --- |
| `ctrl+k` | Check the first link: file name, size, content type and whether the server supports resume |
| `ctrl+o` | Pick the destination folder |
| `ctrl+t` | Pick a `.torrent` or `.metalink` file |
| `ctrl+a` | Advanced aria2 options: split count, connection and speed limits, checksum, headers, proxy, seed ratio |
| `ctrl+s` | Start |
| `esc` | Cancel |

## 3. Watch and control it

Back on the Downloads screen, every card shows progress, speed, ETA and a lifetime speed sparkline. The side panel (`t`) adds a piece map, throughput graphs, disk usage and a log.

| Key | Action |
| --- | --- |
| `space` | Pause or resume |
| `enter` | Details: files, peers, servers and options |
| `x` | Remove (`D` also deletes the files) |
| `r` | Retry a failed download |
| `[` `]` | Lower or raise the global download limit |
| `?` | Every shortcut |

The full reference is in [Terminal UI](terminal-ui.md).

## 4. Try the web UI

```sh
tidefetch serve
```

Open <http://127.0.0.1:8210>. On a loopback address no password is required, and the first-run tour offers to set one. The web UI shows the same queue as the terminal UI, because both talk to the same aria2 daemon.

To reach it from other machines, bind to all interfaces with a password:

```sh
tidefetch serve -host 0.0.0.0 -password 'a-long-unique-password'
```

For a permanent service on a NAS or server, use the [container image](deployment/docker.md) or the [systemd unit](deployment/systemd.md) instead.

## 5. Where things are

| What | Where |
| --- | --- |
| Config, RPC secret, web password hash | `~/.config/tidefetch/config.json` (macOS: `~/Library/Application Support/tidefetch/`) |
| History and the aria2 session file | `~/.local/share/tidefetch/` |
| Downloads | `~/Downloads` by default; change it under Settings → Interface or with `-dir` |

[Data and persistence](data-and-persistence.md) lists every file.

## Next steps

- [Web UI](web-ui.md): screens, authentication and browser preferences.
- [Configuration](configuration.md): flags, environment variables and the config file.
- [HTTP API](api.md): drive Tidefetch from scripts and dashboards.
