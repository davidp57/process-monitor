# pm shortcut: the shortcuts

```powershell
.\pm.exe shortcut <config.yaml> [--profile NAME]
```

`pm shortcut` creates a config's Windows shortcuts: one per profile, or a single one without profiles, each starting `pm.exe` on that config.
With `--profile`, only that profile's.

## The shortcuts section {#section}

The config says which shortcuts it wants:

```yaml
shortcuts:
  mode: run                   # run or watch: the command the shortcut runs (required)
  name: DCS - ${profile}      # shortcut name, without .lnk
  folder: ${start_menu}       # folder to put it in
```

- **`name`**: by default `${name}`, or `${name} - ${profile}` when the config has profiles.
- **`folder`**: one folder, or a list to put each shortcut in several places (`[${desktop}, ${start_menu}]`); by default the config's folder (`${config_dir}`). Two variables give the usual folders: `${desktop}` (the Desktop) and `${start_menu}` (the Start menu). Windows knows them even when the Desktop was moved into OneDrive.
- Without a `shortcuts` section, `pm shortcut` says so and creates nothing. From the [interface](gui.md#configs) or the [terminal menu](installation.md#menu), it offers to add it: two questions (the mode, then one or several folders, ticked), then the section is written at the end of the file, the rest untouched.
  When the config does not say how `pm` shows itself (no `ui` key), the menu asks — notification icon or console window — and adds the line: that is what makes the shortcut open "Minimized".
  The menu also offers an icon to each profile that has none, among the `.ico` files found next to the config: `DEV: {}` becomes `DEV:` followed by an `icon:` line, and nothing else changes.

## The shortcut created {#link}

| Setting | Value |
|---|---|
| Target | the `pm.exe` that ran the command, with `run` or `watch`, the config and `--profile` |
| Start in | the config's folder |
| Run | "Minimized" when the config uses the [notification icon](tray.md) (`ui: tray`), "Normal window" otherwise |
| Icon | the profile's or the config's icon when it is a `.ico`, else the launched program's |

A shortcut of the same name already there is kept as `<name>.lnk.old` before being replaced: to go back, rename the `.old`.

**The taskbar**: Windows lets no program pin a shortcut there without asking. When shortcuts go into the Start menu, `pm shortcut` says how: right-click the shortcut in the Start menu, then "Pin to taskbar".

**Moving `pm.exe`** breaks its shortcuts: running `pm shortcut` from its new place makes them again.
