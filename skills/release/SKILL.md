---
name: release
description: Cut a release — triggers the project's release mechanism (GitHub Actions workflow, npm publish, cargo publish, Docker push, or a project-defined command) and monitors it to completion. Pass "preview" (default) or "production" to pick target. For mobile projects, optionally append "--screenshots[=ios|android|both]" to autonomously capture screenshots of new features via Maestro MCP. Always user-initiated — no other skill auto-invokes this.
user-invocable: true
argument-hint: [preview | production] [--screenshots[=ios|android|both]]
allowed-tools: Bash, Read, Edit, Glob, Grep, mcp__maestro__*
effort: low
---

# Release

Publish what's on `main` as a user-installable artifact. This skill is generic; projects that always release the same way can override it with `./.claude/skills/release/SKILL.md` to skip discovery.

Stop and report if any step fails.

## 0. Project-local override

If `./.claude/skills/release/SKILL.md` exists in the repo root, it already took precedence over this file — you won't be reading this. If you ARE reading this, the project has no override and you need to discover the release mechanism.

## 1. Parse arguments

- **Target** (`$1`): Default to `preview` if empty. Common values:
  - `preview` → internal/test artifact, published as prerelease
  - `production` → user-facing release, published as latest

  If `$1` is anything else (and not a flag), ask the user to clarify before proceeding.

- **Screenshots flag** (`--screenshots` or `--screenshots=<ios|android|both>`): Optional. Mobile projects only. Absent → skip step 7.5. Bare `--screenshots` → both platforms. Stop on any other value. If passed for a non-mobile project, warn and skip — don't fail the release.

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

## 7.5. Capture screenshots (only if `--screenshots` flag passed)

Skip this entire step if the flag was absent.

Goal: autonomously screenshot every new user-facing feature on the requested platforms and commit them to `docs/releases/<tag>/`. The user does not touch the simulator.

#### a. Detect mobile project + app ID

Check, in order:
- **Expo**: `app.json` exists with `expo.ios.bundleIdentifier` or `expo.android.package`. Use those as the app IDs.
- **Bare React Native / native iOS**: `ios/*/Info.plist` → `CFBundleIdentifier`.
- **Bare React Native / native Android**: `android/app/build.gradle` → `applicationId`.
- **Flutter**: `pubspec.yaml` + `ios/Runner.xcodeproj/project.pbxproj` (PRODUCT_BUNDLE_IDENTIFIER) and `android/app/build.gradle`.

If no mobile project is detected, output: "Skipping screenshots — `--screenshots` requires a mobile project (Expo / RN / Flutter) and none was detected." Then proceed to step 8 normally.

#### b. Determine scope

The scope is **every distinct user-visible surface** in this release — not every PR. A single PR can introduce multiple new screens, states, or UI details, and *each one* needs its own screenshot. "≥1 shot per PR" is not sufficient.

- List merged PRs since the previous release tag (skip `chore:`/`ci:` PRs that have no visible feature; keep `docs:` only if they ship visible content): `gh pr list --state merged --base main --search "merged:>$(git log -1 --format=%cI <prev-tag>)" --json number,title,body,url --limit 50`.
- **Prefer the release notes if they already exist** (`gh release view <tag> --json body -q .body`). Every `### Section` header is a feature, and every named surface/state in its prose must be captured. If the release notes haven't been drafted yet, fall back to the PR bodies.
- For each feature, enumerate every distinct surface the PR body / release notes describe. Examples:
  - "Added Preferences, Sounds & Haptics, Appearance, Learning Resources, and About pages" → capture each sub-page *and* the index that lists them (6 surfaces).
  - "Half-spin cross-fade animation" → two endpoints (before + after).
  - "Thicker borders, blue root outline, and a tighter header" → capture every distinct screen where a called-out detail lives, not just one.
  - "Tap the pencil to enter edit mode; custom chords wiggle with a red minus badge" → edit-mode state is its own surface.
- For each surface, infer the navigation path from the PR body/diff: which screen it lives on, what taps reach it, whether a sheet/modal must be opened. Read relevant component files from the source tree if the path is ambiguous.
- If no user-facing feature PRs, output: "No user-facing features in this release — no screenshots captured." Proceed to step 8.

#### c. Prereqs: dev server + devices + app launch (assume cold start)

Assume nothing is running. Do NOT ask the user to start the dev server or boot a simulator — handle it yourself.

1. **Dev server (mobile projects with a dev client)**. Expo / React Native dev clients need Metro on 8081 or the app shows a red-box "No script URL provided" / "Could not connect to development server". If the project has an Expo/RN dev client:
   ```bash
   lsof -iTCP:8081 -sTCP:LISTEN -P -n >/dev/null 2>&1 || \
     (cd <repo-root> && nohup npx expo start > /tmp/metro.log 2>&1 &)
   until lsof -iTCP:8081 -sTCP:LISTEN -P -n >/dev/null 2>&1; do sleep 1; done
   ```
   Run Metro in the background (e.g. `run_in_background: true`); don't block on it. Use the Monitor tool to wait for the port to come up, not a fixed sleep.
   - Release / production builds don't need Metro — skip this sub-step if the installed artifact is a release build.

