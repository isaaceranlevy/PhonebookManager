param([string]$Makensis = '')
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
if (!$Makensis) {
    $command = Get-Command makensis.exe -ErrorAction SilentlyContinue
    if ($command) { $Makensis = $command.Source }
    else { $Makensis = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe' }
}
if (!(Test-Path $Makensis)) { throw 'Install NSIS 3 Unicode or pass -Makensis with its compiler path.' }
New-Item -ItemType Directory -Force bin | Out-Null
$previousOS = $env:GOOS
$previousArch = $env:GOARCH
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    go build -trimpath -o bin/phonebook.exe ./cmd/phonebook
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
} finally {
    $env:GOOS = $previousOS
    $env:GOARCH = $previousArch
}
# Generated, offline HTML guides are committed; regenerating them is optional.
& $Makensis /V3 packaging/phonebook.nsi
if ($LASTEXITCODE -ne 0) { throw 'NSIS compilation failed.' }
Write-Host 'Built bin\NimbusPhonebook-Setup-1.1.0.exe'
