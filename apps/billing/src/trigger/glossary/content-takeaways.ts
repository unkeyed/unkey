import { db } from "@/lib/db-marketing/client";
import { takeawaysSchema } from "@/lib/db-marketing/schemas/takeaways-schema";
import { openai } from "@ai-sdk/openai";
import { task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { eq } from "drizzle-orm";
import { entries, firecrawlResponses } from "../../lib/db-marketing/schemas";
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
        For best practices, include only the 3 most critical and widely-adopted practices.
        For usage in APIs, provide a maximum of 3 short, focused sentences highlighting key terms and primary use cases.
      `,
      prompt: `
        Term: "${term}"
        
        Scraped content summaries:
        ${scrapedContent.map((content) => content.summary).join("\n\n")}
        
        Create structured takeaways covering:
        1. TLDR (brief, clear definition)
        2. Definition and structure - follow these rules:
           - Each value must be instantly recognizable (think of it as a memory aid) in one word or short expression
           - Maximum 1-3 words per value
           - Must be instantly understandable without explanation
           - Examples:
             Bad: "An API gateway is a server that acts as an intermediary..."
             Good: "Client-Server Bridge" or "API Request-Response Routing"
           - Focus on core concepts that can be expressed in minimal words
           
        3. Historical context - follow these exact formats:
           Introduced:
           - Use exact year if known: "1995"
           - Use decade if exact year unknown: "Early 1990s"
           - If truly uncertain: "Est. ~YYYY" with best estimate
           - Never use explanatory sentences
           
           Origin:
           - Format must be: "[Original Context] (${term})"
           - Example: "Web Services (${term})" or "Cloud Computing (${term})"
           - Keep [Original Context] to 1-2 words maximum
           - Never include explanations or evolution
           
           Evolution:
           - Format must be: "[Current State] ${term}"
           - Example: "Standardized ${term}" or "Enterprise ${term}"
           - Maximum 2-3 words
           - Focus on current classification/status only
           
        4. Usage in APIs (max 3 concise sentences covering essential terms and main use cases)
        5. Best practices (only the 3 most important and widely-used practices)
        6. Recommended reading (key resources)
        7. Interesting fact (did you know)
      `,
      schema: takeawaysSchema,
      temperature: 0.2,
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
