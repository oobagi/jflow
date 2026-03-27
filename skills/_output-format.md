# Standard Skill Output Format

All skills follow this output structure. The goal: every skill output is instantly scannable — you know what ran, what happened, and what to do next.

## Structure

```
═══════════════════════════════════════
  <Skill Name> — <Status>
═══════════════════════════════════════

  <Body — skill-specific content>

  Done:
    ✓ Action completed
    ✓ Another action (Xm Ys)
    ✗ Failed action — reason

  Next: /command or specific instruction

═══════════════════════════════════════
```

## Sections

### Header (required)

```
═══════════════════════════════════════
  <Skill Name> — <Status>
═══════════════════════════════════════
```

**Status values:**
| Status | When to use |
|--------|-------------|
| `Starting` | Skill is beginning — showing scope/config |
| `Complete` | Skill finished successfully |
| `Stopped` | Stopped early (failure or user choice) |
| `Needs Work` | Finished but found issues requiring attention |

### Body (skill-specific)

All content indented 2 spaces. Skill-specific — metrics, findings, results, whatever fits. Use these markers consistently:

| Marker | Meaning |
|--------|---------|
| `✓` | Completed / passing |
| `✗` | Failed |
| `•` | Neutral bullet (info) |
| `▸` | In-progress (streaming status) |

When there are key metrics, put them first for quick scanning:

```
  Files changed:  14
  Issues created: 3
  Tests:          passing
```

### Done (required when the skill took actions)

Summarize what was actually done — commands run, files created, PRs opened:

```
  Done:
    ✓ Ran lint — clean
    ✓ Created branch feat/new-feature
    ✓ Opened PR #42
    ✗ CI check — timed out after 5m
```

Include duration when available: `✓ Simplify (2m 14s)`

Skip this section for pure report skills (sitrep, qa manual mode) that don't modify anything.

### Next (required — always)

Every skill output ends with a clear `Next:` line. The user should never wonder "what do I do now?"

Single action:
```
  Next: /test to validate, then /ship
```

Multiple steps:
```
  Next:
    1. Fix the auth error in src/api/auth.ts
    2. Run /test to re-validate
    3. /ship when clean
```

### Footer (required)

```
═══════════════════════════════════════
```

## Variants

### Multi-phase (skills with starting + completion output)

**Starting:**
```
═══════════════════════════════════════
  Polish — Starting
═══════════════════════════════════════

  Scope:
    Modified:  5 files
    New:       1 file
    Lines:     +120 / -45

  Pipeline: simplify > harden > test > ship
  Mode: full

═══════════════════════════════════════
```

### Failure / stopped early

```
═══════════════════════════════════════
  Polish — Stopped
═══════════════════════════════════════

  Done:
    ✓ Simplify    (2m 14s)
    ✗ Harden      — test regression in src/api/auth.ts

  Fix: resolve the regression in src/api/auth.ts
  Resume: /polish skip-simplify

═══════════════════════════════════════
```

### Confirmation prompt (skills that ask before acting)

```
═══════════════════════════════════════
  Issue — Draft
═══════════════════════════════════════

  Title: fix: login button unresponsive on mobile Safari
  Labels: bug, frontend, P1

  [body preview]

  Confirm: create this issue? (or suggest changes)

═══════════════════════════════════════
```

### Report-only (no actions taken)

```
═══════════════════════════════════════
  Sitrep — Complete
═══════════════════════════════════════

  [report content]

  Next: you have uncommitted work on feature-x — pick up there?

═══════════════════════════════════════
```

## Style rules

- 2-space indent inside the box
- 39 `═` characters for the box lines
- Skill name in Title Case
- No blank line between header bar and first content
- One blank line between sections
- No text after the footer bar
- In-progress markers (`▸`) replace themselves with `✓`/`✗` when done
- When running inside a pipeline (/autopilot, /polish), `Next:` describes the pipeline's next step
