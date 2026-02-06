import { describe, expect, it, vi } from "vitest";

import { TRPCError } from "@trpc/server";
import { getStructuredSearchFromLLM, getSystemPrompt } from "./utils";

describe("getSystemPrompt", () => {
  const referenceTime = 1706024400000; // 2024-01-23T12:00:00.000Z

  it("should include all necessary examples and constraints", () => {
    const prompt = getSystemPrompt(referenceTime);

    expect(prompt).toContain('field: "methods"');
    expect(prompt).toContain('field: "status"');
    expect(prompt).toContain('operator: "startsWith"');
    expect(prompt).toContain("GET, POST, PUT, DELETE");
  });
});

describe("getStructuredSearchFromLLM", () => {
  const mockOpenAI = {
    beta: {
      chat: {
        completions: {
          parse: vi.fn(),
        },
      },
    },
  };

  it("should return TRPCError if openai is not configured", async () => {
    await expect(
      getStructuredSearchFromLLM(null as any, "test query", 1706024400000),
    ).rejects.toThrowError(
      new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "OpenAI isn't configured correctly, please check your API key",
      }),
    );
  });

  it("should handle successful LLM response", async () => {
    const mockResponse = {
      choices: [
        {
          message: {
            parsed: {
              field: "methods",
              filters: [{ operator: "is", value: "GET" }],
            },
          },
        },
      ],
    };

    mockOpenAI.beta.chat.completions.parse.mockResolvedValueOnce(mockResponse);

    const result = await getStructuredSearchFromLLM(
      mockOpenAI as any,
      "find GET requests",
      1706024400000,
    );

    expect(result).toEqual({
      field: "methods",
      filters: [{ operator: "is", value: "GET" }],
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
      getStructuredSearchFromLLM(mockOpenAI as any, "invalid query", 1706024400000),
    ).rejects.toThrow(TRPCError);
  });

  it("should handle rate limit error", async () => {
    mockOpenAI.beta.chat.completions.parse.mockRejectedValueOnce({
      response: { status: 429 },
    });

    await expect(
      getStructuredSearchFromLLM(mockOpenAI as any, "test query", 1706024400000),
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
      getStructuredSearchFromLLM(mockOpenAI as any, "test query", 1706024400000),
    ).rejects.toThrowError(
      new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to process your search query. Please try again or contact support@unkey.com if the issue persists.",
      }),
    );
  });
});
