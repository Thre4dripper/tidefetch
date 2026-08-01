# Installation

Tidefetch is a single static binary with no runtime to install. One command
gets you the terminal UI; add `tidefetch serve` later if you want the web UI.

## Quick install

**macOS and Linux**

```sh
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh
```

**Windows** (PowerShell)

```powershell
irm https://thre4dripper.github.io/tidefetch/install.ps1 | iex
```

That is the whole install. The script detects your OS and CPU, downloads the
matching binary from the latest release, verifies its SHA-256 against the
published checksums, and puts it on your `PATH`.

Then confirm everything is wired up:

```sh
tidefetch version
tidefetch doctor
```

Re-run the same command to upgrade. Windows Terminal is recommended — the TUI
uses truecolor and Unicode block glyphs that the legacy console host renders
poorly.

## The aria2 engine

Tidefetch is an interface to [aria2](https://aria2.github.io), so the engine
has to be present.

| How you installed | aria2 |
| --- | --- |
| Windows script | Installed automatically |
| Homebrew | Installed automatically as a dependency |
| Docker | Bundled in the image |
| Linux or macOS script | Install it yourself, see below |

```sh
brew install aria2          # macOS
sudo apt install aria2      # Debian, Ubuntu
sudo dnf install aria2      # Fedora, RHEL
sudo pacman -S aria2        # Arch
sudo apk add aria2          # Alpine
```

`tidefetch doctor` tells you if it is missing.

## Homebrew

Use this if you would rather Homebrew owned upgrades. It pulls in aria2 too.

```sh
brew install thre4dripper/tap/tidefetch
brew upgrade tidefetch
brew uninstall tidefetch
```

Tidefetch is not published to Winget, Scoop, Chocolatey, APT, DNF, AUR or Nix —
the install script replaces all of them.

## Install script options

Set environment variables before `sh` to change the defaults:

```sh
# Pin a version and choose the target directory.
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh \
  | TIDEFETCH_VERSION=v0.2.0 TIDEFETCH_INSTALL_DIR="$HOME/bin" sh

# Never escalate, even if sudo is available.
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | TIDEFETCH_NO_SUDO=1 sh
```

| Variable | Default |
| --- | --- |
| `TIDEFETCH_VERSION` | latest release |
| `TIDEFETCH_INSTALL_DIR` | `/usr/local/bin`, else `~/.local/bin` |
| `TIDEFETCH_NO_SUDO` | unset |

The Windows script accepts the same first two and installs to
`%LOCALAPPDATA%\Programs\Tidefetch`.

Piping a script into a shell means trusting it, so read it first if you prefer:

```sh
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh -o install.sh
less install.sh
sh install.sh
```

### Manual download

Archives are attached to every release for `linux` (`amd64`, `arm64`, `armv7`),
`darwin` (`amd64`, `arm64`) and `windows` (`amd64`, `arm64`):

```sh
curl -fsSL https://github.com/Thre4dripper/tidefetch/releases/latest/download/tidefetch_linux_amd64.tar.gz \
  | tar -xz tidefetch
sudo install -m 0755 tidefetch /usr/local/bin/tidefetch
```

### Uninstall

Delete the binary the installer reported, then remove `~/.config/tidefetch`
and `~/.local/share/tidefetch`.

## Containers

### Docker

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

Images are multi-arch — Docker pulls the right build for your CPU automatically:

```sh
docker pull ghcr.io/thre4dripper/tidefetch:latest   # GitHub Container Registry
docker pull ijlalahmad/tidefetch:latest             # Docker Hub mirror
```

Pin a version for anything you care about:

```sh
docker pull ghcr.io/thre4dripper/tidefetch:0.2.0
```

### Docker Compose

```yaml
services:
  tidefetch:
    image: ghcr.io/thre4dripper/tidefetch:latest
    container_name: tidefetch
    restart: unless-stopped
    environment:
      TIDEFETCH_PASSWORD: replace-this-password
    ports:
      - "8210:8210"
      - "6881:6881"
      - "6881:6881/udp"
    volumes:
      - ./config:/config
      - ./downloads:/downloads
```

### Podman

```sh
podman run -d --name tidefetch --replace \
  -p 8210:8210 \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config:Z \
  -v /srv/downloads:/downloads:Z \
  ghcr.io/thre4dripper/tidefetch:latest
```

### Kubernetes (Helm)

The chart is published as an OCI artifact to GitHub Container Registry:

```sh
helm install tidefetch oci://ghcr.io/thre4dripper/charts/tidefetch \
  --namespace tidefetch --create-namespace \
  --set auth.password='replace-this-password' \
  --set persistence.downloads.size=200Gi
```

See [Kubernetes deployment](deployment/kubernetes.md) for raw manifests and
storage guidance.

## Go toolchain

```sh
go install github.com/Thre4dripper/tidefetch/cmd/tidefetch@latest
```

This builds the binary only — install aria2 separately with your package
manager, then run `tidefetch doctor`.

## From source

Requires Go 1.26.1+, Node.js 20+, Make and aria2.

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
make build
./tidefetch doctor
```

`make install` places the binary in `$(go env GOPATH)/bin`.

Useful targets:

```sh
make build      # web assets + binary
make backend    # Go only, reuses the last web build
make site       # product site
make test       # unit and integration tests
make docker     # container image
```

## Verify a download

`tidefetch doctor` checks the config file, aria2 binary, RPC connectivity,
download directory and data directory.

The install script already verifies checksums for you. To check a manual
download, release artifacts ship a signed checksum manifest:

```sh
curl -fsSLO https://github.com/Thre4dripper/tidefetch/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

On macOS use `shasum -a 256 -c`. On Windows use `Get-FileHash -Algorithm SHA256`.

## First run

```sh
tidefetch                                  # the TUI
tidefetch https://example.com/file.iso     # queue a download on launch
tidefetch serve                            # web UI on http://127.0.0.1:8210
```

Continue with [Configuration](configuration.md), or jump to
[Homelab operations](homelab.md) for server deployments.

## Publishing new releases

Maintainers: see [Publishing](publishing.md) for the release pipeline and the
one-time credentials each registry needs.
