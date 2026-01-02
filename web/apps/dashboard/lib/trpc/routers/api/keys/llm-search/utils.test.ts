import { TRPCError } from "@trpc/server";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { describe, expect, it, vi } from "vitest";
import { getKeysStructuredSearchFromLLM, getKeysSystemPrompt } from "./utils";

describe("getKeysSystemPrompt", () => {
  const referenceTime = 1706024400000; // 2024-01-23T12:00:00.000Z

  it("should include all necessary examples and constraints", () => {
    const prompt = getKeysSystemPrompt(referenceTime);

    // Check for outcome examples
    expect(prompt).toContain("valid");
    expect(prompt).toContain("invalid");

    // Check for time-based examples
    expect(prompt).toContain("show keys from last 30m");
    expect(prompt).toContain("find keys between yesterday and today");

    // Check for name examples
    expect(prompt).toContain("find keys with name");
    expect(prompt).toContain("name containing");

    // Check for key ID examples
    expect(prompt).toContain("find key key_123");

    // Check for field configurations
    expect(prompt).toContain('field: "outcomes"');
    expect(prompt).toContain('field: "names"');
    expect(prompt).toContain('field: "keyIds"');
    expect(prompt).toContain('operator: "contains"');
    expect(prompt).toContain('operator: "is"');

    // Check for dynamic outcome values
    KEY_VERIFICATION_OUTCOMES.forEach((outcome) => {
      expect(prompt).toContain(outcome);
    });
  });
});

describe("getKeysStructuredSearchFromLLM", () => {
  const mockOpenAI = {
    beta: {
      chat: {
        completions: {
          parse: vi.fn(),
        },
      },
    },
  };

  it("should return null if openai is not configured", async () => {
    const result = await getKeysStructuredSearchFromLLM(null as any, "test query", 1706024400000);
    expect(result).toBeNull();
  });

  it("should handle successful LLM response", async () => {
    const mockResponse = {
      choices: [
        {
          message: {
            parsed: {
              field: "outcomes",
              filters: [{ operator: "is", value: "invalid" }],
            },
          },
        },
      ],
    };
    mockOpenAI.beta.chat.completions.parse.mockResolvedValueOnce(mockResponse);

    const result = await getKeysStructuredSearchFromLLM(
      mockOpenAI as any,
      "find invalid keys",
      1706024400000,
    );

    expect(result).toEqual({
      field: "outcomes",
      filters: [{ operator: "is", value: "invalid" }],
    });
  });

  it("should handle complex query with multiple filters", async () => {
    const mockResponse = {
      choices: [
        {
          message: {
            parsed: {
              field: "names",
              filters: [
                { operator: "contains", value: "test" },
                { operator: "is", value: "production-key" },
              ],
            },
          },
        },
      ],
    };
    mockOpenAI.beta.chat.completions.parse.mockResolvedValueOnce(mockResponse);

    const result = await getKeysStructuredSearchFromLLM(
      mockOpenAI as any,
      "find keys containing test with name production-key",
      1706024400000,
    );

    expect(result).toEqual({
      field: "names",
      filters: [
        { operator: "contains", value: "test" },
        { operator: "is", value: "production-key" },
      ],
    });
  });

  it("should handle unparseable response", async () => {
    const mockResponse = {
      choices: [
        {
          message: {
            parsed: null,
          },
        },
      ],
    };
    mockOpenAI.beta.chat.completions.parse.mockResolvedValueOnce(mockResponse);

    await expect(
      getKeysStructuredSearchFromLLM(mockOpenAI as any, "invalid query", 1706024400000),
    ).rejects.toThrow(TRPCError);
  });

  it("should handle rate limit error", async () => {
    mockOpenAI.beta.chat.completions.parse.mockRejectedValueOnce({
      response: { status: 429 },
    });

    await expect(
      getKeysStructuredSearchFromLLM(mockOpenAI as any, "test query", 1706024400000),
    ).rejects.toThrowError(
      new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Search rate limit exceeded. Please try again in a few minutes.",
      }),
    );
  });

  it("should handle general errors", async () => {
    mockOpenAI.beta.chat.completions.parse.mockRejectedValueOnce(new Error("Unknown error"));

    await expect(
      getKeysStructuredSearchFromLLM(mockOpenAI as any, "test query", 1706024400000),
    ).rejects.toThrowError(
      new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to process your search query. Please try again or contact support@unkey.dev if the issue persists.",
      }),
    );
  });
});
