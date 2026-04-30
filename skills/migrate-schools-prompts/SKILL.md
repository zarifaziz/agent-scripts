---
name: migrate-schools-prompts
description: Port one lesson-plan slide prompt at a time from the legacy resources-repo `.go` template into the modality `prompts/resource_types/*.jinja` template. Reads the migration kit at `.tmp/prompt-migration/`, applies the canonical recipe, runs snapshot tests, captures any new lessons back into the recipe, then commits. Use when the user says "port the next slide", "migrate the next prompt", "do the next slide", or names a specific slide like "port exit_ticket".
---

# Migrate Schools Prompts

Drives the per-slide port from `~/Coding/metarepo/backend/app/resources/internal/lesson_plan/prompts/<slide>.go` (legacy mustache) into `<modality-worktree>/prompts/resource_types/<slide>.jinja` (modality jinja2). One slide per invocation, one commit per slide.

## Working Directory

Must be inside the modality worktree where the migration kit lives:

```
<modality-worktree>/.tmp/prompt-migration/
├── plan.md
├── porting-recipe.md
├── examples/
│   ├── english-bank.md
│   └── science-bank.md
└── slides/
    ├── warm_up_questions.md
    ├── exit_ticket.md
    ├── contemplative.md
    ├── activity.md
    ├── scaffolded.md
    ├── multiple_choice.md
    ├── short_answer.md
    ├── situational.md
    ├── misconception.md
    ├── brain_teaser.md
    ├── thought_sparker.md
    └── real_example.md
```

If `.tmp/prompt-migration/` is missing, stop and tell the user — the kit must exist before porting can start.

## Workflow

### Step 1 — Locate the kit and read state

```bash
ls .tmp/prompt-migration/slides/
```

Then read these files in order:

1. `.tmp/prompt-migration/plan.md` — confirms scope, decision log, and the **Status checklist** (see Step 2)
2. `.tmp/prompt-migration/porting-recipe.md` — apply this recipe to every slide; sections §4a (pre-flight scrub) and §4b (verification) are the load-bearing parts

### Step 2 — Pick the next slide

Look at `plan.md` for the **Status checklist** section. Each slide has one of: `[ ]` not started, `[~]` in progress, `[x]` done.

If no checklist exists yet, derive state from `git log --oneline` in the worktree — slides ported land as commits with messages like `feat(prompts): port <slide_name> from legacy go template`. The first slide *without* such a commit (in the order P1 → P2 from `plan.md`) is the next one.

If the user names a specific slide (e.g. "port exit_ticket"), use that one regardless of order.

### Step 3 — Read the slide-specific guide

```bash
cat .tmp/prompt-migration/slides/<slide>.md
```

This contains:
- Legacy file path
- Schema reminder (output keys)
- What to add (`## Target Skills`, `## Lesson Brief`)
- Math-only items to strip
- Slide-specific edits
- 7 examples (3 math + 2 English + 2 science) ready to paste

### Step 4 — Read the legacy source

```bash
cat ~/Coding/metarepo/backend/app/resources/internal/lesson_plan/prompts/<slide>.go
```

Note the mapping: most legacy filenames match modality names, except:
- `application.go` ↔ `real_example.jinja`
- `warm_up_with_context.go` ↔ `warm_up_questions.jinja`

### Step 5 — Read the current jinja state

```bash
cat prompts/resource_types/<slide>.jinja
```

Identify what's already there (canonical shell from the recipe §2) so the port is additive — don't rebuild scaffolding that exists.

### Step 6 — Apply the canonical template

Follow `porting-recipe.md` §2 (canonical jinja shape) and §4a (pre-flight scrub checklist). The structural shell is:

```jinja
{%- include "partials/class_year.jinja" -%}
## Role
{% if skill_titles -%}
## Target Skills
{% endif -%}
## Lesson Brief
## Question Design   ← or "Activity Design", "Discussion Design"
  ### Self-Containment
  ### Question Quality Rules
  ### Self-check before emit
  ### Answers
## Examples              ← 7 examples from the slide guide
## Writing
{% include "partials/writing_locale.jinja" %}
- <slide-specific writing bullets>
{% include "partials/guidelines/katex.jinja" %}
{% include "partials/guidelines/conciseness.jinja" %}
{% if grade_band == "Prep–Grade 2" %}
{% include "partials/guidelines/grade_prep_to_2.jinja" %}
{% elif grade_band == "Grades 3–6" %}
{% include "partials/guidelines/grade_3_to_6.jinja" %}
{% elif grade_band == "Grades 7–10" %}
{% include "partials/guidelines/grade_7_to_10.jinja" %}
{% endif %}
```

### Step 7 — Pre-flight scrub (the silent-failure catchers)

After every paste from legacy, run this grep:

```bash
grep -nE '\{[A-Z]|\{\{[A-Z]' prompts/resource_types/<slide>.jinja
```

