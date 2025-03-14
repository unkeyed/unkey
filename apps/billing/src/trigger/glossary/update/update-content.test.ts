import { tryCatch } from "@/lib/utils/try-catch";
import { updateGlossaryContentTask } from "../../../trigger/glossary/update/update-content";
import { cleanupGlossaryUpdateTask } from "./cleanup-update";
import { task, AbortTaskRunError, metadata } from "@trigger.dev/sdk/v3";
import { z } from "zod";

/**
 * Test cases for the update content task
 * These can be triggered directly from the Trigger.dev cloud console
 */

// Define base test case type with shouldCleanup defaulting to true
type TestCase = {
  name: string;
  input: { inputTerm: string; content: string };
  expectedSuccess: boolean;
  expectedError?: string;
  validate?: (result: { ok: boolean; output?: any; error?: any }) => boolean;
  shouldCleanup?: boolean;
};

// Define test cases with input and expected output validation
const testCases: TestCase[] = [
  {
    name: "basicUpdateTest",
    input: {
      inputTerm: "MIME types", // Using the existing mime-types.mdx term from the glossary
      content:
        "# MIME Types\n\nMIME types (Multipurpose Internet Mail Extensions) are identifiers for file formats and content transmitted on the Internet. They consist of a type and subtype, such as `text/html` or `application/json`.\n\n## Common MIME Types\n\n- `text/html`: HTML documents\n- `application/json`: JSON data\n- `image/jpeg`: JPEG images\n- `application/pdf`: PDF documents\n\n## Usage in APIs\n\nMIME types are crucial in API development for specifying the format of request and response data. They're typically set in the `Content-Type` header.",
    },
    expectedSuccess: true,
    // Using a more generic type that works with both success and error cases
    validate: (result: { ok: boolean; output?: any; error?: any }) => {
      if (result.ok && !result.output.updated) {
        throw new Error("Content was not updated");
      }
      if (result.ok && !result.output.prUrl) {
        throw new Error("No PR URL was returned");
      }
      return true;
    }
  },
  {
    name: "emptyTermTest",
    input: {
      inputTerm: "",
      content: "This content should not be updated because the term is empty.",
    },
    expectedSuccess: false,
    expectedError: "Input term is required"
  },
  {
    name: "emptyContentTest",
    input: {
      inputTerm: "MIME types",
      content: "",
    },
    expectedSuccess: false,
    expectedError: "Content is required"
  },
  {
    name: "nonExistentFileTest",
    input: {
      inputTerm: `non-existent-term-${Date.now()}`,
      content: "This content should not be updated because the file doesn't exist.",
    },
    expectedSuccess: false,
    expectedError: "File not found"
  }
];

/**
 * Define Zod schemas for type-safe metadata
 */

const AllTestsMetadataSchema = z.object({
  totalTests: z.number(),
  completedTests: z.number(),
  passedTests: z.number(),
  failedTests: z.number(),
  status: z.enum(["running", "passed", "failed"]),
  currentTest: z.string().optional(),
  allPassed: z.boolean().optional(),
  results: z.array(z.any()),
  cleanedUpRuns: z.array(z.any())
});

/**
 * Helper functions for type-safe metadata access
 */
function getAllTestsMetadata() {
  return AllTestsMetadataSchema.parse(metadata.current());
}

function safeSetMetadata(key: string, value: any) {
  // Ensure the value is JSON serializable
  const safeValue = JSON.parse(JSON.stringify(value));
  metadata.set(key, safeValue);
}

/**
 * Handles cleanup of successful test runs
 */
