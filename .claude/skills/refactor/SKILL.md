---
name: refactor
description: Structural refactoring pass on changed code. Use after implementing a feature to improve code structure, reduce duplication, and clean up APIs without changing behavior.
---

# Refactor pass

Review the diff against main and perform a structural refactoring pass on all changed code in this branch.

Focus on:

1. **Extract and consolidate**: Pull repeated logic into shared functions. Merge near-duplicate code paths. Collapse unnecessary wrapper layers.

2. **Improve structure**: Flatten deep nesting. Break oversized functions into focused units. Move code closer to where it's used. Colocate related logic.

3. **Clean up APIs and interfaces**: Remove unused parameters, exports, and return values. Simplify function signatures. Make types tighter and more precise.

4. **Reduce indirection**: Inline trivial helpers that obscure intent. Remove pass-through functions that add no value. Shorten unnecessary call chains.

5. **Fix naming**: Rename variables, functions, and types so they say what they mean. Align naming with surrounding code conventions.

Rules:

- **Do not change behavior.** Every input/output must remain identical.
- **Stay within the branch diff.** Only refactor code that was added or modified in this branch. Do not refactor unrelated code unless it is directly entangled.
- **Respect project conventions.** Follow patterns and standards from CLAUDE.md and the surrounding codebase.
- **Don't over-abstract.** Three similar lines are fine. Only extract when duplication is real and likely to grow.
- **Commit-ready output.** The code should compile and pass tests after your changes.

Process:

1. Read the full branch diff (`git diff main...HEAD`)
2. Identify structural issues in the changed code
3. Apply refactors file by file
4. Run the project's type check / lint if available
5. Report a 1-5 sentence summary of what you changed
