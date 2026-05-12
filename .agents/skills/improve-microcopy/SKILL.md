---
name: improve-microcopy
description: Audit UI microcopy in the current branch diff and propose concrete rewrites. Use after writing dashboard work to review button labels, error messages, empty states, dialog copy, form descriptions, tooltips, and toasts. Audit-only — does not edit files.
disable-model-invocation: true
allowed-tools: [Read, Glob, Grep, Bash]
---

# Improve microcopy

Audit user-facing strings introduced or modified in the current branch and report issues with suggested rewrites. **Do not edit files.** The user reads the report and applies what they want.

Microcopy is short UI text (typically under three sentences) that users actually read: button labels, error messages, empty states, form labels and descriptions, placeholders, toast notifications, dialog content, tooltips, confirmation prompts. It supports scanning and is read more often than longer copy — every word counts. Code comments, log lines, and test fixtures are not microcopy.

## Scope

- Source: `git diff main...HEAD` — only added or modified lines.
- Files: `.tsx` and `.ts` under `web/apps/dashboard/`.
- Skip: `*.test.tsx`, `*.test.ts`, `__tests__/`, `*.stories.tsx`, code comments, log/console messages, anything that never renders to a user.
- Do not flag pre-existing copy that wasn't touched in this branch.

## Process

1. `git diff main...HEAD --name-only` → filter to dashboard tsx/ts files.
2. Read each changed file. Extract user-facing strings from added/modified lines: JSX text nodes, `placeholder=`, `title=`, `label=`, `description=`, `aria-label=`, `toast.*()`, `confirm*Text=`, error/success message literals, alert/dialog content.
3. Run **Pass 1: hard rules**. Mechanical, high confidence.
4. Run **Pass 2: taste pass**. Subjective, lower confidence.
5. Emit a single grouped report.

If no user-facing strings changed, say so and stop.

## Voice — the house tone

Unkey's microcopy is **Stripe/Vercel-direct**: confident, dry, specific. The reader is a developer. Plain past tense for success ("Key created"). Active voice with a named actor for failures ("We couldn't…"). No exclamation marks except on first-time success moments (account, workspace, first key). No emojis. No interjections ("Oops", "Hooray"). No apologetic "please".

## Classify each string first

Every piece of microcopy has one of three primary goals (NN/g framework). Identify which one before judging — the standard you hold a string to depends on it.

- **Inform** — communicate facts, status, or consequences. Empty states, error messages, helper text, warning boxes, async-behavior notes, system notifications, success toasts, dialog body copy. Standard: specific, scannable, accurate, names the actual thing. Most dashboard microcopy is Inform.
- **Interact** — tell the user what an element does or capture an action. Button labels, link text, checkbox labels, tab labels, placeholders, breadcrumbs, menu items. Standard: names the action and the object; zero ambiguity about what happens on click.
- **Influence** — drive a conversion or build emotional brand connection. Rare in a product dashboard — mostly billing/upgrade and first-run onboarding. Standard: clear value, honest, never overwrought.

A string can serve more than one goal — pick the dominant one. A confirm-dialog title ("Confirm permission deletion") is Inform-first. A "Get started" CTA in onboarding is Influence-first. A "Delete permission" button is Interact-first even though it also Informs.

**If you find Influence copy in product UI outside billing/upgrade/onboarding, flag it.** Developers use this dashboard daily — they don't need to be sold at every step.

## Pass 1: hard rules

Each finding here is objective. Always propose a concrete rewrite.

### Sentence case everywhere
All UI text is sentence case — buttons, dialog titles, tab labels, table headers, section headings. Proper nouns stay capitalized (Vercel, GitHub, Unkey, API, ID).

- Bad: `"Failed to Create Key"`, `"Delete Identity"`, `"Create New Root Key"`
- Good: `"Failed to create key"`, `"Delete identity"`, `"Create new root key"`

### Banned phrases
Zero tolerance — flag every occurrence:

- `Oops`, `Whoops`, `Uh oh`, `Yikes`
- `Hooray`, `Yay`, `Awesome`, `Great!`
- `Something went wrong` with no specifics
- `An unexpected error occurred`
- Bare `Please try again` with no cause or next step
- `Are you sure?` alone (must name what)

### Error structure
Applies to **Inform** strings (errors, system messages). Every error message must include at least two of:

