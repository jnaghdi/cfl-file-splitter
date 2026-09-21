#Requires -Version 5.1
[CmdletBinding()]
param([switch] $Streaming)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = Split-Path -Parent $PSScriptRoot
$oldOS = $env:GOOS; $oldArch = $env:GOARCH; $oldUtf8 = $env:PYTHONUTF8
function Invoke-Checked([string] $Program, [string[]] $Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed (exit $LASTEXITCODE): $Arguments" }
}
Push-Location $root
try {
    foreach ($program in @('go', 'python')) {
        if (-not (Get-Command $program -ErrorAction SilentlyContinue)) { throw "Install $program and make it available on PATH." }
    }
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    $env:PYTHONUTF8 = '1'
    Invoke-Checked 'go' @('version')
    Invoke-Checked 'go' @('test', '-count=1', './...')
    Invoke-Checked 'go' @('vet', './internal/core', './cmd/cflcli')
    Invoke-Checked 'python' @('tests/repository_test.py')
    Invoke-Checked 'python' @('tests/cross_language_test.py')
    Invoke-Checked 'python' @('tests/auto_mode_test.py')
    if ($Streaming) { Invoke-Checked 'python' @('tests/streaming_test.py') }
    Write-Host 'Requested automated tests passed. This does not test the interactive GUI or Claude uploads.'
} finally {
    $env:GOOS = $oldOS; $env:GOARCH = $oldArch; $env:PYTHONUTF8 = $oldUtf8
    Pop-Location
}
