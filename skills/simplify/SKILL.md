---
name: simplify
description: >
  Deep codebase simplification — dispatches parallel agents to fix DRY violations,
  remove dead code, and simplify logic. Creates reusable helpers for repeated patterns.
  Integrates into /autopilot when the "thorough" flag is used.
user-invocable: true
argument-hint: >
  ["full" for entire codebase | "scope:src/api" to limit scope | "dry-only" | "dead-only" | "logic-only"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Simplify

Deep codebase simplification. Dispatches three specialized agents in parallel — each targeting a specific class of code smell — then consolidates their work into a clean, tested result.

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`full`** — scan the entire codebase, not just recent changes. Default behavior scans files changed since the last simplify commit (or last 10 commits if no prior simplify).
- **`scope:<path>`** (e.g., `scope:src/api`) — limit all agents to files under this path.
- **`dry-only`** — only run the DRY agent.
- **`dead-only`** — only run the dead code agent.
- **`logic-only`** — only run the logic simplification agent.

These can be combined: `full dry-only` means "scan entire codebase but only look for DRY violations."

## 1. Determine scope

### Default scope (no `full` or `scope:` argument)

Find the last simplify commit:

```
git log --oneline --grep="simplify:" -1
```

If found, scope = all files changed since that commit. If not found, scope = files changed in the last 10 commits:

```
git diff --name-only HEAD~10
```

### `full` scope

All source files in the project. Exclude: `node_modules/`, `vendor/`, `dist/`, `build/`, `.git/`, lock files, binary files, generated files (`.pb.go`, `.generated.*`, etc.).

### `scope:<path>` scope

All source files under that path.

In all cases, read the project's `.gitignore` and respect it. Also read the project structure (package.json, Cargo.toml, go.mod, etc.) to understand the language and framework — this informs what "idiomatic" means for each agent.

Report the scope:

> **Simplify scope:** N files across M directories
> Mode: [default | full | scoped to `<path>`]
> Agents: [all | dry-only | dead-only | logic-only]

## 2. Create progress tracker

Use `TaskCreate` to create a parent task for the simplify run, then child tasks for each agent pass and the consolidation step.

## 3. Dispatch agents in parallel

Launch the following agents **in parallel** (single message, multiple Agent tool calls). Each agent receives:

- The list of in-scope file paths
- The detected language/framework
- Instructions to **make changes directly** (Edit tool), not just report

If a `*-only` argument was given, launch only that agent. Otherwise launch all three.

---

### Agent 1: DRY Pass (`subagent_type: "code-simplifier"`)

**Prompt the agent with:**

> You are performing a **DRY (Don't Repeat Yourself) pass** on this codebase.
>
> **Your mission:** Find duplicated patterns and extract them into reusable helpers.
>
> **In-scope files:** [list of file paths]
> **Language/framework:** [detected]
>
> **What to look for:**
>
> 1. **Duplicated code blocks** — 3+ lines that appear in 2+ places with minor variations. Extract into a shared function/method.
> 2. **Repeated patterns** — Similar structures (e.g., error handling wrappers, API response formatting, validation chains, data transformation pipelines) that follow the same shape. Create a generic helper.
> 3. **Copy-paste with parameter differences** — Functions that are near-identical except for a few values. Parameterize into one function.
> 4. **Repeated type constructions** — Identical or near-identical type/struct/interface definitions across files. Consolidate into a shared types file.
> 5. **Boilerplate sequences** — Setup/teardown patterns, configuration scaffolding, middleware chains that repeat. Extract into a builder or factory.
>
> **Rules:**
>
> - Only extract when the pattern appears **3+ times** OR when 2 instances are **10+ lines each**. Three similar lines of code is better than a premature abstraction.
> - Place helpers in the most logical location: a `utils/`, `helpers/`, `shared/`, or `common/` directory that already exists, or adjacent to the most common caller if no such directory exists. Never create a new top-level directory without checking the existing project structure.
> - Name helpers descriptively — `formatApiResponse()` not `helper1()`. The name should make the abstraction's purpose obvious.
> - Preserve all existing behavior exactly. Every extraction must be a pure refactor — same inputs, same outputs, same side effects.
> - Update all call sites to use the new helper. Do not leave any instance of the old duplicated code behind.
> - Add a brief inline comment on the helper explaining what pattern it replaces, **only** if the function name alone doesn't make it obvious.
> - Run the project's linter after making changes (if a lint script exists).
>
> **Output:** List every helper you created, where you placed it, and which call sites you updated. Format:
>
> ```
> CREATED: src/utils/formatApiResponse.ts — extracted from 4 call sites
>   - src/routes/users.ts:42
>   - src/routes/posts.ts:38
>   - src/routes/comments.ts:55
>   - src/routes/auth.ts:71
> ```

---

### Agent 2: Dead Code Pass (`subagent_type: "code-simplifier"`)

**Prompt the agent with:**

> You are performing a **dead code removal pass** on this codebase.
>
> **Your mission:** Find and remove code that is never executed or referenced.
>
> **In-scope files:** [list of file paths]
> **Language/framework:** [detected]
>
> **What to look for:**
>
> 1. **Unused imports/requires** — Modules imported but never referenced in the file.
> 2. **Unused variables and parameters** — Declared but never read. For function parameters, only remove if the function is internal (not part of a public API or interface contract).
> 3. **Unreachable code** — Code after unconditional return/throw/break/continue, dead branches in conditionals (e.g., `if (false)`).
> 4. **Unused functions/methods** — Defined but never called anywhere in the codebase. Search globally before removing — check for dynamic references, reflection, decorators, and route handlers.
> 5. **Unused exports** — Exported symbols that are not imported anywhere else in the project. Be cautious with library code or public APIs — only remove if you can confirm no external consumers exist (i.e., this is an application, not a published package).
> 6. **Stale feature flags and commented-out code** — Code behind permanently-false flags or blocks of commented-out code with no TODO/FIXME annotation.
> 7. **Orphaned files** — Files that are not imported, required, or referenced by any other file in the project (excluding entry points, config files, and test files).
>
> **Rules:**
>
> - **Search globally** before removing anything. Use Grep to verify a symbol is truly unused across the entire project, not just the current file.
> - **Do not remove** test files, config files, entry points, migration files, or files referenced in build configs.
> - **Do not remove** code that is used via reflection, dynamic imports, decorators, or framework conventions (e.g., Next.js page files, Django view functions registered in urls.py, Express route handlers).
> - **Do not remove** interface implementations or trait impls even if the method appears "unused" — it satisfies a contract.
> - When removing an export, check if removing it changes the module's public API in a breaking way.
> - Run the project's linter after making changes (if a lint script exists).
>
> **Output:** List everything you removed and why. Format:
>
> ```
> REMOVED: src/utils/oldParser.ts — unused, 0 importers found
> REMOVED: src/routes/users.ts:5 — unused import `lodash.merge`
> REMOVED: src/models/legacy.ts:42-67 — unreachable code after early return on line 41
> KEPT: src/utils/serialize.ts — appears unused but is loaded dynamically via config.plugins
> ```

---

### Agent 3: Logic Simplification Pass (`subagent_type: "code-simplifier"`)

**Prompt the agent with:**

> You are performing a **logic simplification pass** on this codebase.
>
> **Your mission:** Simplify overly complex logic without changing behavior.
>
> **In-scope files:** [list of file paths]
> **Language/framework:** [detected]
>
> **What to look for:**
>
> 1. **Nested conditionals** — Deeply nested if/else chains (3+ levels) that can be flattened with early returns, guard clauses, or extracted predicates.
> 2. **Overcomplicated boolean expressions** — Expressions like `if (x === true)`, `if (!(!a))`, `condition ? true : false`, or long chains of `&&`/`||` that can be simplified or named.
> 3. **Unnecessary abstractions** — Wrapper functions that just forward arguments, classes with one method that could be a function, inheritance hierarchies that add complexity without value.
> 4. **Verbose patterns with idiomatic alternatives** — Manual loops that could be `map`/`filter`/`reduce`, verbose null checks that could use optional chaining, manual error propagation that could use `?` (Rust) or equivalent.
> 5. **Redundant type assertions/casts** — Casts that don't change the type, `as any` that can be replaced with proper typing, unnecessary generics.
> 6. **Over-defensive code** — Null checks on values that can never be null (e.g., just assigned), try/catch around code that can't throw, validation of internal invariants that are already guaranteed by the type system or caller.
> 7. **Switch/match with single meaningful arm** — Switch statements where all but one case does the same thing (use an if instead).
>
> **Rules:**
>
> - Every simplification must be **behavior-preserving**. If you're not 100% certain the transformation is safe, skip it.
> - Prefer **readability** over cleverness. A clear three-line `if` block is better than a dense one-liner.
> - Do not refactor working code just because you'd write it differently. Only simplify when the current version is objectively harder to understand.
> - When flattening nested conditionals, preserve the same logical paths — don't accidentally invert or drop a condition.
> - Run the project's linter after making changes (if a lint script exists).
>
> **Output:** List every simplification with before/after. Format:
>
> ```
> SIMPLIFIED: src/auth/validate.ts:28-45 — flattened 4-level nested conditional to 3 guard clauses
>   Before: 18 lines, max nesting depth 4
>   After:  12 lines, max nesting depth 1
>
> SIMPLIFIED: src/utils/parse.ts:102 — replaced manual null chain with optional chaining
>   Before: config && config.settings && config.settings.theme
>   After:  config?.settings?.theme
> ```

---

## 4. Consolidate results

After all agents complete, collect their outputs. Check for conflicts:

- **Overlapping edits** — If two agents modified the same file, read the file and verify both sets of changes are compatible. If they conflict, prefer the DRY pass (extractions) over dead code removal, and dead code removal over logic simplification — the more structural change wins.
- **Broken references** — If the dead code agent removed something the DRY agent extracted into a helper, that's a conflict. Re-read the file and resolve manually.

Report the consolidated results:

```
═══════════════════════════════════════
  Simplify — Results
═══════════════════════════════════════

DRY Pass:
  Helpers created: 4
  Call sites updated: 17
  Files touched: 12

Dead Code Pass:
  Items removed: 23
  Lines saved: ~340
  Files deleted: 2

Logic Pass:
  Simplifications: 11
  Nesting reduced: avg 3.2 → 1.4 levels
  Files touched: 8

Conflicts resolved: 1 (overlapping edit in src/routes/users.ts)
```

## 5. Verify

Run the project's full validation suite:

1. **Linter** — run the lint script if it exists. Fix any issues introduced by the agents.
2. **Tests** — run the test suite if it exists. If any test fails:
   - Identify which agent's change caused the failure.
   - Revert that specific change (use `git diff` to isolate it).
   - Report the reverted change so the user knows what was skipped.
3. **Build** — run the build script if it exists. Fix any build errors.

If no lint/test/build scripts exist, do a manual sanity check: read the most heavily modified files and verify they still make sense.

## 6. Summary

Print the final summary using the standard output format:

```
═══════════════════════════════════════
  Simplify — Complete
═══════════════════════════════════════

  Helpers created:     4
  Dead code removed:   21 items (~310 lines)
  Logic simplified:    11 locations
  Total files touched: 24
  Tests:               passing

  New helpers:
    • src/utils/formatApiResponse.ts (4 call sites)
    • src/utils/withRetry.ts (3 call sites)
    • src/utils/validatePagination.ts (5 call sites)
    • src/middleware/requireAuth.ts (6 call sites)

  Reverted: 1 change (test regression in auth module)

  Done:
    ✓ DRY pass — 4 helpers created, 17 call sites updated
    ✓ Dead code pass — 21 items removed (~310 lines)
    ✓ Logic pass — 11 simplifications
    ✓ Lint — clean
    ✓ Tests — passing

  Next: /ship when ready

═══════════════════════════════════════
```

If running inside `/autopilot`, minimize output to just the summary block.

## Error handling

- If an agent fails or times out, report which agent failed and continue with the others' results.
- If the test suite fails after all attempted fixes, revert all changes from the current simplify run (`git checkout -- .`) and report what went wrong. Do not ship broken code.
- If there are no files in scope (e.g., no changes since last simplify), report "Nothing to simplify" and exit cleanly.

## Style guidelines

- Follow the standard output format in _output-format.md
- Be concise — the user wants results, not commentary on each decision.
- Use the structured output formats above so results are scannable.
- Bold the numbers in summaries for quick scanning.
- When running inside autopilot, minimize output to just the summary block.
