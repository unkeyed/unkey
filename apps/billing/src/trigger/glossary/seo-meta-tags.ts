import { z } from "zod";
import { eq, and, or } from "drizzle-orm";
import { keywords, firecrawlResponses, entries } from "../../lib/db-marketing/schemas";
import { task } from "@trigger.dev/sdk/v3";
import { db } from "@/lib/db-marketing/client";
import { generateObject } from "ai";
import { openai } from "@ai-sdk/openai";
import type { CacheStrategy } from "./_generate-glossary-entry";

// Define the job
export const seoMetaTagsTask = task({
  id: "seo_meta_tags",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    term,
    onCacheHit = "stale" as CacheStrategy,
  }: { term: string; onCacheHit?: CacheStrategy }) => {
    // Add check for existing meta tags
    const existing = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      columns: {
        id: true,
        inputTerm: true,
        metaTitle: true,
        metaDescription: true,
        metaH1: true,
      },
      with: {
        dynamicSections: true,
      },
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (
      existing?.metaTitle &&
      existing?.metaDescription &&
      existing?.metaH1 &&
      onCacheHit === "stale"
    ) {
      return existing;
    }

    // Step 1: Fetch keywords associated with the inputTerm
    const relatedKeywords = await db.query.keywords.findMany({
      where: and(
        eq(keywords.inputTerm, term),
        or(eq(keywords.source, "related_searches"), eq(keywords.source, "auto_suggest")),
      ),
    });

    // Step 2: Fetch top 10 ranking pages' data
    const topRankingPages = await db.query.firecrawlResponses.findMany({
      where: eq(firecrawlResponses.inputTerm, term),
      with: {
        serperOrganicResult: true,
      },
    });

    // Step 3: Craft SEO-optimized title and description
    const craftedMetaTags = await generateObject({
      model: openai("gpt-4"),
      system: `
        You are three specialized experts collaborating on creating meta tags for an API documentation glossary:

        TITLE EXPERT (50-60 chars)
        Primary goal: Maximize click-through rate from search results
        Structure outline:
        1. Keyword placement (start)
        2. Current year (2024)
        3. Attention-grabbing character (→, |, ())
        4. Value proposition
        5. "API Glossary" identifier
        Best practices:
        - Use parentheses for year or context
        - Include numbers when relevant (e.g., "5 Best Practices")
        - Stand out among 10 competing results
        Example: "JWT Authentication (2024) → Complete API Glossary Guide"

        DESCRIPTION EXPERT (150-160 chars)
        Primary goal: Convert visibility into clicks
        Structure outline:
        1. Hook with main benefit
        2. Expand on unique value props:
           - "Learn at a glance"
           - "Key takeaways"
           - "Expert examples"
        3. Include secondary keywords
        4. Call-to-action element
        Best practices:
        - Front-load main benefit
        - Use power words (master, discover, unlock)
        - Create urgency or curiosity
        Example: "Master JWT Authentication in minutes with our expert-curated key takeaways. From basic concepts to best practices, get everything you need to implement secure API authentication."

        H1 EXPERT (60-80 chars)
        Primary goal: Validate click & preview content value
        Structure outline:
        1. Main concept introduction
        2. Value proposition bridge
        3. Content scope indicator
        Best practices:
        - Flow naturally from title/description
        - Preview the learning journey
        - Create excitement about content depth
        - Avoid mentioning takeaways (they're prominent anyway)
        Example: "Understanding JWT Authentication: From Theory to Implementation"

        COLLABORATION RULES:
        1. Each element builds upon the previous
        2. Maintain keyword presence across all elements
        3. Create a narrative arc: Promise → Value → Delivery
        4. Technical accuracy is non-negotiable
        5. Consider search intent progression
      `,
      prompt: `
        Term: ${term}
        Content outline:
        ${existing?.dynamicSections.map((section) => `- ${section.heading}`).join("\n")}

        Related keywords: 
        ${relatedKeywords.map((keyword) => keyword.keyword).join("\n")}

        Top ranking pages:
        ${topRankingPages
          .map(
            (page) =>
              `- [${page.serperOrganicResult?.position}] ${page.title}\n  ${page.description}`,
          )
          .join("\n")}

        Create two meta tags and an H1 that form a compelling journey from search result to page content.
        Focus on standing out in search results while maintaining accuracy and user value.
      `,
      schema: z.object({
        title: z.string().max(60),
        description: z.string().max(160),
        h1: z.string().max(80),
        reasoning: z.object({
          titleStrategy: z.string(),
          descriptionStrategy: z.string(),
          h1Strategy: z.string(),
          cohesion: z.string(),
        }),
      }),
      temperature: 0.3,
    });

    // Update database with all three meta tags
    await db
      .update(entries)
      .set({
        metaTitle: craftedMetaTags.object.title,
        metaDescription: craftedMetaTags.object.description,
        metaH1: craftedMetaTags.object.h1,
      })
      .where(eq(entries.inputTerm, term));

    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
  },
});
