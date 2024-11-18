import { db } from "@/lib/db-marketing/client";
import { openai } from "@ai-sdk/openai";
import { task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { and, eq, or } from "drizzle-orm";
import { z } from "zod";
import { entries, firecrawlResponses, keywords } from "../../lib/db-marketing/schemas";
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

        TITLE EXPERT (aim for 45-50 chars, strict max 55)
        Primary goal: Create clear, informative titles that drive clicks
        Structure outline:
        1. Keyword placement (start) - max 25 chars
        2. Concise value proposition - max 20 chars
        3. "Guide" identifier - max 10 chars
        Best practices:
        - Target 45-50 characters total
        - Focus on clarity over attention-grabbing
        - Use simple punctuation (colon, dash)
        - Count characters before submitting
        Example: "JWT Auth: Complete Guide" (27 chars)

        DESCRIPTION EXPERT (aim for 140-145 chars, strict max 150)
        Primary goal: Convert visibility into clicks
        Structure outline:
        1. Hook with main benefit (20-25 chars)
        2. Core value props - pick TWO only:
           - "Learn essentials"
           - "Key takeaways" 
           - "Expert examples"
           - "Info & best practices"
        3. ONE secondary keyword (15-20 chars)
        4. Brief call-to-action (10 chars)
          - omit the punctuatoin at the end to save a character
        Best practices:
        - Front-load main benefit
        - Use short power words (learn, master)
        - Count characters before submitting
        - Leave roughly 10 char buffer
        
        Example: "Master JWT Auth essentials with expert guidance. Learn core concepts and implementation best practices for secure API authentication. Start now." (134 chars)

        H1 EXPERT (aim for 45-50 chars, strict max 60)
        Primary goal: Validate click & preview content value
        Structure outline:
        1. Main concept (20-25 chars)
        2. Value bridge (20-25 chars)
        Best practices:
        - Keep it shorter than title
        - Use fewer modifiers
        - Count characters before submitting
        - Leave 10 char buffer
        Example: "JWT Authentication: Core Concepts & Implementation" (49 chars)

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
        description: z.string().max(190),
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

    // Step 4: Validate and optimize lengths
    const validatedMetaTags = await generateObject({
      model: openai("gpt-4"),
      system: `
        You are an expert SEO consultant with 10 years of experience optimizing content for search engines.
        Your task is to validate and optimize meta tags to ensure they meet strict character limits while
        maintaining their SEO value and readability.

        Key requirements:
        1. Title: Max 60 chars (aim for 50-55)
        2. Description: Max 160 chars (aim for 145-155)
        3. H1: Max 80 chars (aim for 45-50)

        Best practices:   
        - Front-load important keywords
        - Maintain readability and natural language
        - Preserve core message and intent
        - Keep primary keyword visible in truncated versions
        - Use punctuation strategically to create natural breaks

        If tags exceed limits:
        1. Remove unnecessary words while preserving meaning
        2. Replace longer words with shorter synonyms
        3. Restructure for conciseness
        4. Ensure truncation occurs at natural breaks
      `,
      prompt: `
        Original tags:
        Title: ${craftedMetaTags.object.title}
        Description: ${craftedMetaTags.object.description}
        H1: ${craftedMetaTags.object.h1}

        Optimize these tags to meet character limits while maintaining SEO value.
        If they already meet the limits, return them unchanged.
      `,
      schema: z.object({
        title: z.string().max(60),
        description: z.string().max(160),
        h1: z.string().max(80),
        reasoning: z.object({
          titleChanges: z.string(),
          descriptionChanges: z.string(),
          h1Changes: z.string(),
        }),
      }),
      temperature: 0.1, // Low temperature for consistent, focused outputs
    });

    // Update database with validated meta tags
    await db
      .update(entries)
      .set({
        metaTitle: validatedMetaTags.object.title,
        metaDescription: validatedMetaTags.object.description,
        metaH1: validatedMetaTags.object.h1,
      })
      .where(eq(entries.inputTerm, term));

    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
  },
});
