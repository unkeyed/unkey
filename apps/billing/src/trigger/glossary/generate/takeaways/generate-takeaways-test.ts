import { generateTakeawaysTask, type FieldSelection } from "./generate-takeaways";
import { metadata, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";
import { takeawaysSchema } from "@/lib/db-marketing/schemas/takeaways-schema";

// Test Metadata Schema
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

type TestMetadata = z.infer<typeof TestMetadataSchema>;

/**
 * Helper functions for type-safe metadata access
 */
function getAllTestsMetadata(): TestMetadata {
  const current = metadata.current() ?? {};
  return TestMetadataSchema.parse(current.testMetadata ?? {
    totalTests: 0,
    completedTests: 0,
    passedTests: 0,
    failedTests: 0,
    results: [],
    cleanupResults: []
  });
}

function safeSetMetadata(key: string, value: any) {
  const safeValue = JSON.parse(JSON.stringify(value));
  metadata.set(key, safeValue);
}

// Base schema for task run results
const okResultSchema = z.object({
  ok: z.literal(true),
  id: z.string(),
  taskIdentifier: z.string(),
  output: z.any()
});

const errorResultSchema = z.object({
  ok: z.literal(false),
  id: z.string(),
  taskIdentifier: z.string(),
  error: z.unknown()
});

// Test Case Interface
interface TestCase {
  name: string;
  input: {
    term: string;
    fields?: FieldSelection;
  };
  validate(result: Awaited<ReturnType<typeof generateTakeawaysTask["triggerAndWait"]>>): boolean;
  cleanup?: (result: Awaited<ReturnType<typeof generateTakeawaysTask["triggerAndWait"]>>) => Promise<void>;
}

const testCases: TestCase[] = [
  {
    name: "fullGenerationTest",
    input: {
      term: "MIME types"
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(`Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`);
        console.info(validation.error.errors.map(e => e.message).join("\n"));
        return false;
      }
      const expectedOutputSchema = z.object({
        term: z.literal("MIME types"),
        takeaways: takeawaysSchema,
      });
      const output = validation.data.output;
      const outputValidation = expectedOutputSchema.safeParse(output);
      if (!outputValidation.success) {
        console.warn(`Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`);
        console.warn(outputValidation.error.errors.map(e => e.message).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    }
  },
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
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(`Test '${this.name}' failed on result validation. Expected a valid result, but got: ${JSON.stringify(result)}`);
        console.info(validation.error.errors.map(e => e.message).join("\n"));
        return false;
      }

      const expectedOutputSchema = z.object({
        term: z.literal("MIME types"),
        takeaways: z.object({
          tldr: takeawaysSchema.shape.tldr,
          definitionAndStructure: takeawaysSchema.shape.definitionAndStructure,
          usageInAPIs: z.object({
            description: z.string()
          })
        }),
        fields: z.object({
          tldr: z.boolean(),
          definitionAndStructure: z.array(z.number()),
          usageInAPIs: z.object({
            description: z.boolean()
          })
        })
      });
      const outputValidation = expectedOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(`Test '${this.name}' failed on output validation. Expected a valid result, but got: ${JSON.stringify(validation.data.output)}`);
        // print the zod errors in a readable format:
        console.warn(outputValidation.error.errors.map(e => `${e.path}: ${e.message}`).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    }
  },
  {
    name: "invalidTermTest",
    input: {
      term: ""
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(`Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`);
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    }
  }
];

export const generateTakeawaysTest = task({
  id: "generate_takeaways_test",
  retry: {
    maxAttempts: 1,
  },
  onStart: async () => {
    const initialMetadata: TestMetadata = {
      totalTests: testCases.length,
      completedTests: 0,
      passedTests: 0,
      failedTests: 0,
      results: [],
      cleanupResults: []
    };
    safeSetMetadata("testMetadata", initialMetadata);
  },
  run: async () => {
    const startTime = Date.now();

    for (const testCase of testCases) {
      const testStartTime = Date.now();
      const runResult = await generateTakeawaysTask.triggerAndWait(testCase.input);

      const validation = testCase.validate(runResult);

      const metadata = getAllTestsMetadata();
      metadata.completedTests++;

      if (validation) {
        metadata.passedTests++;
        metadata.results.push({
          testCase: testCase.name,
          status: "passed",
          duration: Date.now() - testStartTime,
          output: runResult
        });
      } else {
        metadata.failedTests++;
        metadata.results.push({
          testCase: testCase.name,
          status: "failed",
          duration: Date.now() - testStartTime,
          error: "Validation function failed",
          output: runResult
        });
      }

      if (testCase.cleanup) {
        await testCase.cleanup(runResult);
      }

      safeSetMetadata("testMetadata", metadata);
    }

    // Print final results
    const metadata = getAllTestsMetadata();
    console.info("\n========== TEST RESULTS ==========");
    console.info(`\n${metadata.totalTests === metadata.passedTests ? "✅ All tests passed" : "⚠️ Some tests failed"}`);
    console.info(`Total Tests: ${metadata.totalTests}`);
    console.info(`✓ Passed: ${metadata.passedTests}`);
    console.info(`✗ Failed: ${metadata.failedTests}`);

    const failedTests = metadata.results.filter(r => r.status === "failed");
    if (failedTests.length > 0) {
      console.info("\nFailed Tests:");
      failedTests.forEach(result => {
        console.info(`- ${result.testCase}`);
        if (result.error) {
          console.info(`  Error: ${result.error}`);
        }
      });
    }

    if (metadata.cleanupResults.length > 0) {
      const successfulCleanups = metadata.cleanupResults.filter(r => r.status === "success").length;
      const failedCleanups = metadata.cleanupResults.filter(r => r.status === "failed");
      
      console.info("\nCleanup Results:");
      console.info(`✓ Successfully cleaned up ${successfulCleanups} test(s)`);
      
      if (failedCleanups.length > 0) {
        console.info(`✗ Failed to cleanup ${failedCleanups.length} test(s):`);
        failedCleanups.forEach(result => {
          console.info(`- ${result.testCase}: ${result.error}`);
        });
      }
    }

    console.log(`\nDuration: ${Date.now() - startTime}ms`);
    console.log("===============================");

    return metadata;
  }
}); 