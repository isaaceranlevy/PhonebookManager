# Native acceptance only on a disposable elevated Windows machine. Never an existing customer host.
param([string]$Installer = (Join-Path (Split-Path $PSScriptRoot -Parent) 'bin\NimbusPhonebook-Setup-1.1.0.exe'))
$ErrorActionPreference = 'Stop'
$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (!$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Run elevated on a disposable Windows VM.' }
$installDir = Join-Path $env:ProgramW6432 'NimbusPhonebook'
$dataDir = Join-Path $env:ProgramData 'NimbusPhonebook'
if ((Get-Service NimbusPhonebook -ErrorAction SilentlyContinue) -or (Test-Path $installDir) -or (Test-Path $dataDir)) {
    throw 'Refusing to test on a machine with an existing service, program or data directory.'
}
function Assert-True($condition, [string]$message) { if (!$condition) { throw $message } }
function Wait-Ready {
    for ($attempt=0; $attempt -lt 100; $attempt++) {
        try { return Invoke-RestMethod 'http://127.0.0.1:8765/api/info' -TimeoutSec 1 } catch { Start-Sleep -Milliseconds 200 }
    }
    throw 'Admin HTTP did not become ready.'
}
function Run-Setup {
    $p = Start-Process -FilePath $Installer -ArgumentList @('/S','/SUBNET=127.0.0.0/8') -Wait -PassThru
    Assert-True ($p.ExitCode -eq 0) "Installer exit code: $($p.ExitCode)"
    $null = Wait-Ready
    Assert-True ((Get-Service NimbusPhonebook).Status -eq 'Running') 'Service not running.'
}
Run-Setup
$page = (Invoke-WebRequest 'http://127.0.0.1:8765/' -UseBasicParsing).Content
$token = [regex]::Match($page,'name="phonebook-token" content="([^"]+)"').Groups[1].Value
Assert-True ($token.Length -eq 64) 'Missing page token.'
$headers = @{ 'X-Phonebook-Token' = $token }
$body = @{ revision=0; contacts=@(@{id='installer-test';name='Installer & smoke';phones=@('00123','+9720001')}) } | ConvertTo-Json -Depth 5
$null = Invoke-RestMethod 'http://127.0.0.1:8765/api/book' -Method Put -Headers $headers -ContentType 'application/json' -Body $body
$body = @{ revision=1; contacts=@(@{id='installer-test';name='Installer & smoke';phones=@('00123','+9720001')}) } | ConvertTo-Json -Depth 5
$null = Invoke-RestMethod 'http://127.0.0.1:8765/api/book' -Method Put -Headers $headers -ContentType 'application/json' -Body $body
$xml = (Invoke-WebRequest 'http://127.0.0.1:8088/phonebook.xml' -UseBasicParsing).Content
Assert-True ($xml.Contains('00123') -and $xml.Contains('+9720001') -and $xml.Contains('&amp;')) 'XML content failed.'
$before = Get-Content (Join-Path $dataDir 'book.json') -Raw
$backupCount = @(Get-ChildItem (Join-Path $dataDir 'backups') -Filter '*.json').Count
Assert-True ($backupCount -ge 1) 'Backup missing.'
Restart-Service NimbusPhonebook
$null = Wait-Ready
Assert-True ((Invoke-RestMethod 'http://127.0.0.1:8765/api/book').revision -eq 2) 'Restart lost revision.'
Run-Setup
Assert-True ((Get-Content (Join-Path $dataDir 'book.json') -Raw) -eq $before) 'Upgrade changed book data.'
Assert-True (@(Get-ChildItem (Join-Path $dataDir 'backups') -Filter '*.json').Count -eq $backupCount) 'Upgrade changed backups.'
foreach ($path in @($installDir,(Join-Path $installDir 'phonebook.exe'),$dataDir)) {
    $sid=(Get-Acl $path).GetOwner([System.Security.Principal.SecurityIdentifier]).Value
    Assert-True ($sid -eq 'S-1-5-32-544') "Unexpected owner at $path"
}
$uninstaller = Join-Path $installDir 'Uninstall.exe'
$null = Start-Process -FilePath $uninstaller -ArgumentList '/S' -Wait -PassThru
for ($i=0; $i -lt 100; $i++) {
    if (!(Get-Service NimbusPhonebook -ErrorAction SilentlyContinue) -and !(Test-Path (Join-Path $installDir 'phonebook.exe'))) { break }
    Start-Sleep -Milliseconds 200
}
Assert-True (!(Get-Service NimbusPhonebook -ErrorAction SilentlyContinue)) 'Service survived uninstall.'
Assert-True (!(Test-Path (Join-Path $installDir 'phonebook.exe'))) 'Program survived uninstall.'
Assert-True ((Get-Content (Join-Path $dataDir 'book.json') -Raw) -eq $before) 'Uninstall removed/changed data.'
Assert-True (@(Get-ChildItem (Join-Path $dataDir 'backups') -Filter '*.json').Count -eq $backupCount) 'Uninstall removed backups.'
Write-Host 'PASS: native install, service, HTTP/XML, persistence, restart, upgrade, ownership, uninstall retaining data.'
Write-Host 'Temporary test book intentionally remains in ProgramData for inspection. Dispose of the VM after testing.'
