# Publishing releases

This is the maintainer runbook. Releases are automated with
[GoReleaser](https://goreleaser.com) — tagging a version publishes binaries,
packages, container images and the Helm chart in one pass.

## What a release produces

| Target | Artifact | Automated |
| --- | --- | --- |
| GitHub Releases | archives, checksums, signature | yes |
| Install scripts | `install.sh` / `install.ps1` on GitHub Pages | yes |
| Docker Hub + GHCR | multi-arch `linux/amd64` and `linux/arm64` images | yes |
| Homebrew | cask in `Thre4dripper/homebrew-tap` | yes |
| Helm chart | `oci://ghcr.io/thre4dripper/charts` | yes |

System-level installs are served by the install script rather than by
per-distribution registries or package files. The script reads the same GitHub
release archives listed above, so there is nothing extra to publish for Linux,
macOS or Windows. Tidefetch is deliberately **not** published to Winget, Scoop,
AUR, Nix, MacPorts, Chocolatey or hosted APT/RPM repositories, and no `.deb`,
`.rpm` or `.apk` is produced — each of those needs its own repository,
credential and review cycle per release.

## One-time setup

### Repository secrets

Add these under **Settings → Secrets and variables → Actions**:

| Secret | Purpose |
| --- | --- |
| `HOMEBREW_TAP_TOKEN` | PAT with `contents:write` on `homebrew-tap` |
| `DOCKERHUB_USERNAME` | Docker Hub account name |
| `DOCKERHUB_TOKEN` | Docker Hub access token with write scope |
| `GPG_PRIVATE_KEY` | ASCII-armoured signing key for packages and checksums |
| `GPG_PASSPHRASE` | Passphrase for the above |

`GITHUB_TOKEN` is provided automatically and covers GHCR, GitHub Releases and
the Helm chart push.

### Companion repositories

Create this once:

- `Thre4dripper/homebrew-tap`

### Signing key

```sh
gpg --full-generate-key
gpg --armor --export-secret-keys you@example.com | pbcopy   # → GPG_PRIVATE_KEY
```

The detached signature for `checksums.txt` is attached to each release, so
anyone can verify a download against your public key.

### GitHub Pages

The install scripts are served from the product site. Enable **Settings → Pages
→ Source: GitHub Actions** once, then the `site` workflow publishes
`install.sh` and `install.ps1` to the site root on every push to `main` that
touches `site/**`, `docs/**` or `scripts/install.*`.

## The first release

The install script and the binaries it downloads are hosted independently, so
there is no bootstrap problem — but the order matters, because the script is
useless until a release exists for it to fetch.

1. Enable GitHub Pages and merge to `main`. The `site` workflow publishes
   `https://thre4dripper.github.io/tidefetch/install.sh`. At this point the URL
   resolves but the script has nothing to install yet.
2. Push the first tag: `git tag v0.2.0 && git push origin v0.2.0`. The `release`
   workflow builds the archives and `checksums.txt` and publishes the release.
3. Verify from a clean machine or container:

   ```sh
   docker run --rm -it debian:stable-slim sh -c \
     'apt-get update -qq && apt-get install -y -qq curl ca-certificates >/dev/null &&
      curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh &&
      tidefetch version'
   ```

Users only need `curl` (or `wget`) and `tar`, which ship on effectively every
macOS and Linux system. Nothing has to be preinstalled.

## Release cycle

You choose the version number. Everything else is automated, and the workflow
refuses anything malformed or backwards.

```text
push to main   →  "next" draft release is rebuilt
                  (unreleased commits + real binaries to test)
ready to ship  →  push a tag, or run the workflow with a version
                  → validate → build → publish → draft cleared
```

### The "next" draft release

The `next` workflow keeps a single draft release called **Next (unreleased)** in
sync with `main`. On every push it lists the commits since the last tag and
attaches freshly built binaries for all seven platforms.

It is a draft release rather than a pull request for two reasons: a PR cannot
carry build artifacts, and it would have to modify a file just to exist. Drafts
are invisible to users and excluded from `/releases/latest`, so the install
script keeps serving the last real release.

Use it to download and test `main` before committing to a version number. When
you release, the draft is deleted automatically because its contents just
shipped.

### Cutting a release

Either push a tag:

```sh
git tag v0.3.0
git push origin v0.3.0
```

Or go to **Actions → Release → Run workflow**, type `v0.3.0` in the version box
and untick *dry run*. The workflow creates and pushes the tag for you.

Leaving the version empty with *dry run* ticked builds everything and attaches
it to the workflow run without publishing. That path needs no secrets, so it is
the safest way to prove the pipeline works.

### What gets rejected

The `validate` job runs before anything is built:

| Input | Result |
| --- | --- |
| `0.3.0` | rejected — needs the `v` prefix |
| `v0.3` | rejected — not `MAJOR.MINOR.PATCH` |
| `v0.2.0` when `v0.2.0` exists | rejected — already released |
| `v0.1.0` when `v0.2.0` exists | rejected — older than the latest release |
| `v0.3.0` or `v1.0.0-rc.1` | accepted |

### Choosing the number

Tags are SemVer with a `v` prefix, which is what Go modules and the install
script expect.

| Change | Bump |
| --- | --- |
| Bug fixes only | patch — `0.2.0 → 0.2.1` |
| New features, nothing broken | minor — `0.2.0 → 0.3.0` |
| Breaking change to flags, config or API | minor while pre-1.0, major after |

Stay on `0.x` while CLI flags, the config file format or the web API may still
change — under SemVer, `0.x` explicitly means "anything may break". Cut `v1.0.0`
once those are stable.

### Pre-releases

```sh
git tag v1.0.0-rc.1 && git push origin v1.0.0-rc.1
```

GoReleaser marks it as a prerelease automatically (`prerelease: auto`), and
GitHub excludes prereleases from `/releases/latest`, so the install script keeps
serving the last stable version. Testers opt in explicitly:

```sh
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh \
  | TIDEFETCH_VERSION=v1.0.0-rc.1 sh
```

### Version strings in the repository

`cmd/tidefetch/main.go`, `packaging/helm/tidefetch/Chart.yaml`,
`web/package.json` and `site/package.json` carry a version, but none of them are
authoritative — the tag is. GoReleaser stamps the real version through
`-ldflags`, and the Helm chart is packaged with `--version` from the tag. The
`Makefile` derives its version from `git describe`, so local builds are always
self-describing.

## Dry run

Validate the pipeline without publishing anything:

```sh
make release-check          # goreleaser check + build snapshot
goreleaser release --snapshot --clean --skip=publish
```

Artifacts land in `dist/`. Check the archives and the generated Homebrew cask
before tagging:

```sh
tar -tzf dist/tidefetch_linux_amd64.tar.gz
cat dist/homebrew/Casks/tidefetch.rb
```

## Manual steps

### Unraid Community Applications

The template at `packaging/unraid/tidefetch.xml` is fetched from `main`, so it
updates automatically. Only submit to the Community Applications repository
when the template is added or renamed.

## Verifying a published release

```sh
curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh && tidefetch version
brew install thre4dripper/tap/tidefetch && tidefetch version
docker run --rm thre4dripper/tidefetch:0.3.0 version
helm show chart oci://ghcr.io/thre4dripper/charts/tidefetch | grep version
```

Check the multi-arch manifest:

```sh
docker buildx imagetools inspect thre4dripper/tidefetch:0.3.0
```

## Rolling back

Container tags are immutable per version — point deployments at the previous
tag. For a bad package release, publish a patch version rather than deleting
artifacts; package managers cache aggressively and deletions break clients that
already resolved the version.
