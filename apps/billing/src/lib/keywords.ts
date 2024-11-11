import { sql } from "drizzle-orm";
import { and } from "drizzle-orm";
import { inArray } from "drizzle-orm";
import { eq } from "drizzle-orm";
import { db } from "./db-marketing/client";
import { firecrawlResponses, keywords, serperSearchResponses } from "./db-marketing/schemas";
import { z } from "zod";
import { openai } from "@ai-sdk/openai";
import { generateObject } from "ai";

export const keywordResearchSystemPrompt = `
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

export async function getOrCreateKeywordsFromTitles(args: { term: string }) {
  const { term } = args;
  const existing = await db.query.keywords.findMany({
    where: and(eq(keywords.inputTerm, term)),
  });
  if (existing.length > 0) {
    return existing;
  }

  const searchResponse = await db.query.serperSearchResponses.findFirst({
    where: eq(serperSearchResponses.inputTerm, term),
    with: {
      serperOrganicResults: true,
    },
  });
  if (!searchResponse) {
    throw new Error(
      `Error attempting to get keywords from firecrawl results: No search response found for term ${term}`,
    );
  }

  const promptTitles = `Below is a list of titles separated by semicolons (';') from the top organic search results currently ranking for the term '${term}'.
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

  // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
  await db
    .insert(keywords)
    .values(
      keywordsFromTitles.object.keywords.map((keyword) => ({
        inputTerm: term,
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

  return db.query.keywords.findMany({
    where: and(
      eq(keywords.inputTerm, term),
      eq(keywords.source, "titles"),
      inArray(
        keywords.keyword,
        keywordsFromTitles.object.keywords.map((k) => k.keyword.toLowerCase()),
      ),
    ),
  });
}

export async function getOrCreateKeywordsFromHeaders(args: { term: string }) {
  const { term } = args;
  const existing = await db.query.keywords.findMany({
    where: and(eq(keywords.inputTerm, term), eq(keywords.source, "headers")),
  });
  if (existing.length > 0) {
    return existing;
  }

  const firecrawlResults = await db.query.firecrawlResponses.findMany({
    where: eq(firecrawlResponses.inputTerm, term),
  });

  const context = firecrawlResults
    .map((firecrawlResponse) => {
      if (!firecrawlResponse.markdown || !firecrawlResponse.sourceUrl) {
        return "";
      }
      return `
          ==========
          The headers for the organic result "${firecrawlResponse.sourceUrl}" (ensure you're referencing this url as the sourceUrl for the keyword) are: 
          ${firecrawlResponse.markdown?.match(/^##\s+(.*)$/gm)?.join("\n")}
          ==========
          `;
    })
    .join(";");

  const promptHeaders = `Below is a list of h1 headers, separated by semicolons (';'), from the top organic search results currently ranking for the term '${term}'. Given that some pages might be SEO optimized, there's a chance that we can extract keywords from them.
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

  // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
  await db
    .insert(keywords)
    .values(
      keywordsFromHeaders.object.keywords.map((keyword) => ({
        inputTerm: term,
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
  return db.query.keywords.findMany({
    where: and(
      eq(keywords.inputTerm, term),
      eq(keywords.source, "headers"),
      inArray(
        keywords.keyword,
        keywordsFromHeaders.object.keywords.map((k) => k.keyword.toLowerCase()),
      ),
    ),
  });
}
