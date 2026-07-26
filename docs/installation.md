# Installation

Choose the all-in-one container for a NAS or server. Choose a native binary
for the terminal interface or when an aria2 daemon already exists.

> [!IMPORTANT]
> As of 2026-07-26 this repository has no release tag and no package-manager
> submissions. Source builds and local container builds work now. Registry
> commands below are the supported publication targets and must not be treated
> as available until the project release page marks them **Published**.

## Availability

| Method | Current status | Includes aria2 |
| --- | --- | --- |
| Source checkout (`make build`) | Available now | No |
| Local Docker/Compose build | Available now | Yes |
| `go install` from the default branch | Available after this module path is pushed | No |
| GitHub release archives and packages | Awaiting first tag | No |
| GHCR container image | Awaiting first tag | Yes |
| Homebrew, Winget, Scoop, Chocolatey, AUR, Nix | Not submitted yet | No |

## Requirements

### Native runtime

- aria2 in `PATH`
- A modern terminal for the TUI
- A current browser for `tidefetch serve`

### Building

- Go 1.26.1 or the version declared by `go.mod`
- Node.js 20 or newer and npm when rebuilding the embedded web interface
- Git and Make

The Docker image has no host runtime dependency beyond Docker or Podman.

## Source installation

Install build and runtime dependencies first.

### macOS

```sh
brew install aria2 go node
```

### Debian and Ubuntu

The distribution Go package may be older than this repository requires. Use
the official Go toolchain if `go version` is below the version in `go.mod`.

```sh
sudo apt update
sudo apt install -y aria2 git make nodejs npm
```

### Fedora and RHEL derivatives

```sh
sudo dnf install -y aria2 git make golang nodejs npm
```

### Arch Linux

```sh
sudo pacman -S --needed aria2 git make go nodejs npm
```

### Alpine Linux

```sh
sudo apk add aria2 git make go nodejs npm
```

Build and install:

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
make build
./tidefetch doctor
```

Run `./tidefetch` from the checkout, or install it into `GOBIN`:

```sh
make install
command -v tidefetch
tidefetch version
```

If `command -v` finds nothing, add `$(go env GOPATH)/bin` to `PATH`.

## Go toolchain installation

Once the canonical module path is available from the public default branch:

```sh
go install github.com/Thre4dripper/tidefetch/cmd/tidefetch@latest
```

This installs the binary only. Install aria2 separately and run
`tidefetch doctor` before first use.

## Containers

### Build the current checkout

```sh
git clone https://github.com/Thre4dripper/tidefetch.git
cd tidefetch
mkdir -p packaging/docker/config packaging/docker/downloads
TIDEFETCH_PASSWORD='replace-this-password' \
  docker compose -f packaging/docker/docker-compose.yml up -d --build
```

### Published GHCR image

After the first container release is published:

```sh
docker pull ghcr.io/thre4dripper/tidefetch:latest
docker run -d \
  --name tidefetch \
  --restart unless-stopped \
  -p 8210:8210 \
  -p 6881:6881 \
  -p 6881:6881/udp \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config \
  -v "$PWD/downloads:/downloads" \
  ghcr.io/thre4dripper/tidefetch:latest
```

Podman accepts the same arguments:

```sh
podman run -d --name tidefetch --replace \
  -p 8210:8210 \
  -e TIDEFETCH_PASSWORD='replace-this-password' \
  -v tidefetch-config:/config:Z \
  -v "$PWD/downloads:/downloads:Z" \
  ghcr.io/thre4dripper/tidefetch:latest
```

See [Docker deployment](deployment/docker.md) for permissions, secrets, and
upgrade commands.

## Release archives

After a tagged release, download the archive matching the operating system and
architecture from GitHub Releases. Always verify `checksums.txt` before
installing.

```sh
VERSION=0.2.0
ARCHIVE="tidefetch_${VERSION}_Linux_x86_64.tar.gz"
curl -LO "https://github.com/Thre4dripper/tidefetch/releases/download/v${VERSION}/${ARCHIVE}"
curl -LO "https://github.com/Thre4dripper/tidefetch/releases/download/v${VERSION}/checksums.txt"
grep " ${ARCHIVE}$" checksums.txt | sha256sum -c -
tar -xzf "$ARCHIVE"
sudo install -m 0755 tidefetch /usr/local/bin/tidefetch
```

On macOS, use `shasum -a 256 -c` instead of `sha256sum -c`. On Windows,
compare with `Get-FileHash -Algorithm SHA256`.

## Package-manager publication targets

These are the intended commands after each listing is published. A missing
package is not a local installation problem; use a source or container build.

### Homebrew (macOS and Linux)

```sh
brew tap Thre4dripper/tap
brew install tidefetch
brew services start aria2  # optional existing daemon
```

Upgrade or remove:

```sh
brew update && brew upgrade tidefetch
brew uninstall tidefetch
```

### Winget (Windows)

```powershell
winget install --id Thre4dripper.Tidefetch --exact
winget upgrade --id Thre4dripper.Tidefetch --exact
winget uninstall --id Thre4dripper.Tidefetch --exact
```

Install aria2 separately with `winget install aria2.aria2` if that package is
available in the current Winget source, or use the official aria2 release.

### Scoop (Windows)

```powershell
scoop bucket add tidefetch https://github.com/Thre4dripper/scoop-tidefetch
scoop install tidefetch
scoop update tidefetch
```

### Chocolatey (Windows)

```powershell
choco install tidefetch
choco upgrade tidefetch
choco uninstall tidefetch
```

### AUR (Arch Linux)

Binary package:

```sh
yay -S tidefetch-bin
```

Default-branch build:

```sh
yay -S tidefetch-git
```

### Nix

Once the flake is published in a release:

```sh
nix profile install github:Thre4dripper/tidefetch
nix run github:Thre4dripper/tidefetch -- doctor
```

### Debian and RPM packages

Release assets will use `.deb` and `.rpm` packages:

```sh
sudo apt install ./tidefetch_0.2.0_linux_amd64.deb
sudo dnf install ./tidefetch_0.2.0_linux_amd64.rpm
```

An apt or RPM repository is not configured yet; use release assets directly
until signed repositories are announced.

## Verify the installation

```sh
tidefetch version
tidefetch doctor
```

`doctor` checks the config file, aria2 executable, RPC connectivity, download
directory, and writable data directory.

## First run

Terminal interface:

```sh
tidefetch
```

Local-only web interface:

```sh
tidefetch serve
```

LAN web interface with authentication:

```sh
tidefetch serve -host 0.0.0.0 -password 'replace-this-password'
```

Then open `http://<host>:8210`. Use a reverse proxy for HTTPS before exposing
the service beyond a trusted network.
