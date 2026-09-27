# Bare metal with systemd

Run `tidefetch serve` as a native service on a Raspberry Pi, a small VPS or any host without Docker. Only the web UI needs a service; the terminal UI is an interactive command and should not run under systemd.

## Install

```sh
curl -fsSL https://tidefetch.ijlalahmad.dev/install.sh | sudo sh
sudo apt install aria2        # or dnf, pacman, apk
tidefetch version
```

Create a system user and the directories the unit expects:

```sh
sudo useradd --system --home-dir /var/lib/tidefetch --shell /usr/sbin/nologin tidefetch
sudo mkdir -p /srv/downloads /etc/tidefetch
sudo chown tidefetch:tidefetch /srv/downloads
```

## Password

Store the web password in a file only the service user can read. The unit passes it through `TIDEFETCH_PASSWORD_FILE`, so it never appears in the unit or in `ps`:

```sh
openssl rand -base64 36 | sudo tee /etc/tidefetch/password >/dev/null
sudo chown tidefetch:tidefetch /etc/tidefetch/password
sudo chmod 600 /etc/tidefetch/password
```

## Unit

Install the unit from the repository and enable it:

```sh
sudo curl -fsSL -o /etc/systemd/system/tidefetch.service \
  https://raw.githubusercontent.com/Thre4dripper/tidefetch/main/packaging/systemd/tidefetch.service
sudo systemctl daemon-reload
sudo systemctl enable --now tidefetch
systemctl status tidefetch
```

The unit runs `tidefetch serve -host 0.0.0.0 -dir /srv/downloads` as the `tidefetch` user with `HOME=/var/lib/tidefetch`, so the config, history and aria2 session land in that state directory. It is hardened with `ProtectSystem=strict`, `NoNewPrivileges`, `PrivateTmp` and a writable set limited to `/srv/downloads` and the state directory.

To download somewhere else, change both `-dir` in `ExecStart` and `ReadWritePaths`, then run `sudo systemctl daemon-reload` and restart the service.

## Operate

```sh
journalctl -u tidefetch -f                                         # logs
sudo systemctl restart tidefetch                                   # restart
sudo -u tidefetch env HOME=/var/lib/tidefetch tidefetch doctor     # diagnostics as the service user
```

Upgrade by re-running the install script and restarting:

```sh
curl -fsSL https://tidefetch.ijlalahmad.dev/install.sh | sudo sh
sudo systemctl restart tidefetch
```

Back up `/var/lib/tidefetch` while the service is stopped; see [Data and persistence](../data-and-persistence.md). For TLS in front of the service, see [Reverse proxy and TLS](../reverse-proxy.md).
