<#
.SYNOPSIS
    Tidefetch installer for Windows.

.DESCRIPTION
    Downloads the prebuilt tidefetch binary matching this machine's
    architecture, verifies its SHA-256 against the release checksums file, and
    installs it into a user-writable directory that is added to PATH.

    irm https://thre4dripper.github.io/tidefetch/install.ps1 | iex

.PARAMETER Version
    Install a specific tag (for example v0.2.0). Defaults to the latest release.

.PARAMETER InstallDir
    Target directory. Defaults to %LOCALAPPDATA%\Programs\Tidefetch.
#>

[CmdletBinding()]
param(
    [string]$Version = $env:TIDEFETCH_VERSION,
    [string]$InstallDir = $env:TIDEFETCH_INSTALL_DIR
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Repo = 'Thre4dripper/tidefetch'
$Binary = 'tidefetch'

function Write-Info { param([string]$Message) Write-Host "==> $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "warning: $Message" -ForegroundColor Yellow }

function Get-TargetArchitecture {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        'x86' { throw 'Tidefetch does not ship 32-bit Windows builds.' }
        default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

function Resolve-Version {
    if ($Version) { return $Version }
    Write-Info 'Resolving the latest release...'
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
        -Headers @{ 'User-Agent' = 'tidefetch-installer' }
    if (-not $release.tag_name) { throw 'Could not determine the latest release.' }
    return $release.tag_name
}

# TLS 1.2 is not the default on older Windows PowerShell hosts.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$arch = Get-TargetArchitecture
$tag = Resolve-Version
$archive = "${Binary}_windows_${arch}.zip"
$base = "https://github.com/$Repo/releases/download/$tag"

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\Tidefetch'
}

Write-Info "Installing $Binary $tag for windows/$arch."

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    $archivePath = Join-Path $tmp $archive

    Write-Info "Downloading $archive"
    Invoke-WebRequest -Uri "$base/$archive" -OutFile $archivePath -UseBasicParsing

    # Verify the download against the checksums published with the same release.
    $sumsPath = Join-Path $tmp 'checksums.txt'
    try {
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumsPath -UseBasicParsing
    } catch {
        $sumsPath = $null
        Write-Warn "checksums.txt not published for $tag; skipping verification."
    }

    if ($sumsPath) {
        $line = Get-Content $sumsPath | Where-Object { $_ -match "(\*)?$([regex]::Escape($archive))$" } | Select-Object -First 1
        if (-not $line) { throw "No checksum entry for $archive." }
        $expected = ($line -split '\s+')[0]

        $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash
        if ($expected.ToLower() -ne $actual.ToLower()) {
            throw "Checksum mismatch for ${archive}: expected $expected, got $actual."
        }
        Write-Info 'Checksum verified.'
    }

    Expand-Archive -Path $archivePath -DestinationPath $tmp -Force

    $source = Join-Path $tmp "$Binary.exe"
    if (-not (Test-Path $source)) { throw "Archive did not contain $Binary.exe." }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $source -Destination (Join-Path $InstallDir "$Binary.exe") -Force

    Write-Info "Installed to $InstallDir\$Binary.exe"

    # Persist the directory on the user's PATH if it is not already there.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -notlike "*$InstallDir*") {
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Info "Added $InstallDir to your user PATH."
        Write-Warn 'Open a new terminal for the PATH change to take effect.'
    }
    $env:Path = "$env:Path;$InstallDir"

    if (-not (Get-Command aria2c -ErrorAction SilentlyContinue)) {
        Write-Info 'Installing the aria2 engine...'
        $installed = $false

        foreach ($mgr in @(
            @{ Name = 'winget'; Args = @('install', '--id', 'aria2.aria2', '--exact', '--silent',
                                         '--accept-package-agreements', '--accept-source-agreements') },
            @{ Name = 'scoop'; Args = @('install', 'aria2') }
        )) {
            if (-not (Get-Command $mgr.Name -ErrorAction SilentlyContinue)) { continue }
            try {
                & $mgr.Name @($mgr.Args) | Out-Null
                if ($LASTEXITCODE -eq 0) { $installed = $true; break }
            } catch {
                # Fall through to the next manager.
            }
        }

        if ($installed) {
            Write-Info 'aria2 installed.'
            Write-Warn 'Open a new terminal so aria2c is on your PATH.'
        } else {
            Write-Warn @'
Could not install aria2 automatically. Tidefetch drives the aria2 engine, so install it manually:
  winget install aria2.aria2
'@
        }
    }

    Write-Host ''
    Write-Host "Run `"$Binary doctor`" to verify your setup, or just `"$Binary`" to start." -ForegroundColor DarkGray
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
