#Requires -Version 5.1
<#
Creates a NEW PRIVATE GitHub.com repository and pushes the inventoried source.
Requires Git and the official GitHub CLI, already authenticated with gh auth login.
Does not install software, weaken execution policy, overwrite remotes, force-push,
or handle secrets. Inspect the displayed inventory and type CREATE to authorise.
#>
[CmdletBinding()]
param(
    [ValidatePattern('^(?:[A-Za-z0-9][A-Za-z0-9-]*/)?[A-Za-z0-9][A-Za-z0-9_.-]*$')]
    [string] $Repository = 'cfl-file-splitter'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = (Resolve-Path (Split-Path -Parent $PSScriptRoot)).Path
$oldGHHost = $env:GH_HOST
$env:GH_HOST = 'github.com'
function Invoke-Checked([string] $Program, [string[]] $Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed (exit $LASTEXITCODE). Stop and inspect its message; nothing is force-pushed." }
}
Push-Location $root
try {
    foreach ($program in @('git', 'gh')) {
        if (-not (Get-Command $program -ErrorAction SilentlyContinue)) { throw "Install $program through an approved channel, then reopen PowerShell." }
    }
    & gh auth status --hostname github.com
    if ($LASTEXITCODE -ne 0) { throw 'Authenticate first: gh auth login --hostname github.com --git-protocol https --web' }
    $accountRaw = & gh api --hostname github.com user
    if ($LASTEXITCODE -ne 0) { throw 'Could not read the active GitHub account.' }
    $account = ($accountRaw -join "`n") | ConvertFrom-Json
    $login = [string] $account.login
    $id = [string] $account.id
    if ($login -notmatch '^[A-Za-z0-9-]+$' -or $id -notmatch '^\d+$') { throw 'Unexpected account identity returned by GitHub.' }
    $target = if ($Repository.Contains('/')) { $Repository } else { "$login/$Repository" }

    # Refuse a remote that already exists. A permission/network failure is not
    # treated as permission to modify it; gh repo create must itself succeed.
    $oldPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & gh repo view "github.com/$target" --json nameWithOwner 2>$null | Out-Null
    $existsCode = $LASTEXITCODE
    $ErrorActionPreference = $oldPreference
    if ($existsCode -eq 0) { throw "Repository $target already exists. This script will not modify it." }

    $inventory = Join-Path $root 'SOURCE_SHA256SUMS.txt'
    if (-not (Test-Path -LiteralPath $inventory -PathType Leaf)) { throw 'Missing SOURCE_SHA256SUMS.txt.' }
    $paths = New-Object 'System.Collections.Generic.List[string]'
    $seen = @{}
    $allowedTop = @('README.md','VERSION','go.mod','go.sum','LICENSE','CHANGELOG.md','SECURITY.md','CONTRIBUTING.md','CLAUDE.md','THIRD_PARTY_NOTICES.md','.gitignore','.gitattributes','.editorconfig')
    $allowedDirs = @('cmd','internal','tools','tests','scripts','examples','docs','third-party','.github')
    foreach ($line in [System.IO.File]::ReadAllLines($inventory)) {
        if ([string]::IsNullOrWhiteSpace($line) -or $line.StartsWith('#')) { continue }
        if ($line -notmatch '^([0-9a-f]{64})  (.+)$') { throw 'Malformed source inventory.' }
        $expected = $Matches[1]; $relative = $Matches[2]
        $segments = $relative.Split('/')
        if ($relative -match '[\\:\x00\r\n]' -or $relative.StartsWith('/') -or $segments -contains '..' -or $segments -contains '.' -or $segments -contains '') {
            throw "Unsafe inventory path: $relative"
        }
        if (($allowedTop -notcontains $relative) -and ($allowedDirs -notcontains $segments[0])) { throw "File is not in source allowlist: $relative" }
        if ($seen.ContainsKey($relative)) { throw "Duplicate inventory path: $relative" }
        $seen[$relative] = $true
        $full = [System.IO.Path]::GetFullPath((Join-Path $root $relative))
        $rootPrefix = $root.TrimEnd([System.IO.Path]::DirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
        if (-not $full.StartsWith($rootPrefix, [System.StringComparison]::OrdinalIgnoreCase)) { throw "Path escapes repository: $relative" }
        $cursor = $root
        foreach ($segment in $segments) {
            $cursor = Join-Path $cursor $segment
            $item = Get-Item -LiteralPath $cursor -Force
            if (($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -ne 0) { throw "Symlink/reparse point rejected: $relative" }
        }
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) { throw "Missing source file: $relative" }
        if ((Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) {
            throw "Source hash changed: $relative. Review changes and regenerate inventory with python scripts/source_manifest.py --write."
        }
        $paths.Add($relative)
    }
    if ($paths.Count -lt 10) { throw 'Incomplete source inventory.' }
    $paths.Add('SOURCE_SHA256SUMS.txt')

    if (-not (Test-Path -LiteralPath (Join-Path $root '.git'))) {
        Invoke-Checked 'git' @('init', '-b', 'main')
    }
    $top = & git rev-parse --show-toplevel
    if ($LASTEXITCODE -ne 0 -or [System.IO.Path]::GetFullPath([string] $top) -ne $root) { throw 'Not the expected repository root.' }
    $branch = & git branch --show-current
    if ($LASTEXITCODE -ne 0 -or $branch -ne 'main') { throw 'Expected the main branch. Review your existing checkout before publishing.' }
    $remotes = @(& git remote)
    if ($LASTEXITCODE -ne 0 -or $remotes.Count -gt 0) { throw 'This script only handles a new repository with no existing remotes.' }
    $staged = @(& git diff --cached --name-only)
    if ($LASTEXITCODE -ne 0 -or $staged.Count -gt 0) { throw 'Existing staged changes found. Review them before using this script.' }
    # Do not push hidden historical content from an unrelated existing repository.
    $oldPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & git rev-parse --verify HEAD 2>$null | Out-Null
    $headExists = $LASTEXITCODE -eq 0
    $ErrorActionPreference = $oldPreference
    if ($headExists) { throw 'Existing commit history found. Use a fresh source ZIP or manually review history before publishing.' }

    Write-Host "`nGitHub account: $login"
    Write-Host "Target:         $target"
    Write-Host 'Visibility:     PRIVATE'
    Write-Host ("Verified source files: {0}" -f ($paths.Count - 1))
    Write-Host 'Only the following source files and the inventory will be staged:'
    foreach ($relative in $paths) { Write-Host "  $relative" }
    Write-Host 'No EXEs, build output, tokens or client evidence are intentionally included.'
    Write-Host 'You remain responsible for reviewing the source and fictional fixtures.'
    if ((Read-Host "Type CREATE to create and push $target privately") -cne 'CREATE') {
        Write-Host 'Cancelled. No remote repository was created.'
        return
    }

    # Scope identity changes to this newly initialised repository only.
    $configuredName = & git config user.name
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace([string] $configuredName)) {
        Invoke-Checked 'git' @('config', '--local', 'user.name', $login)
    }
    $configuredEmail = & git config user.email
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace([string] $configuredEmail)) {
        Invoke-Checked 'git' @('config', '--local', 'user.email', "${id}+${login}@users.noreply.github.com")
    }
    # Explicit paths rather than git add .; ignored files must never be forced.
    foreach ($relative in $paths) { Invoke-Checked 'git' @('--literal-pathspecs', 'add', '--', $relative) }
    Invoke-Checked 'git' @('diff', '--cached', '--stat')
    Invoke-Checked 'git' @('commit', '-m', 'CFL FileSplitter For Uploading Large Files To Claude 1.0.2 Windows MIT source repository')
    # The user has already explicitly authorised this remote write above.
    Invoke-Checked 'gh' @('repo', 'create', $target, '--private', '--source', '.', '--remote', 'origin', '--push', '--disable-wiki', '--description', 'Windows file splitter, verifier and joiner with SHA-256-verified self-identifying parts.')
    Write-Host "`nRepository created and source pushed:"
    Invoke-Checked 'gh' @('repo', 'view', "github.com/$target", '--json', 'url', '--jq', '.url')
    Write-Host 'Check the real Actions results. The project uses MIT; repository visibility and release publication remain owner decisions.'
} finally {
    $env:GH_HOST = $oldGHHost
    Pop-Location
}
