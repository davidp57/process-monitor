# pm new: the wizard

```powershell
.\pm.exe new [config.yaml] [--lang fr|en]
```

`pm new` (or a double-click on `pm.exe`, then "create a config") asks a few questions, one at a time, and writes a config that works, with its shortcuts.
The config is written only after the final confirmation, which sums it all up in plain words.
It is the simplest way to start with pm: no YAML to write.

## What it can do {#cases}

| Choice | What you get |
|---|---|
| start DCS World with profiles | a `pm run` config by profiles, set for DCS: launcher, `dcs.exe` followed, SRS stopped at exit if wanted |
| start another game with profiles | the same for any game: give the launcher, the game process if it takes over, and the folders where the game keeps its settings |
| keep a program alive | a `pm watch` config: an instance already running stopped, files to delete before each start, restart delay |
| back up before starting | a dated archive of a folder before each start, keeping the newest |

The common questions come next: the config name (`dcs`, which also names the shortcuts: "dcs - DEV"), the config file to create (suggested, editable) and where the [shortcuts](shortcut.md) go: next to the config, on the Desktop, in the Start menu or in another folder.

`pm new` never changes a config: when the file already exists, it asks for another name.

## Profiles are set up by playing {#learn}

For a game with profiles, the wizard does not ask which files to carry: it finds out.

1. **The profiles**: type their names, one per line (`DEV`, `VR`, `STD`).
   Each profile has its folder next to the config, named after it; the wizard refuses a name whose folder already exists, so it never writes over files it did not make.
2. **The current settings are set aside**: a copy of the game's settings folders (for DCS: `Config` and `Scripts` in Saved Games), in a `.pm-new` folder next to the config.
   For DCS, `Config\authdata.bin` (the login data, which DCS rewrites at every launch) is left alone: never copied, put back or carried.
3. **The profile of the current setup**: a single question, "which profile is the current setup?". That profile takes the settings as they are, without starting the game; "none" has every profile set up by playing.
4. **The other profiles, one after the other**: the original settings are put back, then the wizard asks whether to start from existing files:
    - from an **older pm config** (a profile of `pm-dcs.yaml`, for example): its `sync` lines say where each file goes, and the profile's files are looked for next to the `.yaml` first (a backup moved elsewhere keeps the paths of its old place);
    - from a **folder laid out like the game** (for DCS: `Config\...`, `Scripts\...`): each file goes to the same path;
    - from **files chosen one by one**: for each, choose where it goes among the files of the same name already in the game, the likeliest first (the `Throttle.diff.lua` of an `F-16C_50` folder goes to the F-16's; failing that, the shallowest path, `Config\options.lua` before the presets), or type another path.

   Files outside the watched folders are left out, and the console says so.
   Then the game starts, and the wizard says: *set everything as wanted for this profile, then quit the game*.
   When the game closes, it notes the files changed, created or deleted, by comparing their content, and shows them right away, with readable names for the DCS files ("game options", "controls of …").
   Untick what was not meant to change: this profile keeps its original version.
5. **What is carried**: every file at least one profile changed.
   Each profile keeps its own version; a profile that did not touch it keeps the original.
   A file a profile does not have (a controller added in one profile only) is deleted before that profile starts, so it does not get another profile's.
6. **The confirmation** shows the result in plain words: the files the same in every profile, then each profile's own.
7. **At the end the original settings are put back**: after the wizard, the game is set as before.
8. **One icon per profile**: the wizard offers the `.ico` files found next to the config and in its folders (down to three levels), the one named after the profile first (`dcs-dev.ico` for DEV); or none, or another file. It shows on the shortcuts and in the notification area.

On confirmation the wizard writes the config, fills each profile's folder with its files, creates the shortcuts and removes `.pm-new`.

## Keys {#keys}

| Key | Effect |
|---|---|
| Enter | confirm; on an empty answer, the suggested value |
| ↑ ↓ | choose in a list |
| Space | tick or untick a file |
| Esc | back to the previous question |
| Ctrl+C | give up |

There is no going back past a step that touched the files (setting the settings aside, a game played for a profile).

## If the wizard stops halfway {#interrupted}

Giving up (Ctrl+C on a question) puts the original settings back and writes nothing.
While a game runs for a profile, Ctrl+C has no effect: stopping the wizard would leave the game open, and it would rewrite its files when it closes; quit the game instead.
If the wizard could not put them back (window closed, crash), their copy stays in `.pm-new`: running `pm new` again with the same config file puts them back before anything else.
