import { z } from "zod";
import { task } from "@trigger.dev/sdk/v3";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";

// Serper API Response Schema
export const SerperSearchResultSchema = z.object({
  // Search parameters are part of our request, not the response
  organic: z.array(z.object({
    title: z.string(),
    link: z.string(),
    snippet: z.string(),
    position: z.number().optional(),
    sitelinks: z.array(z.object({
      title: z.string(),
      link: z.string()
    })).optional()
  })),
  peopleAlsoAsk: z.array(z.object({
    question: z.string(),
    snippet: z.string(),
    title: z.string().optional(),
    link: z.string()
  })).optional(),
  relatedSearches: z.array(z.object({
    query: z.string()
  })).optional(),
  // Adding knowledge graph which is sometimes returned
  knowledgeGraph: z.object({
    title: z.string(),
    type: z.string(),
    description: z.string().optional(),
    attributes: z.record(z.string()).optional(),
    imageUrl: z.string().optional(),
  }).optional()
});

export type SerperSearchResult = z.infer<typeof SerperSearchResultSchema>;

export const serperSearchTask = task({
  id: "serper_search",
  retry: {
    maxAttempts: 3,
  },
  run: async ({ inputTerm }: { inputTerm: string }) => {
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
          gl: "us", // Default to US results
          hl: "en", // Default to English
          num: 10,  // Default to 10 results
          type: "search", // Default search type
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new AbortTaskRunError(`Serper API request failed: ${response.status} ${errorData.message || response.statusText}`);
      }

      const data = await response.json();
      const validationResult = SerperSearchResultSchema.safeParse(data);

      if (!validationResult.success) {
        console.error("Validation failed for response:", JSON.stringify(data, null, 2));
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