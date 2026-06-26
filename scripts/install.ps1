# tokless installer for Windows (PowerShell 5.1+).
# Download this script from a trusted source, inspect it if needed, then run:
#   powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1


$ErrorActionPreference = "Stop"
$Owner = "wallentx"
$Repo  = "tokless"

$asset = "tokless-windows-x64.exe"
$url   = "https://github.com/$Owner/$Repo/releases/latest/download/$asset"
$sumsUrl = "https://github.com/$Owner/$Repo/releases/latest/download/SHA256SUMS"
$destDir = Join-Path $env:LOCALAPPDATA "Programs\tokless"
$dest = Join-Path $destDir "tokless.exe"

New-Item -ItemType Directory -Force -Path $destDir | Out-Null
$tmp = [System.IO.Path]::GetTempFileName()
$sums = [System.IO.Path]::GetTempFileName()
try {
    Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    Invoke-WebRequest -Uri $sumsUrl -OutFile $sums -UseBasicParsing

    $expected = $null
    foreach ($line in Get-Content -Path $sums) {
        $parts = $line -split "\s+"
        if ($parts.Length -ge 2 -and $parts[1] -eq $asset) {
            $expected = $parts[0].ToLowerInvariant()
            break
        }
    }
    if (-not $expected) {
        throw "Checksum file does not list $asset"
    }

    $actual = (Get-FileHash -Path $tmp -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Checksum mismatch for $asset"
    }

    Move-Item -Force -Path $tmp -Destination $dest
} catch {
    Write-Host "✖ Verified download failed ($asset). See https://github.com/$Owner/$Repo/releases" -ForegroundColor Red
    Write-Host "  $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue $tmp, $sums
}

$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
$userPath = ""
if ($null -ne $key.GetValue("Path")) {
    $userPath = $key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
}
$expanded = ($userPath -split ";") | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\") }
if ($expanded -notcontains $destDir.TrimEnd("\")) {
    $newPath = if ($userPath) { "$destDir;$userPath" } else { $destDir }
    $key.SetValue("Path", $newPath, [Microsoft.Win32.RegistryValueKind]::ExpandString)
    $env:Path = "$destDir;$env:Path"
}
$key.Close()

$v = & $dest --version 2>$null
Write-Host "✔ tokless $v ready → $dest" -ForegroundColor Green

if ([Environment]::UserInteractive -and -not $env:CI) {
    Write-Host ""
    & $dest
} else {
    Write-Host "Run: tokless" -ForegroundColor Cyan
}
