---
name: design
description: >
  Design system creation and audit. Create mode generates a new design system
  (colors, typography, components, layout patterns). Audit mode reviews an
  existing design for consistency and accessibility.
user-invocable: true
argument-hint: >
  ["create" for new design system | "audit" to review existing | topic or context for targeted work]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, WebSearch, WebFetch
effort: high
---

# Design

Create or audit a design system for the current project. Produces a `DESIGN.md` as the source of truth, plus framework-specific token files.

## 0. Parse arguments and detect mode

Check `$ARGUMENTS` and the project state:

- If `create` is specified, or no design artifacts exist (no `DESIGN.md`, no design tokens, no theme config): **create mode**
- If `audit` is specified, or `DESIGN.md` or design tokens already exist: **audit mode**
- If ambiguous, ask the user with `AskUserQuestion`

Also detect the project's UI framework by reading the codebase:
- **Tailwind CSS** — `tailwind.config.js/ts`, `@tailwind` directives in CSS
- **CSS/SCSS** — stylesheets with custom properties or SCSS variables
- **SwiftUI** — `.swift` files with `Color(`, `Font.system(`
- **React Native** — `StyleSheet.create`, `react-native` in package.json
- **Godot** — `.tres` theme files, `theme_override_` in `.tscn`
- **Terminal/CLI** — no UI framework detected, focus on ANSI color schemes and output formatting
- **None detected** — framework-agnostic tokens only

---

# Create Mode

## 1. Gather context

Ask the user (use `AskUserQuestion` with all unknowns in a single multi-part question):

| Field | Required | Notes |
|---|---|---|
| **Product type** | Yes | Web app, mobile app, game, CLI tool, library docs, etc. |
| **Target audience** | Yes | Developers, consumers, enterprise, gamers, etc. |
| **Brand personality** | Yes | 3-5 keywords (e.g., "minimal, warm, professional") |
| **Existing brand assets** | No | Logo, colors, fonts already chosen |
| **Competitors or inspiration** | No | Links or names for aesthetic research |

Only ask about fields you can't infer from context. If the user said "a dark-themed developer tool", you already know the audience and personality — don't ask again.

If competitors or inspiration URLs are provided, use `WebSearch` and `WebFetch` to research their visual language.

## 2. Generate design system (run agents in parallel)

Launch two agents **in parallel**:

### Technical Artist (`subagent_type: "Technical Artist"`)

Tell it to generate visual foundations for the project. Scope the agent to UI design, NOT game engine pipelines. It should produce:

