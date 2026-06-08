# Review checklists

Hand each list verbatim to the matching read-only sub-agent in step 3 of `SKILL.md`.

## Agent A — Bugs & contracts

Walk the diff against this checklist:

- [ ] Null / undefined deref on values that could be missing
- [ ] Race conditions, broken loading / error states, stale closures
- [ ] Off-by-one in loops, slices, indexing
- [ ] Type signature changes that break existing callers
- [ ] Missing cases in switch / exhaustive checks
- [ ] Behavior regressions vs. prior implementation
- [ ] API / schema / DB-migration changes without a back-compat path
- [ ] Edge cases: empty input, max size, concurrent access, retries, timeouts

## Agent B — Patterns & UX consistency

Walk the diff against this checklist:

- [ ] Reuses existing primitives from the project's UI library (e.g. `@unkey/ui`) instead of reinventing them
- [ ] Uses the project's wrapper component instead of the raw HTML element when one exists (`<button>` → `<Button>`, `<input>` → `<Input>`, `<a>` → `<Link>`, `<dialog>` → `<Dialog>`, etc.)
- [ ] Spacing, colors, typography, radii, shadows use design tokens — no magic values
- [ ] All interaction states present and consistent: hover, focus, active, disabled, selected
- [ ] Loading, error, and empty states match how peer surfaces handle them — don't invent new patterns
- [ ] Responsive: works at mobile (≤375px), tablet, desktop without overflow or broken layout
- [ ] Keyboard nav: tab order is sensible, focus is visible, Enter / Esc / arrow keys behave per existing pattern
- [ ] Copy and microcopy match the project's voice — consistent casing, no jargon shifts, no out-of-place tone
- [ ] Motion / transitions match existing patterns and has prefers reduced motion (or match the absence of motion if peers are static)
- [ ] Naming for new components / props / files matches peer conventions
- [ ] Doesn't regress adjacent UI — a change in one component shouldn't break peer layouts
