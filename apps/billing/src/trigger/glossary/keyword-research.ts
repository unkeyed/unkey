import { db } from "@/lib/db-marketing/client";
import { keywords } from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { sql } from "drizzle-orm";
import { inArray } from "drizzle-orm";
import { and, eq } from "drizzle-orm";
import { z } from "zod";
import { getOrCreateSearchQuery } from "../../lib/search-query";
import { getOrCreateSearchResponse } from "../../lib/serper";
import { getOrCreateFirecrawlResponse } from "../../lib/firecrawl";
import type { FirecrawlResponse } from "@/lib/db-marketing/schemas";

export const keywordResearchTask = task({
  id: "keyword_research",
  retry: {
    maxAttempts: 0,
  },
  run: async (payload: { term: string }) => {
    const searchQuery = await getOrCreateSearchQuery({ term: payload.term });
    console.info(`1/5 - SEARCH QUERY: ${searchQuery?.query}`);

    if (!searchQuery) {
      throw new AbortTaskRunError("Unable to generate search query");
    }

    const searchResponse = await getOrCreateSearchResponse({
      query: searchQuery.query,
      inputTerm: searchQuery.inputTerm,
    });
    console.info(`2/5 - SEARCH RESPONSE: Found ${searchResponse.serperOrganicResults.length} organic results`);

    // Get content for top 3 results
    const THREE = 3;
    const topResults = searchResponse.serperOrganicResults.filter(
      (result) => result.position <= THREE,
    );

    if (topResults.length === 0) {
      console.warn(`No top results found for term: ${payload.term}`);
      throw new AbortTaskRunError(`No search results found for term: ${payload.term}`);
    }

    console.info(`3/5 - Processing top ${topResults.length} results:
      ${topResults.map(r => `\n- ${r.link} (position: ${r.position})`).join('')}
    `);

    const scrapedContent = await Promise.all(
      topResults.map(async (result) => {
        console.info(`Scraping content for ${result.link}...`);
        const response = await getOrCreateFirecrawlResponse(result.link);
        if (!response) {
          console.warn(`Failed to get content for ${result.link}`);
          return null;
        }
        if (!response.success || !response.markdown) {
          console.warn(`No valid content for ${result.link}. Success: ${response.success}, Has markdown: ${!!response.markdown}`);
          return null;
        }
        console.info(`Successfully scraped content for ${result.link}`);
        return response;
      }),
    );

    const validContent = scrapedContent.filter((content): content is FirecrawlResponse => 
      content !== null && content.success && content.markdown !== null
    );

    console.info(`4/5 - Found ${validContent.length} valid content items out of ${scrapedContent.length} attempts`);

    if (validContent.length === 0) {
      throw new AbortTaskRunError(`No valid content found for any of the top ${THREE} results for term: ${payload.term}`);
    }

    const keywordResearchSystemPrompt = `
You are an SEO Expert & Content Writer specializing in creating technical content for Developer Tools that are highly SEO optimized.

**Your Objectives:**
1. **Keyword Extraction:**
   - Extract relevant keywords from the titles of top-ranking organic search results.
   - Focus on technical and context-specific terms related to API development.

2. **Quality Assurance:**
   - **Remove Stopwords:** Ensure that keywords do not include common stopwords (e.g., "for," "and," "the," "of," etc.).
   - **Remove Brand Names:** Ensure that keywords do not include brand names (e.g., "GitHub", "YouTube", "npm", etc.).
   - **Remove README keywords:** Ensure to exclude from instructive headers or titles (e.g., "getting started", "installation", etc.) of readmes.		

**Guidelines:**
- Prioritize keywords that directly relate to the main term and its subtopics.
- Maintain a focus on terms that potential users or developers are likely to search for in the context of API development.
- Branded keywords should be included in the keywordsWithBrandNames and not in the keywords.
`;
    const promptTitles = `Below is a list of titles separated by semicolons (';') from the top organic search results currently ranking for the term '${
      searchQuery.inputTerm
    }'.
		Given that some pages might be SEO optimized, there's a chance that we can extract keywords from the page titles.
		Create a list of keywords that are directly related to the main term and its subtopics form the titles of the pages.
		
		Given that some title contain the brand of the website (e.g. github, youtube, etc.) OR the section of the website (e.g. blog, docs, etc.), ensure to not treat them as keywords.

		==========
        ${searchResponse.serperOrganicResults
          .map(
            (result) =>
              `The title for the sourceUrl "${result.link}" (reference this url as the sourceUrl for the keyword) is: "${result.title}"`,
          )
          .join(";")}
		==========
		`;
    // extract keywords from the title of the organic results
    const keywordsFromTitles = await generateObject({
      model: openai("gpt-4o-mini"),
      system: keywordResearchSystemPrompt,
      prompt: promptTitles,
      schema: z.object({
        keywords: z.array(z.object({ keyword: z.string(), sourceUrl: z.string().url() })),
        keywordsWithBrandNames: z.array(
          z.object({ keyword: z.string(), sourceUrl: z.string().url() }),
        ),
      }),
    });
    console.info(
      `3/5 - KEYWORDS FROM TITLES: ${keywordsFromTitles.object.keywordsWithBrandNames.length} keywords with brand names and ${keywordsFromTitles.object.keywords.length} keywords.`,
    );

    // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
    await db
      .insert(keywords)
      .values(
        keywordsFromTitles.object.keywords.map((keyword) => ({
          inputTerm: searchQuery.inputTerm,
          keyword: keyword.keyword.toLowerCase(),
          sourceUrl: keyword.sourceUrl,
          source: "titles",
        })),
      )
      .onDuplicateKeyUpdate({
        set: {
          updatedAt: sql`now()`,
        },
      });
    const insertedFromTitles = await db.query.keywords.findMany({
      where: and(
        eq(keywords.inputTerm, searchQuery.inputTerm),
        eq(keywords.source, "titles"),
        inArray(
          keywords.keyword,
          keywordsFromTitles.object.keywords.map((k) => k.keyword.toLowerCase()),
        ),
      ),
    });

    const context = validContent
      .map((content) => {
        if (!content?.markdown || !content?.sourceUrl) {
          return "";
        }
        return `
		==========
		The headers for the organic result "${content.sourceUrl}" (ensure you're referencing this url as the sourceUrl for the keyword) are: 
		${content.markdown?.match(/^##\s+(.*)$/gm)?.join("\n")}
		==========
		`;
      })
      .join(";");

    const promptHeaders = `Below is a list of h1 headers, separated by semicolons (';'), from the top organic search results currently ranking for the term '${searchQuery.inputTerm}'. Given that some pages might be SEO optimized, there's a chance that we can extract keywords from them.
		Create a list of keywords that are directly related to the main term and its subtopics form the h1 headers of the pages.

		==========
		${context}
		==========
		`;
    const keywordsFromHeaders = await generateObject({
      model: openai("gpt-4o-mini"),
      system: keywordResearchSystemPrompt,
      prompt: promptHeaders,
      schema: z.object({
        keywords: z.array(z.object({ keyword: z.string(), sourceUrl: z.string().url() })),
        keywordsWithBrandNames: z.array(
          z.object({ keyword: z.string(), sourceUrl: z.string().url() }),
        ),
      }),
    });
    console.info(
      `5/5 - KEYWORDS FROM HEADERS: ${keywordsFromHeaders.object.keywordsWithBrandNames.length} keywords with brand names and ${keywordsFromHeaders.object.keywords.length} keywords.`,
    );

    // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
    await db
      .insert(keywords)
      .values(
        keywordsFromHeaders.object.keywords.map((keyword) => ({
          inputTerm: searchQuery.inputTerm,
          keyword: keyword.keyword.toLowerCase(),
          sourceUrl: keyword.sourceUrl,
          source: "headers",
        })),
      )
      .onDuplicateKeyUpdate({
        set: {
          updatedAt: sql`now()`,
        },
      });
    const insertedFromHeaders = await db.query.keywords.findMany({
      where: and(
        eq(keywords.inputTerm, searchQuery.inputTerm),
        eq(keywords.source, "headers"),
        inArray(
          keywords.keyword,
          keywordsFromHeaders.object.keywords.map((k) => k.keyword.toLowerCase()),
        ),
      ),
    });

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
      `✅ Keyword Research for ${payload.term} completed. Total keywords: ${
        insertedFromTitles.length + insertedFromHeaders.length + insertedRelatedSearches.length
      }`,
    );

    return {
      message: `Keyword Research for ${payload.term} completed`,
      term: searchQuery.inputTerm,
      keywords: [...insertedFromTitles, ...insertedFromHeaders, ...insertedRelatedSearches],
    };
  },
});