- **Color palette:**
  - Primary, secondary, accent colors with hex values
  - Semantic colors: success (#green), warning (#amber), error (#red), info (#blue)
  - Neutral scale: 50 through 950 (background, surface, text hierarchy)
  - Dark mode variants for all colors
- **Typography scale:**
  - Font families (heading, body, mono) — suggest specific fonts
  - Size scale: xs through 4xl with px/rem values
  - Weight scale: light, normal, medium, semibold, bold
  - Line height and letter spacing for each size
- **Spacing scale:** Base unit (e.g., 4px) with multipliers: 0.5, 1, 1.5, 2, 3, 4, 6, 8, 12, 16
- **Other tokens:** Border radii (sm, md, lg, full), shadow scale (sm, md, lg, xl), transition durations (fast, normal, slow)

### Software Architect (`subagent_type: "Software Architect"`)

Tell it to generate structural design patterns for the project. It should produce:

- **Component inventory:** What UI components the project needs based on the product type (e.g., a dashboard app needs: nav, sidebar, card, table, chart, modal, toast)
- **Layout patterns:** Grid system, responsive breakpoints, page templates
- **State patterns:** How to handle loading, empty, error, and success states consistently
- **Design token naming convention:** A systematic naming scheme (e.g., `color.primary.500`, `spacing.4`, `font.heading.lg`)

## 3. Write DESIGN.md

Compile the results into a `DESIGN.md` at the project root. Structure:

```markdown
# Design System

> Brief description of the design direction and personality.

## Colors

### Brand
| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| primary.500 | #3B82F6 | #60A5FA | Primary actions, links |
| ... | ... | ... | ... |

### Semantic
| Token | Light | Dark | Usage |
| ... | ... | ... | ... |

### Neutrals
| Step | Light | Dark |
| ... | ... | ... |

## Typography

| Token | Font | Size | Weight | Line Height |
|-------|------|------|--------|-------------|
| heading.xl | Inter | 30px | 700 | 1.2 |
| ... | ... | ... | ... | ... |

## Spacing

Base unit: 4px

| Token | Value |
|-------|-------|
| 1 | 4px |
| 2 | 8px |
| ... | ... |

## Components

Brief inventory of components with their purpose and state variations.

## Layout

Grid system, breakpoints, page templates.

## Accessibility

Minimum contrast ratios, focus states, motion preferences.
```

## 4. Generate framework-specific tokens

Based on the detected UI framework, also generate token files:

| Framework | Output file | Format |
|---|---|---|
| **Tailwind CSS** | `tailwind.config.js` theme extension | JS object extending `theme.extend` |
| **CSS/SCSS** | `design-tokens.css` | CSS custom properties (`:root { --color-primary: ... }`) |
| **SwiftUI** | `DesignTokens.swift` | `Color` and `Font` extensions |
| **React Native** | `theme.ts` | TypeScript theme object |
| **Godot** | `design_tokens.tres` | Godot theme resource |
| **CLI** | `colors.sh` or `colors.rs` | ANSI color constants |
| **None detected** | `design-tokens.json` | Framework-agnostic JSON tokens |

## 5. Summary

List all generated files and suggest next steps:
- Implement the component inventory
- Set up Storybook / design preview (if web)
- Run `/design audit` after implementation to verify consistency

---

# Audit Mode

## 1. Read existing design artifacts

Read all design-related files in the project:
- `DESIGN.md` — the source of truth
- Framework token files (CSS vars, Tailwind config, Swift extensions, etc.)
- Actual component files — search for color values, font sizes, spacing values used in the codebase

Build an inventory of **declared tokens** (from DESIGN.md / token files) and **used values** (from actual code).

## 2. Run audit agents in parallel

### Technical Artist (`subagent_type: "Technical Artist"`)

Tell it to audit visual consistency (scoped to UI, NOT game pipelines):

- **Color audit:** Find hex/rgb/hsl values in code that are NOT in the design system. Group by similarity — one-off values near a declared token are likely mistakes.
- **Typography audit:** Find font-size, font-weight, font-family values not in the type scale.
- **Spacing audit:** Find margin/padding/gap values that are not multiples of the base unit.
- **Accessibility:** Calculate contrast ratios for all text-on-background color combinations. Flag any below WCAG AA (4.5:1 for normal text, 3:1 for large text).

### Software Architect (`subagent_type: "Software Architect"`)

Tell it to audit structural consistency:

- **Component reuse:** Are there duplicated component patterns that should be consolidated?
- **Naming consistency:** Do token names follow the convention defined in DESIGN.md?
- **Missing states:** Do components handle loading, error, empty, and success states?
- **Token coverage:** Are there token categories in DESIGN.md that aren't actually used in code, or vice versa?

## 3. Report findings

Present findings grouped by severity:

- **Blockers** — Accessibility failures (contrast below WCAG AA), colors/fonts with no design token equivalent
- **Suggestions** — One-off values that should use tokens, missing component states, naming inconsistencies
- **Info** — Token coverage stats, unused tokens

## 4. Fix (with user approval)

For simple fixes (replacing a one-off hex value with the design token equivalent):
- Offer to auto-fix using Edit tool
- Show what would change before applying

For structural issues (missing states, component consolidation):
- Document the issue and recommended fix, but don't auto-apply

Update `DESIGN.md` with any new patterns discovered during the audit.

## 5. Summary

```
Design audit complete.

  Score: 87/100

  Blockers: 1
    • Text color #666 on background #fff has contrast ratio 3.9:1 (needs 4.5:1)

  Suggestions: 4
    • 3 one-off hex values could use design tokens
    • Button component missing loading state

  Token coverage:
    • Colors: 14/16 tokens used (2 unused)
    • Typography: 8/8 tokens used
    • Spacing: 10/12 tokens used (2 unused)
```
