# Web UI

`tidefetch serve` hosts a browser dashboard for the same queue the terminal UI manages. It is built for headless machines: a NAS, a VPS, a Raspberry Pi, or a desktop you want to check from your phone.

## Start it

```sh
tidefetch serve                                    # http://127.0.0.1:8210, this machine only
tidefetch serve -host 0.0.0.0 -password 'secret'   # reachable on the network
```

The server owns one connection to aria2, polls it, and pushes changes to every open browser over a WebSocket, so ten open tabs cost no more than one. In the [container image](deployment/docker.md) it starts automatically with aria2 bundled.

All flags are listed under [Configuration](configuration.md#web-server-flags).

## Authentication

| Listen address | Password set | Result |
| --- | --- | --- |
| Loopback (`127.0.0.1`, `localhost`) | No | Open, no sign-in |
| Any | Yes | Sign-in required |
| Non-loopback | No | Refuses to start. Pass `-password`, set `TIDEFETCH_PASSWORD`, or use `-no-auth` behind your own authenticating proxy |

Passwords are stored as bcrypt hashes. Sessions are HttpOnly, SameSite cookies that last 30 days and are held in memory, so a restart signs everyone out. Eight failed sign-ins block an IP for five minutes.

Set or change the password with `tidefetch serve -password`, the `TIDEFETCH_PASSWORD` or `TIDEFETCH_PASSWORD_FILE` environment variables, the first-run tour, or Settings → Security.

## Screens

**Sidebar.** Filters for Downloads, Active, Queued and Finished with live counts, History, throughput sparklines, storage usage for the download volume, the aria2 version and connection state, and buttons for Settings and sign-out.

**Top bar.** A search box (`/` focuses it), resume all, pause all, clear finished, and Add download.

**Metrics.** Current download and upload speed, bytes moved this session, and a throughput chart.

**Download list.** Virtualised, so thousands of entries scroll smoothly. Each row shows a file-type icon, name, progress, a lifetime speed sparkline, speed and ETA, and a status pill. Hovering reveals pause, resume, retry and remove. Click a row to open its detail drawer.

**Detail drawer.**

| Tab | Contents |
| --- | --- |
| Overview | Progress ring, speed history, status, speed, ETA, connections, seeders, piece map, location, source, info hash, mirrors and error; queue reordering for waiting downloads; remove together with files |
| Files | Per-file progress; tick which torrent files to download, then Apply |
| Peers | Connected BitTorrent peers with speeds and completion (torrents only) |
| Options | The common per-download options as fields, plus every raw option |

**Add download.** Three sources: URLs or magnet links (one per line), a `.torrent` file, or a Metalink file. Check link inspects an HTTP URL before downloading. Browse opens a server-side folder picker that can create folders. Advanced options cover renaming, split and connection counts, speed limits, file allocation, checksum, referer, user agent, proxy, seed ratio, custom headers, resume and add-paused.

**History.** Search, filter by category, download an entry again, delete entries or clear everything.

**Settings.** Transfer, Behaviour, BitTorrent and Network edit aria2's global options. Advanced lists every raw option with a filter. Interface holds browser-only preferences. Security sets or changes the password and shows the version and download directory.

## Browser preferences

These live in the browser, not on the server, under Settings → Interface:

- **Desktop notifications** when a download completes or fails, even in a background tab. The browser asks for permission once.
- **Compact download list** for more rows on screen.
- **Ask before removing** an in-progress download.

## First run

The first visit from a browser shows a short tour. When no password is set, its last step offers to create one. Skipping it leaves the UI open, which is fine on loopback and unsafe anywhere else.

## Reaching it safely

For a public hostname, bind to loopback and put a TLS reverse proxy in front. For private access, a VPN such as Tailscale or WireGuard is simpler. Never forward port 8210 to the internet without TLS and a password, and never expose aria2's RPC port 6800. See [Reverse proxy and TLS](reverse-proxy.md) and [Homelab operations](homelab.md).

## Automation

Everything the dashboard does goes through the [HTTP API](api.md): sign in for a bearer token, then add downloads from a shell alias, pause everything on a schedule, or feed a dashboard widget.
