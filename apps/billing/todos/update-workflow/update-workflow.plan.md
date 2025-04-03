# Update Workflow Plan

## Overview

The Update Workflow is responsible for updating glossary content on the website. It takes the `content` section of an .mdx file and attempts to replace it inside our apps/www inside the content directory.

## File Structure

```sh
apps/billing/src/trigger/glossary/update/
├── update-content.ts # Core task for updating glossary content
├── update-content-cleanup.ts # Task for cleaning up PRs and branches
└── update-content.test.ts # Test harness for the update content task
```

The Update Workflow consists of three main components:

1. **Core Task**: `update-content.ts` - The main task that updates glossary content
2. **Cleanup Task**: `update-content-cleanup.ts` - A task that cleans up after updates (closes PRs, deletes branches)
3. **Test Task**: `test-update-content.ts` - A test harness that verifies the functionality of the update task

These files work together to provide a complete workflow for updating glossary content with proper testing and cleanup.

## Contents

### Update Content Task

**File:** `apps/billing/src/trigger/glossary/update/update-content.ts`

**Purpose:** Updates glossary content on the website by creating a GitHub PR.

**Key Features:**

- Validates input term and content
- Creates a new branch for the update
- Preserves frontmatter when updating existing files
- Creates a pull request for the changes
- Only supports updates (no creation)

**Implementation Details:**

- Uses GitHub API via Octokit to interact with the repository
- Handles both new and existing content
- Returns PR URL and update status

### Clean Up Task

**File:** `apps/billing/src/trigger/glossary/update/cleanup-update.ts`

**Purpose:** Cleans up resources created by the update content task, specifically GitHub PRs and branches.

**Key Features:**

- Closes pull requests created during testing
- Deletes branches created during testing
- Can be triggered manually or automatically after tests

**Implementation Details:**

- Uses GitHub API via Octokit to interact with the repository
- Accepts PR number and branch name as inputs
- Returns status of cleanup operations (PR closed, branch deleted)
- Handles errors gracefully with proper reporting

### Testing Task

**File:** `apps/billing/src/trigger/glossary/update/test-update-content.ts`

**Purpose:** Provides a comprehensive test harness for the update content task.

**Key Features:**

- Defines test cases with inputs and expected outputs
- Runs tests without requiring manual input
- Validates results against expected outcomes
- Automatically cleans up after successful tests
- Provides real-time progress tracking via metadata

**Implementation Details:**

- Uses Trigger.dev metadata API for real-time progress tracking
- Implements type-safe metadata handling with Zod schemas
- Includes test cases for both success and failure scenarios
- Integrates with the cleanup task to remove test resources
- Returns detailed test results with success/failure status

**Test Cases:**

1. `basicUpdateTest`: Tests basic update functionality with a valid term and content
2. `emptyTermTest`: Tests error handling with an empty term
3. `emptyContentTest`: Tests error handling with empty content
4. `nonExistentFileTest`: Tests error handling with a non-existent file

## Running the Workflow

1. **Development Testing:**
   - Start the Trigger.dev CLI: `pnpm -F billing dev`
   - Navigate to the Trigger.dev dashboard
   - Trigger the test task (`glossary-update-content-test-all`) without any input
   - View test results in the dashboard

2. **Production Usage:**
   - Trigger the update content task with term and content parameters
   - Monitor the task execution in the Trigger.dev dashboard
   - Check the created PR in GitHub
