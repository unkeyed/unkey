---
name: dave-write-pr
description: Create Unkey GitHub pull requests with conventional-commit titles and the repo's PR body template (what / type of change / how to test / checklist). Use when the user runs `gh pr create`, pushes a new branch with `-u origin`, asks to draft a PR, open a PR, push up for review, or update an existing PR body.
---

# Write PR

Creates Unkey PRs with titles that follow the `type(scope): summary` convention used across the repo, and bodies that match `.github/pull_request_template.md`.

## PR Title Format

```
<type>(<scope>): <summary>
```

### Types

| Type     | Description                            |
| -------- | -------------------------------------- |
| feat     | New feature                            |
| fix      | Bug fix                                |
| perf     | Performance improvement                |
| refactor | Code change (no fix or feature)        |
| docs     | Documentation only                     |
| test     | Adding/correcting tests                |
| chore    | Routine tasks, maintenance             |
| build    | Build system or dependencies           |
| ci       | CI configuration                       |

### Scopes (optional but recommended)

Pick the smallest accurate scope. Common scopes seen in this repo: `dashboard`, `ui`, `api`, `core`, `db`, `ctrl`, `pscale`, `frontline`, `networking`, `docs`, `skills`. Omit the scope only when the change genuinely spans many areas.

### Summary rules

- Imperative present tense: "Add" not "Added"
- Capitalize the first letter
- No trailing period
- No ticket IDs in the title — link them in the body
- Breaking change: add `!` before the colon → `feat(api)!: Remove v1 endpoints`

## Workflow

1. **Snapshot the diff.**
   ```bash
   git status --short
   git diff --stat
   git log $(git merge-base HEAD origin/main)..HEAD --oneline
   ```
   If nothing's there, stop.

2. **Pick the title parts.** Type from the change. Scope from the most-affected directory (`web/apps/dashboard` → `dashboard`, `web/internal/ui` → `ui`, `go/apps/api` → `api`, etc.). Summary in 5-10 words.

3. **If this is a security fix**: this repo is public, so audit every public-facing artifact (branch name, commit messages, PR title, PR body, Linear URL, test names, code comments) before pushing — see the **Security fixes** section below.

4. **Push the branch.**
   ```bash
   git push -u origin HEAD
   ```

5. **Create the draft PR** with the template from `.github/pull_request_template.md`:
   ```bash
   gh pr create --draft --title "<type>(<scope>): <summary>" --body "$(cat <<'EOF'
   ## What does this PR do?

   <One short paragraph: net change at the product level, then implementation in a second paragraph if needed. Include motivation.>

   Fixes # (issue)

   ## Type of change

   - [ ] Bug fix (non-breaking change which fixes an issue)
   - [ ] Chore (refactoring code, technical debt, workflow improvements)
   - [ ] Enhancement (small improvements)
   - [ ] New feature (non-breaking change which adds functionality)
   - [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
   - [ ] This change requires a documentation update

   ## How should this be tested?

   <Concrete scenarios a reviewer can run. Flag what wasn't tested and why.>

   ## Checklist

   ### Required

   - [ ] Filled out the "How to test" section in this PR
   - [ ] Read [Internal Workflow Guide](./CONTRIBUTING.md)
   - [ ] Self-reviewed my own code
   - [ ] Commented on my code in hard-to-understand areas
   - [ ] Ran `pnpm build`
   - [ ] Ran `pnpm fmt`
   - [ ] Ran `make fmt` on `/go` directory
   - [ ] Checked for warnings, there are none
   - [ ] Removed all `console.logs`
   - [ ] Merged the latest changes from main onto my branch with `git pull origin main`
   - [ ] My changes don't cause any responsiveness issues

   ### Appreciated

   - [ ] If a UI change was made: Added a screen recording or screenshots to this PR
   - [ ] Updated the Unkey Docs if changes were necessary
   EOF
   )"
   ```
   Only tick checklist boxes for work actually completed. Leave the rest unchecked — don't tick to look thorough.

## Body guidelines

- **What does this PR do?** — one short paragraph of prose, not bullets. Lead with the user-visible change; a second paragraph for any reusable primitive or pattern future PRs will care about. Include motivation. Link the GitHub issue with `Fixes #N` / `Closes #N` (auto-closes on merge).
- **Linear** — drop the URL inline if relevant: `https://linear.app/unkey/issue/[TICKET-ID]`. For security fixes, strip the slug (see below).
- **Screenshots** — for any UI change, attach light/dark/mobile, or leave an explicit `_To be added — light, dark, mobile._` placeholder. Don't silently drop the section.
- **Untested edge cases** — name them under "How should this be tested?" with why and likelihood.

## Security fixes

This repo is public. Never expose the attack vector in any public artifact. Describe what the code does, not what threat it prevents.

| Artifact     | Bad                              | Good                                |
| ------------ | -------------------------------- | ----------------------------------- |
| Branch       | `fix-sql-injection-in-webhook`   | `fix-webhook-input-validation`      |
| PR title     | `fix(api): Prevent SSRF`         | `fix(api): Validate outgoing URLs`  |
| Commit msg   | `fix: prevent denial of service` | `fix: add payload size validation`  |
| PR body      | "attacker could trigger SSRF…"   | "validates URL protocol and host"   |
| Linear ref   | URL with slug (leaks title)      | URL without slug, or ticket ID only |
| Test name    | `'should prevent SQL injection'` | `'should sanitize query parameters'`|

Before pushing, verify: branch name, commit messages, PR title, PR body, Linear URL, test names, and code comments give no hint of the vulnerability.

## Examples

| Title                                                       | Notes                              |
| ----------------------------------------------------------- | ---------------------------------- |
| `feat(dashboard): Add workflow performance metrics display` | Feature in dashboard               |
| `fix(core): Resolve memory leak in execution engine`        | Bug fix, single scope              |
| `feat(api)!: Remove deprecated v1 endpoints`                | Breaking change (`!` before `:`)   |
| `chore: Update dependencies`                                | No scope — spans many areas        |
| `refactor(core): Simplify error handling`                   | Refactor, no functional change     |

## Anti-patterns

- A "What does this PR do?" that just says "See ticket #123" or restates the title.
- Inventing section headings that aren't in the template.
- Implementation narration the diff already shows (file paths, hook names, library imports).
- Editorial commentary ("quietly opens the door for feedback", "designed to scale", "generic on purpose").
- Bullet-fragment summaries instead of prose.
- Ticking checklist boxes for work you didn't do.
- Including a Linear URL with the slug on a security fix.

## Validation

Unkey PR titles match:

```
^(feat|fix|perf|refactor|docs|test|chore|build|ci)(\([a-zA-Z0-9 -]+\))?!?: [A-Z].+[^.]$
```
