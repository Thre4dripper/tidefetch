# Unraid

The repository includes an Unraid Docker template with appdata, downloads,
web password, web port, and optional BitTorrent ports.

> [!NOTE]
> The template can be installed manually after the GHCR image is published.
> Community Applications availability requires a separate listing submission
> and is not live yet.

## Install from the template repository

In the Unraid web interface:

1. Open **Settings → Docker** and enable template repositories if the current
   Unraid version exposes that field.
2. Add this repository URL:

   ```text
   https://github.com/Thre4dripper/tidefetch
   ```

3. Open **Docker → Add Container**.
4. Select **Tidefetch** from user templates. If it does not appear, paste the
   raw XML URL into the template URL field:

   ```text
   https://raw.githubusercontent.com/Thre4dripper/tidefetch/main/packaging/unraid/tidefetch.xml
   ```

5. Review every path and set a strong web password.
6. Apply the template and wait for the container to report healthy.

## Manual container fields

| Field | Value |
| --- | --- |
| Repository | `ghcr.io/thre4dripper/tidefetch:latest` |
| Network | `bridge` |
| Web UI port | Host `8210` → container `8210/tcp` |
| Appdata | `/mnt/user/appdata/tidefetch` → `/config` |
| Downloads | `/mnt/user/downloads` → `/downloads` |
| Password variable | `TIDEFETCH_PASSWORD` |
| Timezone variable | `TZ`, for example `America/New_York` |
| Extra parameters | `--security-opt no-new-privileges --cap-drop ALL` |

Optional BitTorrent mappings:

- Host `6881` → container `6881/tcp`
- Host `6881` → container `6881/udp`

Do not add a mapping for aria2 RPC port 6800.

## Permissions

The image runs as UID/GID 1000. From the Unraid terminal:

```sh
mkdir -p /mnt/user/appdata/tidefetch /mnt/user/downloads
chown -R 1000:1000 /mnt/user/appdata/tidefetch
chown 1000:1000 /mnt/user/downloads
chmod 700 /mnt/user/appdata/tidefetch
```

Changing ownership recursively on a shared downloads tree may affect other
containers. Prefer a dedicated share or align all download consumers on the
same group and permissions.

## First start

Open:

```text
http://<unraid-ip>:8210
```

Sign in with the template password. In Settings, confirm the download
directory is `/downloads` and inspect free space.

If the container exits immediately, read its log from the Docker tab. The most
common cause is an empty web password while binding to all interfaces.

## Reverse proxy

With Nginx Proxy Manager, create a Proxy Host:

- Scheme: `http`
- Forward hostname/IP: Unraid host IP
- Forward port: `8210`
- Websockets Support: enabled
- Block Common Exploits: enabled
- SSL: request or select a certificate, force SSL

Preserve Tidefetch's own password unless the proxy provides trusted identity
and access control. See [Reverse proxy and TLS](../reverse-proxy.md).

## Router ports

Forward TCP and UDP 6881 to the Unraid server only when BitTorrent inbound
peers are needed. Do not forward port 8210 without HTTPS, and never forward
6800.

## Backup

Include `/mnt/user/appdata/tidefetch` in the Appdata Backup plugin or normal NAS
backup. Stop Tidefetch during the snapshot so aria2 has a consistent session.

Manual backup:

```sh
docker stop tidefetch
tar -C /mnt/user/appdata -czf /mnt/user/backups/tidefetch-appdata.tar.gz tidefetch
docker start tidefetch
```

## Update and rollback

Use **Check for Updates** in the Docker tab, then update the container. For
predictable rollback, replace `latest` in the Repository field with a fixed
version such as:

```text
ghcr.io/thre4dripper/tidefetch:0.2.0
```

Record the previous tag before updating. Appdata survives container replacement.

## Remove

Removing the container does not delete mapped appdata or downloads. After a
verified backup, remove `/mnt/user/appdata/tidefetch` manually only when the
configuration and history are no longer needed.