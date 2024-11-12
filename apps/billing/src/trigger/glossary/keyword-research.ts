import { db } from "@/lib/db-marketing/client";
import { keywords } from "@/lib/db-marketing/schemas";
import { getOrCreateFirecrawlResponse } from "@/lib/firecrawl";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { sql } from "drizzle-orm";
import { inArray } from "drizzle-orm";
import { and, eq } from "drizzle-orm";
import { getOrCreateKeywordsFromHeaders, getOrCreateKeywordsFromTitles } from "../../lib/keywords";
import { getOrCreateSearchQuery } from "../../lib/search-query";
import { getOrCreateSearchResponse } from "../../lib/serper";
import type { CacheStrategy } from "./_generate-glossary-entry";

export const THREE = 3;

export const keywordResearchTask = task({
  id: "keyword_research",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    term,
    onCacheHit = "stale" as CacheStrategy,
  }: { term: string; onCacheHit?: CacheStrategy }) => {
    const existing = await db.query.keywords.findMany({
      where: eq(keywords.inputTerm, term),
    });

    if (existing.length > 0 && onCacheHit === "stale") {
      return {
        message: `Found existing keywords for ${term}`,
        term,
        keywords: existing,
      };
    }

    const searchQuery = await getOrCreateSearchQuery({ term: term });
    console.info(`1/6 - SEARCH QUERY: ${searchQuery?.query}`);

    if (!searchQuery) {
      throw new AbortTaskRunError("Unable to generate search query");
    }

    const searchResponse = await getOrCreateSearchResponse({
      query: searchQuery.query,
      inputTerm: searchQuery.inputTerm,
    });
    console.info(
      `2/6 - SEARCH RESPONSE: Found ${searchResponse.serperOrganicResults.length} organic results`,
    );

    console.info(`3/6 - Getting content for top ${THREE} results`);
    const topThree = searchResponse.serperOrganicResults
      .sort((a, b) => a.position - b.position)
      .slice(0, THREE);

    // Get content for top 3 results
    const firecrawlResults = await Promise.all(
      topThree.map((result) =>
        getOrCreateFirecrawlResponse({ url: result.link, connectTo: { term: term } }),
      ),
    );

    console.info(`4/6 - Found ${firecrawlResults.length} firecrawl results`);

    const keywordsFromTitles = await getOrCreateKeywordsFromTitles({
      term: term,
    });
    console.info(`5/6 - KEYWORDS FROM TITLES: ${keywordsFromTitles.length} keywords`);

    const keywordsFromHeaders = await getOrCreateKeywordsFromHeaders({
      term: term,
    });

    console.info(`6/6 - KEYWORDS FROM HEADERS: ${keywordsFromHeaders.length} keywords`);

    // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
    await db
      .insert(keywords)
      .values(
        searchResponse.serperRelatedSearches.map((search) => ({
          inputTerm: searchQuery.inputTerm,
          keyword: search.query.toLowerCase(),
          source: "related_searches",
          updatedAt: sql`now()`,
        })),
      )
      .onDuplicateKeyUpdate({
        set: {
          updatedAt: sql`now()`,
        },
      });
    const insertedRelatedSearches = await db.query.keywords.findMany({
      where: and(
        eq(keywords.inputTerm, searchQuery.inputTerm),
        eq(keywords.source, "related_searches"),
        inArray(
          keywords.keyword,
          searchResponse.serperRelatedSearches.map((search) => search.query.toLowerCase()),
        ),
      ),
    });

    console.info(
      `✅ Keyword Research for ${term} completed. Total keywords: ${
        keywordsFromTitles.length + keywordsFromHeaders.length + insertedRelatedSearches.length
      }`,
    );

    return {
      message: `Keyword Research for ${term} completed`,
      term: searchQuery.inputTerm,
      keywords: [...keywordsFromTitles, ...keywordsFromHeaders, ...insertedRelatedSearches],
    };
  },
});
