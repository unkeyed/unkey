import { db } from "@/lib/db-marketing/client";
import {
  firecrawlResponses,
  firecrawlResponses,
  keywords,
  serperOrganicResults,
  serperSearchResponses,
} from "@/lib/db-marketing/schemas";
import { THREE } from "@/trigger/glossary/keyword-research";
import FirecrawlApp from "@mendable/firecrawl-js";
import { and, desc, eq, inArray, isNotNull, lte, sql } from "drizzle-orm";
import { getOrCreateSearchResponse } from "./serper";
import { generateObject, generateText } from "ai";
import { openai } from "@ai-sdk/openai";
import { z } from "zod";
import { keywordResearchSystemPrompt } from "./keywords";
import type { CacheStrategy } from "@/trigger/glossary/_generate-glossary-entry";

const firecrawl = new FirecrawlApp({
  apiKey: process.env.FIRECRAWL_API_KEY!,
});

/**
 * Gets or creates a firecrawl response for a given URL.
 * First checks if we already have the content, if not scrapes it.
 */
export async function getOrCreateFirecrawlResponse(args: {
  url: string;
  connectTo: { term: string };
}) {
  // 1. Check if we already have this URL
  const existing = await db.query.firecrawlResponses.findFirst({
    where: eq(firecrawlResponses.sourceUrl, args.url),
  });
  if (existing?.markdown) {
    return existing;
  }

  // 2. If not, scrape the URL
  try {
    const firecrawlResult = await firecrawl.scrapeUrl(args.url, { formats: ["markdown"] });

    // 3. Handle scraping failure
    if (!firecrawlResult.success) {
      const [response] = await db
        .insert(firecrawlResponses)
        .values({
          sourceUrl: args.url,
          error: firecrawlResult.error || "Unknown error occurred",
          success: false,
        })
        .onDuplicateKeyUpdate({
          set: {
            error: firecrawlResult.error || "Unknown error occurred",
            success: false,
            updatedAt: new Date(),
          },
        })
        .$returningId();

      console.warn(
        `Gracefully continuing after: ⚠️ Failed to scrape URL ${args.url}: ${firecrawlResult.error}. Stored run in DB with id '${response.id}'`,
      );
      return await db.query.firecrawlResponses.findFirst({
        where: eq(firecrawlResponses.sourceUrl, args.url),
      });
    }

    // 4. Store successful result
    await db
      .insert(firecrawlResponses)
      .values({
        success: firecrawlResult.success,
        markdown: firecrawlResult.markdown ?? null,
        sourceUrl: firecrawlResult.metadata?.sourceURL || args.url,
        scrapeId: firecrawlResult.metadata?.scrapeId || "",
        title: firecrawlResult.metadata?.title || "",
        description: firecrawlResult.metadata?.description || "",
        language: firecrawlResult.metadata?.language || "",
        ogTitle: firecrawlResult.metadata?.ogTitle || "",
        ogDescription: firecrawlResult.metadata?.ogDescription || "",
        ogUrl: firecrawlResult.metadata?.ogUrl || "",
        ogImage: firecrawlResult.metadata?.ogImage || "",
        ogSiteName: firecrawlResult.metadata?.ogSiteName || "",
        error: null,
        inputTerm: args.connectTo.term || "",
      })
      .onDuplicateKeyUpdate({
        set: {
          markdown: firecrawlResult.markdown ?? null,
          updatedAt: new Date(),
        },
      });

    return await db.query.firecrawlResponses.findFirst({
      where: eq(firecrawlResponses.sourceUrl, args.url),
    });
  } catch (error) {
    // 5. Handle unexpected errors
    console.error(`Error processing URL ${args.url}:`, error);

    // Store the error and return the response
    await db
      .insert(firecrawlResponses)
      .values({
        sourceUrl: args.url,
        error: error instanceof Error ? error.message : String(error),
        success: false,
      })
      .onDuplicateKeyUpdate({
        set: {
          error: error instanceof Error ? error.message : String(error),
          success: false,
          updatedAt: new Date(),
        },
      });

    return await db.query.firecrawlResponses.findFirst({
      where: eq(firecrawlResponses.sourceUrl, args.url),
    });
  }
}

export async function getOrCreateSummary({
  url,
  connectTo,
  onCacheHit = "stale" as CacheStrategy,
}: {
  url: string;
  connectTo: { term: string };
  onCacheHit?: CacheStrategy;
}) {
  // 1. Check if we already have a summary for this URL
  const existing = await db.query.firecrawlResponses.findFirst({
    where: eq(firecrawlResponses.sourceUrl, url),
  });

  if (existing?.summary && onCacheHit === "stale") {
    return existing;
  }

  // 2. Get the firecrawl response (which includes the markdown)
  const firecrawlResponse = await getOrCreateFirecrawlResponse({
    url,
    connectTo,
  });

  if (!firecrawlResponse?.markdown) {
    console.warn(`No markdown content found for URL ${url}`);
    return firecrawlResponse;
  }

  // 3. Get the position from serper results
  const serperResult = await db.query.serperOrganicResults.findFirst({
    where: eq(serperOrganicResults.link, url),
    columns: { position: true },
  });

  // 4. Generate the summary
  const system = `You are the **Chief Technology Officer (CTO)** of a leading API Development Tools Company with extensive experience in API development using programming languages such as Go, TypeScript, and Elixir and other backend languages. You have a PhD in computer science from MIT. Your expertise ensures that the content you summarize is technically accurate, relevant, and aligned with best practices in API development and computer science.

**Your Task:**
Accurately and concisely summarize the content from the page that ranks #${
    serperResult?.position ?? "unknown"
  } for the term "${connectTo.term}". Focus on technical details, including how the content is presented (e.g., text, images, tables). Ensure factual correctness and relevance to API development.

**Instructions:**
- Provide a clear and concise summary of the content.
- Highlight key technical aspects and insights related to API development.
- Mention the types of content included, such as images, tables, code snippets, etc.
- Cite the term the content is ranking for and its position in the SERP.`;

  const prompt = `Summarize the following content for the term "${connectTo.term}" that's ranking #${serperResult?.position ?? "unknown"}:
=======
${firecrawlResponse.markdown}
=======`;

  const summaryCompletion = await generateText({
    model: openai("gpt-4o-mini"),
    system,
    prompt,
    maxTokens: 500,
  });

  // 5. Store the summary in the database
  await db
    .update(firecrawlResponses)
    .set({ summary: summaryCompletion.text })
    .where(eq(firecrawlResponses.id, firecrawlResponse.id));

  // 6. Return the updated response
  return await db.query.firecrawlResponses.findFirst({
    where: eq(firecrawlResponses.id, firecrawlResponse.id),
  });
}
