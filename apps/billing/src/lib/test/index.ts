import { metadata, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";

/**
 * Base schema for tracking test execution metadata
 */
export const TestMetadataSchema = z.object({
  totalTests: z.number(),
  completedTests: z.number(),
  passedTests: z.number(),
  failedTests: z.number(),
  currentTest: z.string().optional(),
  results: z.array(
    z.object({
      testCase: z.string(),
      status: z.enum(["passed", "failed"]),
      duration: z.number(),
      error: z.string().optional(),
      output: z.any().optional(),
    }),
  ),
  cleanupResults: z.array(
    z.object({
      testCase: z.string(),
      status: z.enum(["success", "failed"]),
      prClosed: z.boolean().optional(),
      branchDeleted: z.boolean().optional(),
      error: z.string().optional(),
    }),
  ),
});

export type TestMetadata = z.infer<typeof TestMetadataSchema>;

/**
 * Base schemas for task run results
 */
export const okResultSchema = z.object({
  ok: z.literal(true),
  id: z.string(),
  taskIdentifier: z.string(),
  output: z.any(),
});

export const errorResultSchema = z.object({
  ok: z.literal(false),
  id: z.string(),
  taskIdentifier: z.string(),
  error: z.unknown(),
});

/**
 * Helper functions for type-safe metadata access
 */
export function getAllTestsMetadata(): TestMetadata {
  const current = metadata.current() ?? {};
  return TestMetadataSchema.parse(
    current.testMetadata ?? {
      totalTests: 0,
      completedTests: 0,
      passedTests: 0,
      failedTests: 0,
      results: [],
      cleanupResults: [],
    },
  );
}

export function safeSetMetadata(key: string, value: any) {
  const safeValue = JSON.parse(JSON.stringify(value));
  metadata.set(key, safeValue);
}

/**
 * Utility types for extracting task input/output types
 */
export type TaskInput<T> = T extends { triggerAndWait: (input: infer I) => any } ? I : never;
export type TaskOutput<T> = T extends { triggerAndWait: (...args: any[]) => Promise<infer O> }
  ? O
  : never;
export type TaskResult<T> = T extends { triggerAndWait: (...args: any[]) => Promise<infer R> }
  ? R
  : never;

/**
 * Base interface for test cases with inferred types
 */
export interface TestCase<TTask> {
  name: string;
  input: TaskInput<TTask>;
  validate(result: TaskResult<TTask>): boolean;
  cleanup?: (result: TaskResult<TTask>) => Promise<void>;
}

/**
 * Creates a test runner task with inferred types
 */
export function createTestRunner<
  TTask extends { triggerAndWait: (input: any) => Promise<any> },
>(options: {
  id: string;
  task: TTask;
  testCases: Array<TestCase<TTask>>;
}) {
  return task({
    id: options.id,
    retry: {
      maxAttempts: 1,
    },
    onStart: async () => {
      const initialMetadata: TestMetadata = {
        totalTests: options.testCases.length,
        completedTests: 0,
        passedTests: 0,
        failedTests: 0,
        results: [],
        cleanupResults: [],
      };
      safeSetMetadata("testMetadata", initialMetadata);
    },
    run: async () => {
      const startTime = Date.now();

      for (const testCase of options.testCases) {
        const testStartTime = Date.now();
        const metadata = getAllTestsMetadata();
        metadata.currentTest = testCase.name;
        safeSetMetadata("testMetadata", metadata);

        try {
          const runResult = await options.task.triggerAndWait(testCase.input);
          const validation = testCase.validate(runResult);

          metadata.completedTests++;

          if (validation) {
            metadata.passedTests++;
            metadata.results.push({
              testCase: testCase.name,
              status: "passed",
              duration: Date.now() - testStartTime,
              output: runResult,
            });
          } else {
            metadata.failedTests++;
            metadata.results.push({
              testCase: testCase.name,
              status: "failed",
              duration: Date.now() - testStartTime,
              error: "Validation function failed",
              output: runResult,
            });
          }

          if (testCase.cleanup) {
            await testCase.cleanup(runResult);
          }
        } catch (error) {
          metadata.failedTests++;
          metadata.results.push({
            testCase: testCase.name,
            status: "failed",
            duration: Date.now() - testStartTime,
            error: error instanceof Error ? error.message : String(error),
          });
        }

        safeSetMetadata("testMetadata", metadata);
      }

      // Print final results
      const metadata = getAllTestsMetadata();
      console.info("\n========== TEST RESULTS ==========");
      console.info(
        `\n${metadata.totalTests === metadata.passedTests ? "✅ All tests passed" : "⚠️ Some tests failed"}`,
      );
      console.info(`Total Tests: ${metadata.totalTests}`);
      console.info(`✓ Passed: ${metadata.passedTests}`);
      console.info(`✗ Failed: ${metadata.failedTests}`);

      const failedTests = metadata.results.filter((r) => r.status === "failed");
      if (failedTests.length > 0) {
        console.info("\nFailed Tests:");
        failedTests.forEach((result) => {
          console.info(`- ${result.testCase}`);
          if (result.error) {
            console.info(`  Error: ${result.error}`);
          }
        });
      }

      if (metadata.cleanupResults.length > 0) {
        const successfulCleanups = metadata.cleanupResults.filter(
          (r) => r.status === "success",
        ).length;
        const failedCleanups = metadata.cleanupResults.filter((r) => r.status === "failed");

        console.info("\nCleanup Results:");
        console.info(`✓ Successfully cleaned up ${successfulCleanups} test(s)`);

        if (failedCleanups.length > 0) {
          console.info(`✗ Failed to cleanup ${failedCleanups.length} test(s):`);
          failedCleanups.forEach((result) => {
            console.info(`- ${result.testCase}: ${result.error}`);
          });
        }
      }

      console.info(`\nDuration: ${Date.now() - startTime}ms`);
      console.info("===============================");

      return metadata;
    },
  });
}
