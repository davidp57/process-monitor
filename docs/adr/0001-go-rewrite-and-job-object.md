# 0001 — Go rewrite, Job Object, and following the game by handle

Status: accepted — 2026-10-23

## Context

process-monitor 1.x (Python, frozen with cx_Freeze) did two things in one loop: keep a program alive, and copy profile files around a DCS session.
It detected the program by polling every running process by name every half second.
That design caused real defects: a second `dcs.exe` (another profile) was adopted and its settings copied back into the wrong profile; the after copies ran up to three times; `q` killed the game; the launcher (`dcs_updater.exe`) and the game having different names needed a lock-file workaround.
Deployment was a folder of several hundred files around `python312.dll`.

## Decision

- Rewrite in **Go**: one static `pm.exe`, no runtime, direct access to the Windows API.
- **One executable, two subcommands**: `pm run` (before/after actions around one run) and `pm watch` (watchdog). They share the config, the launcher and the process follower.
- Launch the program **suspended inside a Job Object**, then resume it: everything it starts belongs to the session.
- **Follow the game by handle** (`track`): found in the job from its creation notification, or by name among processes created after the launch when it left the job. Never poll by name to decide the session is over.
- Guard the copy-back: a session lock per name and per live file, a refusal when the game or its launcher already runs, and a last check before the after actions.

## Consequences

- Windows only: `internal/proc` and `internal/lock` use the Windows API.
- Breakaway is allowed on the job (`JOB_OBJECT_LIMIT_BREAKAWAY_OK`): a launcher that requests it must not fail to start the game; the name fallback covers it.
- A process adopted by name could in theory be an unrelated homonym started in the same seconds; `if_running` guarantees none ran before the launch. Accepted.
- Tests start real processes (copies of the test binary), so they run on Windows only.
