# The notification icon

`--tray`, or `ui: tray` in the config, replaces the window with an icon in the notification area, for `pm run` as for `pm watch`.

Started from a shortcut, pm gives its console window back at once (set the shortcut to "Minimized" so it does not even show).
Started from a terminal, it keeps writing to it, and the double Ctrl+C stays active.

## What the icon shows {#display}

- **Colour**: green while the program runs, orange during the copies (at the start and at the end) or before a restart, grey when paused, red after an error.
- **Tooltip**: the session, the profile and the state.
- **Image**: `--icon FILE`, or `icon:` in the config, or per profile in `profiles:`, replaces the disc with your image (`.png`, or `.ico` in 32-bit or holding a PNG), with a small dot of the state's colour in its bottom-right corner. An unreadable image is reported in the log, and pm keeps the disc.

## The menu {#menu}

- **Open the log**.
- **Stop and finish**: stops the program, then runs the *after* actions.
- **Detach…**: after a confirmation that says what it costs at that moment (cancel at start, end during a restart delay, otherwise leave the program without its copy-back). If the session changes state while the question is shown, it is asked again.
- In `watch`: **Pause**, **Resume**, **Restart**.

## The end of the session {#end}

- On failure (an error, or a non-zero exit code in `pm run`), the icon turns red and a Windows message box says what failed and where the log is.
- A cancellation you asked for is not a failure: no error box.
- Otherwise the icon keeps its final state for `exit.delay`, then goes away; with `exit.wait_key: always`, a "Session over" box shows.

`exec` commands do not open in a console window, nor does the launched program once pm has no console.
