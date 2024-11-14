import { task } from "@trigger.dev/sdk/v3";
import { db } from "@/lib/db-marketing/client";
import { entries, firecrawlResponses, insertEntrySchema } from "../../lib/db-marketing/schemas";
import { eq } from "drizzle-orm";
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import type { CacheStrategy } from "./_generate-glossary-entry";

export const contentTakeawaysTask = task({
  id: "content_takeaways",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    term,
    onCacheHit = "stale" as CacheStrategy,
  }: {
    term: string;
    onCacheHit?: CacheStrategy;
  }) => {
    const existing = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      columns: {
        id: true,
        inputTerm: true,
        takeaways: true,
      },
    });

    if (existing?.takeaways && onCacheHit === "stale") {
      return existing;
    }

    // Get scraped content for context
    const scrapedContent = await db.query.firecrawlResponses.findMany({
      where: eq(firecrawlResponses.inputTerm, term),
      columns: {
        markdown: true,
        summary: true,
      },
    });

    const takeaways = await generateObject({
      model: openai("gpt-4"),
      system: `
        You are an API documentation expert. Create comprehensive takeaways for API-related terms.
        Focus on practical, accurate, and developer-friendly content.
        Each section should be concise but informative.
      `,
      prompt: `
        Term: "${term}"
        
        Scraped content summaries:
        ${scrapedContent.map((content) => content.summary).join("\n\n")}
        
        Create structured takeaways covering:
        1. TLDR (brief, clear definition)
        2. Definition and structure (key components)
        3. Historical context (evolution and significance)
        4. Usage in APIs (practical implementation)
        5. Best practices (implementation guidelines)
        6. Recommended reading (key resources)
        7. Interesting fact (did you know)
      `,
      // define the zod schema for the takeaways separately, use it in content-collections & then infer the type from it for drizzle.
      schema: insertEntrySchema.pick({
        takeaways: true,
      }),
      temperature: 0.3,
    });

    await db
      .update(entries)
      .set({
        takeaways: takeaways.object,
      })
      .where(eq(entries.inputTerm, term));

    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
    });
  },
});
