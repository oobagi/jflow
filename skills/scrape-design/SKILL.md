---
name: scrape-design
description: >
  Scrape a website and produce a high-fidelity design document capturing colors,
  typography, layout patterns, component inventory, and design philosophy —
  detailed enough to rebuild the site from scratch.
user-invocable: true
argument-hint: >
  <url>
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, WebFetch, WebSearch, TaskCreate, TaskUpdate
effort: high
---

# Scrape Design

Analyze a live website and produce a comprehensive design specification document. The output should be detailed enough that a developer could rebuild the site's visual identity and layout structure without ever seeing the original.

## 0. Parse arguments and validate

Extract the URL from `$ARGUMENTS`. If no URL is provided, use `AskUserQuestion` to request one.

Normalize the URL (add `https://` if missing, strip trailing slashes).

---

## 1. Fetch the site

Use `WebFetch` to retrieve the target URL. Fetch the **full HTML** — we need the raw markup, inline styles, stylesheet links, and meta tags.

From the initial fetch, extract:
- **Navigation links** — identify 3-5 distinct page types (e.g., homepage, about, pricing, blog post, docs). Pick links that look structurally different, not just different content.
- **External stylesheet URLs** — linked CSS files (`<link rel="stylesheet" href="...">`). Fetch the primary stylesheet(s) with `WebFetch` to extract the full design token set.
- **Meta information** — `<title>`, `<meta name="description">`, Open Graph tags, favicon, theme-color meta tag.
- **Font references** — Google Fonts links, Adobe Fonts, `@font-face` declarations, or font CDN URLs.

**Always fetch subpages.** After the homepage, fetch 2-4 additional pages that are structurally distinct (e.g., a content page, a listing page, a form/input page, a minimal page). Prioritize pages that reveal different layout patterns or component types not visible on the homepage.

**Important:** Capture all raw CSS custom properties (`:root { --var: value }`) and Tailwind class patterns. These are the ground truth for the design system.

---

## 2. Analyze — launch agents in parallel

Create a parent task with `TaskCreate`: "Scraping design from [URL]"

Launch **three agents in parallel** to analyze different dimensions of the design:

### Agent A: Visual Foundations (`subagent_type: "Technical Artist"`)

Scope this agent to **UI/web design analysis**, NOT game engine pipelines. Provide the fetched HTML and CSS content. Tell it to extract:

**Color System:**
- Every unique color value used (hex, rgb, hsl, oklch, CSS variables) — group into:
  - **Brand colors** — primary, secondary, accent (the dominant 2-3 colors)
  - **Semantic colors** — success, warning, error, info tones
  - **Neutral scale** — backgrounds, surfaces, borders, text colors (ordered light to dark)
  - **Decorative colors** — gradients, overlays, shadows with color
- For each color: exact value, where it's used (background, text, border, accent), and frequency
- Note the **overall color temperature** (warm, cool, neutral) and **saturation strategy** (muted, vibrant, mixed)
- Dark mode: if the site has a dark mode toggle or `prefers-color-scheme` media queries, capture both palettes

**Typography System:**
- All font families used — heading fonts, body fonts, monospace fonts, display fonts
- The complete type scale: every unique font-size found, mapped to a logical scale (xs, sm, base, lg, xl, 2xl, etc.)
- For each scale step: font-size (px and rem), font-weight, line-height, letter-spacing
- Where each type style is used (h1, h2, body, caption, nav, button, etc.)
- **Text sizing philosophy** — how does the hierarchy work? Large contrast between heading and body? Subtle? How many distinct sizes are actually used?
- Font loading strategy (Google Fonts, self-hosted, system fonts)

**Spacing System:**
- Detect the **base unit** — is spacing built on 4px, 8px, 5px, or something else?
- Catalog all unique padding, margin, and gap values
- Map them to a scale (1, 2, 3, 4, 6, 8, 12, 16, etc. x base unit)
- Note **padding patterns** — what padding does the hero section use? Cards? Sections? Buttons?
- **Section spacing** — how much vertical space exists between major page sections? Is it consistent?
- **Content max-width** — what's the maximum content width? Does it vary by section?

