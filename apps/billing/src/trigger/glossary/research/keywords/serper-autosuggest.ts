import { task } from "@trigger.dev/sdk/v3";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";
import { z } from "zod";
import { KeywordSchema } from "./serper-search";

// Serper Autosuggest Response Schema
export const SerperAutosuggestResultSchema = z.object({
  searchParameters: z.object({
    q: z.string(),
    type: z.string(),
    engine: z.string(),
  }),
  suggestions: z.array(
    z.object({
      value: z.string(),
    }),
  ),
  credits: z.number(),
});

// Task Output Schema
export const TaskOutputSchema = z.object({
  inputTerm: z.string(),
  searchResult: SerperAutosuggestResultSchema,
  keywords: z.array(KeywordSchema),
});

export type SerperAutosuggestResult = z.infer<typeof SerperAutosuggestResultSchema>;
export type Keyword = z.infer<typeof KeywordSchema>;
export type TaskOutput = z.infer<typeof TaskOutputSchema>;

export const serperAutosuggestTask = task({
  id: "serper_autosuggest",
  retry: {
    maxAttempts: 3,
  },
  run: async (params: { inputTerm: string }): Promise<TaskOutput> => {
    const { inputTerm } = params;
    if (!inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    const apiKey = process.env.SERPER_API_KEY;
    if (!apiKey) {
      throw new AbortTaskRunError("SERPER_API_KEY environment variable is not set");
    }

    try {
      const response = await fetch("https://google.serper.dev/autocomplete", {
        method: "POST",
        headers: {
          "X-API-KEY": apiKey,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          q: inputTerm,
          gl: "us", // Default to US results
          hl: "en", // Default to English
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new AbortTaskRunError(
          `Serper API request failed: ${response.status} ${errorData.message || response.statusText}`,
        );
      }

      const data = await response.json();
      const validationResult = SerperAutosuggestResultSchema.safeParse(data);
      if (!validationResult.success) {
        console.error("Validation failed for response:", JSON.stringify(data, null, 2));
        throw new AbortTaskRunError(
          `Invalid API response format: ${validationResult.error.message}`,
        );
      }

      const searchResult = validationResult.data;

      // Transform autosuggest results into our standard keyword format
      const keywords = searchResult.suggestions.map((suggestion) => ({
        keyword: suggestion.value,
        source: "autosuggest" as const,
        confidence: 1.0,
        context: "Direct Google autocomplete suggestion",
      }));

      return {
        inputTerm,
        searchResult,
        keywords,
      };
    } catch (error) {
      if (error instanceof AbortTaskRunError) {
        throw error;
      }
      throw new AbortTaskRunError(
        `Failed to fetch autocomplete suggestions: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  },
});
