---
name: release
description: Cut a release — triggers the project's release mechanism (GitHub Actions workflow, npm publish, cargo publish, Docker push, or a project-defined command) and monitors it to completion. Pass "preview" (default) or "production" to pick target. Always user-initiated — no other skill auto-invokes this.
user-invocable: true
allowed-tools: Bash, Read, Edit, Glob, Grep
effort: low
---

# Release

Publish what's on `main` as a user-installable artifact. This skill is generic; projects that always release the same way can override it with `./.claude/skills/release/SKILL.md` to skip discovery.

Stop and report if any step fails.

## 0. Project-local override

If `./.claude/skills/release/SKILL.md` exists in the repo root, it already took precedence over this file — you won't be reading this. If you ARE reading this, the project has no override and you need to discover the release mechanism.

## 1. Parse the argument

`$1` is the release target. Default to `preview` if empty. Common values:
- `preview` → internal/test artifact, published as prerelease
- `production` → user-facing release, published as latest

If `$1` is anything else, ask the user to clarify before proceeding.

## 2. Discover the release mechanism

Look for one of these, in order:

1. **GitHub Actions workflow** — `ls .github/workflows/*.yml`. Scan for a workflow with a name matching `/build|release|publish|deploy|ship/i` that has a `workflow_dispatch` trigger. If exactly one match, use it. If multiple, show the list and ask which one.
2. **npm publish** — `package.json` with no `private: true` and a build/release script. Target is npm registry.
3. **cargo publish** — `Cargo.toml` with `[package]` section.
4. **Python publish** — `pyproject.toml` with `[project]` or `[tool.poetry]`.
5. **Docker** — `Dockerfile` + a build workflow.
6. **Custom** — a `scripts/release.sh` or similar.

If none found, stop and ask the user how releases work in this repo, then suggest creating a project-local override so this discovery can be skipped next time.

## 3. Discover the version file

Look for one of these, in order:
- `app.json` (Expo) → `expo.version`
- `package.json` → `version`
- `Cargo.toml` → `[package] version`
- `pyproject.toml` → `[project] version` or `[tool.poetry] version`
- `ios/*/Info.plist` → `CFBundleShortVersionString`
- `VERSION` file

Stash the current version.

## 4. Preflight (production only)

For `production`:

1. `git fetch --tags`
2. Check if tag `v{version}` already exists (`git tag -l "v{version}"` and `git ls-remote --tags origin "v{version}"`). If it exists → stop and tell the user: "Production tag `v{version}` already exists. Bump the version before releasing." Offer to bump minor (`0.2.0 → 0.3.0`) or patch (`0.2.0 → 0.2.1`) via a small PR, then continue.
3. Show: current version, tag that will be created, SHA of `main`, commits since last production tag (`git log v{prev}..main --oneline | head -20`).
4. Ask the user to confirm before triggering. (Production is expensive and public — always confirm.)

For `preview`, skip — preview artifacts typically append a unique suffix (run number, timestamp, commit SHA) so collisions don't matter.

## 5. Check for uncommitted changes

If `git status --porcelain` is non-empty, stop. The release runs against `origin/main`, so local dirty state would silently not be included. Tell the user to commit/stash/revert first.

## 6. Trigger

### GitHub Actions workflow

```
gh workflow run "<workflow name>" --field profile=<profile>
# or whatever input name the workflow uses
```

Wait 3 seconds, list runs (`gh run list --workflow="<name>" --limit 1 --json databaseId,status,createdAt`), confirm the new run appears. If not, stop.

### npm / cargo / others

Follow the project's published command. Don't invent commands. If unsure, ask.

## 7. Monitor

For GH Actions, poll every 60 seconds (long-running builds; faster polling wastes cache cycles):

```
until [ "$(gh run view <id> --json status -q .status)" = "completed" ]; do sleep 60; done
```

For local commands (`npm publish`, etc.), run synchronously and capture stderr.

## 8. Report

**Success**:
- Fetch release/tag URL (`gh release list --limit 1 --json tagName,url,isPrerelease` for GH; `https://www.npmjs.com/package/<name>` for npm; etc.)
- Show what landed and where.
- Offer to open in browser (`gh release view <tag> --web`, `open https://npmjs.com/...`).

**Failure**:
- Fetch the failing log (`gh run view <id> --log-failed | tail -80` for Actions)
- One-line likely cause (quota, auth, bundle, signing, version conflict)
- Suggest next step.

## 9. Hand-off

On success, remind the user of follow-ups if relevant:
- Install the artifact (APK URL, `npm install`, etc.)
- Update CHANGELOG / release notes if not auto-generated
- Post announcement (Discord/Slack/Twitter) if the project has a channel

## Output format

```
═══════════════════════════════════════
  Release — <Status>
═══════════════════════════════════════

  Mechanism: <GH Actions / npm / cargo / ...>
  Target: <preview | production>
  Version: <x.y.z>
  Tag: <vX.Y.Z[-preview.N]>

  Done:
    ✓ Triggered <workflow/command>
    ✓ Build finished in <Xm Ys>
    ✓ Published to <where> (<url>)

  Next: <install command | /release production | etc.>

═══════════════════════════════════════
```

## Notes

Release is always a user decision — no other skill triggers `/release` automatically. It only publishes; it never modifies code. Run it when *you* decide something is shippable.

## Style guidelines
- Follow the standard output format in `_output-format.md`
- Poll interval ≥60s for long builds
- Never edit version files without explicit user confirmation
- Never trigger `production` without showing the commit diff since last prod tag and getting confirmation
- If the repo is unfamiliar, suggest creating `./.claude/skills/release/SKILL.md` to lock in answers for next time
