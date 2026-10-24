# The interface

```powershell
.\pm.exe gui
```

A double-click on `pm.exe` does the same: `pm` opens its interface in the default browser.
It is a page served by `pm` itself on the machine (`127.0.0.1`), with nothing to install.

A **pm — interface** icon shows in the notification area: "Open the interface" opens the page again, "Quit the interface" stops it.
When no page of the interface has been open for 30 seconds, it stops by itself, unless a wizard is under way: its game runs in the interface, and its original settings must be put back.
The sessions running always go on when the interface stops.
A second double-click on `pm.exe` opens the page of the interface already running, or starts a new one.
When the interface running comes from another `pm.exe` (a previous version, or a copy elsewhere), it stops and the new one takes its place, unless a wizard is under way in it.

Top right:

- the light or dark theme, kept by the browser;
- the **settings**: `pm`'s **language** (French, English, or Windows'), which holds for the interface, the menus and every config without a `lang:` key (see [language](installation.md#language)), and the folders to look for configs in;
- the keyboard shortcuts: `1`, `2`, `3` for the tabs, `/` to filter the configs, `Enter` to answer a wizard question, `Esc` to close a window;
- a link to this documentation.

Beside every field that takes a path, "Browse…" opens the Windows window to choose the file or the folder: a program, an icon, a config, a folder, or "Save as" for a config to create.
`pm` opens it, since a web page never learns where a file given to it lives; if it does not show, it may be behind the browser.

## Sessions {#sessions}

Every `pm run` and `pm watch` session running, wherever it was started (a shortcut, a terminal, the interface): its state, its PID, its log live.
The buttons do what the [notification icon](tray.md) does: pause, resume, restart (for `pm watch`), **stop and finish** (the program is stopped, then the after actions run), **detach** (`pm` stops, the program goes on, the after actions do not run).

## Configs {#configs}

The interface lists:

- the configs `pm` remembers: each is noted as soon as `pm` runs it, checks it, makes its shortcuts or creates it with the wizard;
- the `.yaml` files of the folders given to it, and of those it looks into by default: the folder of `pm.exe` and the one above, `Saved Games\DCS\Profiles` (and `DCS.openbeta`), and each folder of `Documents\pm`, where the wizard puts configs (one level; a `.yaml` that `pm` cannot read is not shown).

A config or a folder can be added by its path, and a config removed from the list: the file itself is never touched.

The list is on the left; the config chosen shows on the right.
At the top, **Launch**: choose the profile and the mode (`run` or `watch`), and a `pm` starts on its own, as from a shortcut.
The profile chosen also holds for the tabs below:

| Tab | What it shows |
|---|---|
| Overview | what the session would do, as [`pm check`](run.md#check), drawn: the profile's files on the left, the program's on the right, linked by an arrow per pair (copied before the session, copied back after; a missing source in red), then the steps before, during and after; the text of `pm check` unfolds below |
| Shortcuts | creates the shortcuts; when the config does not say yet, asks the mode, the folders, the display and an icon per profile (see [pm shortcut](shortcut.md)) |
| Logs | the profile's logs, read in the page |
| Edit | edits the config through forms |

"Forget" removes the config from the list.
A config `pm` refuses is marked invalid, with the message saying why, and can be fixed in "Edit".

## Editing a config {#edit}

Every key of the [configuration](configuration.md) has its field, grouped by theme: program, session and display, variables, files and actions, shortcuts, keep alive, ready, logs, profiles.
The keys the forms do not show are kept as they are.

On saving, `pm` reads the new config with every profile: if it refuses it, nothing is written and the error shows.
Otherwise the previous version is kept as `.bak` beside it.
The file's comments are kept; its layout may change (quotes, alignment).

## New config {#wizard}

The [wizard](new.md) of `pm new`, in the page: the same questions, the same steps.
The answers already given stay shown beside it; "Back" returns to the last one.
For a game with profiles, the game starts on "Start the game", and its log scrolls during the game.
In each profile's report, "See what changed" shows, for a text file, the lines removed and added against the original; for a binary or large file, its size before and after.

"Give up" puts the original settings back and writes nothing.

## Security {#security}

The page is served on `127.0.0.1` only, and only to the browser `pm` opened: the address opened carries a secret token, which the page then keeps in a cookie no other site gets.
Another program on the machine, or a website open in the browser, can neither read the interface nor drive the sessions.
Each session listens the same way on `127.0.0.1`, with its own token, written in `%LOCALAPPDATA%\pm\locks`.
