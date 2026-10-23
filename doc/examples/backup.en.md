# Example: back up before starting

**The need**: before every game, keep a copy of a game's saves, in case a game damages them, without piling up archives forever.

**What Process Monitor does**:

1. before the launch, it creates a dated `.zip` archive of the saves folder;
2. it starts the game and waits for it to close;
3. afterwards, it keeps only the 10 most recent archives.

If the archive cannot be created, the game is not started: no playing without a safety net.

## The config {#config}

The comments in the file are in French; the lines are explained below.

```yaml
--8<-- "examples/backup.yaml"
```

- The `exec` actions run PowerShell here. In a YAML single-quoted string, a quote is doubled (`''`); `${archives}` is replaced by Process Monitor before running.
- `timeout: 5m` stops a command that does not return.
- There is no `track`: the game starts directly, without a launcher.

## Run {#run}

```powershell
.\pm.exe check .\backup.yaml
.\pm.exe run .\backup.yaml
```

`pm check` shows both commands as they will run, variables replaced.

!!! note "A unique program name"
    Process Monitor refuses to start when a program with the same name already runs. For a game, that is the guard you want; but a program with a very common name (`cmd.exe`, `powershell.exe`) would nearly always be "already running". See [Troubleshooting](../troubleshooting.md#already-running).
