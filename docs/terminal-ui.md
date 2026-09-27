# Terminal UI

`tidefetch` opens a full-screen terminal interface with keyboard and mouse control. This page walks through each screen and lists every shortcut. Press `?` inside the app for the same reference.

## Layout

The header shows global download and upload speed, a sparkline, queue counts and the aria2 connection state. Tabs below it switch between the Downloads screen and the Add, Files, History and Settings screens. The footer lists the shortcuts for the current screen, and every entry is clickable.

Mouse support: click tabs, rows, buttons and form fields; click a selected row again to open it; the wheel scrolls lists. Modals close when you click outside them.

Terminals narrower than 110 columns get a compact tab strip; the side panel needs at least 100 columns.

## Downloads

Every download is a card: name and status badge; a progress gauge with sizes, speed, ETA and a lifetime speed sparkline; and a context line with the destination, source host and connection count, or the error message for a failed download. Tabs `1` to `4` filter by All, Active, Queued and Finished.

The side panel (`t`) stacks the selected download with its speed history and progress, a piece map for torrents, global traffic graphs, disk usage for the destination volume, session details and the last few log lines.

### Navigation

| Key | Action |
| --- | --- |
| `j` `k`, `↑` `↓` | Move the cursor |
| `g` `G`, `Home` `End` | First or last row |
| `ctrl+d` `ctrl+u`, `PgDn` `PgUp` | Half a page |
| `1` `2` `3` `4` | All, Active, Queued, Finished |
| `/` | Search by name or URL; `enter` keeps the filter, `esc` clears it |
| `S` | Cycle sort: default, name, size, speed, progress |
| `t` | Toggle the side panel |

### Selected download

| Key | Action |
| --- | --- |
| `space`, `p` | Pause or resume |
| `enter`, `i`, `l` | Open details |
| `x`, `d`, `Delete`, `Backspace` | Remove from aria2; asks first unless disabled in Settings |
| `D` | Remove and delete the files from disk; always asks |
| `r` | Retry a failed download from its original URLs |
| `J` `K` | Move a queued download down or up |
| `o` | Open the destination folder in your file manager |
| `y` | Copy the URL, or the magnet link for a torrent |

### Everywhere on this screen

| Key | Action |
| --- | --- |
| `a` | Add downloads |
| `P` `R` | Pause all, resume all |
| `c` | Clear finished, failed and removed results; asks first |
| `w` | Save the aria2 session now |
| `[` `]` | Global download limit down or up by 256 KiB/s; `0` is unlimited |
| `{` `}` | Global upload limit down or up by 256 KiB/s |
| `f` `h` `s` | Files, History, Settings |
| `?` | Help |
| `q`, `ctrl+c` | Quit; the aria2 daemon keeps running |
| `Q` | Quit and shut aria2 down; the session is saved first |

## Add

Paste one download per line into the text area: `http(s)://`, `ftp://`, `sftp://` or `magnet:` links, or the path of a local `.torrent`, `.metalink` or `.meta4` file. Mirrors of the same file go on one line separated by spaces.

| Key | Action |
| --- | --- |
| `tab` `shift+tab` | Next or previous field; `↑` `↓` also work outside the text area |
| `←` `→` | Change a selector value |
| `space` | Toggle a checkbox |
| `ctrl+k` | Check the first link: name, size, type and resume support |
| `ctrl+o` | Browse for the destination folder |
| `ctrl+t` | Pick a `.torrent` or `.metalink` file |
| `ctrl+a` | Show or hide advanced options |
| `ctrl+s`, `enter` on a field | Start |
| `esc` | Cancel |

The basic fields are the destination, a filename override, the split count, connections per server and start-paused. Advanced options add resume, file allocation, per-download speed limits, retries, checksum verification, user agent, referer, up to five custom headers, a proxy and the BitTorrent seed ratio. The destination row shows free space on the chosen volume.

## Details