**Visual Effects:**
- Border radii used (sharp, slightly rounded, pill-shaped, circular)
- Shadow styles (subtle, dramatic, layered, colored)
- Transitions and animations (hover effects, entrance animations, scroll-triggered)
- Backdrop filters, blurs, overlays
- Image treatments (rounded corners, overlapping, bleeding to edge, contained)

### Agent B: Layout & Component Architecture (`subagent_type: "Software Architect"`)

Provide the fetched HTML content. Tell it to extract:

**Page-Level Layout:**
- **Overall page structure** — what are the major sections from top to bottom? (nav, hero, features, social proof, CTA, footer, etc.)
- **Section shapes** — does the site use full-bleed sections? Contained sections? Alternating backgrounds? Angled dividers? Overlapping sections?
- **Grid system** — is it a 12-column grid? Flexbox-based? CSS Grid? What are the column patterns? (1-col, 2-col, 3-col, 4-col — where is each used?)
- **Responsive strategy** — breakpoints detected, how the layout adapts (stack, hide, reflow)
- **Content width constraints** — max-width values, when content goes full-bleed vs contained
- **Whitespace rhythm** — is there a consistent vertical rhythm? How does spacing scale between sections?

**Component Inventory:**
For EACH distinct component type found on the site, document:
- **Component name** (e.g., "Feature Card", "Testimonial Carousel", "Pricing Table")
- **Visual description** — shape, proportions, what's inside it
- **Layout pattern** — how it's arranged internally (image left + text right, icon top + text bottom, etc.)
- **Variants** — if the same component appears in different sizes, colors, or orientations
- **State indicators** — hover effects, active states, selected states
- **Content structure** — what content slots exist (title, description, image, badge, CTA, etc.)

**Section Patterns (document each unique section):**
- **Purpose** — what job does this section do? (hero, social proof, feature showcase, CTA, FAQ, etc.)
- **Layout** — how is content arranged? (centered single column, side-by-side, grid of cards, etc.)
- **Visual weight** — is this section visually heavy (dark background, large text) or light?
- **Content density** — how much content is in this section vs whitespace?
- **Unique touches** — anything distinctive about this section's design

**Navigation Patterns:**
- Header layout (logo position, nav items, CTA button)
- Mobile navigation pattern (hamburger, bottom nav, slide-out)
- Footer structure (columns, link groups, social links)
- In-page navigation (sticky headers, scroll indicators, breadcrumbs)

### Agent C: Design Philosophy & Identity (`subagent_type: "UX Researcher"`)

Provide the fetched HTML content and tell it to analyze the site's design philosophy, visual pacing, and identity from a user experience perspective:

**Design Philosophy:**
- **Overall aesthetic** — what design movement does this site align with? (minimalist, maximalist, brutalist, neomorphic, glassmorphic, organic, corporate, playful, etc.)
- **Design era** — does it feel modern (2024+), classic (2018-2022), or retro?
- **Key design principles** — what 3-5 principles seem to guide every design decision? (e.g., "whitespace as a feature", "content-first", "progressive disclosure", "visual delight")

**Visual Pacing & Flow:**
- **Information hierarchy** — how does the page guide the eye from most important to least?
- **Scroll rhythm** — is there a pattern of dense section → breathing room → dense section?
- **Visual anchors** — what elements grab attention first? (large hero image, bold headline, animated element)
- **Contrast strategy** — how does the site create emphasis? (size contrast, color contrast, whitespace contrast, weight contrast)
- **Content density progression** — does the page start sparse and get denser, or stay consistent?

**Image & Media Philosophy:**
- **Image role** — are images decorative, informational, emotional, or functional?
- **Image style** — photography, illustrations, icons, 3D renders, abstract shapes, screenshots?
- **Image placement** — full-bleed, contained, overlapping, floating, background?
- **Image-to-text ratio** — is this an image-heavy or text-heavy design?
- **Absence of images** — if sections lack images, what fills the visual space? (typography, whitespace, color blocks, patterns)

