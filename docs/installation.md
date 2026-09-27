# Installation

Tidefetch is a single static binary. Pick one install method, make sure aria2 is present, then run `tidefetch doctor`.

## Choose a method

| Method | Platforms | aria2 | Upgrade |
| --- | --- | --- | --- |
| [Install script](#install-script) | macOS, Linux, Windows | Windows: installed for you. macOS and Linux: [install it yourself](#install-aria2) | Re-run the script |
| [Homebrew](#homebrew) | macOS | Installed as a dependency | `brew upgrade tidefetch` |
| [Container image](#container-image) | Docker, Podman, Kubernetes, Unraid | Bundled | Pull a new tag |
| [Go toolchain](#go-toolchain) | Anywhere Go runs | Install it yourself | Re-run `go install` |
| [Manual download](#manual-download) | macOS, Linux, Windows | Install it yourself | Download again |

Tidefetch is not in Winget, Scoop, Chocolatey, APT, DNF, the AUR or Nix. The install script covers those systems.

## Install script

**macOS and Linux**

```sh
curl -fsSL https://tidefetch.ijlalahmad.dev/install.sh | sh
```

**Windows** (PowerShell)

```powershell
irm https://tidefetch.ijlalahmad.dev/install.ps1 | iex
```

The script detects your OS and CPU, downloads the matching archive from the latest GitHub release, verifies its SHA-256 against the published `checksums.txt`, and installs the binary:

- macOS and Linux: `/usr/local/bin` when it is writable or the script runs as root, otherwise `~/.local/bin`. It warns if that directory is not on your `PATH`. Use `curl … | sudo sh` for a system-wide install.
- Windows: `%LOCALAPPDATA%\Programs\Tidefetch`, added to your user `PATH`. The script also installs aria2 through winget or Scoop when it is missing.

Builds exist for Linux amd64, arm64 and armv7 (32-bit Raspberry Pi), macOS Intel and Apple silicon, and Windows amd64 and arm64.

### Options

Set environment variables before `sh` to change the defaults:

| Variable | Default | Purpose |
| --- | --- | --- |
| `TIDEFETCH_VERSION` | latest release | Install a specific tag, for example `v0.1.0` |
| `TIDEFETCH_INSTALL_DIR` | `/usr/local/bin`, else `~/.local/bin` | Target directory |
| `TIDEFETCH_NO_SUDO` | unset | Set to `1` to never escalate |

```sh
curl -fsSL https://tidefetch.ijlalahmad.dev/install.sh \
  | TIDEFETCH_INSTALL_DIR="$HOME/bin" sh
```

The Windows script honours `TIDEFETCH_VERSION` and `TIDEFETCH_INSTALL_DIR` as well.

Piping a script into a shell means trusting it. To read it first:

```sh
curl -fsSL https://tidefetch.ijlalahmad.dev/install.sh -o install.sh
less install.sh
sh install.sh
```

## Homebrew

The cask lives in the `thre4dripper/tap` tap, not in Homebrew core, and pulls in the `aria2` formula:

```sh
brew install thre4dripper/tap/tidefetch
```

Upgrade with `brew upgrade tidefetch` and remove with `brew uninstall tidefetch`. The cask is built for macOS; on Linux use the install script.

## Container image

The image bundles aria2 and runs the web UI. It is published to GitHub Container Registry with a Docker Hub mirror, for `linux/amd64` and `linux/arm64`:

```text
ghcr.io/thre4dripper/tidefetch:latest
ijlalahmad/tidefetch:latest
```

Every release adds its own tags, for example `0.1.0` and `0.1`. Use `latest` to try it out and a release tag for anything you rely on.

```sh
docker run -d \
  --name tidefetch \
  --restart unless-stopped \
  -p 8210:8210 \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config \
  -v /srv/downloads:/downloads \
  ghcr.io/thre4dripper/tidefetch:latest
```

The password is required: the container listens on all interfaces, and Tidefetch refuses to do that without one. Open `http://<host>:8210` and sign in.

[Docker and Podman](deployment/docker.md) covers Compose, secrets, volumes and permissions. On Kubernetes, use the Helm chart:

```sh
helm install tidefetch oci://ghcr.io/thre4dripper/charts/tidefetch \
  --namespace tidefetch --create-namespace \
  --set auth.password='replace-this-password'
```

See [Kubernetes and k3s](deployment/kubernetes.md).

## Go toolchain

Requires Go 1.26.1 or newer. The embedded web UI is committed to the repository, so Node.js is not needed:

```sh
go install github.com/Thre4dripper/tidefetch/cmd/tidefetch@latest
```

Then [install aria2](#install-aria2).

## Manual download

Archives for every platform are attached to each [GitHub release](https://github.com/Thre4dripper/tidefetch/releases) together with `checksums.txt`:

```sh
curl -fsSLO https://github.com/Thre4dripper/tidefetch/releases/latest/download/tidefetch_linux_amd64.tar.gz
curl -fsSLO https://github.com/Thre4dripper/tidefetch/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
tar -xzf tidefetch_linux_amd64.tar.gz tidefetch
sudo install -m 0755 tidefetch /usr/local/bin/tidefetch
```

On macOS use `shasum -a 256 -c`, on Windows `Get-FileHash -Algorithm SHA256`. Every archive also carries a GitHub build provenance attestation:

```sh
gh attestation verify tidefetch_linux_amd64.tar.gz --repo Thre4dripper/tidefetch
```

## Install aria2

Tidefetch drives aria2 over RPC, so `aria2c` must be on your `PATH` unless you use the container image or the Windows script. Version 1.36 or newer.

```sh
brew install aria2          # macOS
sudo apt install aria2      # Debian, Ubuntu
sudo dnf install aria2      # Fedora, RHEL
sudo pacman -S aria2        # Arch
sudo apk add aria2          # Alpine
winget install aria2.aria2  # Windows
```

## Verify

```sh
tidefetch version
tidefetch doctor
```

`doctor` reports the config file, the aria2 binary and its version, whether the RPC endpoint answers, and whether the download and data directories are writable.

## Upgrade

| Installed with | Upgrade |
| --- | --- |
| Install script | Re-run the same one-liner |
| Homebrew | `brew upgrade tidefetch` |
| Container | Pull the new tag and recreate the container |
| Go | Re-run `go install …@latest` |

The config file is forward-compatible. Run `tidefetch doctor` afterwards to confirm everything still resolves.

## Uninstall

Delete the binary the installer reported. To remove your settings and history as well, delete:

- Config: `~/.config/tidefetch` on Linux, `~/Library/Application Support/tidefetch` on macOS, `%AppData%\tidefetch` on Windows
- Data: `~/.local/share/tidefetch`

## Build from source

Requires Go 1.26.1 or newer, Node.js 22, Make and aria2:

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
make build          # web UI + Go binary with embedded assets
./tidefetch doctor
```

`make install` places the binary in `$(go env GOPATH)/bin`. Other targets: `make backend` (Go only, reuses the last web build), `make test`, `make site` and `make docker`.

Next: [Quick start](quick-start.md).
