# Example: keep SimAppPro alive

**The need**: SimAppPro, the software driving simulation devices, must run for the whole gaming session; if it crashes, it must be restarted, and its temporary files pile up.

**What Process Monitor does**:

1. it stops a SimAppPro instance already running, to start from a known state;
2. it cleans the temporary files, then starts SimAppPro;
3. every time it stops, it restarts it, cleaning first; if it crashes in a loop, it spaces the restarts out up to one minute;
4. when the session is stopped, it cleans one last time.

## The config {#config}

The comments in the file are in French; the lines are explained below.

```yaml
--8<-- "examples/simapppro.yaml"
```

- `if_running: kill`: an instance already running is stopped instead of blocking the start.
- `capture: true` and `ELECTRON_NO_ATTACH_CONSOLE`: SimAppPro's output goes to the log, without opening a console.
- `watch`: always restart, 2 seconds after the stop, up to one minute for a crash series.
- `hooks.before` runs again before every restart; `hooks.after` only runs at the end of the session.
- `ui: tray` and `icon:`: SimAppPro's icon in the notification area, with the state dot. If the image file does not exist on your machine, pm says so in the log and keeps the coloured disc.

## Run {#run}

```powershell
.\pm.exe watch .\simapppro.yaml
```

The icon's menu pauses, restarts, stops or detaches. In a console, without `ui: tray`, the same actions are on the `p`, `r`, `k` and `q` keys (see [pm watch](../watch.md)).