Expect zero matches except KaTeX expressions (`\text{H}^+`, `\text{CO}_2`). Anything else is a leak.

Common leaks to scrub (per recipe §4a):

| Pattern | Action |
|---|---|
| `{{ClassYear}}`, `{{Country}}` (mustache double-brace) | Drop. Partials handle these. Jinja silently emits empty for these in modality's relaxed-render config — broken but invisible. |
| `{Skill}`, `{Skills}`, `{Context}` (single-brace) | Replace literally — `{Skill}` → "the target skill", `{Context}` → "the lesson theme". Jinja matches only `{{ }}`, so single-brace renders verbatim to the LLM. |
| `{{#GradeLevel_PrepTo2}}...{{/GradeLevel_PrepTo2}}` | Convert to `{% if grade_band == "Prep–Grade 2" %}...{% endif %}`. |
| `{{#Context}}...{{/Context}}` / `{{^Context}}...{{/Context}}` | Drop both branches. |
| `{{#RelevantContentExamples}}...{{/RelevantContentExamples}}` | Drop — DB-injected examples not plumbed in modality. |

After bulk `replace_all` of `{Skill}` → "the target skill", grep for double-ups:

```bash
grep -E "the the |the current the " prompts/resource_types/<slide>.jinja
```

Fix any matches manually.

### Step 8 — Strip math-only language

Common math-only fragments to genericise (per recipe §4):

| Legacy phrase | Replacement |
|---|---|
| "must yield a numeric or algebraic answer expressible in KaTeX" | Dual-format rule: "Use KaTeX `$...$` for numeric/algebraic answers; use plain text for word answers." |
| "{{ClassYear}} maths teacher" | "school teacher" |
| "valid math skills" / "mathematical concepts" / "mathematical knowledge" | "valid skills" / "concepts" / "knowledge" |
| "Show how math helps achieve the goal" | "Show how the lesson skill helps achieve the goal" |

Keep math-flavoured rules that are still subject-neutral ("Avoid circular logic", "Questions end with `?`").

### Step 9 — Don't re-paste partial-covered blocks

These legacy whole-blocks are already covered by partials at the bottom of the canonical template. Pasting them duplicates output:

| Legacy block | Covered by |
|---|---|
| `# WRITING\n- Write in {{Country}} English...\n- Write at a {{ClassYear}} reading level...` | `partials/writing_locale.jinja` |
| `# CONCISENESS GUIDELINE:\n- Students are in school all day...` | `partials/guidelines/conciseness.jinja` |
| `# KaTeX Guideline...` | `partials/guidelines/katex.jinja` |
| Any `<root>...</root>` XML output spec | Tool-call schema (modality is structured-output, not XML) |
| `<Slide>EditInstructionTemplate` / `SelectionSystemTemplate` | Out of scope — separate stages |

