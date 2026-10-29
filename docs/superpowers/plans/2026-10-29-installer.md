# Simple Windows installer Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans inline; one independent review at completion.

**Goal:** Deliver one offline Windows installer and separate simple Hebrew support/customer guides.
**Architecture:** NSIS wraps safe Go setup helpers and the existing Windows service; JSON data remains in ProgramData.
**Tech Stack:** Go with existing dependencies, NSIS Modern UI, local Markdown/RTL HTML.
**Spec:** ../specs/2026-10-29-installer-design.md

## Global Constraints
- Preserve book/backups/config on upgrade and uninstall. No purge, no silent expansion of allowed networks.
- UAC elevation only at installation; everyday browser shortcut needs none.
- Same loopback admin and LAN-only XML contract. No new runtime dependency.
- Do not publish the cloud environment. Update the same GitHub project as previously authorized.

## Review Focus
- Refuse existing services registered to another executable or custom data directory.
- Invalid or ambiguous subnet must not mutate an existing install.
- Failed upgrades must preserve the existing book and recover the old executable/service.
- Installer must not report success unless service/HTTP readiness passes.
- Uninstall failure must not remove executable from under a running service.

### Task 1: Setup helper commands
Files: internal/deploy/{network.go,deploy.go,*_test.go}, cmd/phonebook/main.go, internal/winsvc/{windows.go,other.go}.
- [x] Test subnet suggestions (multiple adapters, loopback, IPv6), validation and existing config preservation before implementation.
- [x] Implement setup-info, setup-prepare and setup-finish commands. Prepare verifies registration/paths, validates supplied subnet and stops/removes only the owned service. Finish creates minimal missing config, installs/restarts service and verifies loopback /api/info.
- [x] Test registration/upgrade policy through platform-independent helpers and run go test -race ./...; cross-build Windows.

### Task 2: Installer and reproducible build
Files: packaging/phonebook.nsi, scripts/build-installer.sh, tests/installer_contract_test.py.
- [x] Write contract checks for elevation, subnet selection, shortcuts, readiness errors, safe rollback and preserving data.
- [x] Implement Hebrew Modern UI wizard, offline files, old-binary recovery, service setup/removal, safe uninstall.
- [x] Compile with NSIS and verify binary/payload; document native acceptance limits.

### Task 3: Separate customer/support guides and delivery
Files: packaging/INSTALL_SUPPORT_HE.md, packaging/USER_GUIDE_HE.md, packaging/*.html, README.md, docs/INSTALLER_VERIFICATION.md.
- [x] Write short task-oriented documents and printable offline HTML.
- [x] Full race/vet, Windows cross-build, NSIS compile, payload verification and independent code review.
- [x] Commit/push source and deliver Setup.exe through a reliable GitHub download path; record actual Windows execution status honestly.

## Evidence

See ../../INSTALLER_VERIFICATION.md for commands, 39 passing Go tests, 6 packaging checks, browser/runtime validation, independent review fix, payload integrity, and native Windows acceptance still outstanding. The two-guide separation and support-owned phone configuration were confirmed by the user.
