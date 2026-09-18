import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod";
import { SEARCH_MODEL } from "./client";
import { buildSystemPrompt } from "./prompt";
import type { FilterOutput, SearchSpec } from "./spec";

export type SearchRun = {
  result: FilterOutput;
  inputTokens: number;
  outputTokens: number;
};

function unreadable(spec: SearchSpec): TRPCError {
  const fields = Object.keys(spec.fields).slice(0, 4).join(", ");
  return new TRPCError({
    code: "UNPROCESSABLE_CONTENT",
    message: `Could not read that as a filter. Try naming what you want to narrow by, such as ${fields}.\nFor additional help, contact support@unkey.com`,
  });
}

export async function runSearch(
  spec: SearchSpec,
  openai: OpenAI | null,
  query: string,
  referenceMs: number,
): Promise<SearchRun> {
  if (!openai) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "OpenAI isn't configured correctly, please check your API key",
    });
  }

  try {
    const completion = await openai.chat.completions.parse({
      model: SEARCH_MODEL,
      temperature: 0.1,
      n: 1,
      messages: [
        { role: "system", content: buildSystemPrompt(spec, referenceMs) },
        { role: "user", content: query },
      ],
      response_format: zodResponseFormat(spec.outputSchema, "searchQuery"),
    });

    const parsed = spec.outputSchema.safeParse(completion.choices[0].message.parsed);
    if (!parsed.success) {
      throw unreadable(spec);
    }

    return {
      result: parsed.data,
      inputTokens: completion.usage?.prompt_tokens ?? 0,
      outputTokens: completion.usage?.completion_tokens ?? 0,
    };
  } catch (error) {
    if (error instanceof TRPCError) {
      throw error;
    }
    if ((error as { status?: number }).status === 429) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Search rate limit exceeded. Please try again in a few minutes.",
      });
    }
    console.error(
      `${spec.subject} search failed. Input: ${JSON.stringify(query)}\n Output ${(error as Error).message}`,
    );
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "Failed to process your search query. Please try again or contact support@unkey.com if the issue persists.",
    });
  }
}

export function createSearch(spec: SearchSpec) {
  return async (openai: OpenAI | null, query: string, referenceMs = Date.now()) => {
    const { result } = await runSearch(spec, openai, query, referenceMs);
    return result;
  };
}
