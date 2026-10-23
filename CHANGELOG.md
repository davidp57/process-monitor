# Changelog

All notable changes to pm are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A pull request adds its entry at the **end** of `[Unreleased]` and never touches the version: the release commit renames the heading and the tag carries the number.

## [Unreleased]

### Changed

- **Rewritten in Go as a single `pm.exe`** (breaking): the Python `process-monitor` 1.x is replaced. Two subcommands split its two jobs: `pm run` (actions before, launch, wait for the end, actions after) and `pm watch` (watchdog with a restart policy and a terminal interface), plus `pm check` to validate a config without running anything. New YAML format with profiles and variables; see the migration table in the README.
- **The program runs in a Windows Job Object** and the real game process is followed by handle (`track`), including through a launcher that hands over, a process that leaves the job, or a same-name successor. Polling by name every half second is gone.
- **Once the game has exited, pm waits for its launcher only**, not for what the game started on the side (a radio client): the copy-back no longer waits for SRS to be closed. See ADR 0002.
- **`pm watch` runs the before actions again before every restart**, as 1.x ran its cleanup; the sync copies happen once per session (1.x also re-copied its initial copies at every restart), and a replay makes no new `.bak`.
- **Cancelling before the launch** (double Ctrl+C or `q`×2 while the files are copied) launches nothing and puts the live files back; the console says what a second Ctrl+C will do.
- **`pm run` exit code**: the program's own code; 1 when pm failed or detached; 2 for a config or usage error. Exec actions run in the config's folder.

### Added

- **`sync` lines** (`profile <-> live`): one line per file, copied in before and back after, replacing the mirrored copy lists.
- **Guards against overwriting a profile**: a session lock per name and per live file, a refusal when the game or its launcher already runs outside pm (`if_running: abort`, or `kill`), live files put back when a before step fails, and no copy-back while the game runs unfollowed.
- **`kill_on_exit`**: processes stopped once the followed one has exited.
- **`hooks.timeout`** on exec actions; atomic copies with a `.bak` of the overwritten file.
- **Notification-area icon** (`--tray`, or `ui: tray` in the config): `pm run` and `pm watch` without a console window. The icon's colour and tooltip follow the session; its menu opens the log, stops, detaches (with a confirmation that says what it costs at that moment) and, in `watch`, pauses and restarts. A failed session ends with a Windows message box (#4).

### Fixed

- The after actions no longer run up to three times.
- `q` no longer kills the program; it detaches after a confirmation, and `k` stops it.

## [1.5.1] and earlier

Python versions; see the git history before the Go rewrite.
