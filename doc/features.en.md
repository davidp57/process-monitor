# Features

Each feature is set in the configuration file (YAML). The snippets below combine in one file; the [Configuration](configuration.md) page lists every key, and the [examples](examples/dcs.md) show complete configs.

## The program to run {#program}

```yaml
command: C:\jeux\DCS World\bin\dcs_updater.exe   # the program
args: [--quiet]                                  # its arguments (optional)
workdir: C:\Users\me\Saved Games\DCS             # its working folder (optional)
env: { MY_VARIABLE: value }                      # extra environment variables (optional)
```

A bare name (`notepad`) is looked up in `PATH`; a relative path starts from the config's folder.

## Profiles and synced files {#sync}

**When**: one program, several sets of settings.

A `sync` line links a file of the profile (left) and the file the program really uses, the "live" file (right). At start, the profile's file is copied over the live one; at the end, the live file is copied back into the profile, with the changes made during the session.

```yaml
vars:
  saved: ${env:USERPROFILE}\Saved Games\DCS
  prof: ${saved}\Profiles\${profile}

hooks:
  sync:
    - ${prof}\options.lua <-> ${saved}\Config\options.lua

profiles:
  DEV: {}
  VR: {}
```

- `--profile DEV` picks the profile; `${profile}` is then `DEV`.
- A profile can add its own lines, variables or icon:

    ```yaml
    profiles:
      DEV:
        hooks:
          sync:
            - '${prof}\joystick.diff.lua <-> ${saved}\Config\Input\joystick.diff.lua'
    ```

- Every overwritten file is kept as `.bak`, and a copy is never half-written.

## Before and after actions {#actions}

**When**: to prepare or clean up around the program.

```yaml
hooks:
  before:                       # before the launch, after the sync copies
    - delete: ${prof}\cache     # a file or a folder; missing = nothing to do
    - copy: { from: template.ini, to: '${saved}\config.ini' }
    - exec: [cmd, /c, echo, ready]
  after:                        # after the stop, after the sync copy-back
    - delete: ${prof}\cache
  timeout: 10m                  # longest an exec command may run
```

- If a *before* action fails, nothing is launched, and the live files already replaced by the `sync` lines are put back as they were. The `copy` and `delete` actions already done are not undone (a `copy` leaves the previous file as `.bak`).
- If an *after* action fails, the following ones still run.
- An `exec` command runs in the config's folder, without opening a window; its output goes to the log.

## Following the real program {#track}

**When**: the program goes through a launcher, which starts the real program and exits.

```yaml
command: C:\jeux\DCS World\bin\dcs_updater.exe
track: dcs.exe
```

Process Monitor follows `dcs.exe`, not the launcher: the session lasts as long as the game runs, and the settings are brought back only once the game has closed. The details are on the [How pm follows a program](tracking.md) page.

## Guards {#guards}

**When**: always; they are on by default.

```yaml
name: dcs              # every profile of this config shares this lock
if_running: abort      # abort (default): refuse when the game already runs; kill: stop it first
```

- Two sessions with the same name never run together: starting the VR profile while DEV runs is refused, before any copy.
- If the game, or its launcher, already runs outside Process Monitor, `abort` refuses to start; `kill` stops it, then starts.

## Restarts {#watch}

**When**: a program to keep alive. Start it with `pm watch` instead of `pm run`.

```yaml
watch:
  restart: always       # always | on-failure | never
  delay: 2s             # wait before a restart
  max_delay: 1m         # ceiling, when the program crashes in a loop
  max_restarts: 0       # 0 = no limit
```

The *before* actions (a cleanup) run again before every restart; the `sync` lines are copied at the start and the end only. See [pm watch](watch.md).

## Side programs {#kill-on-exit}

**When**: the program opens others that have no purpose once it is gone.

```yaml
kill_on_exit: [SR-ClientRadio.exe]
```

With `track`, Process Monitor does not wait for these programs before bringing the settings back, and those in the list are stopped once the followed program has exited. Without `track`, the session waits for every program it started, these included: `kill_on_exit` only stops them afterwards, so it is only useful with `track`.

## The notification icon {#tray}

**When**: no console window, or to see the state at a glance.

```yaml
ui: tray                       # or --tray on the command line
icon: Icons\dcs.ico            # optional; or --icon, or per profile
```

The icon is green while the program runs, orange during the copies or before a restart, red after an error; its menu opens the log, stops or detaches. See [The notification icon](tray.md).

## Logs and language {#logs}

```yaml
capture: true                       # the program's output into the log
log: { dir: '${config_dir}\logs', keep: 5 }
lang: en                            # fr or en; --lang and PM_LANG win over it; with none, the Windows language
```

Each session writes its log: steps, copies, errors, and the program's output with `capture`. Only the `keep` most recent are kept.

## End of the session {#exit}

```yaml
exit: { delay: 2s, wait_key: on-error }   # wait_key: never | on-error | always
```

In a console, `wait_key: on-error` keeps the window open on an error so it can be read. With the icon, an error opens a Windows message box describing it.
