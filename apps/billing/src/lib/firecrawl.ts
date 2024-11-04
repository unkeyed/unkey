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
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import { z } from "zod";
import { keywordResearchSystemPrompt } from "./keywords";

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
