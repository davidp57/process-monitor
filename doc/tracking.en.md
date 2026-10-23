# How pm follows a program

The program is launched in a Windows **Job Object**: every process it creates stays attached to it.
Without `track`, the session ends when the last process of the group has stopped.

## Launchers {#launchers}

Many programs go through a launcher: `dcs_updater.exe` starts `dcs.exe` and exits.
`track: dcs.exe` tells pm to follow that process, by its handle, rather than the launcher:

- pm finds it in the group as soon as it is created;
- if it left the group, pm looks for it by name among the processes created after the launch;
- when it stops, pm waits 3 more seconds, in case it hands over to another process of the same name (Windows 11's Notepad does that);
- if the **launcher** still runs (to start the game again after an update, say), pm waits for it and says so in the log;
- the other processes the game started (a radio client such as SRS, an overlay) are **not** waited for: the log names them, and `kill_on_exit` can stop them;
- if it does not appear within 10 seconds after everything else has ended, the session ends in error.

## Guards {#guards}

- **Locks**: a session takes a lock on its `name`, and one per live file of its `sync` lines. Two sessions with the same name, or two configs writing the same live file, cannot run together: the second is refused before any copy. Windows releases the locks when pm exits, even after a crash.
- **Program already running outside pm** (`if_running`), checked on the followed process **and** on the launcher: `abort` (default) refuses, nothing is copied or launched; `kill` stops the existing instances, launcher first, then checks nothing runs any more.
- **Last safety net** (with `track`): if, when the after actions are due, the followed process still runs unfollowed, or the process list cannot be read, the after actions do not run and the session ends in error. A missed copy-back, reported, is better than a profile overwritten while the game runs.
- **Safe copies**: written to a temporary file then renamed, never half-written; the previous file is kept as `.bak`.

## Logs {#logs}

Each session writes `<name>[-<profile>]-<date>-<time>.log` in `log.dir`: pm's steps, the copies, the program's output with `capture`.
Only the `log.keep` most recent are kept.