`enter` on a download opens five panels, switched with `tab`, `shift+tab` or `←` `→`.

| Panel | Shows |
| --- | --- |
| Info | Progress, speed history, size, speed, ETA, connections, directory, GID, torrent metadata, error and source URL |
| Files | Every file with size and progress; `space` toggles whether a torrent file is downloaded |
| Peers | Connected BitTorrent peers with speeds and flags |
| Servers | Connected HTTP and FTP servers with per-server speed |
| Options | Every aria2 option on this download; `enter` edits the selected one |

| Key | Action |
| --- | --- |
| `+` `-` | Per-download download limit up or down by 256 KiB/s |
| `p` | Pause or resume |
| `x`, `D` | Remove, or remove and delete files |
| `o` `y` | Open the folder, copy the URL |
| `esc`, `q` | Back |

## Files

A browser for the download directory and the rest of the filesystem.

| Key | Action |
| --- | --- |
| `enter`, `l` | Enter a folder, or open a file with its default application |
| `Backspace`, `h` | Parent folder |
| `o` | Reveal in the file manager |
| `x`, `Delete` | Delete the file or folder from disk; asks first |
| `n` | New folder |
| `d` `~` `/` | Jump to the download directory, home or root |
| `.` | Show hidden files |
| `r` | Refresh |
| `esc`, `q`, `f` | Back |

The folder picker used by the Add form takes the same keys; `space`, `s` or `ctrl+s` chooses the current folder.

## History

Finished and failed downloads are recorded with a category derived from the file extension: Video, Audio, Documents, Archives, Programs, Images, Torrents or Other.

| Key | Action |
| --- | --- |
| `enter`, `r` | Download again: opens the Add form with the URL and folder filled in |
| `/` | Search by name or URL |
| `c` | Cycle the category filter |
| `x`, `d` | Delete the entry |
| `C` | Clear the whole history; asks first |
| `o` `y` | Open the folder, copy the URL |
| `esc`, `q`, `h` | Back |

History is capped by the `history_limit` setting, 2000 entries by default.

## Settings

Seven categories, switched with `←` `→` (or `tab`, `h`, `l`). `enter` edits the selected option. For choice lists, `←` `→` cycle values, `enter` saves and `esc` reverts; the colour theme previews as you cycle.

| Category | Contents |
| --- | --- |
| Transfer | Global download and upload limits, concurrent downloads, split, connections per server, minimum split size |
| Behaviour | Resume, file allocation, retries, retry wait, auto-rename, overwrite, integrity checks |
| BitTorrent | Seed ratio and time, peer limits, peer speed target, listen and DHT ports |
| Network | Proxy, user agent, connect and I/O timeouts |
| Interface | Colour theme, default download directory, refresh interval, side panel, compact cards, remove confirmation, history limit |
| Advanced | Every live aria2 global option, filtered with `/` |
| Security | Web listen address and port, authentication state, web password |

Transfer, Behaviour, BitTorrent, Network and Advanced write to the running aria2 daemon. Interface and Security are stored in Tidefetch's config file; the web settings apply the next time `tidefetch serve` starts. `r` reloads the options from aria2.

## Themes

Thirteen palettes ship with the terminal UI. Pick one under Settings → Interface → Colour theme; it is saved as `theme` in the config file.

| Value | Theme |
| --- | --- |
| `surge` | Surge (default) |
| `tide` | Tide |
| `tokyonight` | Tokyo Night |
| `catppuccin` | Catppuccin Mocha |
| `gruvbox` | Gruvbox Dark |
| `nord` | Nord |
| `dracula` | Dracula |
| `rosepine` | Rosé Pine |
| `everforest` | Everforest |
| `kanagawa` | Kanagawa |
| `solarized` | Solarized Dark |
| `ayu` | Ayu Dark |
| `monokai` | Monokai Pro |

An unknown value falls back to `surge`. Themes use truecolor, which most modern terminals support. On Windows, use Windows Terminal rather than the legacy console.
