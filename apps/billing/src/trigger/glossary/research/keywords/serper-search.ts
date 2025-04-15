import { openai } from "@ai-sdk/openai";
import { task } from "@trigger.dev/sdk/v3";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { z } from "zod";

// Serper API Response Schema
export const SerperSearchResultSchema = z.object({
  // Search parameters are part of our request, not the response
  organic: z.array(
    z.object({
      title: z.string(),
      link: z.string(),
      snippet: z.string(),
      position: z.number().optional(),
      sitelinks: z
        .array(
          z.object({
            title: z.string(),
            link: z.string(),
          }),
        )
        .optional(),
    }),
  ),
  peopleAlsoAsk: z
    .array(
      z.object({
        question: z.string(),
        snippet: z.string(),
        title: z.string().optional(),
        link: z.string(),
      }),
    )
    .optional(),
  relatedSearches: z
    .array(
      z.object({
        query: z.string(),
      }),
    )
    .optional(),
  // Adding knowledge graph which is sometimes returned
  knowledgeGraph: z
    .object({
      title: z.string(),
      type: z.string(),
      description: z.string().optional(),
      attributes: z.record(z.string()).optional(),
      imageUrl: z.string().optional(),
    })
    .optional(),
});

// Keyword Schema as defined in MVP
export const KeywordSchema = z.object({
  keyword: z.string(),
  source: z.enum(["related_search", "llm_extracted", "autosuggest"]),
  confidence: z.number().min(0).max(1).optional(),
  context: z.string().optional(),
});

export type SerperSearchResult = z.infer<typeof SerperSearchResultSchema>;
export type Keyword = z.infer<typeof KeywordSchema>;

// Schema for task output
export const TaskOutputSchema = z.object({
  inputTerm: z.string(),
  searchResult: SerperSearchResultSchema,
  keywords: z.array(KeywordSchema),
});

export type TaskOutput = z.infer<typeof TaskOutputSchema>;

export const serperSearchTask = task({
  id: "serper_search",
  retry: {
    maxAttempts: 3,
  },
  run: async (params: { inputTerm: string }) => {
    const { inputTerm } = params;
    if (!inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    const apiKey = process.env.SERPER_API_KEY;
    if (!apiKey) {
      throw new AbortTaskRunError("SERPER_API_KEY environment variable is not set");
    }

    try {
      // 1. Fetch search results from Serper
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
          num: 10, // Default to 10 results
          type: "search", // Default search type
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new AbortTaskRunError(
          `Serper API request failed: ${response.status} ${errorData.message || response.statusText}`,
        );
      }

      const data = await response.json();
      const validationResult = SerperSearchResultSchema.safeParse(data);

      if (!validationResult.success) {
        console.error("Validation failed for response:", JSON.stringify(data, null, 2));
        throw new AbortTaskRunError(
          `Invalid API response format: ${validationResult.error.message}`,
        );
      }

      const searchResult = validationResult.data;

      // 2. Extract keywords from related searches
      const relatedKeywords = (searchResult.relatedSearches || []).map(({ query }) => ({
        keyword: query,
        source: "related_search" as const,
        confidence: 1,
        context: "Directly from Google related searches",
      }));

      // 3. Extract keywords using LLM from organic results
      const organicContent = searchResult.organic
        .map((result) => `${result.title}\n${result.snippet}`)
        .join("\n\n");

      // Also include "People Also Ask" questions for better context
      const paaContent = searchResult.peopleAlsoAsk
        ? `\n\nPeople Also Ask:\n${searchResult.peopleAlsoAsk
            .map((item) => `Q: ${item.question}\nA: ${item.snippet}`)
            .join("\n\n")}`
        : "";

      const llmResponse = await generateObject({
        model: openai("gpt-4o-mini"),
        system:
          "You are an SEO expert specializing in keyword research. Your task is to analyze search results and extract relevant keywords.",
        prompt: `Analyze these search results for '${inputTerm}' and extract relevant keywords.

Focus on keywords that would be valuable for SEO optimization, including:
- Main topic keywords
- Related concepts and terms
- Technical variations
- Common use cases
- Related technologies or standards

For each keyword:
- Set source as "llm_extracted"
- Assign a confidence score (0-1) based on relevance
- Add context explaining why this keyword is relevant

Search Results:
${organicContent}
${paaContent}`,
        schema: z.array(KeywordSchema),
        output: "array",
      });

      // 4. Combine and deduplicate keywords
      const allKeywords = [...relatedKeywords, ...llmResponse.object[0]];
      const uniqueKeywords = Array.from(
        new Map(allKeywords.map((item) => [item.keyword.toLowerCase(), item])).values(),
      );

      return {
        inputTerm,
        searchResult,
        keywords: uniqueKeywords,
      } as const;
    } catch (error) {
      if (error instanceof AbortTaskRunError) {
        throw error;
      }
      throw new AbortTaskRunError(
        `Failed to fetch search results: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  },
});
