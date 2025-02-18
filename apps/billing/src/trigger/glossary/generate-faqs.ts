import { db } from "@/lib/db-marketing/client";
import { entries } from "@/lib/db-marketing/schemas";
import { faqSchema } from "@/lib/db-marketing/schemas/entries";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { eq } from "drizzle-orm";
import { z } from "zod";
import type { CacheStrategy } from "./_generate-glossary-entry";

export const generateFaqsTask = task({
  id: "generate_faqs",
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
      with: {
        searchQuery: {
          with: {
            searchResponse: {
              with: {
                serperPeopleAlsoAsk: true,
              },
            },
          },
        },
      },
      orderBy: (entries, { asc }) => [asc(entries.createdAt)],
    });

    if (existing?.faq && existing.faq.length > 0 && onCacheHit === "stale") {
      return existing;
    }

    if (!existing?.searchQuery?.searchResponse?.serperPeopleAlsoAsk) {
      throw new AbortTaskRunError(`No 'People Also Ask' data found for term: ${term}`);
    }

    const peopleAlsoAsk = existing.searchQuery.searchResponse.serperPeopleAlsoAsk;

    // Generate comprehensive answers for each question
    const faqs = await generateObject({
      model: openai("gpt-4"),
      system: `You are an API documentation expert. Your task is to provide clear, accurate, and comprehensive answers to frequently asked questions about API-related concepts.

      Guidelines for answers:
      1. Be technically accurate and precise
      2. Use clear, concise language
      3. Include relevant examples where appropriate
      4. Focus on practical implementation details
      5. Keep answers focused and relevant to API development
      6. Maintain a professional, technical tone
      7. Ensure answers are complete but not overly verbose`,
      prompt: `
        Term: "${term}"
        
        Generate comprehensive answers for these questions from "People Also Ask":
        ${peopleAlsoAsk
          .map(
            (q) => `
        Question: ${q.question}
        Current snippet: ${q.snippet}
        Source: ${q.link}
        `,
          )
          .join("\n\n")}

        Provide clear, accurate answers that improve upon the existing snippets while maintaining technical accuracy.
      `,
      schema: z.object({ faq: faqSchema }),
      temperature: 0.2,
    });

    // Update the database with the generated FAQs
    await db
      .update(entries)
      .set({
        faq: faqs.object.faq,
      })
      .where(eq(entries.inputTerm, term));

    return db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { asc }) => [asc(entries.createdAt)],
    });
  },
});
