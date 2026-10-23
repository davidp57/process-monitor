# Issue tracker: GitHub Issues

Specs and tickets for this repo live as **GitHub Issues** on `davidp57/process-monitor`, driven with the `gh` CLI.
There is no local backlog directory.

## Conventions

- A spec (the output of `to-spec`) is one issue, titled after the feature, with the spec template as its body.
- A ticket (the output of `to-tickets`) is one issue per tracer-bullet slice.
  Its body starts with `## Parent` (the spec issue, `#N`) and ends with `## Blocked by` (`#N` per blocker, or "None").
  Publish blockers first, so every reference points at a real issue.
- Status is carried by **one** triage label at a time (see `triage-labels.md`), plus the issue's open/closed state.
- A pull request names the tickets it delivers with `Closes #N` in its description; merging closes them.
  One lot = one branch = one PR, whatever the number of tickets.

## Commands

```bash
gh issue create --title "..." --body-file body.md --label ready-for-agent
gh issue list --label ready-for-agent
gh issue view N --comments
gh issue edit N --add-label ready-for-human --remove-label ready-for-agent
gh issue close N --reason "not planned"   # wontfix
```

## When a skill says "publish to the issue tracker"

- A spec → `gh issue create` with the `ready-for-agent` label.
- Tickets → one `gh issue create` each, in dependency order, with `ready-for-agent`.
- Never close or edit a parent issue from a ticket-creating skill.

## What stays out of the tracker

- David's own to-dos (a test to fly, a shortcut to switch) go to his Obsidian vault; the issue holds the technical detail and the vault task points to it.
- Review comments stay on their pull request.
