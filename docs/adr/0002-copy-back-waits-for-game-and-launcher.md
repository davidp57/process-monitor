# 0002 — The copy-back waits for the game and its launcher, not for leftovers

Status: accepted — 2026-10-23, decided by David

## Context

When the followed game exits, other processes of the session may still run.
Two kinds, with opposite needs:

- the **launcher**, which may start the game again (after an update): copying the profile back now would save it mid-session and release the lock while the game returns;
- **leftovers** the game started for itself: with `srsAutoLaunchEnabled = true` (set in all of David's DCS profiles), DCS starts the SRS radio client, which inherits the job. Waiting for the whole job meant the copy-back waited until SRS was closed by hand, and a double Ctrl+C to get out of that wait lost the copy-back entirely.

1.x waited for nothing but the game. A review asked for the opposite: wait for every descendant.

## Decision

- After the followed process exits (and the handover grace), pm waits only for processes of the job whose image name is the launcher's.
- Every other process still running is **named in the log** and not waited for.
- `kill_on_exit` lists image names to stop once the followed process has exited, wherever they run. David's DCS config lists `SR-ClientRadio.exe`.
- Without `track` nothing changes: the session ends when the whole job has exited (SimAppPro's Electron children are the program).

## Consequences

- A leftover that writes a live file after the game exits could race the copy-back. None is known; if one appears, it goes in `kill_on_exit` (stopped before the copy-back).
- An SRS running elevated cannot be stopped by a non-elevated pm: the log says "access denied" and the copy-back still runs.
