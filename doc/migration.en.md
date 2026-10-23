# Migrating from 1.x

Version 2 is a complete Go rewrite of the old Python `process-monitor`; the config format has changed.

## Options {#options}

| 1.x | 2.x |
|---|---|
| `process-monitor.exe --config x.yaml` | `pm run x.yaml` (run once) or `pm watch x.yaml` (restart) |
| `process_name` | `track` (or nothing when it is the launched program) |
| `executable` | `command` |
| `working_dir` | `workdir` |
| `monitor_process: false` / `true` | `pm run` / `pm watch` |
| `kill_existing: true` | `if_running: kill` |
| `lock_file` | `ready.file`, with a timeout |
| mirrored `initial_file_copies` + `final_file_copies` | one `sync` line per file |
| `initial_file_cleanup` / `final_file_cleanup` | `hooks.before` / `hooks.after` with `delete` |
| `capture_output` + `output_log` | `capture` + `log` |
| `log_cleanup_*` | `log.keep` |
| `exit_delay`, `wait_keypress` | `exit.delay`, `exit.wait_key` |
| `interval`, `startup_delay` | removed: pm no longer polls, it waits for the process to end |
| `tui`, `verbose` | `pm watch` has the interface, `pm run` the console; everything goes to the log |

## Changed behaviour {#changes}

- `q` no longer kills the program: it detaches, after a confirmation. `k` stops it.
- The after actions run once (1.x could run them up to three times).
- A program already running is no longer "adopted": pm refuses (or stops it with `if_running: kill`).
- A failing *before* action prevents the launch (1.x logged it and launched anyway).
- `exit.wait_key` also applies to the `pm watch` interface (1.x ignored `wait_keypress` in TUI mode).
- The copy-back waits for the game and its launcher, not for the other processes it started (1.x waited for the game only).
- In `pm watch`, each restart runs the *before* actions again but not the `sync` copies; 1.x also re-copied its `initial_file_copies` at every restart, which overwrote the settings changed during the previous run.
