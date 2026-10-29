# Simple Windows deployment and separate Hebrew guides

User request: support sends a single installer to a customer, who installs and starts using Nimbus without Git, Go, PowerShell or manual copying. Provide separate initial installation instructions for support and everyday instructions for the customer. Existing authorization: continue implementing the agreed Nimbus plan, which included optional Windows installer packaging; this request promotes that packaging into the deliverable.

Use an offline NSIS installer containing the Windows application and local guides. NSIS produces a standard Windows Setup.exe and can be compiled reproducibly on this Linux host; Inno Setup is a valid alternative but would require an unavailable Windows compiler/runner here. MSI enterprise administration is not needed for this support-led single-host workflow.

Install into a fixed administrator-controlled Program Files directory, elevate through UAC, register and start the existing Windows service, configure the existing restricted firewall rule and put a browser shortcut on the desktop/start menu. The service must continue after logout/reboot. The user's book/backups remain in ProgramData and survive upgrades/uninstall. No automatic purge. UI remains loopback-only and phones use the computer's LAN URL on 8088.

One short setup page selects the phone subnet, offering detected IPv4 subnets as suggestions. Choose automatically only when unambiguous; never infer that every connected network is approved. Support may provide /SUBNET=... for unattended installation. Installation with missing/invalid subnet fails clearly before changing an existing install. No change to IP addresses, DHCP, phone provisioning, or customer network profile.

Upgrades must stop and remove only a service pointing to this installation's executable/data directory, copy the new binary, reinstall/start and verify HTTP readiness. Existing configuration and book must be preserved. Keep a temporary old binary for recovery; on upgrade failure restore it and attempt restoring the old service, reporting incomplete recovery. Refuse unsupported external/custom service layouts instead of overwriting them.

Customer guide: open shortcut, add/edit/delete/save, CSV, backups and contact support. Support guide: prerequisites, stable IP/DHCP reservation, allowed subnet, installer run, one-time Yealink XML URL, readiness, restart, upgrade/uninstall and minimal troubleshooting. Include printable RTL HTML companions that open locally without external dependencies.

Build and test helper logic on Linux, cross-build Windows, compile Setup.exe, validate included payload and installer source checks; Windows setup/service/reboot acceptance remains a documented native check unless actually executed. Signing requires a customer-provided trusted signing identity; do not imply the installer is signed.
