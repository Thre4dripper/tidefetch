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
2. Merge the release PR that `release-please` opens. That tags the commit and
   runs the release workflow, which builds the archives and `checksums.txt`.
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

Releases are driven by [release-please](https://github.com/googleapis/release-please)
and Conventional Commits. There is no manual tagging and no version file to
edit — the only human step is merging a pull request.

```text
commit to main  →  release-please updates the Release PR
                   (version bump + CHANGELOG.md)
merge that PR   →  tag vX.Y.Z + GitHub Release
                →  GoReleaser attaches binaries, images and the chart
```

### Commit messages decide the version

| Commit prefix | Effect while pre-1.0 | Effect after 1.0 |
| --- | --- | --- |
| `fix:` | patch — `0.2.0 → 0.2.1` | patch |
| `feat:` | minor — `0.2.0 → 0.3.0` | minor |
| `feat!:` or `BREAKING CHANGE:` | minor — `0.2.0 → 0.3.0` | **major** |
| `docs:` `refactor:` `perf:` | patch | patch |
| `chore:` `ci:` `test:` `build:` | no release | no release |

Anything that is not a Conventional Commit is ignored for versioning, so
`chore:` and `ci:` commits never cut a release on their own.

### The Release PR

`release-please` keeps one open PR titled `chore(main): release X.Y.Z`. It
rewrites it on every push to `main`, so it always reflects everything unreleased.
Merging it is the release. Closing it without merging simply defers the release.

Merging that PR bumps the version in every place it appears:

- `cmd/tidefetch/main.go` and `packaging/helm/tidefetch/Chart.yaml` (annotated
  with `x-release-please-version`)
- `web/package.json` and `site/package.json`
- `CHANGELOG.md`

The `Makefile` is deliberately not in that list — it derives the version from
`git describe`, so local builds are always self-describing.

### Version scheme

Tags are SemVer with a `v` prefix (`v0.3.0`), which is what Go modules and the
install script expect.

Stay on `0.x` while CLI flags, the config file format or the web API may still
change — under SemVer, `0.x` explicitly means "anything may break". Cut `v1.0.0`
once those are stable, and from then on `feat!:` triggers a major bump.

### Pre-releases

Tag a release candidate to publish without disturbing existing users:

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

### Manual releases

Pushing a `v*` tag by hand still works and runs the same pipeline — useful for
release candidates or a hotfix from a branch.

## Cutting a release by hand

Only needed if you are bypassing `release-please`.

1. Make sure `main` is green.
2. Tag and push:

   ```sh
   git tag -a v0.3.0 -m "v0.3.0"
   git push origin v0.3.0
   ```

3. The `release` workflow runs GoReleaser and publishes every target above.
   Remember to bump `.release-please-manifest.json` to match, or the next
   Release PR will propose a version you have already shipped.

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