**What Makes This Site Unique:**
- List 5-10 specific design decisions that differentiate this site from a generic template
- For each: what is it, why it works, and how it contributes to the brand identity
- Note any **unconventional choices** — things that break common patterns but work well
- **Micro-interactions** — small hover effects, button animations, scroll behaviors that add polish

**Emotional Impact:**
- What feeling does the site evoke? (trust, excitement, calm, urgency, playfulness, sophistication)
- How do the design choices support this feeling? (color warmth, type elegance, spacing openness, image tone)

---

## 3. Compile the design document

Collect all three agent results. Resolve any conflicts (e.g., different agents naming the same color differently). Cross-reference to ensure completeness.

### 3a. Check for existing design docs

Before writing, check if `docs/design/` already exists. Use `Glob` to scan for `docs/design/**/*.md`.

- **If no existing docs:** create `docs/design/` and write all files fresh.
- **If existing docs found:** read each existing file. Compare the scraped data against what's already documented. Use `AskUserQuestion` to present a summary of major differences and ask which files to replace vs merge vs skip. For example:

  ```
  Existing design docs found in docs/design/:

  Files with major differences (scraped site diverges significantly):
    - colors.md — current: 6 brand colors, warm palette. Scraped: 4 brand colors, cool palette.
    - typography.md — current: Inter/Mono stack. Scraped: Geist/Geist Mono stack.

  Files with minor differences (scraped site mostly aligns):
    - spacing.md — same 4px base, slightly different scale steps.

  Files not yet documented (new from scrape):
    - effects.md — not present, will create.

  For each file with major differences, should I:
    (a) Replace with scraped version
    (b) Merge (keep existing, add scraped as "Reference: [site name]" section)
    (c) Skip
  ```

  Respect the user's choices. Default to **merge** for files with major differences if the user doesn't specify.

### 3b. Write the design docs

Create `docs/design/` if it doesn't exist. Write the following files, each with a focused scope:

---

#### `docs/design/overview.md` — Overview & Philosophy

The index file. Links to all other design docs. Contains the high-level identity.

```markdown
# Design System — [Site Name]

> Scraped from [URL] on [date]
>
> This directory captures the complete visual identity, layout architecture,
> and design philosophy — detailed enough to rebuild the site from scratch.

## Design Philosophy

### Aesthetic & Identity
[Overall aesthetic description, design era, brand personality]

### Core Principles
[3-5 principles with explanation of how each manifests in the design]

### Emotional Target
[What feeling the site aims to evoke and how design choices support it]

## Documents

| File | Scope |
|------|-------|
| [colors.md](colors.md) | Color system — brand, semantic, neutrals, strategy |
| [typography.md](typography.md) | Font stacks, type scale, sizing philosophy |
| [spacing.md](spacing.md) | Base unit, spacing scale, section rhythm, content width |
| [effects.md](effects.md) | Borders, shadows, transitions, decorative elements |
| [layout.md](layout.md) | Page structure, grid system, section patterns |
| [components.md](components.md) | Component inventory with variants and states |
| [navigation.md](navigation.md) | Header, footer, in-page navigation patterns |
| [media.md](media.md) | Image philosophy, style, placement, density |
| [identity.md](identity.md) | What makes this site unique, rebuild notes |
```

---

#### `docs/design/colors.md` — Color System

```markdown
# Colors

> Part of the [Site Name] design system — [see overview](overview.md)

## Brand Palette
| Token | Value | Usage | Notes |
|-------|-------|-------|-------|

## Semantic Colors
| Token | Value | Usage |
|-------|-------|-------|

## Neutral Scale
| Step | Value | Usage |
|------|-------|-------|

## Gradients & Decorative Colors
[Gradient definitions, overlay colors, shadow tints]

## Color Strategy
[Overall temperature, saturation approach, dark mode behavior, how color creates hierarchy and mood]
```

