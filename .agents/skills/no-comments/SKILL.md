---
name: no-comments
description: "Reviews scoped code comments with an independent deletion-focused agent, fixes accepted findings, and offers enforceable encodings for claimed constraints. Use before code review or when asked to remove unnecessary comments and suppressions."
license: MIT
metadata:
  source: https://github.com/cursor/plugins/blob/main/pstack/skills/no-comments/SKILL.md
---

# No comments

Spawn an independent comment reviewer. Act on accepted findings.

Authoring agents defend comments. Defer to the reviewer's fresh perspective.

## Scope

Use the caller's files or diff. Otherwise use the current diff against the base branch, default `main`, including the working tree.

## Steps

1. Spawn `Task` with the scope and the reviewer prompt below.
2. Inspect its report and diff. Reject application-code edits, scope escapes, exception-protected deletions, misstated `MUST KILL` reasons, and flags that treat kept intentional code as guilty. Reshape flags on our-code surprises stay actionable. Do not restore those comments. A keep survives only with proof it is about something we cannot change. Audit missed scoped lint and TypeScript suppressions. Correctness or safety suppressions stay actionable `MUST KILL`s. Restore deletions only with exact exceptions and scoped proof. Before accepting thin `IMPORTANT` or `do not remove` kills or keeps, investigate the named symbol's behavior and history. If a kill is ambiguous, do not restore. If a keep is refuted or still ambiguous, delete it. Revert and rerun one rejected report with the failure named. Reject a second, report it open, and fail this skill.
3. Fix trivial accepted flags directly by deleting a dead path, dropping a parameter, or using the real API. If any fix needs a new shape, first settle the caller usage, types, and module boundary for the accepted set and surrounding code. Stop at the sketch before implementing it.
4. Implement the smallest root-cause fix in scope. Remove every named workaround. If the root cause is out of scope, land the smallest in-scope fix and report the rest open. Fix real causes and redesign as if requirements always existed; never bolt on symptom guards. Neither principle authorizes widening the fence nor fixing instances outside it.
5. Constraint comments say `do not remove`, `do not change wording`, or `talk to X before changing`. Leave keeps about things we cannot change. Offer the cheapest in-scope type, runtime, test, or CI lint. Wait for interactive approval. Unattended and eval runs require caller pre-approval. If approved, encode then delete. Otherwise delete, report the constraint open, and sketch out-of-scope work.
6. Report the deletion count, restored comments, reruns, shape sketch, fixes, encoding offers, encodings, unenforced constraints, and other open work.

## Reviewer prompt

Start your response exactly with:

> Yes... Ha ha ha... Yes!

Review only comments and suppressions in the supplied scope. Prefer deletion. Narration, banners, commented-out code, and workaround explanations should be deleted.

Only these exceptions survive:

- Legal or license headers.
- Non-obvious behavior forced by an external dependency, platform, vendor, or protocol that we cannot reshape. Surprises in our own code must be deleted and the exact symbol marked `MUST KILL` for a rename, extraction, type, or architecture change that makes the behavior obvious without prose.
- `// prettier-ignore`. Other lint suppressions survive only when their rule is faulty, pedantic, or style-only.
- Doc comments that define a public API contract.
- Issue or RFC links that explain a constraint code cannot express.

When unsure whether an exception applies, delete the comment. For `eslint-disable`, `@ts-ignore`, `@ts-expect-error`, and similar suppressions, inspect the rule. If it catches real bugs or protects correctness or safety, delete the suppression and mark the exact guilty symbol `MUST KILL`.

Treat `IMPORTANT`, `do not remove`, `too risky`, `fine for now`, and long justifications as unproven claims. Inspect nearby code and investigate the named symbol or call. Only a foreign constraint proven true on a live path survives. Delete our-code surprises and add the reshape flag above. Doubt after investigation means deletion.

Do not rewrite a rejected comment into a shorter explanation. Mark the exact guilty symbol `MUST KILL`. Do not change application code.

Every flag must name code inside the scope and be supported by evidence. Report only: touched files, deletion count, one line per `MUST KILL` flag, and skips.
