# pm

`pm` is a single Windows executable, with no dependency, that runs a program and frames it.

- **`pm run`** runs actions before a program starts, then others after it exits: copy a profile's files, clean up, run a command.
- **`pm watch`** keeps a program alive (watchdog) and restarts it when it stops.
- **`pm check`** validates a config and shows what would be done, without running anything.

The first two show themselves in a console, a terminal interface, or just an icon in the notification area.

## Quick start {#quick-start}

1. [Install pm](installation.md).
2. Write a config. For example, to start a game with a profile's settings and get them back when it closes:

    ```yaml
    vars:
      saved: ${env:USERPROFILE}\Saved Games\DCS
      prof: ${saved}\Profiles\${profile}
    command: C:\jeux\DCS World\bin\dcs_updater.exe
    track: dcs.exe
    hooks:
      sync:
        - ${prof}\options.lua <-> ${saved}\Config\options.lua
    profiles:
      DEV: {}
      VR: {}
    ```

3. Check it, then run it, in PowerShell:

    ```powershell
    .\pm.exe check .\dcs.yaml --profile DEV
    .\pm.exe run .\dcs.yaml --profile DEV
    ```

## Going further {#next}

- [`pm run`](run.md): how a session unfolds, and its guards.
- [`pm watch`](watch.md): restarts and the terminal interface.
- [The notification icon](tray.md): no console window.
- [How pm follows a program](tracking.md): launchers, the Job Object, what pm waits for.
- [Configuration](configuration.md): every key.
- [Migrating from 1.x](migration.md) and [Troubleshooting](troubleshooting.md).