---

#### `docs/design/typography.md` — Typography

```markdown
# Typography

> Part of the [Site Name] design system — [see overview](overview.md)

## Font Stack
| Role | Family | Source | Fallbacks |
|------|--------|--------|-----------|

## Type Scale
| Token | Size | Weight | Line Height | Letter Spacing | Usage |
|-------|------|--------|-------------|----------------|-------|

## Text Sizing Philosophy
[How hierarchy is created — size contrast ratio between heading and body, how many distinct sizes are used, weight as hierarchy tool, when letter-spacing is applied]

## Font Loading
[Google Fonts, self-hosted, system fonts — how fonts are loaded and what fallback strategy is used]
```

---

#### `docs/design/spacing.md` — Spacing & Rhythm

```markdown
# Spacing & Rhythm

> Part of the [Site Name] design system — [see overview](overview.md)

## Base Unit
[Value (e.g., 4px) and how the scale is derived from it]

## Spacing Scale
| Token | Value | Common Usage |
|-------|-------|-------------|

## Section Spacing
[Vertical rhythm between major page sections — is it consistent or variable? How does padding inside sections compare to gaps between them?]

## Content Width
[Max-width values for different contexts — main content, wide sections, full-bleed. When content is constrained vs when it breaks out.]

## Padding Patterns
[Specific padding values for common elements: hero sections, cards, buttons, form inputs, nav items. Why these values — what rhythm do they create?]
```

---

#### `docs/design/effects.md` — Visual Effects

```markdown
# Visual Effects

> Part of the [Site Name] design system — [see overview](overview.md)

## Borders & Radii
| Token | Value | Usage |
|-------|-------|-------|

## Shadows
| Token | Value | Usage |
|-------|-------|-------|

## Transitions & Animations
[Hover effects, entrance animations, scroll-triggered behaviors, transition durations and easing curves]

## Decorative Elements
[Gradients, overlays, backdrop filters, blurs, divider styles, background patterns]

## Image Treatments
[How images are styled — rounded corners, overlapping, bleeding to edge, contained, masked shapes]
```

---

#### `docs/design/layout.md` — Layout Architecture

```markdown
# Layout Architecture

> Part of the [Site Name] design system — [see overview](overview.md)

## Page Structure
[Top-to-bottom section order for each page type scraped. Visual flow description.]

## Grid System
[Column count, gutter width, implementation (CSS Grid, Flexbox, etc.), column patterns used (1-col, 2-col, 3-col, asymmetric)]

## Responsive Strategy
[Breakpoints detected, how layout adapts at each — stack, hide, reflow, resize]

## Section Patterns

### [Section Name] (e.g., Hero)
- **Purpose:** [what job this section does]
- **Layout:** [arrangement description — centered, side-by-side, grid, etc.]
- **Background:** [color, image, gradient, or transparent]
- **Content:** [what goes inside — headline, subtext, CTA, image, etc.]
- **Spacing:** [internal padding, element gaps]
- **Visual weight:** [heavy/light — dark background + large text vs airy whitespace]
- **Content density:** [how much content vs whitespace]
- **Unique traits:** [what makes this section distinctive]

[Repeat for each unique section type]

## Whitespace Rhythm
[Is there a pattern of dense → breathing room → dense? How does spacing scale between sections? What creates the scroll rhythm?]
```

---

#### `docs/design/components.md` — Component Inventory

```markdown
# Component Inventory

> Part of the [Site Name] design system — [see overview](overview.md)

## [Component Name] (e.g., Feature Card)
- **Shape:** [proportions, aspect ratio, overall silhouette]
- **Internal layout:** [how content is arranged — image top + text bottom, icon left + text right, etc.]
- **Content slots:** [title, description, image, icon, badge, CTA, metadata, etc.]
- **Variants:** [size, color, orientation, density variations observed]
- **States:** [default, hover, active, focused, disabled — describe each]
- **Spacing:** [internal padding, gap between child elements]
- **Visual style:** [background, border, shadow, radius specific to this component]

[Repeat for every distinct component type found on the site]
```