async function handleCleanup(testCaseName: string, result: any) {
  if (!result.ok || !result.output || !result.output.prUrl || !result.output.branch) {
    return null;
  }
  
  console.log("\n========== CLEANUP STARTED ==========");
  console.log(`Cleaning up after successful test: ${testCaseName}`);
  console.log(`PR URL to close: ${result.output.prUrl}`);
  console.log(`Branch to delete: ${result.output.branch}`);
  
  safeSetMetadata("cleanupStatus", "started");
  
  const { data: cleanupResult, error: cleanupError } = await tryCatch(
    cleanupGlossaryUpdateTask.triggerAndWait({
      prNumber: result.output.prUrl,
      branch: result.output.branch,
    })
  );

  console.log(`Cleanup task completed with ID: ${cleanupResult?.id || 'unknown'}`);
  console.log("========== CLEANUP FINISHED ==========\n");

  if (cleanupError) {
    const errorMessage = cleanupError instanceof Error ? cleanupError.message : String(cleanupError);
    console.error(`❌ Error during cleanup: ${errorMessage}`);
    
    safeSetMetadata("cleanupStatus", "failed");
    safeSetMetadata("cleanupError", errorMessage);
    
    return {
      cleaned: false,
      error: errorMessage
    };
  }

  if (!cleanupResult.ok) {
    console.error("❌ Failed to clean up PR and branch");
    console.error(`Error: ${JSON.stringify(cleanupResult.error)}`);
    
    safeSetMetadata("cleanupStatus", "failed"); 
    safeSetMetadata("cleanupError", JSON.stringify(cleanupResult.error));
    
    return {
      cleaned: false,
      error: JSON.stringify(cleanupResult.error)
    };
  }

  console.log("✅ Successfully cleaned up PR and branch");
  
  safeSetMetadata("cleanupStatus", "completed");
  safeSetMetadata("cleanupDetails", {
    prClosed: cleanupResult.output.prClosed,
    branchDeleted: cleanupResult.output.branchDeleted,
    prNumber: cleanupResult.output.prNumber || "",
    branch: cleanupResult.output.branch || ""
  });

  return {
    cleaned: true,
    prClosed: cleanupResult.output.prClosed,
    branchDeleted: cleanupResult.output.branchDeleted,
    prNumber: cleanupResult.output.prNumber || "",
    branch: cleanupResult.output.branch || ""
  };
}

/**
 * Extracts error message from various error formats
 */
function extractErrorMessage(error: any): string {
  if (!error) {
    return "";
  }
  
  if (typeof error === 'object') {
    if ('message' in error && error.message) {
      return String(error.message);
    }
    if ('details' in error && error.details) {
      return String(error.details);
    }
    return JSON.stringify(error);
  }
  
  return String(error);
}

/**
 * Task to run a specific test case
 */
export const runTestCase = task({
  id: "glossary-update-content-test",
  run: async ({ testCaseName }: { testCaseName: string }) => {
    // Find the test case by name
    const testCase = testCases.find(tc => tc.name === testCaseName);
    
    if (!testCase) {
      throw new AbortTaskRunError(`Test case "${testCaseName}" not found`);
    }

    // Initialize metadata for tracking
    safeSetMetadata("testCase", testCase.name);
    safeSetMetadata("status", "running");
    safeSetMetadata("expectedSuccess", testCase.expectedSuccess);
    safeSetMetadata("input", {
      inputTerm: testCase.input.inputTerm,
      content: testCase.input.content
    });
    
    console.log(`Running test case: ${testCaseName}`);
    console.log(`Input: ${JSON.stringify(testCase.input)}`);

    // Run the update content task
    const result = await updateGlossaryContentTask.triggerAndWait(testCase.input);

    console.log(`Task result: ${JSON.stringify(result)}`);
    
    // Store safe serializable version of the result
    safeSetMetadata("taskResult", {
      ok: result.ok,
      output: result.ok && result.output ? JSON.parse(JSON.stringify(result.output)) : null,
      error: !result.ok && result.error ? JSON.stringify(result.error) : null
    });

    // Expected failure but got success
    if (!testCase.expectedSuccess && result.ok) {
      safeSetMetadata("status", "failed");
      safeSetMetadata("reason", `Expected error "${testCase.expectedError || 'unknown'}" but task succeeded`);
      
      return {
        success: false,
        message: `Test failed: Expected error "${testCase.expectedError || 'unknown'}" but task succeeded`,
        details: { result },
      };
    }

    // Expected failure and got failure
    if (!testCase.expectedSuccess && !result.ok) {
      const errorMessage = extractErrorMessage(result.error);
      
      safeSetMetadata("errorMessage", errorMessage);
      
      // got expected failure and error message matches expected
      if (errorMessage.includes(testCase.expectedError || '')) {
        safeSetMetadata("status", "passed");
        
        return {
          success: true,
          message: `Test passed: Got expected error "${testCase.expectedError}"`,
          details: { error: errorMessage },
        };
      }
      
      // got expected failure but error message doesn't match expected
      safeSetMetadata("status", "failed");
      safeSetMetadata("reason", `Error message doesn't match expected`);
      
      return {
        success: false,
        message: `Test failed: Error message doesn't match expected. Got: "${errorMessage}", Expected to include: "${testCase.expectedError || 'unknown'}"`,
        details: { error: errorMessage },
      };
    }

    // Expected success and got success
    if (testCase.expectedSuccess && result.ok) {
      // Run validation if provided
      if (testCase.validate) {
        testCase.validate(result);
      }

      // Handle cleanup if needed
      let cleanupDetails = null;
      if (testCase.shouldCleanup !== false) {
        cleanupDetails = await handleCleanup(testCaseName, result);
      }

      safeSetMetadata("status", "passed");
      if (cleanupDetails) {
        // Store serializable version of cleanup details
        safeSetMetadata("cleanup", JSON.parse(JSON.stringify(cleanupDetails)));
      }
      
      return {
        success: true,
        message: `Test passed: ${testCaseName}`,
        details: { result },
        cleanup: cleanupDetails
      };
    }

    // Expected success but got failure
    if (testCase.expectedSuccess && !result.ok) {
      safeSetMetadata("status", "failed");
      safeSetMetadata("reason", "Expected success but got error");
      
      return {
        success: false,
        message: "Test failed: Expected success but got error",
        details: { error: result.error },
      };
    }

    // Default case (should not reach here)
    safeSetMetadata("status", "failed");
    safeSetMetadata("reason", "Unexpected test result");

    
    return {
      success: false,
      message: "Test failed: Unexpected test result",
      details: { result },
    };
  }
});

