# Example: DCS World with profiles

**The need**: play DCS World with several sets of settings (a profile for the screen, one for virtual reality, one for mission development), without copying them by hand, and without one profile overwriting another.

**What Process Monitor does**:

1. it copies the chosen profile's files (`options.lua`, `export.lua`, control bindings) to their place in `Saved Games\DCS`;
2. it starts `dcs_updater.exe`, then follows `dcs.exe`, which the launcher starts before exiting;
3. once DCS has closed, it also closes SRS if it runs, then copies the files back into the profile, with the settings changed in game;
4. if DCS already runs, or another profile is in use, it refuses to start.

## Prepare the profiles {#profiles}

One folder per profile in `Saved Games\DCS\Profiles`, holding a copy of the files to follow:

```text
Saved Games\DCS\Profiles\
  DEV\  export.lua  options.lua  joystick.diff.lua
  VR\   export.lua  options.lua
  STD\  export.lua  options.lua
  Icons\ dcs-dev.ico  dcs-vr.ico  dcs.ico
```

To create a profile, copy DCS's current files (`Scripts\export.lua`, `Config\options.lua`…) into its folder.

## The config {#config}

The comments in the file are in French; the lines are explained below.

```yaml
--8<-- "examples/dcs.yaml"
```

Line by line:

- `name: dcs`: the three profiles share one lock; two cannot run at once.
- `prof` is the folder of the profile chosen with `--profile`.
- `track: dcs.exe`: the session follows the game, not the launcher.
- `kill_on_exit`: SRS, started by DCS, is closed with it.
- `ready`: the log notes when DCS has finished starting (the `dcs.lock` file appears).
- `sync`: two files common to every profile; DEV adds a third.
- `ui: tray` and a per-profile `icon:`: no console window, an icon in the profile's image.
- `lang: fr`: pm speaks French; set `en`, or remove the line to follow Windows.

## Check, then run {#run}

```powershell
.\pm.exe check .\dcs.yaml --profile DEV
.\pm.exe run .\dcs.yaml --profile DEV
```

`pm check` reports a missing profile file before anything is launched.

## One shortcut per profile {#shortcuts}

- **Target**: `C:\...\pm.exe run "C:\...\dcs.yaml" --profile DEV`
- **Start in**: the profile's folder
- **Run**: Minimized
- **Icon**: the same as the profile's (`Icons\dcs-dev.ico`)

A double-click starts DCS with that profile; the icon shows in the notification area and goes away once the settings are back.
