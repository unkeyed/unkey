# Trigger.dev Testing Approach

## Overview

This document outlines our approach to testing Trigger.dev tasks, which require a different testing methodology than traditional unit tests. Since Trigger.dev tasks run in a cloud environment and can't be directly tested with Jest or other testing frameworks, we've developed a pattern for end-to-end testing of tasks.

## Testing Architecture

Our testing approach consists of three key components:

1. **Core Task File**: The actual implementation of the task that performs the real work
2. **Test Task File**: A test harness that defines test cases and runs the core task with different inputs
3. **Cleanup Task File**: A task that reverses the actions of the core task to clean up after tests

## Component Structure

### 1. Core Task File

The core task file contains the actual implementation of the functionality we want to test. For example, `update-content.ts` contains the task that updates glossary content by creating a GitHub PR.

```typescript
// Example: update-content.ts
export const updateGlossaryContentTask = task({
  id: "update_glossary_content",
  run: async ({ inputTerm, content }) => {
    // Implementation that creates a PR with the content
    return {
      inputTerm,
      updated: true,
      prUrl,
      branch,
    };
  },
});
```

### 2. Test Task File

The test task file defines test cases and runs the core task with different inputs to verify its behavior. It should run without requiring any input parameters when triggered from the Trigger.dev console.

```typescript
// Example: update-content.test.ts
const testCases = [
  {
    name: "basicUpdateTest",
    input: { inputTerm: "MIME types", content: "..." },
    expectedSuccess: true,
    validate: (result) => { /* validation logic */ }
  },
  // More test cases...
];

export const runAllTests = task({
  id: "glossary-update-content-test-all",
  run: async () => {
    // Run all test cases without requiring input
    for (const testCase of testCases) {
      // Run the test case and collect results
    }
    return { /* test results */ };
  }
});
```

### 3. Cleanup Task File

The cleanup task file contains a task that reverses the actions of the core task. For example, `update-content-cleanup.ts` contains a task that closes PRs and deletes branches created by the update content task.

```typescript
// Example: update-content-cleanup.ts
export const cleanupGlossaryUpdateTask = task({
  id: "cleanup_glossary_update",
  run: async ({ prNumber, branch }) => {
    // Close PR and delete branch
    return {
      prClosed: true,
      branchDeleted: true,
      prNumber,
      branch,
    };
  },
});
```

## Testing Workflow

1. **Define Test Cases**: Create test cases with inputs, expected outputs, and validation logic
2. **Implement Test Harness**: Create a test task that runs all test cases without requiring input
3. **Implement Cleanup Logic**: Create a cleanup task and integrate it into the test harness
4. **Run Tests**: Trigger the test task from the Trigger.dev console
5. **View Results**: Check the test results in the Trigger.dev console

## Real-time Progress Tracking

To provide visibility into test progress, we use Trigger.dev's metadata API:

1. **Initialize Metadata**: Set up metadata at the start of the test run
2. **Update Progress**: Update metadata as tests run
3. **Track Results**: Store test results in metadata
4. **View in Dashboard**: View real-time progress in the Trigger.dev dashboard

## Example Implementation

Our glossary update workflow testing consists of:

1. **Core Task**: `update-content.ts` - Updates glossary content by creating a GitHub PR
2. **Test Task**: `update-content.test.ts` - Runs test cases for the update content task
3. **Cleanup Task**: `update-content-cleanup.ts` - Closes PRs and deletes branches created during tests

The test task includes:

- Predefined test cases with inputs and expected outputs
- A task to run a single test case
- A task to run all test cases without requiring input
- Metadata tracking for real-time progress visibility
- Integration with the cleanup task to clean up after tests

## Running Tests

1. Start the Trigger.dev CLI: `pnpm -F billing dev`
2. Navigate to the Trigger.dev dashboard
3. Find the test task (e.g., `glossary-update-content-test-all`)
4. Trigger the task without any input
5. View the test results in the task output

