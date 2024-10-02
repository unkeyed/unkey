import { db } from "@/lib/db-marketing/client";
import { openai } from "@ai-sdk/openai";
import { generateObject } from "ai";
import { eq, sql } from "drizzle-orm";

import { insertSearchQuerySchema, searchQueries } from "@/lib/db-marketing/schemas";

export async function getOrCreateSearchQuery(args: { term: string }) {
  const { term } = args;
  // Try to find existing search query
  const existingQuery = await db.query.searchQueries.findFirst({
    where: eq(searchQueries.inputTerm, term),
  });

  if (existingQuery) {
    return existingQuery;
  }

  // Generate new search query
  const generatedQuery = await generateObject({
    model: openai("gpt-4o"),
    system: `You are a Senior Content Writer who specialises in writing technical content for Developer Tools that are SEO optimized.
For every term, you conduct a search on Google to gather the data you need.
You're goal is to create a search query that will return a SERP with the most relevant information for the term.

Make sure to always include the exact term in the search query at the beginning of the query.
If the term is clearly associated to API development, use the term as-is for query.

If the term is ambiguous with non-API development related fields and could result in unrelated results, add context to the search query to clarify the search & return the reason for the ambiguity.

Keep the search query as short and as simple as possible, don't use quotes around the search query.

`,
    prompt: `Create the search query for the term "${term}."`,
    schema: insertSearchQuerySchema,
  });

  // NB: drizzle doesn't support returning ids in conjunction with handling duplicates, so we get them afterwards
  await db
    .insert(searchQueries)
    .values({
      ...generatedQuery.object,
    })
    .onDuplicateKeyUpdate({
      set: {
        updatedAt: sql`now()`,
      },
    });
  const insertedQuery = await db.query.searchQueries.findFirst({
    where: eq(searchQueries.inputTerm, generatedQuery.object.inputTerm),
  });

  if (!insertedQuery) {
    throw new Error("Failed to insert or update search query");
  }

  return insertedQuery;
}
