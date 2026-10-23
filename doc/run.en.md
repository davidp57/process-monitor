# pm run

```powershell
.\pm.exe run <config.yaml> [--profile NAME] [--tray] [--icon FILE] [--lang fr|en]
```

## How a session unfolds {#flow}

1. **Session lock**: one session per `name`, and one per live file (see [guards](tracking.md#guards)).
2. **Program already running?** pm refuses, or stops it with `if_running: kill`.
3. **Before actions**: the `sync` lines copy the profile's files to the live ones, then the `before` actions run.
4. **Launch**, then **wait for the end** of the followed program (see [how pm follows a program](tracking.md)).
5. **After actions**: the `sync` lines copy the live files back to the profile, then the `after` actions run.

If a *before* action fails, nothing is launched, and the live files already replaced are put back as they were.
If an *after* action fails, the following ones still run, and the error is shown.

Each `exec` command is bounded by `hooks.timeout` (10 minutes by default) and runs in the config's folder; a program it leaves in the background does not hold pm.

## Ctrl+C and the window {#interrupt}

Ctrl+C does not cut at once: a second Ctrl+C within 5 seconds is needed, and the console first says what it will do:

- while the starting files are copied, it **cancels**: nothing is launched, the live files are put back as they were;
- while the program runs, it **detaches** pm: the program goes on without it, and the *after* actions do not run;
- in `watch`, during the delay before a restart, it **ends** the session, *after* actions included.

Closing the console window cuts pm with no confirmation possible: the *after* actions do not happen.
To have no window to close by mistake, use the [notification icon](tray.md).

## Exit codes {#exit-codes}

| Code | Meaning |
|---|---|
| the program's | the session went through; pm returns the followed program's code |
| 1 | pm hit an error, or detached (the copy-back did not happen) |
| 2 | config or command-line error |

## Before running: pm check {#check}

`.\pm.exe check <config.yaml> [--profile NAME]` runs nothing: it shows what would be copied, deleted or run, and flags missing source files.
Run it after every config change.
