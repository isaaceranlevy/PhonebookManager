# Verification record — 2026-10-29

## Delivered workflow

Single Go executable with embedded Hebrew RTL UI, transactional JSON storage and ten rolling backups, two Yealink XML formats, CSV preview/merge/replace/export, independent loopback administration and read-only LAN listeners, and Windows service lifecycle commands. A Windows amd64 distribution is generated under `bin/` (ignored build outputs).

## Executed successfully in this cloud Linux instance

- Official Go 1.27.3 linux/amd64 toolchain archive SHA-256 verified against official release metadata before extraction. Pinned Go dependencies verified with `go mod verify`.
- `go test -race -count=1 -json ./...`: **30 tests passed, 0 failed, 0 skipped tests**. The embedded `web` package has no Go tests; its interactions are covered by Chromium acceptance. The Windows-only security test is excluded by build tags on Linux and is **not counted as passed**.
- `go vet ./...` and `GOOS=windows GOARCH=amd64 go vet ./...`: passed.
- Linux and Windows amd64 builds: passed. `file bin/phonebook.exe` identifies a PE32+ x86-64 Windows console executable. This proves compilation, not execution on Windows.
- `GOOS=windows GOARCH=amd64 go test -c -o bin/winsvc.test.exe ./internal/winsvc`: Windows test binary compiled; **not executed**.
- `node --check web/app.js`: passed.
- `python3 tests/ui_smoke.py`: **passed** in Chromium. Creates its own temporary data directory and cleans up its application process. Verified create, unsaved draft CSV export, save/reload persistence, search, edit, import preview/merge, XML content, backup restore, delete, mobile overflow, actual beforeunload warning cancellation, two independent browser windows causing a real 409 conflict, draft-preserving export after that conflict and explicit reload confirmation. No JavaScript errors recorded.
- `python3 tests/runtime_smoke.py`: **passed** for both `directory` and `phonebook` fixtures. Verified real XML over HTTP, Hebrew/+ and leading zeroes, real LAN interface access to XML, LAN `/api/book` returning 404, refusal of admin connections via all three discovered LAN interfaces, and clean SIGTERM shutdown. This exercises Linux listeners; Windows firewall behavior remains unverified.
- Desktop (1440 px) and mobile (390 px) screenshots inspected. Generated under `bin/desktop.png` and `bin/mobile.png`.

## Independent code review and regression fixes

One separate reviewer inspected the user specification and source, ran the race suite, and reported five important findings. No critical findings. All five were addressed in a single fix pass:

| Finding | Change | Evidence and limit |
|---|---|---|
| Export after conflict lost visible draft | Dirty UI exports the current draft with CSV escaping/BOM; clean UI downloads committed CSV | Browser regression failed before fix, passed after; real two-window 409 checked |
| Existing unprivileged owner survived ACL replacement | Set Administrators owner as well as protected SYSTEM/Administrators DACL recursively | Trusted descriptor policy test passed; native Windows ownership test compiled but awaits elevated Windows execution |
| Installation rollback ignored stop errors | Disable autostart/recovery, wait through pending states, verify Stopped before deletion and report rollback failures | Fake SCM boundary tests exercise real lifecycle policy for pending/running/stopped and rejected Stop; native SCM still manual |
| Corrupt configuration blocked status/uninstall | Dispatch config-independent lifecycle commands before configuration loading | CLI regression passed with malformed config |
| Purge could delete an unrelated default dataset for a custom service | Read the registered SCM command line and validate requested + registered + default directory match before any removal | Directory policy regression passed; SCM integration still manual |

Additional regression tests cover missing-primary recovery, structurally invalid stored books, required PUT fields, config null/unknown fields, valid-but-oversized JSON, content type and phone source subnet rejection.

## Decisions made during implementation

- The supplied plan targets Windows; this host is Linux. Use Go cross-compilation plus native Linux core/browser/runtime checks and explicitly reserve Windows and handset checks for the target hardware.
- Use native Windows security APIs with well-known SIDs rather than language-dependent `icacls` output; protect the owner and DACL. Failure to apply security fails startup/installation. A target Windows acceptance run remains essential.
- Validate and retain ten **valid** rolling backups. Preserve corrupt backup files for diagnosis. Pruning happens after the commit and does not produce a false failed-save response after a successful primary replacement; operators may need to clean stale files if filesystem maintenance fails.
- CSV preview is JSON `{data: base64}` to honor the JSON-only mutation contract while preserving cp1255 bytes. The decoded CSV size limit still applies.
- No optional Inno Setup wrapper, signing, general XML import or separate portable variant. `run` provides console operation.
- Retain the existing checkout and cloud configuration. No worktree, remote push, environment draft change or publication performed for this implementation.

## Required hands-on acceptance, not executed here

1. Elevated Windows run, installation, SCM Running, recovery actions, firewall subnet/profile/program scope, ownership/ACLs and uninstall preserving ProgramData. Run the Windows-only ownership test on an elevated host (`go test ./internal/winsvc`) if Go is available there.
2. Windows reboot with no logged-in user and automatic service availability.
3. Yealink model/firmware compatibility: actual Hebrew rendering, both formats if needed, dialing leading-zero/+ numbers, observed refresh behavior and `features.remote_phonebook.flash_time`.
4. Customer approval for unencrypted LAN HTTP and a customer pilot.

Use the 12-row acceptance matrix and device report template in `packaging/SUPPORT_HE.md`. The executable is unsigned. It has not been installed on a customer machine and has not been tested on a physical telephone.
