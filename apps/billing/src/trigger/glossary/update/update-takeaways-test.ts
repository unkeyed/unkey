import { metadata, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";
import type { FieldSelection } from "../generate/takeaways/generate-takeaways";
import { updateTakeawaysTask } from "./update-takeaways";
import { updateTakeawaysCleanupTask } from "./update-takeaways-cleanup";
import { takeawaysSchema } from "@/lib/db-marketing/schemas/takeaways-schema";

// Test Metadata Schema
const TestMetadataSchema = z.object({
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

type TestMetadata = z.infer<typeof TestMetadataSchema>;

/**
 * Helper functions for type-safe metadata access
 */
function getAllTestsMetadata(): TestMetadata {
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

function safeSetMetadata(key: string, value: any) {
  const safeValue = JSON.parse(JSON.stringify(value));
  metadata.set(key, safeValue);
}

// Base schema for task run results
const okResultSchema = z.object({
  ok: z.literal(true),
  id: z.string(),
  taskIdentifier: z.string(),
  output: z.any(),
});

const errorResultSchema = z.object({
  ok: z.literal(false),
  id: z.string(),
  taskIdentifier: z.string(),
  error: z.unknown(),
});

// Pre-generated takeaways from previous run
const preGeneratedTakeaways = {
  tldr: "MIME types are crucial for APIs to specify the format of data being sent and received. They ensure that clients and servers correctly interpret the content. Always use the `Content-Type` header.",
  definitionAndStructure: [
    {
      key: "Definition",
      value:
        "MIME (Multipurpose Internet Mail Extensions) types are standard identifiers used to indicate the nature and format of a document, file, or assortment of bytes. They are used in HTTP headers to specify the media type of content transmitted between a server and a client.",
    },
    {
      key: "Structure",
      value:
        "A MIME type consists of a type and a subtype, separated by a forward slash (/). Optionally, it can include parameters. For example: `application/json; charset=utf-8`.",
    },
    {
      key: "Type",
      value:
        "Represents the general category of the content (e.g., `application`, `text`, `image`, `audio`, `video`).",
    },
    {
      key: "Subtype",
      value:
        "Specifies the specific format or type within the general category (e.g., `json`, `xml`, `png`, `jpeg`).",
    },
    {
      key: "Parameters",
      value:
        "Provide additional information about the content, such as character set (`charset`), encoding, or version. Parameters are key-value pairs separated by semicolons (`;`).",
    },
  ],
  historicalContext: [
    {
      key: "Origin",
      value:
        "MIME was originally designed to extend the capabilities of email (SMTP) to support different content types beyond plain text.",
    },
    {
      key: "Adoption",
      value:
        "It was later adopted by HTTP to standardize the way web servers and browsers exchange data.",
    },
    {
      key: "Evolution",
      value:
        "Over time, MIME types have become a fundamental part of the internet, enabling the transfer of diverse media types.",
    },
  ],
  usageInAPIs: {
    tags: ["Content-Type", "Accept", "HTTP Headers", "Serialization", "Deserialization"],
    description:
      "MIME types are used in APIs primarily within the `Content-Type` and `Accept` HTTP headers. The `Content-Type` header in a response specifies the format of the response body (e.g., `application/json`). The `Accept` header in a request indicates the media types the client can handle, allowing the server to provide the most appropriate response format.",
  },
  bestPractices: [
    "Always specify the `Content-Type` header in your API responses to indicate the media type of the response body.",
    "When accepting data, validate the `Content-Type` header in the request to ensure it matches the expected format.",
    "Support multiple MIME types for both requests and responses to provide flexibility to API consumers.",
    "Use specific and accurate MIME types. Avoid generic types like `application/octet-stream` unless absolutely necessary.",
    "Document the supported MIME types clearly in your API documentation.",
  ],
  recommendedReading: [
    {
      title: "MDN Web Docs: HTTP Content-Type",
      url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Type",
    },
    {
      title: "IETF RFC 6838: Media Type Specifications and Registration Procedures",
      url: "https://datatracker.ietf.org/doc/html/rfc6838",
    },
    {
      title: "W3C: Content Negotiation",
      url: "https://www.w3.org/Protocols/rfc2616/rfc2616-sec12.html",
    },
  ],
  didYouKnow:
    "The `Content-Disposition` header, often used with a `filename` parameter, can be used in conjunction with the `Content-Type` header to suggest a filename for the downloaded resource.",
};

// Test Case Interface
interface TestCase {
  name: string;
  input: {
    term: string;
    takeaways: typeof preGeneratedTakeaways;
    fields?: FieldSelection;
  };
  validate(result: Awaited<ReturnType<(typeof updateTakeawaysTask)["triggerAndWait"]>>): boolean;
  cleanup?: (
    result: Awaited<ReturnType<(typeof updateTakeawaysTask)["triggerAndWait"]>>,
  ) => Promise<void>;
}

const testCases: TestCase[] = [
  {
    name: "fullUpdateTest",
    input: {
      term: "MIME types",
      takeaways: preGeneratedTakeaways,
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed on result validation. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const expectedOutputSchema = z.object({
        inputTerm: z.literal("MIME types"),
        updated: z.literal(true),
        prUrl: z.string().url(),
        branch: z.string(),
        updatedFields: z.record(z.any()),
      });

      const output = validation.data.output;
      const outputValidation = expectedOutputSchema.safeParse(output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
    async cleanup(result) {
      if (!result.ok) {
        return;
      }

      const { prUrl, branch } = result.output;
      const cleanupResult = await updateTakeawaysCleanupTask.triggerAndWait({ prUrl, branch });

      const metadata = getAllTestsMetadata();
      if (cleanupResult.ok) {
        metadata.cleanupResults.push({
          testCase: this.name,
          status: "success",
          prClosed: cleanupResult.output.prClosed,
          branchDeleted: cleanupResult.output.branchDeleted,
        });
      } else {
        metadata.cleanupResults.push({
          testCase: this.name,
          status: "failed",
          error:
            cleanupResult.error instanceof Error
              ? cleanupResult.error.message
              : String(cleanupResult.error),
        });
      }
      safeSetMetadata("testMetadata", metadata);
    },
  },
  {
    name: "partialUpdateTest",
    input: {
      term: "MIME types",
      takeaways: preGeneratedTakeaways,
      fields: {
        tldr: true,
        definitionAndStructure: [0, 1],
        usageInAPIs: {
          description: true,
        },
      },
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const expectedOutputSchema = z.object({
        inputTerm: z.literal("MIME types"),
        updated: z.literal(true),
        prUrl: z.string().url(),
        branch: z.string(),
        updatedFields: z.object({
          tldr: takeawaysSchema.shape.tldr,
          definitionAndStructure: takeawaysSchema.shape.definitionAndStructure,
          usageInAPIs: z.object({
            description: takeawaysSchema.shape.usageInAPIs.shape.description
          }),
        }),
      });

      const outputValidation = expectedOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
    async cleanup(result) {
      if (!result.ok) {
        return;
      }

      const { prUrl, branch } = result.output;
      const cleanupResult = await updateTakeawaysCleanupTask.triggerAndWait({ prUrl, branch });

      const metadata = getAllTestsMetadata();
      if (cleanupResult.ok) {
        metadata.cleanupResults.push({
          testCase: this.name,
          status: "success",
          prClosed: cleanupResult.output.prClosed,
          branchDeleted: cleanupResult.output.branchDeleted,
        });
      } else {
        metadata.cleanupResults.push({
          testCase: this.name,
          status: "failed",
          error:
            cleanupResult.error instanceof Error
              ? cleanupResult.error.message
              : String(cleanupResult.error),
        });
      }
      safeSetMetadata("testMetadata", metadata);
    },
  },
  {
    name: "invalidTermTest",
    input: {
      term: "",
      takeaways: preGeneratedTakeaways,
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

export const updateTakeawaysTest = task({
  id: "update_takeaways_test",
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
      cleanupResults: [],
    };
    safeSetMetadata("testMetadata", initialMetadata);
  },
  run: async () => {
    const startTime = Date.now();

    for (const testCase of testCases) {
      const testStartTime = Date.now();

      const runResult = await updateTakeawaysTask.triggerAndWait(testCase.input);
      const validation = testCase.validate(runResult);

      const metadata = getAllTestsMetadata();
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

    console.log(`\nDuration: ${Date.now() - startTime}ms`);
    console.log("===============================");

    return metadata;
  },
});
