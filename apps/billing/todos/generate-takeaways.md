# Generate Takeaways Workflow Specification

## Type Definitions

IMPORTANT: Always follow the rules defined inside .cursor/rules directory. These rules take precedence over any other implementation details.
E.g. we should minimize explicit type definitions and instead infer types from Zod schemas and task definitions wherever possible.

```ts
// Import the source of truth
import { takeawaysSchema } from //...

// We'll use the existing takeawaysSchema as source of truth and create the field selection schema:
const fieldSelectionSchema = z.object({
  tldr: z.boolean().optional(),
  definitionAndStructure: z.union([z.boolean(), z.array(z.number())]).optional(),
  historicalContext: z.union([z.boolean(), z.array(z.number())]).optional(),
  usageInAPIs: z.union([
    z.boolean(),
    z.object({
      tags: z.boolean().optional(),
      description: z.boolean().optional()
    })
  ]).optional(),
  bestPractices: z.union([z.boolean(), z.array(z.number())]).optional(),
  recommendedReading: z.union([z.boolean(), z.array(z.number())]).optional(),
  didYouKnow: z.boolean().optional()
});

// Field Selection - Similar to Prisma's select API
// Type is inferred from the schema above
type FieldSelection = z.infer<typeof fieldSelectionSchema>;

// Test Case Interface - Using inferred types
interface TestCase {
  name: string;
  input: {
    term: string;
    fields?: FieldSelection;
  };
  // Use the actual task result type
  expectedTaskRunResult: TaskRunResult<typeof generateTakeawaysTask>;
  validate?: (result: TaskRunResult<typeof generateTakeawaysTask>) => boolean;
  cleanup?: (result: TaskRunResult<typeof generateTakeawaysTask>) => Promise<void>;
}

// Metadata Schema
// IMPORTANT: Follow the type safety pattern from the Trigger rule using Zod
const TestMetadataSchema = z.object({
  totalTests: z.number(),
  completedTests: z.number(),
  passedTests: z.number(),
  failedTests: z.number(),
  currentTest: z.string().optional(),
  results: z.array(z.object({
    testCase: z.string(),
    status: z.enum(["passed", "failed"]),
    duration: z.number(),
    error: z.string().optional(),
    output: z.any().optional()
  })),
  cleanupResults: z.array(z.object({
    testCase: z.string(),
    status: z.enum(["success", "failed"]),
    prClosed: z.boolean().optional(),
    branchDeleted: z.boolean().optional(),
    error: z.string().optional()
  }))
});

// Type is inferred from schema
type TestMetadata = z.infer<typeof TestMetadataSchema>;
```

## Test Cases

1. Full Generation Test

```typescript
{
  name: "fullGenerationTest",
  input: {
    term: "MIME types"
  },
  expectedTaskRunResult: {
    ok: true,
    output: {
      term: "MIME types",
      takeaways: {
        // Will be inferred from takeawaysSchema
      }
    }
  }
}
```

2. Partial Generation Test

```typescript
{
  name: "partialGenerationTest",
  input: {
    term: "MIME types",
    fields: {
      tldr: true,
      definitionAndStructure: [0, 1],
      usageInAPIs: {
        description: true
      }
    }
  },
  expectedTaskRunResult: {
    ok: true,
    output: {
      term: "MIME types",
      takeaways: {
        // Will be inferred from takeawaysSchema and FieldSelection
      }
    }
  }
}
```

3. Error Cases

```typescript
{
  name: "invalidTermTest",
  input: {
    term: ""
  },
  expectedTaskRunResult: {
    ok: false,
    error: {
      message: "Term is required",
      code: "INVALID_INPUT"
    }
  }
}
```

## Implementation Requirements

### Test Implementation

Generate:

- Location: `/generate/generate-takeaways-test.ts`
- Uses inferred types from task definitions
- Tracks test progress using TestMetadataSchema
- Provides cleanup functionality for PRs
- Logs final results to console

Update:

- Location: `/update/update-takeaways.test.ts`
- Uses inferred types from task definitions
- Tracks test progress using TestMetadataSchema
- Provides cleanup functionality for PRs
- Logs final results to console

### Generate Task

- Location: `/generate/generate-takeaways.ts`
- Uses Vercel AI SDK with Google Gemini
- Supports partial field generation using FieldSelection type
- Implements metadata tracking using TestMetadataSchema
- Error handling with AbortTaskRunError

### Update Task

- Location: `/update/update-takeaways.ts`
- Creates GitHub PR for changes
- Can refer to update-content.ts for a "similar implementation"
- This should support partial field updates using FieldSelection type though (different to update-content)
- Preserves all existing .mdx content other than the above fields

## Metadata Tracking Example

```typescript
// Example metadata during test run - type is inferred from TestMetadataSchema
{
  totalTests: 5,
  completedTests: 3,
  passedTests: 2,
  failedTests: 1,
  currentTest: "partialGenerationTest",
  results: [
    {
      testCase: "fullGenerationTest",
      status: "passed",
      duration: 1234,
      output: { ... }
    }
  ],
  cleanupResults: [
    {
      testCase: "fullGenerationTest",
      status: "success",
      prClosed: true,
      branchDeleted: true
    }
  ]
}
```

## Console Output Format

Final test results should be logged in the following format:

```
========== TEST RESULTS ==========
Total Tests: 5
✓ Passed: 4
✗ Failed: 1

Failed Tests:
- invalidTermTest
  Error: Term is required
  Code: INVALID_INPUT

Cleanup Results:
✓ Closed 3 PRs
✓ Deleted 3 branches
✗ Failed to cleanup 1 PR (invalidTermTest)

Duration: 1234ms
===============================
```
