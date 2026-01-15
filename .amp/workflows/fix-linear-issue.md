# Fix Linear Issue Workflow

This workflow handles fixing a Linear issue end-to-end: from fetching the issue details to creating a pull request.

## Guidelines

Before implementing any fix, review and follow these standards:

- **Code Style**: `web/apps/engineering/content/docs/contributing/code-style.mdx`
- **Documentation**: `web/apps/engineering/content/docs/contributing/documentation.mdx`
- **Testing**: `web/apps/engineering/content/docs/contributing/testing/`

## Trigger

When the user provides a Linear issue identifier.

## Steps

1. **Fetch issue details** - Use `mcp__linear__get_issue` with the provided identifier to get:
   - Title and description (contains what to fix and where)
   - Suggested git branch name (`gitBranchName`)
   - File locations mentioned in the description

2. **Check for child issues** - Use `mcp__linear__list_issues` with `parentId` set to the issue ID.

   > **IMPORTANT**: If child issues exist, spawn a subagent for EACH child issue using the Task tool.
   > Do NOT work on the parent issue directly. Each subagent should follow steps 3-7 independently
   > and create a separate PR per child issue.

3. **Understand the problem** - Read the relevant file(s) at the specified line numbers to understand what needs to change.

4. **Implement the fix** - Make the necessary code changes using `edit_file`.

5. **Run tests** - Verify the fix works:
   ```bash
   # For Go code
   bazel test //path/to:test_target --test_output=errors
   
   # For TypeScript code
   pnpm --dir=web test
   ```

6. **Self-review** - Before committing, spawn a subagent to review your changes:
   ```
   Use the Task tool to create a reviewer subagent with this prompt:
   
   "Review the following code changes for a PR. Act as a senior engineer reviewer.
   
   Check against our guidelines:
   - Code style: web/apps/engineering/content/docs/contributing/code-style.mdx
   - Documentation: web/apps/engineering/content/docs/contributing/documentation.mdx  
   - Testing: web/apps/engineering/content/docs/contributing/testing/
   
   Files changed: <list the modified files>
   
   Provide specific feedback if changes are needed, or approve if ready to merge."
   ```
   
   Address any feedback from the reviewer before proceeding.

7. **Create branch and commit**:
   ```bash
   git checkout -b <gitBranchName from issue>
   git add <changed files>
   git commit -m "<type>: <description from issue title>"
   ```

8. **Create PR**:
   ```bash
   git push -u origin <branch>
   gh pr create --title "<type>: <description>" --body "## Summary
   <what was changed and why, derived from issue>
   
   Closes <issue identifier>"
   ```
   
   > **Note**: Linear status updates automatically when the issue ID (e.g., ENG-1234) is referenced in the branch name or commit message. Do NOT manually update issue status.

## Example: Single Issue

```
User: fix issue ENG-1234

Agent:
1. Fetches issue details from Linear
2. Checks for child issues (none found)
3. Reads the file(s) mentioned in the issue description
4. Implements the fix as described
5. Runs relevant tests
6. Spawns reviewer subagent to self-review changes
7. Addresses any feedback
8. Creates branch, commits, pushes, and creates PR
```

## Example: Issue with Children

```
User: fix issue ENG-1000

Agent:
1. Fetches issue details from Linear
2. Checks for child issues → finds ENG-1001, ENG-1002, ENG-1003
3. Spawns 3 subagents in parallel, one per child issue
4. Each subagent independently:
   - Implements the fix for its child issue
   - Runs tests
   - Self-reviews
   - Creates a separate branch and PR
5. Reports summary of all PRs created
```
