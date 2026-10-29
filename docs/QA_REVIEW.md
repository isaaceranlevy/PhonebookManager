# Independent review — 2026-10-29

One read-only reviewer inspected the complete implementation against the user-provided design and implementation plan, and ran `go test -race ./...`. The repository started empty; the review covered the newly created source rather than an existing commit range.

The review reported five important findings and no critical findings:

1. Conflict instructions recommended export, but export contained the saved book instead of the unsaved draft.
2. Replacing the data-directory DACL did not replace a pre-existing unprivileged owner.
3. Installation rollback ignored Stop failure and did not wait for Stopped before deleting the service.
4. Malformed config.json prevented status and uninstall from reaching SCM.
5. Purge validated only the requested default directory, not the registered service's directory.

All five were fixed with regression evidence. See [verification record](VERIFICATION.md) for exact scope and limits. No second review was requested; the fix pass was verified through regressions, the full race suite, browser acceptance, Windows static analysis and cross-compilation. Real Windows service/ACL execution remains a required manual check.

No minor findings were deferred. The reviewer treated placement of the service executable in an administrator-controlled directory as a documented deployment precondition; the application does not enforce ACLs on the executable's parent directory. Operators must follow that requirement before installing a LocalSystem service.
