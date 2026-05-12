---
name: improve-microcopy
description: Write or review UX microcopy for dashboard interfaces. Use when reviewing button labels, errors, empty states, dialogs, helper text, tooltips, toasts, onboarding, or any user-facing product copy.
disable-model-invocation: true
allowed-tools: [Read, Glob, Grep, Bash]
---

# Improve Microcopy

Write or review UX copy for product UI. Be concise, suggestive, and light on theory.

Do not edit product files when this skill is invoked. Return recommendations first. The human chooses what to apply.

## What I Need

Ask only if missing and not discoverable:

- Context: What screen, flow, or feature?
- User state: What is the user trying to do? Are they blocked, deciding, or confirming?
- Tone: Neutral, reassuring, direct, celebratory?
- Constraints: Character limits, layout, platform rules?

For branch reviews, infer context from `git diff main...HEAD` and changed files under `web/apps/dashboard/`. Skip tests, stories, logs, comments, identifiers, and copy outside the diff. Do not patch the dashboard.

## Principles

- Clear: Say exactly what you mean. No ambiguity.
- Concise: Use the fewest words that preserve the full meaning.
- Consistent: Use the same term for the same thing on the same surface.
- Useful: Answer the user's next question before they get stuck.
- Human: Use everyday words a person would say out loud.
- Calm: No hype, jokes, emojis, melodrama, or blame.

## Friction Check

Before rewriting, ask what job the copy has:

- Before action: give the user a reason, expectation, or warning.
- During action: explain what to enter, choose, or check.
- After action: confirm what happened or explain how to recover.

Then check:

- What might users misunderstand?
- What might make them hesitate?
- What would make them ask support or give up?

## Unkey Voice

- Lead with the user's task.
- Use concrete nouns already in the UI: key, API, identity, namespace, permission, role, workspace.
- Use "Unkey" only when the product is truly the actor or boundary.
- Use active voice when ownership matters: "We couldn't create the key".
- Use plain past tense for success: "Key created", "Changes saved".
- Avoid "please", "simply", "just", "easily", "obviously", "successfully", and exclamation marks.

## Active and Passive Voice

- Use active voice for actions, errors, and next steps: "We couldn't delete the key", "Create a namespace first".
- Use passive voice only when the actor does not matter: "Changes saved", "Key deleted", "Request blocked".
- Name the actor when it changes responsibility: "GitHub denied access to this repo" is better than "Access denied" if the user needs to know where to fix it.
- Avoid vague passive errors: "The key could not be created", "An error occurred", "Request failed".

## Copy Patterns

### CTAs
Start with a verb. Name the object when context is not obvious: "Create key", "Save changes", "Delete permission". Avoid "Submit", "Confirm", "Continue", and "OK".

### Errors
Structure: blocker + cause + next step. Put the most useful information first. Use "Name already in use. Choose a different key name." Avoid "Something went wrong. Please try again."

### Empty States
Structure: what this is + why it is empty + how to start. Use "No permissions yet. Add a permission to control access to this API." Avoid "No data".

Never leave a dead end unless the user truly cannot act.

### Confirmation Dialogs
Make the action clear, describe consequences, and label buttons with the action: "Delete permission?" / "Requests that rely on this permission may fail. This action cannot be undone." / "Delete permission" and "Keep permission".

Do not mechanically rewrite a whole dialog. If the current warning is clear, keep it and add only the missing consequence or undo note. Avoid "gone for good", "destroyed forever", and "permanently destroys its analytics".

### Helper Text
Explain why the field matters, not what the label already says. Use "Shown in logs and the dashboard." Avoid "A name to identify this item."

### Tooltips, Loading, Onboarding
Tooltips should clarify unfamiliar terms, icons, limits, or hidden behavior. Loading states should set expectations only when timing matters. Onboarding should reveal one concept at a time. Persuasive copy belongs mostly in onboarding, billing, and upgrade moments, not routine dashboard workflows.

## Output

For a single screen, flow, or focused review:

```md
## UX Copy: [Context]

### Recommended Copy
**[Element]**: [Copy]

### Alternatives
| Option | Copy | Tone | Best for |
|--------|------|------|----------|
| A | [Copy] | [Tone] | [When to use] |
| B | [Copy] | [Tone] | [When to use] |

### Rationale
[Short explanation: user context, clarity, action-orientation.]

### Localization Notes
[Anything translators should know. Omit if not relevant.]
```

For branch audits, group by file and keep only high-signal findings:

```md
## UX Copy Review

### web/apps/dashboard/example.tsx

#### [Element], line 42
Current: "[Current copy]"
Recommended: "[Best rewrite]"
Alternative: "[Optional second choice]"
Why: [One short sentence.]

N recommendations across X files.
```

Limit alternatives to two unless the user asks for exploration. Prefer one strong recommendation over a menu of tiny variations.

After giving recommendations, offer to preview them as a small self-contained HTML page if visual context would help. Use realistic UI contexts: dialogs, toasts, empty states, form rows, or buttons. Show before and after side by side with the file, line, recommendation, optional alternative, and one short rationale.
