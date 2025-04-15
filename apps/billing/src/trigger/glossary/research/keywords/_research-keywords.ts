import { AbortTaskRunError, batch, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";
import { enrichKeywordsTask } from "./enrich-keywords";
import { relatedKeywordsTask } from "./related-keywords";
import { serperAutosuggestTask } from "./serper-autosuggest";
import { serperSearchTask } from "./serper-search";

// Input schema
const ParentTaskPayload = z.object({
  inputTerm: z.string(),
});

// Output schema for unified keyword data
const UnifiedKeywordSchema = z.object({
  keyword: z.string(),
  volume: z.number(),
  cpc: z.number(),
  competition: z.number(),
  source: z.enum(["massiveonlinemarketing.nl", "related_search", "autosuggest", "llm_extracted"]),
});

const ParentTaskOutput = z.object({
  keywords: z.array(UnifiedKeywordSchema),
  metadata: z.object({
    totalKeywords: z.number(),
    sources: z.object({
      relatedKeywords: z.number(),
      serperSearch: z.number(),
      serperAutosuggest: z.number(),
    }),
    deduplication: z.object({
      total: z.number(),
      skippedEnrichment: z.number(),
      duplicatesRemoved: z.number(),
    }),
  }),
});

// Helper to normalize keywords for comparison
const normalizeKeyword = (keyword: string) => keyword.toLowerCase().trim();

/**
 * Performs comprehensive keyword research by combining and enriching data from multiple sources.
 *
 * Flow:
 * 1. Executes three tasks in parallel:
 *    - Related keywords from massiveonlinemarketing.nl
 *    - Serper search results
 *    - Serper autosuggest results
 *
 * 2. Processes results in priority order:
 *    a. Related keywords are processed first as primary source
 *    b. Serper search results are enriched if confidence >= 0.8
 *    c. Serper autosuggest results are enriched if confidence >= 0.8
 *
 * 3. Deduplication:
 *    - Uses normalized keywords (lowercase, trimmed) as unique keys
 *    - Skips enrichment for already known keywords
 *    - Tracks metadata about deduplication process
 *
 * @param payload - Contains the input term for keyword research
 * @param payload.inputTerm - The seed keyword to research
 *
 * @returns {Promise<{
 *   keywords: Array<{
 *     keyword: string;
 *     volume: number;
 *     cpc: number;
 *     competition: number;
 *     source: "massiveonlinemarketing.nl" | "relatedSearch" | "autosuggest" | "llm_extracted";
 *   }>;
 *   metadata: {
 *     totalKeywords: number;
 *     sources: {
 *       relatedKeywords: number;
 *       serperSearch: number;
 *       serperAutosuggest: number;
 *     };
 *     deduplication: {
 *       total: number;
 *       skippedEnrichment: number;
 *       duplicatesRemoved: number;
 *     };
 *   };
 * }>}
 *
 * @throws {AbortTaskRunError} If inputTerm is empty
 */
export const researchKeywords = task({
  id: "research_keywords",
  run: async (payload: z.infer<typeof ParentTaskPayload>) => {
    if (!payload.inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    // 1. Execute initial tasks in parallel
    const { runs } = await batch.triggerByTaskAndWait([
      { task: relatedKeywordsTask, payload: { inputTerm: payload.inputTerm } },
      { task: serperSearchTask, payload: { inputTerm: payload.inputTerm } },
      { task: serperAutosuggestTask, payload: { inputTerm: payload.inputTerm } },
    ]);

    // 2. Process results and handle errors
    const [relatedResult, serperSearchResult, serperAutosuggestResult] = runs;

    // Initialize tracking variables
    let totalKeywordsBeforeDedup = 0;
    let skippedEnrichment = 0;

    // Process related keywords first as they're our primary source
    const keywordMap = new Map<string, z.infer<typeof UnifiedKeywordSchema>>();
    if (relatedResult.ok) {
      relatedResult.output.keywordIdeas.forEach((kw) => {
        const normalized = normalizeKeyword(kw.keyword);
        keywordMap.set(normalized, {
          keyword: kw.keyword,
          volume: kw.avg_monthly_searches,
          cpc: kw.high_top_of_page_bid_micros / 1_000_000,
          competition: kw.competition,
          source: "massiveonlinemarketing.nl",
        } satisfies z.infer<typeof UnifiedKeywordSchema>);
      });
      totalKeywordsBeforeDedup += relatedResult.output.keywordIdeas.length;
    }

    // Helper to filter out keywords we already have data for so that we don't enrich them again
    const filterNewKeywords = (keywords: Array<{ keyword: string; confidence?: number }>) => {
      const newKeywords = keywords.filter(
        (kw) => (kw.confidence ?? 0) >= 0.8 && !keywordMap.has(normalizeKeyword(kw.keyword)),
      );
      skippedEnrichment += keywords.length - newKeywords.length;
      return newKeywords;
    };

    // Enrich Serper search results with keyword metrics
    if (serperSearchResult.ok) {
      const newSearchKeywords = filterNewKeywords(serperSearchResult.output.keywords);

      if (newSearchKeywords.length > 0) {
        const enrichedSearch = await enrichKeywordsTask.triggerAndWait({
          keywords: newSearchKeywords.map((kw) => ({
            keyword: kw.keyword,
            source: "llm_extracted" as const,
            confidence: kw.confidence,
            context: (kw as { context?: string }).context ?? "Search result",
          })),
        });

        if (enrichedSearch.ok) {
          enrichedSearch.output.enrichedKeywords.forEach((kw) => {
            const normalized = normalizeKeyword(kw.keyword);
            if (!keywordMap.has(normalized)) {
              keywordMap.set(normalized, {
                keyword: kw.keyword,
                volume: kw.volume,
                cpc: kw.cpc,
                competition: kw.competition,
                source: "llm_extracted",
              } satisfies z.infer<typeof UnifiedKeywordSchema>);
            }
          });
          totalKeywordsBeforeDedup += enrichedSearch.output.enrichedKeywords.length;
        }
      }
    }

    // Enrich Serper autosuggest results with keyword metrics
    if (serperAutosuggestResult.ok) {
      const newAutosuggestKeywords = filterNewKeywords(serperAutosuggestResult.output.keywords);

      if (newAutosuggestKeywords.length > 0) {
        const enrichedAutosuggest = await enrichKeywordsTask.triggerAndWait({
          keywords: newAutosuggestKeywords.map((kw) => ({
            keyword: kw.keyword,
            source: "autosuggest" as const,
            confidence: kw.confidence,
            context: (kw as { context?: string }).context ?? "Autosuggest result",
          })),
        });

        if (enrichedAutosuggest.ok) {
          enrichedAutosuggest.output.enrichedKeywords.forEach((kw) => {
            const normalized = normalizeKeyword(kw.keyword);
            if (!keywordMap.has(normalized)) {
              keywordMap.set(normalized, {
                keyword: kw.keyword,
                volume: kw.volume,
                cpc: kw.cpc,
                competition: kw.competition,
                source: "autosuggest",
              } satisfies z.infer<typeof UnifiedKeywordSchema>);
            }
          });
          totalKeywordsBeforeDedup += enrichedAutosuggest.output.enrichedKeywords.length;
        }
      }
    }

    // Get final unique keywords array
    const uniqueKeywords = Array.from(keywordMap.values());

    // Calculate source counts for metadata
    const sourceCounts = {
      relatedKeywords: relatedResult.ok ? relatedResult.output.keywordIdeas.length : 0,
      serperSearch: serperSearchResult.ok
        ? serperSearchResult.output.keywords.filter((k) => (k.confidence ?? 0) >= 0.8).length
        : 0,
      serperAutosuggest: serperAutosuggestResult.ok
        ? serperAutosuggestResult.output.keywords.filter((k) => (k.confidence ?? 0) >= 0.8).length
        : 0,
    };

    // Return unified results
    return ParentTaskOutput.parse({
      keywords: uniqueKeywords.sort((a, b) => b.volume - a.volume),
      metadata: {
        totalKeywords: uniqueKeywords.length,
        sources: sourceCounts,
        deduplication: {
          total: totalKeywordsBeforeDedup,
          skippedEnrichment,
          duplicatesRemoved: totalKeywordsBeforeDedup - uniqueKeywords.length,
        },
      },
    });
  },
});
