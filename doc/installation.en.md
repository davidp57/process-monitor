# Installation

pm runs on Windows. It is a single file, `pm.exe`, that needs nothing else.

## Download {#download}

Every published version is on the repository's [Releases](https://github.com/davidp57/process-monitor/releases) page: download `pm.exe` and put it in a folder, next to your configs for instance.

## The menu {#menu}

A double-click on `pm.exe` opens a menu: create a config with the [wizard](new.md), create its [shortcuts](shortcut.md), check it, run it.
No command line to type; started without arguments from a terminal, `pm.exe` shows its help as before.

## Build it yourself {#build}

You need Go 1.27 or newer (`scoop install go`), then, in the repository folder:

```powershell
go build -o dist\pm.exe .\cmd\pm
```

A local build shows the version `dev` (`.\pm.exe version`).

## Start from a shortcut {#shortcut}

A Windows shortcut is the simplest way to start a session.
[`pm shortcut`](shortcut.md) creates them from the config; by hand, these are the settings:

- **Target**: the path of `pm.exe`, followed by the arguments, for example `run "C:\...\dcs.yaml" --profile DEV --tray`;
- **Start in**: the profile's folder, or the config's;
- **Run**: "Minimized", when the session uses the [notification icon](tray.md): the console Windows opens for the shortcut then does not even show.

## Language {#language}

pm speaks French and English.
It takes the first language set among: the `--lang fr|en` option, the `PM_LANG` environment variable, the config's `lang:` key, the Windows display language; failing that, English.
