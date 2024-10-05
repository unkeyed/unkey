import { db } from "@/lib/db-marketing/client";
import { firecrawlResponses } from "@/lib/db-marketing/schemas";
import FirecrawlApp from "@mendable/firecrawl-js";
import { and, eq, inArray, isNotNull } from "drizzle-orm";

const firecrawl = new FirecrawlApp({
  apiKey: process.env.FIRECRAWL_API_KEY!,
});

export async function getTopResultsContent(args: { urls: string[] }) {
  // Check for existing responses with markdown content
  const existingResponses = await db.query.firecrawlResponses.findMany({
    where: and(
      inArray(firecrawlResponses.sourceUrl, args.urls),
      isNotNull(firecrawlResponses.markdown),
    ),
  });

  const existingUrlSet = new Set(existingResponses.map((r) => r.sourceUrl));
  const urlsToScrape = args.urls.filter((url) => !existingUrlSet.has(url));

  // Scrape new URLs
  const newResponses = await Promise.all(urlsToScrape.map(scrapeAndStoreUrl)); // Combine existing and new responses
  return [...existingResponses, ...newResponses];
}
async function scrapeAndStoreUrl(url: string) {
  try {
    const firecrawlResult = await firecrawl.scrapeUrl(url, { formats: ["markdown"] });

    if (!firecrawlResult.success) {
      console.error(`Firecrawl error for URL ${url}:`, firecrawlResult.error);
      // store the error in the database
      const [errorResponse] = await db
        .insert(firecrawlResponses)
        .values({
          sourceUrl: url,
          error: firecrawlResult.error || "Unknown error occurred",
          success: false,
        })
        .$returningId();
      return await db.query.firecrawlResponses.findFirst({
        where: eq(firecrawlResponses.id, errorResponse.id),
      });
    }

    const [insertedResponse] = await db
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
      .$returningId();

    return await db.query.firecrawlResponses.findFirst({
      where: eq(firecrawlResponses.id, insertedResponse.id),
    });
  } catch (error) {
    console.error(`Error scraping URL ${url}:`, error);
    const [errorResponse] = await db
      .insert(firecrawlResponses)
      .values({
        sourceUrl: url,
        error: error instanceof Error ? error.message : String(error),
        success: false,
      })
      .$returningId();
    return await db.query.firecrawlResponses.findFirst({
      where: eq(firecrawlResponses.id, errorResponse.id),
    });
  }
}

export async function getScrapedContentMany(urls: string[]) {
  // Check which URLs we already have responses for
  const existingResponses = await db.query.firecrawlResponses.findMany({
    where: inArray(firecrawlResponses.sourceUrl, urls),
  });

  const existingUrls = new Set(existingResponses.map((r) => r.sourceUrl));
  const urlsToScrape = urls.filter((url) => !existingUrls.has(url));

  // Scrape the URLs that we don't have responses for
  const newResults = await Promise.all(urlsToScrape.map(scrapeAndStoreUrl));

  // Combine existing and new results
  return [...existingResponses, ...newResults];
}
