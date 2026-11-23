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

The config's `shortcuts` section describes them; one command creates them all:

```powershell
.\pm.exe shortcut .\dcs.yaml
```

Each one has these settings:

- **Target**: `C:\...\pm.exe run "C:\...\dcs.yaml" --profile DEV`
- **Start in**: the config's folder
- **Run**: Minimized
- **Icon**: the same as the profile's (`Icons\dcs-dev.ico`)

A double-click starts DCS with that profile; the icon shows in the notification area and goes away once the settings are back.

## DCS and SteamVR on separate cores {#steamvr}

**The need**: in virtual reality, leave DCS its cores, and run SteamVR on the others.

**Why through Process Monitor**: the `affinity_mask` key of `autoexec.cfg` has no effect in DCS 2.9.31. The `dcs.log` file always reports every core, whatever its value. DCS reads, when it starts, the cores Windows gave its process. They must therefore be set before it is launched, not after: that is what the `affinity` block does, and it replaces `affinity_mask`.

For a processor with 16 logical processors (8 cores with SMT), in the VR profile of the config above:

```yaml
profiles:
  VR:
    icon: ${saved}\Profiles\Icons\dcs-vr.ico
    affinity:
      cores: 0-11
      others:
        cores: 12-15
        processes: [vrserver.exe, vrcompositor.exe, vrmonitor.exe, vrdashboard.exe, vrwebhelper.exe, vrstartup.exe]
```

- DCS, and what it starts, runs on cores 0 to 11; the STD and DEV profiles keep every core.
- SteamVR's processes are held on cores 12 to 15 during the session, then get their cores back. If SteamVR already runs, it is moved before DCS starts.
- The list comes from SteamVR 2.19 with Steam Link. Another headset or runtime may add processes of its own: look for them in the Task Manager, Details tab, during a VR session.
- Keep whole physical cores: `12-15`, not `13-15`.

**Check**: after a launch by Process Monitor, the `process affinity mask:` line at the top of `Saved Games\DCS\Logs\dcs.log` has one digit per core. With `0-11`, it reads `111111111111` (12 digits). Sixteen `1`s mean DCS got every core: it was not launched by Process Monitor, or another program started it (Process Monitor's log then says so).

```powershell
Select-String -Path "$env:USERPROFILE\Saved Games\DCS\Logs\dcs.log" -Pattern "process affinity mask" | Select-Object -First 1
```

For SteamVR, Process Monitor's log notes each process placed (`affinity: vrserver.exe (pid …) placed on cores 12-15`).

**Measure the gain (optional)**: nothing guarantees there is one. On a single-die processor, whose cache all cores share, it will be modest, and DCS loses 4 logical processors. To judge:

1. Replay the same track (`.trk`) twice in VR, once with the `affinity` block, once without.
2. Record frame times for three minutes on each pass, with CapFrameX or fpsVR.
3. Compare the average and the slowest 1 %.
