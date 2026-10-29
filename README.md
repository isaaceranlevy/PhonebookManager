# Nimbus Phonebook

A local, Hebrew RTL phonebook manager for Windows and Yealink desk phones. A single Go executable embeds the UI, persists JSON transactionally, and exposes a separate read-only XML listener.

- **Admin:** `127.0.0.1:8765` (loopback only, Host/Origin/CSRF protection).
- **Phones:** `0.0.0.0:8088/phonebook.xml` (GET/HEAD only, UTF-8, ETag/Last-Modified).
- **Storage:** `%ProgramData%\NimbusPhonebook\` on Windows; `./data` on other platforms. JSON is authoritative; XML is generated in memory.
- **Dependencies:** Go 1.24+; only pinned `golang.org/x/sys` and `golang.org/x/text`. No Node build, CDN, database or external service.

## Develop and build

```sh
go mod download
go test -race ./...
go vet ./...
go run ./cmd/phonebook run --data-dir ./data
# Visit http://127.0.0.1:8765 on the SAME computer.
go build -trimpath -o bin/phonebook ./cmd/phonebook
GOOS=windows GOARCH=amd64 go build -trimpath -o bin/phonebook.exe ./cmd/phonebook
```

PowerShell: `$env:GOOS='windows'; $env:GOARCH='amd64'; go build -trimpath -o bin/phonebook.exe ./cmd/phonebook`.

## Simple installer (1.1)

Support distributes **NimbusPhonebook-Setup-1.1.0.exe**, an offline Hebrew Windows installer; customers need no Go, Git or commands. It installs in 64-bit Program Files, requests UAC, starts the automatic service, creates browser/guide shortcuts and retains ProgramData on upgrade/uninstall. The one network page proposes detected subnets and requires an explicit approved selection; it never silently approves all connected networks. Support configures the phones once.

- [Initial installation for support (Hebrew)](packaging/INSTALL_SUPPORT_HE.md) · [printable PDF](packaging/INSTALL_SUPPORT_HE.pdf)
- [Everyday customer guide (Hebrew)](packaging/USER_GUIDE_HE.md) · [printable PDF](packaging/USER_GUIDE_HE.pdf)
- [Installer validation and native acceptance status](docs/INSTALLER_VERIFICATION.md)

The compiled installer and SHA-256 are distributed under `releases/` so authenticated users can download from GitHub without cloud-workspace links. It is **unsigned**; an organization that requires signed software must provide a code-signing process before customer rollout.

Developer build (Go + NSIS 3 Unicode; no new application dependencies): `scripts/build-installer.sh` on Linux, or `powershell -File scripts/build-installer.ps1` on Windows. `MAKENSIS`/`-Makensis` select the compiler. To regenerate local HTML guides run `python3 scripts/render-guides.py` (standard library only).

Native acceptance on a **disposable elevated Windows VM**, never an existing customer installation: `powershell -File tests/windows-installer-smoke.ps1`. It refuses existing program/data/service paths, then exercises install, service/HTTP/XML, persistence, restart, upgrade and uninstall retaining data. A real reboot remains a separate manual gate.

## Windows usage

Move `phonebook.exe` into a stable administrator-controlled directory (for example `C:\Program Files\NimbusPhonebook`) before installing; the service runs that exact path. Run an **elevated** terminal:

```powershell
.\phonebook.exe run
# Or install and start an automatic LocalSystem service:
.\phonebook.exe install --subnet 192.168.1.0/24
.\phonebook.exe status
.\phonebook.exe uninstall
# Explicit irreversible data deletion, default data directory only:
.\phonebook.exe uninstall --purge
```

Use the actual permitted subnet, not the example. Install sets restart recovery and a program/port/subnet firewall rule for Domain/Private profiles. Installation requires an explicit IPv4 CIDR of /8 or narrower. Existing firewall rules remain the administrator's responsibility. The UI itself does not require elevation; file access and service operations do. Run/installation secure the data directory to SYSTEM and Administrators through Windows ACL APIs using SIDs, independent of system language. The owner is set to Administrators as well as replacing the DACL, so a pre-existing unprivileged owner is not retained. All users on a multi-user/RDS host can still reach loopback administration. This is a trusted-computer tool, not a user-authenticated system.

`serve-fixture` uses an isolated temporary book (never your production data):

```powershell
.\phonebook.exe serve-fixture
.\phonebook.exe serve-fixture --format phonebook
```

It exposes both standard listeners while running. Stop with Ctrl+C. To test over LAN, a Windows administrator must separately permit the fixture executable/port for the test subnet. Confirm real Hebrew rendering and dialing on your device before rollout. If the service is already running, stop it before running fixture mode because the ports are shared.

## Configuration

Stop the process/service before editing `config.json` in the data directory. Missing fields use defaults; invalid settings fail startup. For service restart use `Restart-Service NimbusPhonebook` in an elevated PowerShell session. Example:

```json
{
  "adminPort": 8765,
  "phonePort": 8088,
  "format": "directory",
  "allowedSubnet": "192.168.1.0/24",
  "limits": {
    "contacts": 1000,
    "name": 64,
    "number": 32,
    "jsonBytes": 1048576,
    "csvBytes": 2097152
  }
}
```

`format` is `directory` (`YealinkIPPhoneDirectory`, default) or `phonebook` (`YealinkIPPhoneBook`). `allowedSubnet`, when set, adds an application-level source-address check on the phone listener. If config.json already exists, installation requires its allowedSubnet to match `--subnet` exactly; a missing config is created during install. Changing ports/subnets requires updating the firewall rule too. IPv6 and HTTPS are outside v1; the customer must approve LAN HTTP.

## CSV

Exact header: `name,phone1,phone2,phone3`. Quoting follows RFC CSV. UTF-8 (with/without BOM) and fallback Windows-1255 are supported. Export is UTF-8 with BOM and CRLF. A leading formula marker (`= + - @`) or apostrophe is escaped with one apostrophe; import removes one escape apostrophe. Phones remain strings. When the screen contains unsaved changes, the UI exports the visible draft as `nimbus-phonebook-draft.csv`; otherwise it downloads the saved book. **Opening CSV directly in Excel can remove leading zeroes: import the telephone columns as Text.** Preview does not change storage; merge/replace only changes the draft until Save. Merge skips identical name-and-numbers rows and reports the count.

## API

`GET /api/book`, `PUT /api/book {revision,contacts}`, `GET /api/info`, `GET /api/export.csv`, `GET /api/backups`, `POST /api/restore {id,revision}`, `POST /api/import/preview {data}`. Import `data` is the base64-encoded raw CSV bytes, so cp1255 survives JSON transport; the decoded CSV limit still applies. Contact: `{id,name,phones:["00123","+972..."]}` (up to three). Writes require the page's per-process token in `X-Phonebook-Token`, a same-host Origin if provided, and `application/json`. There is no CORS. Conflicts return 409, validation 400, size limits 413, storage failures 500. A token expires on process restart; refresh the page.

## Durability and recovery

Save serializes writers, validates, builds XML, backs up a valid primary, writes/syncs/closes a same-directory temporary JSON, atomically replaces the primary (Windows `MoveFileEx` with REPLACE_EXISTING/WRITE_THROUGH), then publishes an immutable snapshot. A failed precommit leaves the previous primary and snapshot intact. Backup pruning is postcommit maintenance and cannot turn a committed write into an error response. Ten valid rolling backups are retained; corrupt backup files are excluded from restoration and retained for diagnosis. Linux rename is atomic, but power-loss durability is targeted at the Windows implementation.

Startup recovers the latest valid backup if the primary is corrupt or missing, logs a warning and shows a degraded banner. If evidence of previous data exists but no valid backup remains, startup fails rather than silently resetting the book. Restoring an earlier book uses the current revision and creates a new one. Do not run multiple instances against the same data directory. Do not edit book.json while the service is running. Logs append to `logs/phonebook-<startup UTC date>.log`; operators should archive/prune old logs.

## Acceptance and support

See [Hebrew customer guide](packaging/README_HE.md), [support and 12 acceptance checks](packaging/SUPPORT_HE.md), and [verification record](docs/VERIFICATION.md). Windows service, reboot, actual handset compatibility and customer pilot require hands-on acceptance and are not claimed verified by a cross-build.

Browser smoke: install Python Playwright in your own tooling environment, build `bin/phonebook`, ensure ports 8765/8088 are free, then run `python tests/ui_smoke.py`. It starts and stops an isolated temporary test instance and cleans up its data. Set `UI_BINARY` and `CHROMIUM_PATH` to override executable locations (Chromium defaults to `/usr/bin/chromium`). Screenshots are written under `bin/`.