2. **Devices**. For each platform in scope, `mcp__maestro__list_devices`. Pick a connected device if one exists; otherwise `mcp__maestro__start_device` with `platform: "ios"` or `"android"`.

3. **Launch app**. `mcp__maestro__launch_app` with the discovered app ID. Inspect the view hierarchy — if you see a red-box error, Metro is still booting or crashed; re-check 8081 and retry the launch. If the app itself isn't installed, log `"App not installed on <platform> — skipping that platform"` and continue with the other platform if applicable. Do NOT install the app from this skill.

#### d. Capture loop

For each platform, for each **feature**, for each **surface** of that feature:

1. `mcp__maestro__stop_app` then `mcp__maestro__launch_app` — reset to home screen
2. Navigate to the surface:
   - `mcp__maestro__inspect_view_hierarchy` — see what's on screen
   - `mcp__maestro__tap_on` — navigate (prefer text matches, fall back to accessibility id)
   - For sheets: tap the trigger, brief wait for animation
   - Repeat for each nav step until you're on the surface
3. `mcp__maestro__take_screenshot` → `docs/releases/<tag>/<platform>/<NN>-<feature-slug>-<MM>-<surface-slug>.png`
4. On any failure (target not found, navigation broke, app crashed): log it, mark the surface as uncovered for that platform, and move on. Partial coverage > no coverage — but the gap MUST surface in the output (step e).

`mkdir -p docs/releases/<tag>/{ios,android}` (only platforms in scope) before the first screenshot.

#### e. Index + coverage report + commit

Write `docs/releases/<tag>/README.md` with feature-grouped `![]()` references, one subsection per feature, one row per surface within it.

**Print a coverage report before committing**, grouped by feature × surface × platform. Example:

```
Screenshot coverage — <tag>

settings-sub-pages
  index              iOS ✓  Android ✓
  preferences        iOS ✓  Android ✓
  sounds-haptics     iOS ✓  Android ✓
  appearance         iOS ✗  Android ✗   (navigation failed)
  learning-resources iOS ✓  Android ✓
  about              iOS ✓  Android ✓
chord-builder-animation
  before             iOS ✓  Android ✓
  after              iOS ✓  Android ✓

Result: 2 gaps — see "appearance" above.
```

Gaps do not block the release, but they MUST be visible in the report and called out in the final output (step 8). Never silently skip a surface.

Commit:
```bash
git add docs/releases/<tag>/
git commit -m "docs: add release screenshots for <tag>"
git push origin main
```

If the push is rejected by branch protection, open a tiny PR via `gh pr create` titled `docs: release screenshots for <tag>` and let auto-merge handle it. Never bypass branch protection (no `--admin`, no force push).

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

## Expo / React Native: displayed version source

For mobile projects that show a version string in the UI (e.g. About/Settings screens), the version MUST be read from the native binary, not the JS-side app config snapshot. Concretely:

- **Use** `expo-application`: `Application.nativeApplicationVersion` (and `nativeBuildVersion` for the build number).
- **Do NOT use** `Constants.expoConfig?.version` (or `Constants.manifest?.version`). That value is read from a manifest snapshot baked at prebuild time. `npx expo run:ios|android` does NOT re-run prebuild after an `app.json` version bump, so the displayed version silently drifts from the installed binary's actual version.

If during release-related work you spot a screen reading version from `expo-constants`, flag it and offer to swap to `expo-application`. Install with `npx expo install expo-application`.

After bumping `app.json` `expo.version`, local dev clients still need `npx expo prebuild --clean && npx expo run:<platform>` to flow the new version into `Info.plist` (`CFBundleShortVersionString`) and `android/app/build.gradle` (`versionName`). EAS Build always re-runs prebuild on its own, so released artifacts are always correct — it's only the local install that goes stale.

## Style guidelines
- Follow the standard output format in `_output-format.md`
- Poll interval ≥60s for long builds
- Never edit version files without explicit user confirmation
- Never trigger `production` without showing the commit diff since last prod tag and getting confirmation
- Screenshots (`--screenshots`): never block the release on capture failures — log and continue. Never install the app from this skill — assume it's installed. Maestro MCP must be configured in the user's Claude Code MCP settings (`claude mcp add -s user maestro -- maestro mcp`); if it isn't, output a one-liner with that command and skip the step
- If the repo is unfamiliar, suggest creating `./.claude/skills/release/SKILL.md` to lock in answers for next time
