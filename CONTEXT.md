# pm — Context

Glossary of the words used in the code, the config and the docs. Definitions only; how it works is in the code and the ADRs.

## Sessions

**Session**:
One run of `pm run` or `pm watch` on one config: lock, before actions, launch (and restarts for watch), after actions.
Identified by its `name`.
_Avoid_: job (that is the Windows object), instance.

**Profile**:
A named set of overrides in one config (`--profile DEV`): variables and extra hooks.
All profiles of a config share one session name, so two profiles of the same game never run together.

**Launcher**:
The program `command` starts. It may be the game itself, or a program that starts the game and exits (`dcs_updater.exe`).

**Followed process** (`track`):
The process whose end ends the session: the game, found by its image name in the session's job, or among processes created after the launch when it left the job.
Without `track`, the session ends when every process of the job has exited.

**Handover**:
The followed process exiting while a process of the same name takes over (a launcher named like its target). pm waits a short grace before concluding the game is over.

**Leftover**:
A process the game started that still runs after it (a radio client). Not waited for; named in the log; stopped if listed in `kill_on_exit`. In `pm watch`, every leftover of the job is stopped at the end of each run, before the restart.

## Files

**Live file**:
A file the game reads and writes where it lives (`Saved Games\DCS\Config\options.lua`).

**Profile file**:
The copy of a live file kept with a profile.

**Sync line**:
`profile file <-> live file`: copied in before the session, copied back after.
_Avoid_: mirror, copy pair.

**Copy-back**:
The after half of the sync lines: live files saved into the profile. The step every guard exists to protect.

**Ready file** (`ready.file`):
A file the program creates once started; informational only.

## Ending

**Detach**:
pm stops supervising and exits; the program keeps running; the after actions do **not** run.
_Avoid_: quit (ambiguous with stopping the program).

**Stop** (`k`):
pm stops the program, then runs the after actions.

**Last safety net**:
Before the after actions, with `track`: if the followed process runs unfollowed, or the process list cannot be read, the after actions are skipped and the session ends in error.
