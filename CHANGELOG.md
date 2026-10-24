# Changelog

All notable changes to pm are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A pull request adds its entry at the **end** of `[Unreleased]` and never touches the version: the release commit renames the heading and the tag carries the number.

## [Unreleased]

### Added

- **`pm new`, a configuration wizard** (#10): a few questions in the terminal, then a working config and its shortcuts, for the four use cases of the docs (DCS by profiles, another game by profiles, keep a program alive, back up before starting). Game profiles are learned by playing: the wizard starts the game for each profile, shows the files that changed as soon as the game closes, and carries the ones kept (DCS's login data `authdata.bin` is never touched); a profile can start from existing files (an older pm config, a folder laid out like the game, or files placed one by one), and a game left running is offered to be stopped; the original settings are put back before each profile and at the end. It never overwrites an existing config.
- **`pm menu`**: what pm can do (create a config, its shortcuts, check it, run it), chosen from a list in the terminal, like `veaf-tools.exe`.
- **`pm shortcut` and the `shortcuts:` section** (#9): one Windows shortcut per profile, pointing at the running `pm.exe`, minimized with `ui: tray`, with the profile's `.ico` or the program's icon; an existing shortcut is kept as `.lnk.old`. New built-in variables `${desktop}` and `${start_menu}` (`desktop` and `start_menu` are now reserved names in `vars`).
- **`pm gui`, the interface** (#11), also opened by a double-click on `pm.exe`: a page served on 127.0.0.1 and opened in the browser, with a notification icon to reopen or quit it; it stops by itself 30 s after its last page is closed, unless a wizard is under way. It lists the sessions running, wherever they were started, with their log live, and drives them as the notification icon does; lists the configs pm remembers and those of folders given to it, and for each launches a profile, checks it, reads its logs, makes its shortcuts, and edits it through forms (saved only if pm loads it, `.bak` kept, comments kept); hosts the wizard, with what each profile changed in each file. Every session now opens a control channel on 127.0.0.1 with its own token; pm remembers the configs it uses in `%APPDATA%\pm\configs.json`, and also lists those next to `pm.exe` and where the wizard puts them. The language can be set in the interface: it comes after the config's `lang:` and before Windows'.

### Fixed

- **Stopping a game from the notification icon no longer reports an error.** The followed process was terminated twice, by the job and then by its own handle while it was dying, and Windows answered "Access is denied"; it is now terminated on its own only when it left the job. And a run stopped on request no longer shows "the program exited with code 1": that code is the one Windows gives a terminated process.

## [2.0.0] — 2026-10-23

### Changed

- **Rewritten in Go as a single `pm.exe`** (breaking): the Python `process-monitor` 1.x is replaced. Two subcommands split its two jobs: `pm run` (actions before, launch, wait for the end, actions after) and `pm watch` (watchdog with a restart policy and a terminal interface), plus `pm check` to validate a config without running anything. New YAML format with profiles and variables; see "Migrating from 1.x" on the documentation site.
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
- **Notification-area icon** (`--tray`, or `ui: tray` in the config): `pm run` and `pm watch` without a console window. The icon's colour and tooltip follow the session; its menu opens the log, stops, detaches (with a confirmation that says what it costs at that moment) and, in `watch`, pauses and restarts. A failed session ends with a Windows message box (#4). A custom image (`--icon`, `icon:`, per profile too; `.png` or `.ico`) gets a state dot in its corner. Exec actions no longer open console windows.
- **French and English**: every message comes from an embedded catalog; the language is `--lang`, else `PM_LANG`, else `lang:` in the config, else the Windows display language, else English (#6).
- **Documentation site** in French and English (MkDocs Material, GitHub Pages, one version per release with mike); the README is now a short bilingual overview pointing to it (#6).
- **Documentation**: an entry page that says what Process Monitor is for, a Features page with the configuration of each feature, and worked examples (DCS World with profiles, SimAppPro, a backup before starting) whose configs are the tested files of `examples/`.

### Fixed

- The after actions no longer run up to three times.
- `q` no longer kills the program; it detaches after a confirmation, and `k` stops it.

## [1.5.1] and earlier

Python versions; see the git history before the Go rewrite.
