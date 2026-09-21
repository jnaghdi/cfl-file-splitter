#Requires -Version 5.1
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = Split-Path -Parent $PSScriptRoot
$oldOS = $env:GOOS; $oldArch = $env:GOARCH; $oldCGO = $env:CGO_ENABLED
function Invoke-Go([string[]] $Arguments) {
    & go @Arguments
    if ($LASTEXITCODE -ne 0) { throw "go failed (exit $LASTEXITCODE): $Arguments" }
}
Push-Location $root
try {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Install a supported Go toolchain and reopen PowerShell.' }
    # Tests execute natively. The distribution targets Windows x64 explicitly.
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    $env:CGO_ENABLED = '0'
    Invoke-Go -Arguments @('version')
    Invoke-Go -Arguments @('test', './...')
    Invoke-Go -Arguments @('vet', './internal/core', './cmd/cflcli')
    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
    $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
    Invoke-Go -Arguments @('build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w', '-o', 'bin/CFL_Splitter_CLI.exe', './cmd/cflcli')
    Invoke-Go -Arguments @('build', '-trimpath', '-buildvcs=false', '-ldflags=-s -w -H=windowsgui', '-o', 'bin/CFL_File_Splitter.exe', './cmd/cflsplit')
    Write-Host 'Built Windows x64 GUI and CLI in bin/. Interactive GUI testing is still required.'
} finally {
    $env:GOOS = $oldOS; $env:GOARCH = $oldArch; $env:CGO_ENABLED = $oldCGO
    Pop-Location
}
