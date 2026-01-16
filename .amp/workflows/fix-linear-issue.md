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
   > DO NOT run agents in parallel, let one agent finish its issue before moving onto the next one.
   > Do NOT work on the parent issue directly. Each subagent should follow steps 3-9 independently
   > and create a separate PR per child issue.
   >
   > **When writing the Task prompt for subagents, you MUST include ALL steps explicitly:**
   > - Steps 3-6: Prepare workspace, understand problem, implement fix, build and test
   > - **Step 7: Self-review (REQUIRED)** - Instruct subagent to spawn its own reviewer subagent
   > - Steps 8-9: Create branch, commit, and create PR
   >
   > Do NOT abbreviate or skip steps when writing subagent prompts.

3. **Prepare workspace** - Start from a clean, up-to-date main branch:
   ```bash
   git checkout main
   git pull origin main
   ```

4. **Understand the problem** - Read the relevant file(s) at the specified line numbers to understand what needs to change.

5. **Implement the fix** - Make the necessary code changes using `edit_file`.

6. **Build and test** - Verify the fix works:
   ```bash
   # For Go code - always run full build and test, bazel caching handles efficiency
   bazel build //...
   bazel test //...
   
   # For TypeScript code
   pnpm --dir=web build
   pnpm --dir=web test
   ```

7. **Self-review (REQUIRED - DO NOT SKIP)** - Before committing, spawn a subagent to review your changes:

   > ⚠️ **This step is mandatory.** Every PR must be self-reviewed before creating the branch and committing.
   
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
   
   Address any feedback from the reviewer before proceeding to step 8.

8. **Create branch and commit**:
   ```bash
   git checkout -b <gitBranchName from issue>
   git add path/to/file1.go path/to/file2.go  # List ONLY the specific files you modified
   git commit -m "<type>: <description from issue title>"
   ```
   
   > **CRITICAL for subagents**: Never use `git add -A`, `git add .`, or `git add --all`.
   > Always explicitly list each file path you modified. Parallel subagents share
   > the same working directory, so using `-A` will stage other subagents' changes.

9. **Create PR**:
   ```bash
   git push -u origin <branch>
   gh pr create --draft --title "<type>: <description>" --body "## Summary
   <detailed explanation of what was changed and why>
   
   ## Changes
   - <specific change 1 with file and reasoning>
   - <specific change 2 with file and reasoning>
   
   ## Testing
   <how the changes were verified, test commands run>
   
   Closes <issue identifier>"
   ```
   
   > **NOTE**: The PR description should be verbose and detailed. Include all context
   > a reviewer needs to understand the change. Do not summarize changes to the user -
   > put that detail in the PR description instead.
   
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
3. Spawns subagents ONE AT A TIME (not in parallel), one per child issue
4. Each subagent independently follows ALL steps:
   - Prepares workspace (checkout main, pull)
   - Reads and understands the problem
   - Implements the fix
   - Runs build and tests
   - **Spawns its own reviewer subagent for self-review** ← DO NOT SKIP
   - Addresses any review feedback
   - Creates a separate branch and PR (staging ONLY its own files)
5. Reports summary of all PRs created
```
