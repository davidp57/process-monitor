# Process Monitor

**Process Monitor** runs a program on Windows and frames it: it prepares what the program needs before it starts, watches it while it runs, and tidies up after it when it stops.

It is a single file, `pm.exe`, with nothing to install and no dependency. You start it from a shortcut or a terminal, with a configuration file that describes the program and what to do around it.

## What it is for {#use-cases}

- **Start a game with a profile of settings, and get them back when it closes.**
  Several sets of settings for one game (screen, virtual reality, development…): Process Monitor puts the chosen profile's files in place, starts the game, waits until it has really closed, then copies the settings changed in game back into the profile.
  → [Example: DCS World with profiles](examples/dcs.md)
- **Keep a program alive.**
  A utility that crashes now and then, a server, a companion app: Process Monitor restarts it every time it stops, spacing the restarts out if it crashes in a loop.
  → [Example: SimAppPro](examples/simapppro.md)
- **Clean up before and after.**
  Delete temporary files, empty a cache, run a command or a script before the start and after the stop.
  → [Example: back up before starting](examples/backup.md)
- **Close what the program opened on the side.**
  A radio client, an overlay started by the game: Process Monitor can close them when the game stops (see [Side programs](features.md#kill-on-exit)).
- **Never mix two profiles.**
  If the game already runs, or another profile of the same game is in use, Process Monitor refuses to start rather than overwrite settings.
- **Run quietly.**
  Instead of a console window, an icon in the notification area, with the image of your choice, shows the session's state and lets you stop it.

## What it does, in short {#features}

| Feature | What it brings |
|---|---|
| [Profiles and synced files](features.md#sync) | one line per file: copied at start, brought back at the end |
| [Before and after actions](features.md#actions) | copy, delete, run a command |
| [Following the real program](features.md#track) | wait for the game, not its launcher |
| [Guards](features.md#guards) | refusal when the program already runs, a lock between profiles |
| [Restarts](features.md#watch) | keep a program alive (`pm watch`) |
| [Side programs](features.md#kill-on-exit) | close what the program opened |
| [Notification icon](features.md#tray) | no console window, the state in colour |
| [Logs and language](features.md#logs) | one log per session, in French or English |

Each feature and its configuration are detailed on the [Features](features.md) page; every key, on the [Configuration](configuration.md) page.

## Quick start {#quick-start}

1. [Install](installation.md) `pm.exe`.
2. Run the [wizard](new.md), which asks a few questions and writes the config and its shortcuts:

    ```powershell
    .\pm.exe new
    ```

    Or write the config by hand, starting from an [example](examples/dcs.md).
3. Check it, then run it, in PowerShell:

    ```powershell
    .\pm.exe check .\dcs.yaml --profile DEV
    .\pm.exe run .\dcs.yaml --profile DEV
    ```

The commands: `pm run` runs once with its before and after actions, `pm watch` keeps the program alive, `pm check` shows what would be done without running anything, `pm new` writes a config by asking questions, `pm shortcut` creates its shortcuts.
