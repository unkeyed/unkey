---
name: dave-ship
description: Run the full pre-PR pipeline — refactor pass, parallel review agents, auto-fix safe issues, build/fmt verify, surface remaining design decisions one at a time, then open a draft PR. Use when implementation feels done and you want to wrap up.
disable-model-invocation: true
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash, Agent, Skill, TaskCreate, TaskUpdate]
---

# Ship

You've finished implementation. This walks the diff through refactor → review → checkpoint → PR. The **checkpoint is the load-bearing part**: surface decisions one at a time so the user can react cleanly. Do not dump everything at once.

## Steps

### 1. Snapshot the diff

```bash
git status --short
git diff HEAD
git log $(git merge-base HEAD origin/main 2>/dev/null || git merge-base HEAD origin/master)..HEAD --oneline
gh pr list --head "$(git branch --show-current)" --json number,url
```

- If working tree is clean and no commits ahead of base: abort with "nothing to ship."
- If `gh pr list` returns a PR: this is an **update**, not a new PR. Note it; step 8 changes accordingly.

### 2. Cleanup passes

Run these auto-fixing passes **in order**, in the main thread. Each applies fixes inline; record anything a pass flags-but-doesn't-fix for the checkpoint (step 7). Order matters: later passes assume earlier ones ran, so each owns a distinct scope and **must not re-litigate what an earlier pass owns**.

| Order | Skill                         | When                                               | Owns (and only this)                                                              |
| ----- | ----------------------------- | -------------------------------------------------- | --------------------------------------------------------------------------------- |
| 1     | `refactor`                    | Always                                             | Mechanical structure: extract duplication, flatten nesting, tighten signatures, fix naming. No behavior change. |
| 2     | `self-review`                 | If project-local (e.g. Unkey's `.claude/skills/`)  | Quality-doc compliance + entropy: testing gaps, doc standards, shortcuts. **Skip structural nits — refactor already owns those.** |
| 3     | `dave-react-best-practices`   | `git diff --name-only HEAD` contains a `.tsx` file | React-specific quality per the BP checklist.                                      |
| 4     | `dave-rams`                   | `git diff --name-only HEAD` contains a `.tsx` file | Accessibility (WCAG) and visual design on touched components.                     |

The `dave-deslop` pass (AI-slop removal) runs later, in step 5 — after review fixes land — so it cleans the cumulative result instead of slop these passes might reintroduce.

Briefly report what each pass changed (1-3 sentences total).

### 3. Parallel review (read-only agents)

Launch **two `general-purpose` sub-agents in parallel — in one message**. Both are read-only: instruct them explicitly **not to call Edit, Write, or NotebookEdit**. Fixes happen in step 5 after triage, not inside the agents.

Shared prompt prefix:

> **Read-only review.** Do not call Edit, Write, or NotebookEdit — this is analysis only. Run `git diff HEAD` to see the diff. Read whichever of `AGENTS.md`, `CLAUDE.md`, and the docs they reference exist, for project standards. Be conservative — pure style nits and hypothetical "future risk" don't count. Cap output at 25 findings.
>
> Return findings in this exact format, one per line:
>
> ```
> <file>:<line> | <high|med|low> | <one-line issue> | <one-line suggested fix>
> ```

Give each agent its checklist from [REVIEW-CHECKLISTS.md](REVIEW-CHECKLISTS.md):

- **Agent A — Bugs & contracts**: null derefs, races, off-by-one, broken caller contracts, missing switch cases, behavior regressions, back-compat, edge cases.
- **Agent B — Patterns & UX consistency**: reuse of UI primitives, design tokens, interaction/loading/empty states, responsive + keyboard nav, copy voice, motion, naming, no adjacent-UI regressions.


### 4. Triage findings

Walk each finding from step 3's agents:

- **Auto-apply** if `high` severity + no design implication (concrete bug with an obvious fix, missing null check on internal data, clear security issue with a one-line patch).
- **Defer to checkpoint** if it needs a design call, has a UX trade-off, or risks behavior change.
- **Drop** if it's a low-confidence "future risk" with no clear failure mode.

When in doubt, defer. Skills in step 2 already applied their own fixes — this step is only for the agents' findings.

### 5. Deslop pass

Invoke the `dave-deslop` skill via the Skill tool. It runs **last among the auto-fixing passes**, after refactor, the cleanup skills, and triage fixes have all touched the diff — so it strips AI slop from the cumulative result, including slop those earlier passes may have introduced. It targets: gratuitous comments a human wouldn't write, defensive checks / try-catch abnormal for the area, `any`-casts that paper over types, and style inconsistent with the file. Apply fixes inline; report a 1-3 sentence summary.

### 6. Verify

Run the project's formatter and build for the touched files. Detect commands from `AGENTS.md` / `CLAUDE.md` / `package.json` / file extensions; in monorepos, filter to the changed package(s). **If the build fails, stop and fix before continuing** — do not enter the checkpoint with a broken build.

### 7. Checkpoint — one at a time

Aggregate:
- Findings deferred in steps 2 and 4
- Open design questions from the prior conversation (unresolved grilling threads, deferred decisions)
- Untested edge cases worth flagging in the PR

Present them **one at a time**, prefixed `[N of M]`. For each:
- **The question/issue** in 1-2 sentences
- **Why it matters** — 1 short line
- **Suggested action** — your recommended fix, or "your call" if purely design
- Then **wait** for the user before moving on

Responses: "fix it" / "yes" → apply, continue. "skip" / "leave it" / "note in PR" → record for PR Notes. A different fix → apply theirs, continue.

After all items: brief final summary (auto-applied count; checkpoint decisions; PR-notes items), then ask: **"Ready to ship?"**

### 8. Ship

Once the user confirms:

- **New PR**: invoke the `dave-write-pr` skill. Pass it the flagged-for-PR items so they land in the **Notes for reviewers** section.
- **Existing PR** (detected in step 1): commit the session's changes with a focused message (what changed — no "address review feedback" meta), `git push`, and post any newly-surfaced design questions as a PR comment (don't edit the body).

## Anti-patterns

- **Dumping all findings at once.** One at a time is the whole point of the checkpoint.
- **Auto-fixing things that need a design call.** Defer.
- **Sequential when parallel works.** Step 3's two agents go in one message.
- **Letting review agents edit code.** Step 3 is read-only — fixes happen in step 4 after triage.
- **Calling `dave-write-pr` before the user says "ready to ship".**
- **Hard-coding build commands** — detect them, don't assume.

## When to NOT use this skill

- The user said "just commit and push" — respect that.
- Branch is mid-implementation / broken.
- No diff exists.
