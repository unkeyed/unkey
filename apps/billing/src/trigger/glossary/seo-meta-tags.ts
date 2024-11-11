import { Trigger } from "@trigger.dev/sdk";
import { z } from "zod";
import { eq, and, or } from "drizzle-orm";
import { keywords, firecrawlResponses, entries } from "../../lib/db-marketing/schemas";
import { task } from "@trigger.dev/sdk/v3";
import { db } from "@/lib/db-marketing/client";
import { generateObject, generateText } from "ai";
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
      },
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (existing?.metaTitle && existing?.metaDescription && onCacheHit === "stale") {
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
      model: openai("gpt-4o"),
      system: `
        You are an SEO expert specializing in technical content, particularly API development. You are given an API-related term and need to craft an SEO-optimized title and description for a glossary entry about this term.

        Follow these best practices when crafting the title and description:

        For the title:
        - Keep it concise, ideally between 50-60 characters.
        - Include the API term at the beginning of the title.
        - If the term is an acronym, consider including both the acronym and its full form.
        - Make it informative and clear, indicating that this is a definition or explanation.
        - Include "API Glossary" or your brand name at the end if space allows.
        - Use a pipe (|) or dash (-) to separate title elements if needed.

        For the description:
        - Aim for 150-160 characters for optimal display in search results.
        - Start with a clear, concise definition of the API term.
        - Include the phrase "Learn about [term]" or similar to indicate the educational nature.
        - Mention that this is part of an API development glossary.
        - If relevant, briefly mention a key benefit or use case of the term.
        - Use technical language appropriately, but ensure it's understandable for developers.
        - Include a call-to-action like "Explore our API glossary for more terms."

        Additional guidelines:
        - Ensure accuracy in technical terms and concepts.
        - Balance SEO optimization with educational value.
        - Consider the context of API development when explaining terms.
        - For complex terms, focus on clarity over comprehensiveness in the meta description.
        - If the term is commonly confused with another, briefly differentiate it.

        Example format:
        Title: "HATEOAS in REST APIs | Unkey API Glossary"
        Description: "What is HATEOAS in REST APIs? Learn about HATEOAS in REST APIs. Discover how it enhances API navigation and discoverability. Explore our API development glossary for more terms."

        Remember, the goal is to create meta tags that are both SEO-friendly and valuable to developers seeking to understand API terminology.
        `,
      prompt: `
        Term: ${term}
        List of related keywords: 
        - ${relatedKeywords.map((keyword) => keyword.keyword).join("\n- ")}

        A markdown table of the title & description of the top 10 ranking pages along with their position:
        \`\`\`
        | Position | Title | Description |
        | -------- | ----- | ----------- |
        ${topRankingPages
          .map(
            (page) => `${page.serperOrganicResult?.position} | ${page.title} | ${page.description}`,
          )
          .join("\n")}
        \`\`\`

        The title and description should be SEO-optimized for the keywords provided.
      `,
      schema: z.object({
        title: z.string(),
        description: z.string(),
      }),
      temperature: 0.5,
    });

    await db
      .update(entries)
      .set({
        metaTitle: craftedMetaTags.object.title,
        metaDescription: craftedMetaTags.object.description,
      })
      .where(eq(entries.inputTerm, term));
    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
  },
});
