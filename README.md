# aria2tui

A fast, polished terminal UI for [aria2](https://aria2.github.io) — the
download utility that does HTTP(S), FTP, SFTP, BitTorrent and Metalink.
Surge-style dark interface, IDM-style workflow: queue control, categories,
history, directory browser, live speed charts.

```
 ⬡ aria2tui   ▼ 12.4 MB/s  ▲ 1.2 MB/s  ▁▂▃▅▇█▆▅▃▂  active 3 · queued 2 · stopped 8      ● aria2 1.37.0
  1 All   2 Active   3 Queued   4 Finished  │  ＋ Add   ⟲ History   ⚙ Settings
┃ ⇣ ubuntu-24.04.2-desktop-arm64.iso                                              ⧉ ACTIVE
  ██████████████████████▌───────────────────────  46.8%  2.5 GB / 5.3 GB  12.4 MB/s  eta 3m52s  seeds 74
  ⏸ linux-6.9.tar.xz                                                                PAUSED
  ████████▏──────────────────────────────────────  17.2%  24 MB / 142 MB  paused
```

## Features

- **Live dashboard** — global ↓/↑ speeds, sparkline history, per-download
  progress bars (⅛-cell precision), ETA, connections, seeders
- **btop-style side panel** — per-task braille speed graph for the selected
  download, global traffic graphs, disk usage of the target volume, session
  stats; toggle with `t`
- **Full mouse support** — click tabs, rows (twice for details), form fields,
  dialog buttons, the bottom button bar; scroll with the wheel
- **File browser** — browse the download folder in the TUI: open files,
  reveal in Finder/Explorer, delete, free-space readout (`f`)
- **Link inspector** — IDM-style pre-download check (`ctrl+k` in the add
  form): does the server support resume? real size, server filename, type
- **Full aria2 option set when adding** — advanced panel with resume/continue,
  file allocation (none/prealloc/trunc/falloc), per-task speed limit, retries,
  checksum verification, custom headers, referer, user-agent, proxy, seed ratio
- **Selection-based settings** — common options pick from sensible presets
  with `←/→`, free-text where it matters (proxy, user-agent, dir)
- **Surge-style chunk map** — live piece grid for the selected download
- **Every download type aria2 supports** — HTTP(S), FTP, SFTP, magnet links,
  `.torrent` files, `.metalink`/`.meta4` files, multi-mirror downloads
- **Full queue control** — pause/resume (single or all), reorder queue,
  remove, remove + delete files from disk, retry failed downloads
- **Directory lookup** — built-in filesystem browser to pick the save
  directory (or create one) and to pick torrent/metalink files, with free
  disk space shown as you browse
- **Download history** — persisted across runs, IDM-style automatic
  categories (Video, Audio, Documents, Archives, Programs, Images, …),
  search, filter by category, one-key re-download
- **Details panel** — live speed graph, files (with per-file selection for
  torrents), peers, servers, piece map, per-download speed limit
- **Live settings** — edit daemon options over RPC (concurrency, split,
  speed limits, seed ratio, proxy, …), saved into the aria2 session
- **Speed limiting** — global and per-download, adjustable with one key
- **Daemon management** — attaches to a running aria2c or spawns one with
  session persistence; downloads keep running after you quit the UI
- **Startup queueing** — `aria2tui <URL>...` adds downloads immediately

## Install

Requires [aria2](https://aria2.github.io) (`brew install aria2` /
`apt install aria2`) and Go 1.22+.

```sh
go install github.com/turbostart/aria2c-tui@latest   # installs `aria2c-tui`
# or from a checkout:
make install
```

## Usage

```sh
aria2tui                                  # open the UI
aria2tui https://example.com/file.iso     # open and start downloading
aria2tui "magnet:?xt=urn:btih:..."        # magnets work too
aria2tui -url ws://nas:6800/jsonrpc -secret s3cret   # remote daemon
aria2tui -no-spawn                        # never start a local daemon
```

On first run a config file is created at `~/.config/aria2tui/config.json`
with a random RPC secret. History and the aria2 session live in
`~/.local/share/aria2tui/`.

## Keys

| Key | Action |
| --- | --- |
| `j/k` `↑/↓` | move · `g/G` top/bottom · `ctrl+d/u` half page |
| `1–4` | filter: All / Active / Queued / Finished |
| `space` / `p` | pause · resume selected |
| `P` / `R` | pause all · resume all |
| `enter` / `i` | details (info · files · peers · servers) |
| `a` | add downloads (URLs, magnets, torrent files) |
| `x` | remove (with confirmation) |
| `D` | remove **and delete files from disk** |
| `r` | retry a failed download |
| `J/K` | move in queue |
| `S` | cycle sort: default · name · size · speed · progress |
| `/` | search; `esc` clears |
| `o` / `y` | open folder · copy URL/magnet |
| `t` | toggle the side panel |
| `f` | file browser (open · reveal · delete) |
| `[` `]` / `{` `}` | global ↓ / ↑ speed limit −/+ |
| `+` `-` | per-download ↓ limit (in details) |
| `c` | clear finished results |
| `w` | save session |
| `h` / `s` / `?` | history · settings · help |
| `q` | quit — daemon keeps downloading |
| `Q` | quit **and** shut the daemon down |

Add form: `tab` next field · `ctrl+o` browse directory · `ctrl+t` pick a
torrent/metalink file · `ctrl+s` start. One URL per line; multiple mirrors
for the same file go on one line separated by spaces.

Mouse: click tabs and rows (click a selected row again for details), click
the buttons in the bottom bar, scroll lists with the wheel.

## Standalone RPC client

The aria2 JSON-RPC client is an independent package with no TUI
dependencies — use it in your own tools:

```go
import "github.com/turbostart/aria2c-tui/pkg/aria2"

client, _ := aria2.Dial(ctx, "ws://127.0.0.1:6800/jsonrpc", "secret")
gid, _ := client.AddURI(ctx, []string{"https://example.org/file.iso"},
    aria2.Options{aria2.OptDir: "/downloads", aria2.OptSplit: "16"})

for n := range client.Notifications() {
    fmt.Println(n.Method, n.GID) // aria2.onDownloadComplete abc123…
}
```

Covers the full method surface: add (URI/torrent/metalink), tell*, pause,
remove, queue position, per-download and global options, global stats,
`system.multicall`, session save/shutdown, and event notifications over
WebSocket.

## Development

```sh
make build      # build ./aria2tui
make test       # unit + integration tests (integration skips without aria2c)
make run        # build & run
```

## License

MIT
