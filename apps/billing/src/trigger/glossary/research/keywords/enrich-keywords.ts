import { task } from "@trigger.dev/sdk/v3";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";
import { z } from "zod";
import { KeywordSchema } from "./serper-search";

// Keywords Everywhere API Response Schema
const KeywordsEverywhereResponseSchema = z.object({
  data: z.array(
    z.object({
      keyword: z.string(),
      vol: z.number(),
      cpc: z.object({
        currency: z.string(),
        value: z.string(),
      }),
      competition: z.number(),
      trend: z
        .array(
          z.object({
            month: z.string(),
            year: z.number(),
            value: z.number(),
          }),
        )
        .optional()
        .default([]),
    }),
  ),
  credits: z.number(),
  credits_consumed: z.number(),
  time: z.number(),
});

// Enriched keyword schema extends the base keyword schema
const EnrichedKeywordSchema = KeywordSchema.extend({
  volume: z.number(),
  cpc: z.number(),
  competition: z.number(),
  trends: z
    .array(
      z.object({
        month: z.string(),
        year: z.number(),
        value: z.number(),
      }),
    )
    .optional(),
});

// Task input schema
const TaskInputSchema = z.object({
  keywords: z.array(KeywordSchema),
});

// Task output schema
export const TaskOutputSchema = z.object({
  enrichedKeywords: z.array(EnrichedKeywordSchema),
  metadata: z.object({
    totalProcessed: z.number(),
    enrichedCount: z.number(),
    skippedCount: z.number(),
    skippedKeywords: z.array(z.string()),
    creditsUsed: z.number(),
    creditsRemaining: z.number(),
    processingTime: z.number(),
    timestamp: z.string(),
  }),
});

export const enrichKeywordsTask = task({
  id: "enrich-keywords",
  run: async (payload: z.infer<typeof TaskInputSchema>) => {
    const { keywords } = payload;

    // Check if we have too many keywords
    if (keywords.length > 100) {
      throw new AbortTaskRunError("Cannot process more than 100 keywords at once");
    }

    // Prepare API request
    const params = new URLSearchParams();
    keywords.forEach((kw) => params.append("kw[]", kw.keyword));
    params.append("country", "us");
    params.append("currency", "usd");
    params.append("dataSource", "gkp");

    // Make API request
    const response = await fetch("https://api.keywordseverywhere.com/v1/get_keyword_data", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${process.env.KEYWORDS_EVERYWHERE_API_KEY}`,
      },
      body: params,
    });

    if (!response.ok) {
      throw new AbortTaskRunError(
        `Keywords Everywhere API error: ${response.status} ${response.statusText}`,
      );
    }

    const rawData = await response.json();
    const keData = KeywordsEverywhereResponseSchema.parse(rawData);

    // Track skipped keywords
    const skippedKeywords: string[] = [];

    // Map and filter the enriched data
    const enrichedKeywords = keywords
      .map((keyword) => {
        const enrichment = keData.data.find(
          (d) => d.keyword.toLowerCase() === keyword.keyword.toLowerCase(),
        );

        if (!enrichment) {
          skippedKeywords.push(keyword.keyword);
          return null;
        }

        // Skip if no meaningful data (all metrics are zero)
        if (
          enrichment.vol === 0 &&
          Number.parseFloat(enrichment.cpc.value) === 0 &&
          enrichment.competition === 0
        ) {
          skippedKeywords.push(keyword.keyword);
          return null;
        }

        return {
          ...keyword,
          volume: enrichment.vol,
          cpc: Number.parseFloat(enrichment.cpc.value),
          competition: enrichment.competition,
          trends: enrichment.trend,
        };
      })
      .filter((k): k is NonNullable<typeof k> => k !== null);

    // Prepare output with enhanced metadata
    const output = {
      enrichedKeywords,
      metadata: {
        totalProcessed: keywords.length,
        enrichedCount: enrichedKeywords.length,
        skippedCount: skippedKeywords.length,
        skippedKeywords,
        creditsUsed: keData.credits_consumed,
        creditsRemaining: keData.credits,
        processingTime: keData.time,
        timestamp: new Date().toISOString(),
      },
    };

    return TaskOutputSchema.parse(output);
  },
});

// Export types for external use
export type EnrichedKeyword = z.infer<typeof EnrichedKeywordSchema>;
export type TaskInput = z.infer<typeof TaskInputSchema>;
export type TaskOutput = z.infer<typeof TaskOutputSchema>;
