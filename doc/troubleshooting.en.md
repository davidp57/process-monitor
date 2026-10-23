# Troubleshooting

## PowerShell and paths {#powershell}

The examples in these docs are written for **PowerShell**, Windows' default terminal.

- **Starting pm**: write `.\pm.exe`, not `pm.exe`. PowerShell does not look for programs in the current folder; `cmd.exe` accepts both forms, so `.\` is the only one that works everywhere.
- **Environment variables**: `$env:USERPROFILE` in PowerShell, `%USERPROFILE%` in `cmd.exe`. Typed in PowerShell, `%USERPROFILE%` reaches pm as is, and pm then looks for a folder called `%USERPROFILE%`:

    ```powershell
    .\pm.exe run "$env:USERPROFILE\Saved Games\DCS\Profiles\pm-dcs.yaml" --profile DEV
    ```

- In the YAML **config**, pm reads the variables itself, with its own syntax: `${env:USERPROFILE}`.

## "another pm session is already active" {#busy}

A session with the same `name`, or writing the same live files, is already running; the message gives its profile.
The lock goes away with the pm process holding it, even after a crash: if it stays, a pm still runs (see the notification icon, or the Task Manager).

## "… already runs outside this session" {#already-running}

The game, or its launcher, was started without pm. pm refuses so as not to overwrite the settings in use: close the program, or set `if_running: kill` if that is what you want.

The check goes by the program's name: a very common name (`cmd.exe`, `powershell.exe`) is nearly always "already running". Start the real program directly: `track` does not get around this refusal, since the launched program is checked too.

## The copy-back did not happen {#no-copy-back}

The log says why: session detached (double Ctrl+C, `q` twice, window closed), or the last safety net (the followed process still ran unfollowed).
The live files have not moved: starting the session again and ending it normally copies the settings back.

## A process the game started stays open {#leftovers}

pm only waits for the game and its launcher; the other processes (a radio client, an overlay) are named in the log.
To stop them at the end, list them in `kill_on_exit`. A program running as administrator cannot be stopped by a pm started without elevation: the log shows "access denied".

## The icon does not show my image {#icon}

pm reads `.png` files and `.ico` files in 32-bit or holding a PNG. A 256-colour `.ico` is refused: the log says so, and pm keeps the coloured disc. Use the matching `.png`, or save the icon again in 32-bit.

## pm speaks the wrong language {#language}

pm follows the Windows display language. To force it: `--lang fr`, the `PM_LANG` variable, or `lang: fr` in the config.