/**
 * Task to run all test cases
 */
export const runAllTests = task({
  id: "glossary-update-content-test-all",
  run: async () => {
    // Initialize metadata for tracking all tests
    safeSetMetadata("totalTests", testCases.length);
    safeSetMetadata("completedTests", 0);
    safeSetMetadata("passedTests", 0);
    safeSetMetadata("failedTests", 0);
    safeSetMetadata("status", "running");
    safeSetMetadata("results", []);
    safeSetMetadata("cleanedUpRuns", []);

    console.log(`Running all ${testCases.length} test cases`);

    for (const testCase of testCases) {
      console.log(`\n========== STARTING TEST: ${testCase.name} ==========`);
      
      // Update metadata to show current test
      safeSetMetadata("currentTest", testCase.name);
      
      const { data: result, error } = await tryCatch(runTestCase.triggerAndWait({
        testCaseName: testCase.name,
      }));

      if (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        console.log(`Error running test ${testCase.name}: ${errorMessage}`);
        
        // Update metadata for failed test
        const metadata = getAllTestsMetadata();
        safeSetMetadata("completedTests", metadata.completedTests + 1);
        safeSetMetadata("failedTests", metadata.failedTests + 1);
        
        // Add to results
        safeSetMetadata("results", [
          ...metadata.results,
          {
            testCase: testCase.name,
            success: false,
            message: `Failed to run test: ${errorMessage}`
          }
        ]);
        continue;
      }

      console.log(`Test ${testCase.name} completed with result: ${JSON.stringify(result)}`);
      
      // Update completed tests count
      const metadata = getAllTestsMetadata();
      safeSetMetadata("completedTests", metadata.completedTests + 1);
      
      // Track test result - ensure it's serializable
      const testResult = {
        testCase: testCase.name,
        ...(result.ok 
          ? JSON.parse(JSON.stringify(result.output)) 
          : { success: false, message: `Task run failed: ${JSON.stringify(result.error)}` })
      };
      
      // Update results array in metadata
      safeSetMetadata("results", [...metadata.results, testResult]);
      
      // Check if the task run was successful
      if (result.ok) {
        // Track cleanup information if available
        if (result.output.cleanup) {
          safeSetMetadata("cleanedUpRuns", [
            ...metadata.cleanedUpRuns,
            {
              testCase: testCase.name,
              cleanup: JSON.parse(JSON.stringify(result.output.cleanup))
            }
          ]);
        }
        
        // Update passed/failed tests count
        if (result.output.success) {
          safeSetMetadata("passedTests", metadata.passedTests + 1);
        } else {
          safeSetMetadata("failedTests", metadata.failedTests + 1);
        }
      } else {
        safeSetMetadata("failedTests", metadata.failedTests + 1);
      }
    }

    // Update metadata for completed all tests
    const metadata = getAllTestsMetadata();
    safeSetMetadata("status", metadata.status);
    safeSetMetadata("completedTests", metadata.completedTests);
    safeSetMetadata("passedTests", metadata.passedTests);
    safeSetMetadata("failedTests", metadata.failedTests);
    safeSetMetadata("allPassed", metadata.passedTests === metadata.totalTests);

    return {
      success: true,
      message: `All ${testCases.length} test cases completed`,
      details: {
        totalTests: metadata.totalTests,
        completedTests: metadata.completedTests,
        passedTests: metadata.passedTests,
        failedTests: metadata.failedTests,
        status: metadata.status,
        currentTest: metadata.currentTest,
        allPassed: metadata.allPassed,
        results: metadata.results,
        cleanedUpRuns: metadata.cleanedUpRuns
      }
    };
  }
});