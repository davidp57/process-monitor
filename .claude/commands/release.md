# Release

Guide the release of a new pm version, one step at a time, waiting for David's answer where a step says so.

## Context

- Trunk: feature PRs land on `main`; a release is a PR from `release/x.y.z` to `main`, then a tag.
- The version exists only in the tag `vx.y.z`; `.github/workflows/release.yml` builds `pm.exe` with it and publishes the GitHub Release, whose text is `RELEASE_NOTES.md`.

## Steps

1. **Sync**: `git fetch`, `git switch main`, `git pull --ff-only`. Check `main` is green (`gh run list --branch main --limit 1`).
2. **Read `[Unreleased]`** in `CHANGELOG.md` and propose the number (SemVer: breaking → major, feature → minor, fixes only → patch). **Wait for David's go on the number.**
3. **Branch** `release/x.y.z`.
4. **CHANGELOG**: rename `[Unreleased]` to `[x.y.z] — YYYY-MM-DD` and open a new empty `[Unreleased]` above it.
5. **RELEASE_NOTES.md**: rewrite it entirely for this version, **in French, for the user**: what changes for the user, in the user's words, no implementation vocabulary; a breaking change says what to do (config to rewrite, shortcut to update). **Show it to David and wait for the go.**
6. **Quality gate**: `gofmt -l .`, `go vet ./...`, `go test -race ./...`.
7. **Commit** `chore(release): x.y.z`, push, open the PR to `main`, run `pr-code-review`, merge when green (merge commit or squash, either: there is no second long-lived branch).
8. **Tag on `main`**: `git switch main`, `git pull --ff-only`, `git tag -a vx.y.z -m "pm x.y.z"`, `git push origin vx.y.z`.
9. **Check the release**: `gh run watch` is forbidden (long wait); look once with `gh run list --workflow release.yml --limit 1`, then `gh release view vx.y.z` once it is done. The same tag runs the Docs workflow, which publishes the version on the site and moves `latest` to it. Report the release URL and the site URL.
