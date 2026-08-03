# HTTP API

`tidefetch serve` exposes the same REST + WebSocket API its web UI runs on.
It is the supported way to drive Tidefetch from dashboards, scripts and
custom frontends: one origin, password auth, live push updates — and the
aria2 RPC secret never leaves the server.

Base URL: `http://<host>:8210/api`. Every request and response is JSON.

> [!NOTE]
> The API is stable in spirit but still pre-1.0: additive changes land
> without notice, breaking changes only with a minor version bump and a
> changelog entry.

## Choosing a surface

| You are building | Use |
| --- | --- |
| A dashboard widget, script or custom UI | **This API** |
| A tool that already speaks aria2 JSON-RPC | aria2's own RPC, see below |

Tidefetch adds what raw aria2 does not have: bcrypt password auth with rate
limiting, WebSocket deltas instead of polling, persistent history, link
probing, and file-system browsing. Going straight to
[aria2's JSON-RPC](https://aria2.github.io/manual/en/html/aria2c.html#rpc-interface)
makes sense only when an existing integration already speaks it — in that
case point it at the daemon (default `ws://127.0.0.1:6800/jsonrpc`) with the
RPC secret from `config.json`. Do not expose that port beyond localhost;
the secret authorizes full control, including arbitrary file writes.

## Authentication

Three modes, decided at startup:

| Bind | Password set | Behaviour |
| --- | --- | --- |
| Loopback | no | Auth disabled — requests need no credentials |
| Any | yes | Login required |
| Non-loopback | no | Server refuses to start (use `-password` or `-no-auth`) |

Log in once, then use the returned token:

```sh
TOKEN=$(curl -s http://nas:8210/api/login \
  -d '{"password":"your-password"}' | jq -r .token)

curl -s http://nas:8210/api/state -H "Authorization: Bearer $TOKEN"
```

`POST /api/login` → `200 {"ok":true,"authRequired":true,"token":"…"}`.
The same value is also set as an HttpOnly `tf_session` cookie for browsers;
scripts should prefer the `Authorization: Bearer` header. Sessions live for
30 days and are held in memory — a server restart signs everyone out.
`POST /api/logout` revokes the session.

Failed logins are rate limited per IP: 8 wrong attempts block that IP for
5 minutes (`429`).

Cross-origin, state-changing requests are rejected (`403`): browsers send an
`Origin` header and Tidefetch requires it to match the host. Server-side
callers without an `Origin` header are unaffected.

## Errors

Non-2xx responses carry `{"error":"human-readable message"}`.

| Status | Meaning |
| --- | --- |
| `400` | Malformed request or unknown action |
| `401` | Missing/expired session, or wrong password |
| `403` | Cross-origin request rejected |
| `404` | Unknown GID |
| `429` | Login rate limit |
| `502` | aria2 rejected the operation or is unreachable |

## Real-time updates

`GET /api/ws` upgrades to a WebSocket. The server pushes; the client never
needs to send anything.

```js
const ws = new WebSocket("ws://nas:8210/api/ws");
ws.onmessage = (e) => {
  const msg = JSON.parse(e.data);
  // msg.type: "snapshot" | "delta" | "conn"
};
```

| `type` | Fields | When |
| --- | --- | --- |
| `snapshot` | `tasks`, `stat` | On connect, and after bulk changes |
| `delta` | `updated` (changed tasks), `removed` (gone GIDs), `stat` | Every poll tick with changes |
| `conn` | `connected` (bool) | aria2 link lost or regained |

Browsers authenticate the upgrade with the session cookie automatically; the
`Authorization` header works for non-browser clients. If WebSockets are
inconvenient, poll `GET /api/state` instead.

## The Task object

Every task in `state`, `snapshot`, `delta` and task detail responses:

```json
{
  "gid": "c37c75438f281a1a",
  "name": "ubuntu-24.04.2-desktop-amd64.iso",
  "status": "active",
  "total": 6343219200, "done": 2617245696, "uploaded": 0,
  "downSpeed": 861184, "upSpeed": 0,
  "conns": 1, "seeders": 0,
  "seeding": false, "torrent": false,
  "dir": "/downloads",
  "uri": "https://releases.ubuntu.com/…",
  "numFiles": 1,
  "progress": 0.4126,
  "speeds": [812032, 845311, …]
}
```

`status` is one of `active · waiting · paused · complete · error · removed`.
`speeds` is a downsampled lifetime speed series (≤48 points) for sparklines.
Error tasks add `errorCode` and `errorMsg`.

The `stat` object carries the global picture: `downSpeed`, `upSpeed`,
`numActive`, `numWaiting`, `numStopped`, `sessionDown`, `sessionUp`,
`diskFree`, `diskTotal`.

## Endpoints

### State

```
GET /api/state
```

Full snapshot: `version`, `aria2` (engine version), `connected`,
`downloadDir`, `authEnabled`, `tasks`, `stat`. This is the one endpoint a
dashboard widget needs.

### Add downloads

```
POST /api/add
```

```json
{ "kind": "uri", "uris": ["https://example.com/file.iso", "magnet:?xt=…"],
  "options": { "dir": "/downloads/isos", "max-download-limit": "2M" } }
```

`kind` is `uri` (default), `torrent` or `metalink`; the latter two send the
file as base64 in `payload` instead of `uris`. Each URI becomes its own
task. `options` accepts any
[aria2 input option](https://aria2.github.io/manual/en/html/aria2c.html#input-file);
`dir` defaults to the configured download directory. Returns
`{"gids":["…"]}`.

### Inspect a task

```
GET /api/tasks/{gid}
```

Returns `task`, per-file rows (`files`), live `peers` (torrents), `servers`
(HTTP), a ≤240-bucket `pieces` completion map, the full `speedHistory`, and
`bt` metadata.

### Act on a task

```
POST /api/tasks/{gid}/action
```

```json
{ "action": "pause" }
```

| `action` | Effect |
| --- | --- |
| `pause` / `resume` | Pause or unpause |
| `remove` | Stop and remove; `"deleteFiles": true` also deletes data |
| `removeResult` | Drop a finished/failed entry from the list |
| `retry` | Re-queue a failed download from its original URI |

Bulk variant — `POST /api/tasks/actions` with `{"action":"pauseAll"}`,
`resumeAll` or `purge` (clear all finished results).

### Tune a task

```
POST /api/tasks/{gid}/files      {"indices":[1,3,4]}     select torrent files
GET  /api/tasks/{gid}/options                            current aria2 options
PUT  /api/tasks/{gid}/options    {"max-download-limit":"1M"}
POST /api/tasks/{gid}/position   {"pos":0,"how":"POS_SET"}  queue order
```

### Global options

```
GET /api/options
PUT /api/options   {"max-overall-download-limit":"5M"}
```

Reads and writes aria2's global option set — the same 100+ keys the
Settings screens edit.

### History

```
GET    /api/history?q=ubuntu&category=Software
DELETE /api/history/{gid}
DELETE /api/history
```

Returns `{"entries":[…],"categories":[…]}`. Entries persist across
restarts and record name, URL, size, category and completion time.

### Filesystem helpers

```
GET  /api/browse?path=/srv          list directories (server-side)
POST /api/browse/mkdir              {"path":"/srv","name":"isos"}
GET  /api/probe?url=https://…       pre-download link inspection
```

`browse` walks up to the nearest existing directory instead of failing and
reports free space. `probe` answers with filename, size, content type and
whether the server supports resume:

```json
{ "filename": "file.iso", "size": 6343219200,
  "contentType": "application/x-iso9660-image",
  "resumable": true, "via": "accept-ranges", "finalUrl": "…" }
```

### Account

```
POST /api/login      {"password":"…"}
POST /api/logout
POST /api/password   {"current":"…","new":"…"}
```

Setting a first password via `/api/password` turns authentication on.

## Recipes

**Dashboard widget** (Homepage, Homarr, custom): call `GET /api/state`,
show `stat.downSpeed`, `stat.numActive` and the first few `tasks`.

**Add a download from a shell alias:**

```sh
tfadd() {
  curl -s http://nas:8210/api/add \
    -H "Authorization: Bearer $TIDEFETCH_TOKEN" \
    -d "{\"kind\":\"uri\",\"uris\":[\"$1\"]}"
}
```

**Pause everything at night** (cron):

```sh
curl -s http://nas:8210/api/tasks/actions \
  -H "Authorization: Bearer $TIDEFETCH_TOKEN" \
  -d '{"action":"pauseAll"}'
```

For reverse-proxy setups, TLS and exposing the API safely beyond your LAN,
see [Reverse proxy & TLS](reverse-proxy.md) and
[Homelab operations](homelab.md).
