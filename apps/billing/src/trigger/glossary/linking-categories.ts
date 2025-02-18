import { db } from "@/lib/db-marketing/client";
import { openai } from "@ai-sdk/openai";
import { task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { eq } from "drizzle-orm";
import { z } from "zod";
import { entries } from "../../lib/db-marketing/schemas";
import type { CacheStrategy } from "./_generate-glossary-entry";

// NB: These are hardcoded for now, but we should have them be dicatated by search volume as it allows for clusterization
const API_CATEGORIES = [
  "Authentication & Authorization",
  "API Design",
  "Security",
  "Performance",
  "Protocols",
  "Data Formats",
  "Error Handling",
  "Versioning",
  "Testing",
  "Documentation",
  "Monitoring",
  "Integration",
] as const;

export const linkingCategoriesTask = task({
  id: "linking_categories",
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
        categories: true,
      },
    });

    if (existing?.categories && existing.categories.length > 0 && onCacheHit === "stale") {
      return existing;
    }

    const categories = await generateObject({
      model: openai("gpt-4o-mini"),
      system: `
        You are an API documentation expert. Categorize API-related terms into relevant categories.
        Only use categories from the predefined list. A term can belong to multiple categories if relevant.
        Consider the term's primary purpose, related concepts, and practical applications.
      `,
      prompt: `
        Term: "${term}"
        Available categories: ${API_CATEGORIES.join(", ")}
        
        Analyze the term and assign it to the most relevant categories. Consider:
        1. Primary function and purpose
        2. Related API concepts
        3. Implementation context
        4. Technical domain
      `,
      schema: z.object({
        categories: z.array(z.enum(API_CATEGORIES)),
        reasoning: z.string(),
      }),
      temperature: 0.3,
    });

    await db
      .update(entries)
      .set({
        categories: categories.object.categories,
      })
      .where(eq(entries.inputTerm, term));

    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
    });
  },
});
