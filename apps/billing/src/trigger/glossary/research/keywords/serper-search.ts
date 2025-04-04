import { z } from "zod";
import { task } from "@trigger.dev/sdk/v3";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";

// Serper API Response Schema
export const SerperSearchResultSchema = z.object({
  searchParameters: z.object({
    q: z.string(),
    gl: z.string(),
    hl: z.string(),
    num: z.number(),
    type: z.string()
  }),
  searchInformation: z.object({
    totalResults: z.string(),
    formattedTotalResults: z.string(),
    formattedSearchTime: z.string()
  }),
  organic: z.array(z.object({
    title: z.string(),
    link: z.string(),
    snippet: z.string(),
    position: z.number(),
    sitelinks: z.array(z.object({
      title: z.string(),
      link: z.string()
    })).optional()
  })),
  peopleAlsoAsk: z.array(z.object({
    question: z.string(),
    snippet: z.string(),
    title: z.string(),
    link: z.string()
  })).optional(),
  relatedSearches: z.array(z.object({
    query: z.string()
  })).optional()
});

export type SerperSearchResult = z.infer<typeof SerperSearchResultSchema>;

export const serperSearchTask = task({
  id: "serper_search",
  retry: {
    maxAttempts: 3,
  },
  run: async ({ inputTerm }: { inputTerm: string }): Promise<{ inputTerm: string; result: SerperSearchResult }> => {
    if (!inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    const apiKey = process.env.SERPER_API_KEY;
    if (!apiKey) {
      throw new AbortTaskRunError("SERPER_API_KEY environment variable is not set");
    }

    try {
      const response = await fetch("https://google.serper.dev/search", {
        method: "POST",
        headers: {
          "X-API-KEY": apiKey,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          q: inputTerm,
        }),
      });

      if (!response.ok) {
        throw new AbortTaskRunError(`Serper API request failed: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      const validationResult = SerperSearchResultSchema.safeParse(data);

      if (!validationResult.success) {
        throw new AbortTaskRunError(`Invalid API response format: ${validationResult.error.message}`);
      }

      return {
        inputTerm,
        result: validationResult.data,
      };
    } catch (error) {
      if (error instanceof AbortTaskRunError) {
        throw error;
      }
      throw new AbortTaskRunError(`Failed to fetch search results: ${error instanceof Error ? error.message : String(error)}`);
    }
  },
}); 