---

#### `docs/design/navigation.md` — Navigation Patterns

```markdown
# Navigation

> Part of the [Site Name] design system — [see overview](overview.md)

## Header
[Logo position, nav item layout, CTA button placement, background treatment, behavior on scroll (sticky, shrink, hide, change background), mobile adaptation]

## Mobile Navigation
[Pattern used — hamburger menu, bottom nav bar, slide-out drawer, full-screen overlay. Transition animation. What items are shown vs hidden.]

## Footer
[Column structure, content organization, link groups, social links, legal text, newsletter signup, background treatment]

## In-Page Navigation
[Sticky sub-headers, scroll indicators, breadcrumbs, table-of-contents sidebars, anchor links, progress bars]
```

---

#### `docs/design/media.md` — Image & Media Strategy

```markdown
# Image & Media Strategy

> Part of the [Site Name] design system — [see overview](overview.md)

## Image Philosophy
[Why images are used — decorative, informational, emotional, functional? What role do they serve in the overall design narrative?]

## Image Style
[Photography style (candid, staged, abstract), illustration style (flat, 3D, hand-drawn), icon style (outlined, filled, duotone), screenshots, abstract shapes]

## Image Placement Patterns
[Full-bleed vs contained, overlapping other elements, floating, background usage, aspect ratios, cropping strategy]

## Media Density
[Image-to-text ratio across sections, where media is concentrated vs absent, how media density changes as you scroll]

## Absence of Images
[In sections without images, what fills the visual space? Typography, whitespace, color blocks, patterns, icons?]
```

---

#### `docs/design/identity.md` — Unique Identity & Rebuild Notes

```markdown
# What Makes This Site Unique

> Part of the [Site Name] design system — [see overview](overview.md)

## Distinctive Design Decisions

[Numbered list of 5-10 specific design decisions that differentiate this site from a generic template. For each:]

### N. [Decision Name]
- **What:** [describe the design choice]
- **Why it works:** [why this is effective]
- **Brand contribution:** [how it reinforces the site's identity]

## Unconventional Choices
[Things that break common design patterns but work well — and why]

## Micro-Interactions
[Small hover effects, button animations, scroll behaviors, loading states, transitions that add polish]

## Visual Pacing & Flow
[How the page guides the eye — information hierarchy, scroll rhythm, visual anchors, contrast strategy, content density progression]

---

## Rebuild Notes

### Priority Order
[Recommended order for implementing: 1. tokens/variables, 2. layout/grid, 3. typography, 4. components, 5. effects/polish, 6. responsive, 7. micro-interactions]

### Critical Details
[Subtle things easy to miss but important for fidelity — specific spacing between logo and nav, exact border-radius on cards vs buttons, shadow direction consistency, etc.]

### Common Pitfalls
[What would make a rebuild look "off" — wrong font weight, spacing too tight/loose, missing hover states, wrong neutral gray tone, etc.]
```

---

Update the parent task to "completed".

---

## 4. Summary

Present a brief summary:

```
Design scrape complete: docs/design/

  Files written:
    overview.md      — philosophy, principles, emotional target
    colors.md        — [N] unique values across [N] categories
    typography.md    — [font families listed]
    spacing.md       — [base unit]px base, [N]-step scale
    effects.md       — borders, shadows, transitions
    layout.md        — [N] unique section patterns
    components.md    — [N] distinct component types
    navigation.md    — header, footer, in-page nav
    media.md         — image philosophy and placement
    identity.md      — [N] unique traits, rebuild notes

  Top unique traits:
    1. [most interesting design decision]
    2. [second most interesting]
    3. [third most interesting]
```

If the user has a DESIGN.md in the project root (from `/design create`), suggest reconciling it with the new `docs/design/` directory — either migrating DESIGN.md content into the directory structure, or keeping both with cross-references.
