import { db } from "@/lib/db-marketing/client";
import { firecrawlResponses } from "@/lib/db-marketing/schemas";
import FirecrawlApp from "@mendable/firecrawl-js";
import { eq } from "drizzle-orm";

const firecrawl = new FirecrawlApp({
  apiKey: process.env.FIRECRAWL_API_KEY!,
});

/**
 * Gets or creates a firecrawl response for a given URL.
 * First checks if we already have the content, if not scrapes it.
 */
export async function getOrCreateFirecrawlResponse(url: string) {
  // 1. Check if we already have this URL
  const existing = await db.query.firecrawlResponses.findFirst({
    where: eq(firecrawlResponses.sourceUrl, url),
  });
  
  if (existing) {
    return existing;
  }

  // 2. If not, scrape the URL
  try {
    const firecrawlResult = await firecrawl.scrapeUrl(url, { formats: ["markdown"] });

    // 3. Handle scraping failure
    if (!firecrawlResult.success) {
      const [response] = await db
        .insert(firecrawlResponses)
        .values({
          sourceUrl: url,
          error: firecrawlResult.error || "Unknown error occurred",
          success: false,
        })
        .onDuplicateKeyUpdate({
          set: {
            error: firecrawlResult.error || "Unknown error occurred",
            success: false,
            updatedAt: new Date(),
          },
        }).$returningId();

      console.warn(`Gracefully continuing after: ⚠️ Failed to scrape URL ${url}: ${firecrawlResult.error}. Stored run in DB with id '${response.id}'`);
      return await db.query.firecrawlResponses.findFirst({
        where: eq(firecrawlResponses.sourceUrl, url),
      });
    }

    // 4. Store successful result
    await db
      .insert(firecrawlResponses)
      .values({
        success: firecrawlResult.success,
        markdown: firecrawlResult.markdown ?? null,
        sourceUrl: firecrawlResult.metadata?.sourceURL || url,
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
      })
      .onDuplicateKeyUpdate({
        set: {
          markdown: firecrawlResult.markdown ?? null,
          updatedAt: new Date(),
        },
      });

    return await db.query.firecrawlResponses.findFirst({
      where: eq(firecrawlResponses.sourceUrl, url),
    });

  } catch (error) {
    // 5. Handle unexpected errors
    console.error(`Error processing URL ${url}:`, error);
    
    // Store the error and return the response
    await db
      .insert(firecrawlResponses)
      .values({
        sourceUrl: url,
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
      where: eq(firecrawlResponses.sourceUrl, url),
    });
  }
}