Lines from legacy `# WRITING` that ARE worth keeping (because they're not in the partials):
- "Ensure scenarios are realistic and possible."
- "Verify all context-specific facts are accurate."

These belong in the bottom `## Writing` block, *under* the `{% include "partials/writing_locale.jinja" %}` line.

### Step 10 — Verify (the build-cache trap)

`cargo test` does **not** reliably rebuild after `.jinja` changes — templates are baked in via `include_str!` and Cargo's dependency graph for `concat!()`-derived paths is unreliable. **You must touch the prompt mod to force rebuild:**

```bash
touch crates/platform/ai/src/prompt/mod.rs
cargo test -p modality_ai -- prompt::tests::<slide>
```

Snapshots will fail on first run after edits — that's expected. Accept them in batch:

```bash
cargo insta test -p modality_ai --accept --test-runner=cargo-test -- prompt::tests::<slide>
```

The `composed_slide_kinds` test is special — it loops over every slide and short-circuits on first failure. Use:

```bash
cargo insta test -p modality_ai --accept --test-runner=cargo-test -- composed_slide_kinds
```

After accepting, **re-run the test to confirm green**:

```bash
touch crates/platform/ai/src/prompt/mod.rs
cargo test -p modality_ai -- prompt::tests::<slide>
```

If a partial (e.g. `partials/writing_locale.jinja`) changed, every `composed_whiteboard_<kind>.snap` drifts simultaneously. Accept the full sweep:

```bash
cargo insta test -p modality_ai --accept --test-runner=cargo-test
```

Eyeball one or two diffs to confirm the change matches intent before committing.

### Step 11 — Eyeball the rendered snapshot

Open the rendered prompt and skim it as if you were the LLM:

```bash
cat crates/platform/ai/src/prompt/snapshots/modality_ai__prompt__tests__composed_whiteboard_<slide>.snap | less
```

Checklist:
- [ ] `## Role` block renders subject-agnostic ("school teacher", not "maths teacher")
- [ ] `## Target Skills` shows the test fixture's `skill_titles`
- [ ] `## Lesson Brief` either renders the brief or the no-brief default
- [ ] All 7 examples render (3 math + 2 English + 2 science)
- [ ] No literal `{Skill}`, `{Context}`, `{{ClassYear}}`, `{{Country}}` in the output
- [ ] No duplicate writing/conciseness sections
- [ ] Grade-band partial at the bottom matches the test fixture's `grade_band`

### Step 12 — Capture new lessons back into the recipe

If during the port you encountered:

- A pattern not covered by `porting-recipe.md` §4a (e.g. a new mustache fragment, a new slide-specific gotcha)
- A new partial dependency you discovered
- A subject-agnostic phrasing that worked particularly well

…**add it to `.tmp/prompt-migration/porting-recipe.md`** before committing. The recipe is the durable artefact; the slide guides are the per-slide application of it.

If the lesson is slide-specific, add it to the next-up slide guide in `.tmp/prompt-migration/slides/<next>.md` so the next port doesn't re-trip on it.

### Step 13 — Commit and push

One commit per slide. Format:

```bash
git add prompts/resource_types/<slide>.jinja \
        crates/platform/ai/src/prompt/snapshots/modality_ai__prompt__tests__composed_whiteboard_<slide>.snap \
        crates/platform/ai/src/prompt/snapshots/modality_ai__prompt__tests__<slide>_*.snap

# If the recipe was updated, also stage:
git add .tmp/prompt-migration/porting-recipe.md  # only if .tmp/ is in .gitignore — see below

git commit -m "$(cat <<'EOF'
feat(prompts): port <slide_name> from legacy go template

- Subject-agnostic role framing
- 7 examples (3 math, 2 English, 2 science)
- Drops: <math-only items removed, e.g. KaTeX-mandate, ClassYear inline refs>
- Uses: skill_titles, generation_context, grade_band gating
EOF
)"
git push
```

**Note on `.tmp/`:** by default `.tmp/prompt-migration/` is untracked (an in-progress kit, not part of the PR). If the user wants the recipe updates committed, they need to either move the kit out of `.tmp/` or add `!.tmp/prompt-migration/porting-recipe.md` to a tracked file. Check with the user before staging anything from `.tmp/`.

### Step 14 — Update the status checklist

Open `.tmp/prompt-migration/plan.md` and tick the slide in the **Status checklist**. If no checklist exists yet, create one at the bottom of `plan.md`:

```markdown
## Status checklist

P1 — pattern setters (manual port by user):
- [x] warm_up_questions
- [ ] exit_ticket
- [ ] contemplative

P2 — Claude follows pattern:
- [ ] activity
- [ ] scaffolded
- [ ] multiple_choice
- [ ] short_answer
- [ ] situational
- [ ] misconception
- [ ] brain_teaser
- [ ] thought_sparker
- [ ] real_example
```

### Step 15 — Report

End the session with one paragraph for the user:

- Slide ported, commit SHA, files changed
- Snapshots accepted (count)
- Any lesson added to the recipe (with one-line summary)
- Next slide in queue

## Gotchas — read before every port

1. **Build cache lies.** `cargo test` after a `.jinja` edit will pass against stale embedded templates unless you `touch crates/platform/ai/src/prompt/mod.rs` first. This is the single most common silent failure.
2. **Mustache `{{X}}` renders to empty string.** Modality's jinja config doesn't error on undefined variables — it silently emits nothing. Tests pass with broken prompts. The pre-flight grep is the only catch.
3. **Single-brace `{Skill}` is NOT jinja.** It renders as the literal string "{Skill}" to the LLM. This is the second-most-common silent failure.
4. **Partials are auto-included.** Don't paste `# WRITING`, `# CONCISENESS GUIDELINE`, or `# KaTeX Guideline` blocks from legacy — the partials at the bottom of the template already inject them.
5. **`composed_slide_kinds` short-circuits.** Plain `cargo test` reports only the first slide-kind mismatch. Use `cargo insta test --accept` to iterate over all of them.
6. **Partial changes propagate.** Editing `partials/writing_locale.jinja` (or any partial) drifts ALL slide composed snaps. Expect ~50 modified `.snap` files; accept them as a single sweep.
7. **PR #320 is the home for these commits.** Don't open a new PR per slide.

## Reference: recipe + plan locations

| File | Purpose |
|---|---|
| `.tmp/prompt-migration/plan.md` | Scope, decision log, status checklist |
| `.tmp/prompt-migration/porting-recipe.md` | Canonical jinja shape, pre-flight scrub (§4a), verification recipe (§4b) |
| `.tmp/prompt-migration/examples/english-bank.md` | Vetted English Input/Output examples (Pool A/B/C) |
| `.tmp/prompt-migration/examples/science-bank.md` | Biology + chemistry examples (Pool BIO/CHEM) |
| `.tmp/prompt-migration/slides/<slide>.md` | Per-slide diff plan + 7 ready-to-paste examples |

If any of these are missing, stop and ask the user — don't improvise.
