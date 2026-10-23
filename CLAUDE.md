# pm — Claude Code Instructions

> `pm` is a single Windows executable written in Go: `pm run` wraps a program with actions before and after it (profile files copied in and back), `pm watch` keeps a program alive.
> Its real users are DCS World profile switching and SimAppPro; see `README.md` and `CONTEXT.md`.

---

## 1. Language

- **Communication**: the language the user writes in.
- **Code, comments, commit messages, PR descriptions, `CLAUDE.md`, `docs/`, `CONTEXT.md`, ADRs**: English.
- **User-facing text in French**: the program's messages, `README.md`, `RELEASE_NOTES.md`. The user is French; these are what he reads.

---

## 2. Behaviour

- **Surgical changes**: never touch adjacent code, comments or formatting unrelated to the request.
- **Simplicity**: the minimum code that solves the problem; no speculative options.
- **No assumptions**: an ambiguous or contradictory instruction → stop and ask.
- **Testing never touches the user's desktop.** Tests and manual checks run copies of the test binary, hidden, never a GUI program on David's session; a process is killed by the PID you started, never by name. Ask before anything that opens a window or stops a program he did not start through you.

---

## 3. Quality gate (the CI runs exactly this)

```bash
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...   # needs cgo: gcc on PATH
```

Resolve errors rather than silencing them. `-race` is not optional: `internal/proc` and `internal/session` are concurrent by nature.

---

## 4. TDD and coverage

- Write the failing test first, then the code.
- New behaviour ships with its test; changed behaviour updates its test.
- A test must be able to fail for the reason it names: assert the outcome (a marker file written at exit, the captured text, the process gone), not only that a call returned nil or that some time elapsed.
- Process tests use copies of the test binary driven by `PM_HELPER` (see `internal/proc/proc_windows_test.go`); never a real program.

---

## 5. Git

- Work on a branch from `main` (`feat/<slug>`, `fix/<slug>`, `chore/<slug>`); never commit to `main`.
- Conventional Commits in English (`type(scope): description`, `!` for a breaking change).
- One lot = one branch = one PR, however many tickets it closes.
- Line endings are LF (`.gitattributes`); `gofmt` rejects CRLF.

---

## 6. Default workflow

0. **Sync first**: `git fetch` then `git pull --ff-only` on `main` (or rebase the branch on `origin/main`) before reading issues or code.
1. **Analyse** the request; when starting on an issue, restate it in 1–3 sentences (what, and what "done" means) before any tool call.
2. **Track it**: if no issue covers the work, create one (`docs/agents/issue-tracker.md`).
3. **Branch** from `main`.
4. **Implement**: test first, code, docs (`README.md` when behaviour or config changes).
5. **Quality gate** (§3).
6. **`CHANGELOG.md`**: one entry under `[Unreleased]`, appended at the **end** of its section. Do not create a version heading and do not touch the version (§8).
7. **Manual check needed?** Stop and wait for David's go.
8. **Commit, push, open the PR** to `main`, with `Closes #N`.
9. **Review**: run `pr-code-review` on the PR; fix what it finds on the same branch.
10. **Merge** when CI is green **and** `pr-code-review` is done; then back to `main`, pull, and say so.

---

## 7. Pull requests

- `pr-code-review` (full by default, light when David asks) is the review "merge au vert" waits for.
- **Sourcery constrains nothing.** Read its comments like any opinion; its failing status check is not a gate. Never split a lot for it, never silence a finding to turn it green.
- Do not request a Copilot review.

---

## 8. Versions and releases

- The version lives **only** in the git tag: the release workflow injects it (`-ldflags "-X main.version=..."`); a local build says `dev`.
- A PR never bumps anything: two open PRs must not conflict on a number.
- Releasing is `/release` (`.claude/commands/release.md`): `[Unreleased]` becomes `[x.y.z]`, `RELEASE_NOTES.md` is rewritten for the user, the release PR merges, then the tag `vx.y.z` on `main` triggers `.github/workflows/release.yml`, which builds `pm.exe` and publishes the GitHub Release.

---

## Agent skills

### Issue tracker

GitHub Issues, through `gh`. See `docs/agents/issue-tracker.md`.

### Triage labels

`needs-triage` · `needs-info` · `ready-for-agent` · `ready-for-human` · `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` + `docs/adr/`. See `docs/agents/domain.md`.
Read the ADRs before changing how a session ends or what it waits for: those are decisions, not accidents.
