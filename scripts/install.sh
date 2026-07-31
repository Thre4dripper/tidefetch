#!/bin/sh
# Tidefetch installer for macOS and Linux.
#
#   curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh
#
# Nothing is compiled: this downloads the prebuilt static binary that matches
# your platform, verifies its SHA-256 against the signed checksums file from the
# same release, and installs it onto your PATH.
#
# Environment overrides:
#   TIDEFETCH_VERSION      install a specific tag, e.g. v0.2.0 (default: latest)
#   TIDEFETCH_INSTALL_DIR  target directory (default: /usr/local/bin or ~/.local/bin)
#   TIDEFETCH_NO_SUDO=1    never escalate; fall back to a user-writable directory

set -eu

REPO="Thre4dripper/tidefetch"
BINARY="tidefetch"

bold=""
dim=""
red=""
green=""
yellow=""
reset=""
if [ -t 2 ] && [ -z "${NO_COLOR:-}" ]; then
	bold="$(printf '\033[1m')"
	dim="$(printf '\033[2m')"
	red="$(printf '\033[31m')"
	green="$(printf '\033[32m')"
	yellow="$(printf '\033[33m')"
	reset="$(printf '\033[0m')"
fi

info() { printf '%s==>%s %s\n' "$green" "$reset" "$1" >&2; }
warn() { printf '%swarning:%s %s\n' "$yellow" "$reset" "$1" >&2; }
die() {
	printf '%serror:%s %s\n' "$red" "$reset" "$1" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

# ── Platform detection ────────────────────────────────────────────────────────

detect_os() {
	os="$(uname -s)"
	case "$os" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	MINGW* | MSYS* | CYGWIN* | Windows_NT)
		die "Windows detected. Run the PowerShell installer instead:
  irm https://thre4dripper.github.io/tidefetch/install.ps1 | iex"
		;;
	*) die "unsupported operating system: $os" ;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	armv7l | armv7 | armhf) echo armv7 ;;
	*) die "unsupported architecture: $arch" ;;
	esac
}

# ── HTTP helpers ──────────────────────────────────────────────────────────────

if command -v curl >/dev/null 2>&1; then
	http_get() { curl -fsSL "$1"; }
	http_download() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	http_get() { wget -qO- "$1"; }
	http_download() { wget -qO "$2" "$1"; }
else
	die "either curl or wget is required"
fi

# ── Release resolution ────────────────────────────────────────────────────────

resolve_version() {
	if [ -n "${TIDEFETCH_VERSION:-}" ]; then
		echo "$TIDEFETCH_VERSION"
		return
	fi
	# Ask the API for the newest published (non-draft) release.
	tag="$(http_get "https://api.github.com/repos/$REPO/releases/latest" |
		sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
		head -n 1)"
	[ -n "$tag" ] || die "could not determine the latest release; set TIDEFETCH_VERSION"
	echo "$tag"
}

# ── Checksum verification ─────────────────────────────────────────────────────

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	elif command -v sha256 >/dev/null 2>&1; then
		sha256 -q "$1"
	else
		echo ""
	fi
}

verify_checksum() {
	archive_path="$1"
	archive_name="$2"
	sums_path="$3"

	expected="$(awk -v name="$archive_name" '$2 == name || $2 == "*" name {print $1}' "$sums_path" | head -n 1)"
	[ -n "$expected" ] || die "no checksum entry for $archive_name"

	actual="$(sha256_of "$archive_path")"
	if [ -z "$actual" ]; then
		warn "no SHA-256 tool available; skipping checksum verification"
		return
	fi
	if [ "$expected" != "$actual" ]; then
		die "checksum mismatch for $archive_name
  expected $expected
  actual   $actual"
	fi
	info "Checksum verified."
}

# ── Install location ──────────────────────────────────────────────────────────

SUDO=""

choose_install_dir() {
	if [ -n "${TIDEFETCH_INSTALL_DIR:-}" ]; then
		echo "$TIDEFETCH_INSTALL_DIR"
		return
	fi
	# Prefer a system location, but only if we can write there without prompting
	# for a password in a piped shell.
	if [ -w /usr/local/bin ] 2>/dev/null; then
		echo /usr/local/bin
		return
	fi
	if [ "$(id -u)" = "0" ]; then
		echo /usr/local/bin
		return
	fi
	if [ "${TIDEFETCH_NO_SUDO:-}" != "1" ] && command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
		SUDO="sudo"
		echo /usr/local/bin
		return
	fi
	echo "$HOME/.local/bin"
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
	need uname
	need tar
	need awk

	OS="$(detect_os)"
	ARCH="$(detect_arch)"
	VERSION="$(resolve_version)"

	archive="${BINARY}_${OS}_${ARCH}.tar.gz"
	base="https://github.com/$REPO/releases/download/$VERSION"

	info "Installing ${bold}${BINARY} ${VERSION}${reset} for ${OS}/${ARCH}."

	tmp="$(mktemp -d 2>/dev/null || mktemp -d -t tidefetch)"
	trap 'rm -rf "$tmp"' EXIT INT TERM

	info "Downloading $archive"
	http_download "$base/$archive" "$tmp/$archive" ||
		die "no release asset named $archive in $VERSION"

	if http_download "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
		verify_checksum "$tmp/$archive" "$archive" "$tmp/checksums.txt"
	else
		warn "checksums.txt not published for $VERSION; skipping verification"
	fi

	tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY" ||
		die "archive did not contain a $BINARY binary"
	chmod +x "$tmp/$BINARY"

	dir="$(choose_install_dir)"
	$SUDO mkdir -p "$dir" || die "could not create $dir"

	# install(1) is not portable enough across BSD/GNU here, so copy then chmod.
	if ! $SUDO cp "$tmp/$BINARY" "$dir/$BINARY"; then
		die "could not write to $dir. Re-run with:
  TIDEFETCH_INSTALL_DIR=\$HOME/.local/bin sh"
	fi
	$SUDO chmod 0755 "$dir/$BINARY"

	info "Installed to ${bold}$dir/$BINARY${reset}"

	# ── Post-install guidance ────────────────────────────────────────────────
	case ":$PATH:" in
	*":$dir:"*) ;;
	*)
		warn "$dir is not on your PATH. Add this to your shell profile:
  export PATH=\"$dir:\$PATH\""
		;;
	esac

	if ! command -v aria2c >/dev/null 2>&1; then
		warn "aria2 was not found. Tidefetch drives the aria2 engine, so install it:
  macOS         brew install aria2
  Debian/Ubuntu sudo apt install aria2
  Fedora        sudo dnf install aria2
  Arch          sudo pacman -S aria2
  Alpine        sudo apk add aria2"
	fi

	printf '\n%sRun %s%s doctor%s to verify your setup, or just %s%s%s to start.%s\n' \
		"$dim" "$reset$bold" "$BINARY" "$reset$dim" "$reset$bold" "$BINARY" "$reset$dim" "$reset" >&2
}

main "$@"
