# Data & persistence

Everything Tidefetch keeps is plain files. This page documents exactly what is
stored, where, and what you must put on a volume.

The short version for self-hosting: **mount two volumes — `/config` and
`/downloads` — and you have persisted everything.**

## What is stored

| Data | File | Volume | Rebuildable? |
| --- | --- | --- | --- |
| Settings, RPC secret, web password hash, theme | `config.json` | `/config` | No — back this up |
| Download history + metadata | `history.json` | `/config` | No |
| aria2 queue (unfinished/paused downloads) | `session.aria2` | `/config` | No — losing it loses the queue |
| DHT routing tables (BitTorrent) | `dht.dat`, `dht6.dat` | `/config` | Yes, rebuilt automatically |
| Completed files | your files | `/downloads` | No |
| Resume state for in-progress downloads | `<file>.aria2` | `/downloads` | No — losing it restarts the file |
| Saved torrent metadata from magnets | `<infohash>.torrent` | `/downloads` | Yes |
| Web UI login sessions | in memory only | — | Restart requires re-login |
| Live speed-history charts | in memory only | — | Rebuilds as downloads run |

## Container layout

The image sets `HOME=/config`, so every state file resolves beneath it:

```text
/config/
├── .config/tidefetch/
│   └── config.json                 # settings, RPC secret, bcrypt password hash, theme
├── .local/share/tidefetch/
│   ├── history.json                # completed + failed download history
│   └── session.aria2               # aria2 queue, restored on restart
└── .cache/aria2/
    ├── dht.dat                     # IPv4 DHT routing table (BitTorrent only)
    └── dht6.dat                    # IPv6 DHT routing table

/downloads/
├── ubuntu-24.04.iso                # completed file
├── debian-13.iso                   # in-progress file
├── debian-13.iso.aria2             # its resume/control file
└── <infohash>.torrent              # metadata saved from magnet links
```

> [!IMPORTANT]
> `config.json` contains the aria2 RPC secret and the bcrypt hash of your web
> password. Treat the `/config` volume as sensitive: restrict permissions and
> encrypt backups.

### Minimum volumes

```yaml
volumes:
  - ./config:/config
  - /srv/downloads:/downloads
```

Named volumes work equally well:

```yaml
volumes:
  - tidefetch-config:/config
  - tidefetch-downloads:/downloads
```

Nothing else needs to persist. Mounting only `/downloads` will lose your
settings, history and queue on every restart.

## Native layout

Outside a container, Tidefetch follows platform conventions.

| Platform | Config | Data |
| --- | --- | --- |
| Linux | `${XDG_CONFIG_HOME:-~/.config}/tidefetch/` | `~/.local/share/tidefetch/` |
| macOS | `~/Library/Application Support/tidefetch/` | `~/.local/share/tidefetch/` |
| Windows | `%AppData%\tidefetch\` | `%UserProfile%\.local\share\tidefetch\` |

Confirm the resolved paths on any machine with:

```sh
tidefetch doctor
```

## File formats

### config.json

Plain JSON, mode `0600`, written on every settings change.

```json
{
  "rpc_url": "ws://127.0.0.1:6800/jsonrpc",
  "secret": "…",
  "download_dir": "/downloads",
  "theme": "surge",
  "web_password_hash": "$2a$10$…"
}
```

### history.json

Every completed or failed download, capped by `history_limit` (default 2000,
oldest entries dropped first). Each entry records name, URI, size, destination,
status, category and timestamps. It is Tidefetch's own record — independent of
aria2, which forgets downloads once they leave its result list.

### session.aria2

Written by aria2, not Tidefetch. It holds unfinished and paused downloads with
their URIs and options, and is saved every 20 seconds plus on clean shutdown.
On startup it is fed back to aria2 so the queue survives restarts.

> [!NOTE]
> Because it saves every 20 seconds, an unclean kill can lose up to 20 seconds
> of queue changes. Stop the container with `docker compose stop` (SIGTERM)
> rather than `docker kill` so aria2 flushes first.

### `.aria2` control files

aria2 writes `<filename>.aria2` next to each in-progress download to track
which pieces are complete. Delete one and that download restarts from zero.
They are removed automatically on completion.

This is why partial downloads must live on the same volume you keep — a
separate `/incomplete` mount is fine, but it has to persist too.

## Backups

`/config` is small (kilobytes) and the only irreplaceable part. Stop the
service first so aria2 flushes its session:

```sh
docker compose stop tidefetch
tar -C /srv/tidefetch -czf tidefetch-config-$(date +%F).tar.gz config
docker compose start tidefetch
```

For a named volume:

```sh
docker run --rm \
  -v tidefetch-config:/source:ro \
  -v "$PWD:/backup" \
  alpine tar -C /source -czf /backup/tidefetch-config.tar.gz .
```

Restore into a stopped service, then fix ownership:

```sh
docker compose down
tar -C /srv/tidefetch/config -xzf tidefetch-config-2026-07-26.tar.gz --strip-components=1
chown -R 1000:1000 /srv/tidefetch/config
docker compose up -d
```

Back up `/downloads` according to what the files are worth — most are
re-downloadable, so it is usually excluded from off-site backups.

## Permissions

The container runs as UID/GID `1000`. Both volumes must be writable by it:

```sh
chown -R 1000:1000 /srv/tidefetch/config /srv/downloads
chmod 700 /srv/tidefetch/config
```

On SELinux hosts add `:Z` to bind mounts. On NFS, confirm root squashing still
lets UID 1000 create, rename and delete — aria2 renames files on completion.

## Migrating to another host

1. Stop Tidefetch on the old host.
2. Copy the whole `/config` volume, and `/downloads` if you want the files.
3. Start on the new host with the same volume paths.

Everything — settings, password, history and the in-flight queue — comes back.
If you copy `/config` without `/downloads`, unfinished downloads restart from
the beginning because their `.aria2` control files are gone.

## Using an external aria2

When Tidefetch attaches to an aria2 you run yourself (`-no-spawn`), that daemon
owns `session.aria2` and the control files at whatever paths you configured.
Tidefetch still keeps `config.json` and `history.json` in its own directory, so
`/config` remains worth persisting.
