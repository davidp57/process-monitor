# pm watch

```powershell
.\pm.exe watch <config.yaml> [--profile NAME] [--tray | --no-tui] [--icon FILE] [--lang fr|en]
```

`pm watch` restarts the program every time it stops, as `watch.restart` says: `always` (default), `on-failure` (only on a non-zero exit code) or `never`.

## Restarts {#restarts}

- A stop less than 10 seconds after the launch counts as a crash loop: the first waits `watch.delay`, each next one doubles the delay, up to `watch.max_delay`.
- `watch.max_restarts` only counts automatic restarts, not those asked with `r`.
- The `sync` lines are copied once at the start and copied back once at the end of the session.
- The other *before* actions (a cleanup, for instance) run again before every restart; the *after* actions only run at the end.
- At the end of each run, pm stops what is left of the process group, so leftovers do not pile up from one run to the next.

## Keys {#keys}

| Key | Effect |
|---|---|
| `p` | pauses / resumes the program (all its processes) |
| `r` | stops and restarts the program |
| `k` | stops the program, runs the *after* actions, quits |
| `q` twice | detaches: pm quits, the program goes on, the *after* actions do **not** run; at start, cancels; during the delay before a restart, ends the session, *after* actions included |
| arrows, PgUp/PgDn | scroll the log |

`--no-tui` replaces the interface with a plain text log, for a service for instance.
`--tray` replaces it with the [notification icon](tray.md).
