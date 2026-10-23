# Configuration

A config is a YAML file. Complete examples are in the repository's [`examples`](https://github.com/davidp57/process-monitor/tree/main/examples) folder.

## Every key {#keys}

```yaml
name: dcs                       # session and lock name; default: the file name
lang: en                        # pm's language: fr or en (see Installation)
vars:                           # variables, usable anywhere as ${name}
  saved: ${env:USERPROFILE}\Saved Games\DCS
  prof: ${saved}\Profiles\${profile}

command: C:\jeux\DCS World\bin\dcs_updater.exe
args: []
workdir: ${prof}
env: { KEY: value }             # added to pm's environment
track: dcs.exe                  # process to follow instead of the launched one
kill_on_exit: [SR-ClientRadio.exe]  # stopped once the followed process has exited, wherever they run
if_running: abort               # abort (default) or kill
capture: false                  # program output into the log (and no console window)
ui: console                     # console (default) or tray: the notification icon
icon: Icons\dcs.ico             # image of the notification icon; per profile too
ready: { file: '${saved}\dcs.lock', timeout: 3m }
backup: true                    # keeps a .bak of every overwritten file (default)

hooks:
  sync:                         # "profile <-> live": copied before, copied back after
    - ${prof}\options.lua <-> ${saved}\Config\options.lua
  before:
    - copy: { from: a.txt, to: b.txt }
    - delete: ${prof}\cache     # file or folder; missing = nothing to do
    - exec: [cmd, /c, echo, ok]
  after: []
  timeout: 10m                  # longest an exec command may run (default)

profiles:                       # pm run ... --profile DEV
  DEV:
    vars: { other: value }      # replaces or adds variables
    icon: Icons\dcs-dev.ico     # replaces the icon
    hooks:                      # added to the common hooks
      sync: [ '${prof}\joy.lua <-> ${saved}\Input\joy.lua' ]
  VR: {}

log: { dir: '${config_dir}\logs', keep: 5 }
watch: { restart: always, delay: 2s, max_delay: 1m, max_restarts: 0 }
exit: { delay: 0s, wait_key: on-error }   # never | on-error | always
```

## Rules {#rules}

- **Variables**: `${name}` reads `vars`, which may refer to each other. Also available: `${profile}` (when there are profiles), `${name}`, `${config_dir}` (the config file's folder) and `${env:NAME}`. An unknown variable is an error.
- **Unknown keys**: a typo in a key is an error, never an ignored option.
- **Paths**: between single quotes `'...'` YAML leaves `\` alone; between double quotes `"..."`, they must be doubled.
- **Relative paths**: resolved from the config's folder. For `command` and an `exec` program, a bare name (`cmd`) is looked up in `PATH`, a path (`tools\app.exe`) starts from the config's folder. An `exec` command runs in the config's folder; the launched program starts in `workdir`, or else the current folder.
- **`track`** and **`kill_on_exit`**: process names alone, extension included (`dcs.exe`).
- **`sync`**: a file may appear in one line only, on the profile side as on the live side.
- **One document**: a `---` that would cut the file in two is an error.
- **Order**: before = `sync` (in) then `before`; after = `sync` (back) then `after`.
- **ready**: pm deletes the old file before the launch, then reports when it shows up again, or that it did not within the timeout. It is information: the end of the session depends on the followed process.