1. **What** action failed (specific verb + object)
2. **Why** it failed (cause or condition, when knowable)
3. **What next** the user should do (retry, fix something, contact support)

Use active voice with a named actor when *we* failed:

- Bad: `"Your password couldn't be changed"` — passive, no actor, no reason
- Good: `"We couldn't change your password — it must be at least 8 characters"`

### Contextual labels
Applies to **Interact** strings (buttons, links, menu items). Flag bare action verbs that don't say what they act on when context is ambiguous (dialogs with multiple buttons, dense forms):

- `Submit`, `Confirm`, `Continue`, `Next`, `Save`, `Cancel` standing alone — usually flag
- Prefer: `Save changes`, `Create key`, `Delete identity`, `Cancel rotation`

Exception: a single-button wizard step where the parent context makes the object obvious.

### Vague filler
Flag and propose removal:

- `please` (allow at most one apologetic use per dialog; never in errors)
- `very`, `extremely`, `really`, `actually`, `simply`, `just`
- `successfully` in success toasts is usually redundant: `"Member removed successfully"` → `"Member removed"`

### Destructive-action template
Any dialog with a destructive primary action (delete, revoke, rotate, regenerate, cancel subscription) must include:

- **What is lost or changed** — "Deleting this identity removes its metadata and ratelimits."
- **Scope clarification** when applicable — "Associated keys will not be affected."
- **Irreversibility note** when applicable — "This action cannot be undone."
- **Typed-name confirmation** for high-impact actions (delete API, delete namespace, delete project, delete workspace).

Flag dialogs missing any element that applies.

## Pass 2: taste pass

Subjective. Mark each finding `[taste]`. Use judgment; bias toward fewer, higher-confidence flags.

### Specificity
Name the actual thing. Not "the item", "this entry", "your data".

- Weaker: `"Save failed"`
- Stronger: `"We couldn't save your changes"`

### Expectations
Set async behavior, irreversibility, and one-time disclosures **before** the click, not after:

- `"You'll see this secret only once."`
- `"Changes may take up to 60s to propagate."`
- `"The current key remains valid for the grace period you select."`

Flag dialogs and forms that perform consequential actions without these.

### Empty states
Applies to **Inform** strings. An empty state must answer two things:

1. **What is this surface?** ("Verification logs show each time this key was used.")
2. **What can the user do here?** (CTA or docs link.)

Bare "No data" or "Nothing here yet" without both — flag.

### Helper text and field descriptions
Applies to **Inform** strings beside form fields. One sentence. Explain **why**, not what — the label already says what.

- Weak: label `"Name"`, description `"A name to identify this rate limit rule"` — adds nothing.
- Good: label `"Workspace name"`, description `"Not customer-facing. Choose a name that is easy to recognize."`

### Brevity
Microcopy can't rescue a confusing UI. Flag instructional copy longer than ~2 sentences that explains "how to use this" — that's a sign the interaction itself needs work, not more words.

### Local terminology consistency
If the diff itself uses two different terms for the same concept on the same surface, flag it:

- `"API key"` and `"key"` interchangeably in one dialog
- `"Identity"` and `"user"` for the same entity

Do **not** validate against a global canonical glossary — out of scope.

## What NOT to flag

- Pre-existing strings outside the diff
- Log messages, console output, monitoring labels, error codes
- Strings in test files or stories
- Code identifiers, route paths, type names
- aria-labels that duplicate visible text for screen readers (judge whether the label is also user-visible)
- Marketing/landing copy — this skill is for product UI only

## Report format

Output one markdown report. Group by file. Within each file: hard findings first, then taste findings.

```
### web/apps/dashboard/components/foo/bar.tsx

**[hard] Sentence case** — line 42
Current: "Failed to Create Key"
Suggested: "Failed to create key"

**[hard] Vague error** — line 87
Current: "Something went wrong. Please try again."
Suggested: "We couldn't save your changes — check your connection and retry."
Reason: name what failed, name the next step.

**[taste] Helper text adds nothing** — line 120
Field: label "Name" + description "A name to identify this rate limit rule"
Suggested: drop the description, or explain why naming matters ("Shown in logs and the dashboard — keep it human-readable.")
```

End with one line: `N hard, M taste findings across X files.`
