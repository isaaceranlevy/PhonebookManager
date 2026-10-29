# Installer 1.1 — verification and handoff (2026-10-29)

## What was requested and delivered

Support sends one offline Windows Setup.exe to the customer. The installer requests elevation once, selects the approved phone subnet, installs a Windows service and creates a browser shortcut. The customer needs no Git, Go, command prompt or separate runtime. Support performs the one-time XML URL configuration on the phones; this division was explicitly confirmed by the user.

Separate Hebrew documents are provided as Markdown, offline printable HTML and PDF:

- `packaging/USER_GUIDE_HE.*`: everyday customer use (PDF: 2 pages).
- `packaging/INSTALL_SUPPORT_HE.*`: initial installation, network prerequisites, phone connection and handoff (PDF: 3 pages).

## Actual verification in this Linux environment

- **39 Go tests passed**, 0 failed and 0 skipped tests in a fresh `go test -race -count=1 -json ./...` run. Existing Windows-only native ACL tests are excluded by build tags and are not counted as passed.
- `go vet ./...`, `GOOS=windows GOARCH=amd64 go vet ./...`, `go mod verify`, and Windows amd64 application compilation passed.
- `GOOS=windows GOARCH=amd64 go test -c` compiled the Windows ACL test executable; not run here.
- **6 installer contract checks passed** (`python3 tests/installer_contract_test.py`). They check packaging invariants; they are source checks, not native installation acceptance.
- Existing Chromium acceptance passed: contacts, persistence/reload, search/edit/delete, CSV import/draft export, backup restore, real two-window conflict and beforeunload warning, mobile layout, no JavaScript errors.
- Both real runtime fixture formats passed: Hebrew/leading-zero/+ XML, LAN XML reachability, phone API 404 isolation, admin LAN refusal and graceful shutdown.
- The setup-info command was actually run and its UTF-16 INI parsed. This host has three IPv4 subnet suggestions; the helper correctly leaves the selection empty instead of silently approving one.
- NSIS **3.11-1** compiled a Unicode Hebrew offline installer successfully, including its UAC elevation manifest and uninstaller. The NSIS compiler/resources were obtained from Debian signed repository metadata in workspace-only caches; no signature/TLS verification was disabled and no system packages were changed.
- The compiled NSIS executable was extracted with 7-Zip and its embedded Windows application, guides and notices were compared byte-for-byte to the build inputs. The final distributed installer SHA-256 is saved in `releases/SHA256SUMS.txt`.
- Both PDFs were rendered in Chromium, their RTL HTML layout visually inspected and PDF page count/text extraction checked.

## Independent review and fix

One independent read-only reviewer ran the race suite and inspected the installer, upgrade/rollback paths and separate guides. It reported one important issue: NSIS's standard `/D=` override could select a user-writable directory for a LocalSystem service executable.

The fix rejects any destination outside the fixed 64-bit Program Files NimbusPhonebook directory in both NSIS and the Go helper. The Windows helper also recursively sets Administrators ownership and a protected DACL on the installation tree: SYSTEM/admins may modify, Users may read/execute the program and guides. The sensitive ProgramData book directory retains its separate SYSTEM/admin-only policy.

Regression evidence: fixed-path and install-ACL policy tests first failed because the protection was absent, then passed after implementation. A Windows-only real ACL test and the native installation acceptance script remain available for a target host. The existing install/rollback checks also refuse an unfinished previous-binary recovery instead of overwriting that recovery copy.

A separate readiness regression demonstrated that an unrelated HTTP 200 must not count as a healthy Nimbus instance; the helper now requires valid Nimbus `/api/info` JSON with a revision and supported XML format. Retry behavior was exercised against real temporary local HTTP servers.

## Data and deployment guarantees implemented

- No book/config/backup deletion on upgrade or uninstall; the installer never calls purge.
- Existing registered executable and data-directory paths must match this installation before its service can be stopped/removed.
- Existing config/subnet must match the selected subnet. Ports, XML format, limits and all data remain unchanged.
- Only the selected subnet is allowed by the existing firewall/handler policy. Multiple detected networks are suggestions and require a choice.
- The old binary is backed up before service changes; failure attempts removal of the new service and restoration/reinstallation of the old binary. Recovery uses legacy `install` so version 1.0 binaries without new setup helper commands are supported. Incomplete recovery is reported and its backup retained.
- Successful completion follows actual service startup and a valid loopback API response. A failed stop blocks uninstall before deleting the executable.
- Program Files is fixed; no arbitrary custom destination is supported by this installer. Existing advanced/custom CLI installations require support-led migration.

## Decisions and practical limits

- NSIS was chosen over the previously optional Inno Setup wrapper because a trusted native compiler is available for this Linux host. Both produce a familiar offline Windows installer; no new application dependency was introduced.
- Direct GitHub API access returned Forbidden. The installer is therefore checked into `releases/` with its SHA-256 so it can be downloaded through the existing authenticated GitHub repository, rather than through a cloud-workspace file link. No GitHub Release API publication is claimed.
- No cloud environment configuration changes/publication were made.
- **The installer is unsigned.** If the customer's organization requires signed executables, use its approved code-signing process before deployment.

## Native Windows acceptance still required

This environment cannot execute a Windows installer/SCM. Compilation, extraction and source tests do not prove native install, upgrade, shortcuts, UAC, firewall or reboot behavior. The user previously reported that the core application and phone XML flow work; this does not replace acceptance of the new installer.

Before sending broadly, run `tests/windows-installer-smoke.ps1` elevated on a disposable Windows VM. It refuses an existing program/data/service installation, then tests silent install, SCM Running, HTTP/XML, save/reload, restart, upgrade preserving data/backups, trusted ownership and uninstall retaining data. The test intentionally leaves its ProgramData fixture for inspection. A real reboot/no-login check and customer network/phone handoff remain manual. The PowerShell test itself has not been executed on Windows here.

No minor review findings were deferred. The installer and guides are ready for the first native pilot, not a claim of completed Windows acceptance.
