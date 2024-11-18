import { db } from "@/lib/db-marketing/client";
import { openai } from "@ai-sdk/openai";
import { generateObject } from "ai";
import { eq } from "drizzle-orm";

import { entries, insertSearchQuerySchema, searchQueries } from "@/lib/db-marketing/schemas";
import type { CacheStrategy } from "@/trigger/glossary/_generate-glossary-entry";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";

export async function getOrCreateSearchQuery({
  term,
  onCacheHit = "stale",
}: { term: string; onCacheHit: CacheStrategy }) {
  // Try to find existing search query
  const existing = await db.query.entries.findFirst({
    where: eq(entries.inputTerm, term),
    with: {
      searchQuery: true,
    },
    orderBy: (searchQueries, { asc }) => [asc(searchQueries.createdAt)],
  });

  if (existing?.searchQuery && onCacheHit === "revalidate") {
    return existing;
  }

  if (!existing) {
    throw new AbortTaskRunError(
      `Entry not found for term: ${term}. It's likely that the keyword-research task failed.`,
    );
  }

  // Generate new search query
  // NOTE: THE PROMPTING HERE REQUIRES SOME IMPROVEMENTS (ADD EVALS) -- FOR API RATE LIMITING IT GENERAATED:
  // "API Rate Limiting best practices and implementation", which is not the best keyword to search for.
  const generatedQuery = await generateObject({
    model: openai("gpt-4o-mini"),
    system: `You are a Senior Content Writer who specialises in writing technical content for Developer Tools that are SEO optimized.
For every term, you conduct a search on Google to gather the data you need.
You're goal is to create a search query that will return a SERP with the most relevant information for the term.

Make sure to always include the exact term in the search query at the beginning of the query.
If the term is clearly associated to API development, use the term as-is for query.

If the term is ambiguous with non-API development related fields and could result in unrelated results, add context to the search query to clarify the search & return the reason for the ambiguity.

Keep the search query as short and as simple as possible, don't use quotes around the search query.

`,
    prompt: `Create the search query for the term "${term}."`,
    schema: insertSearchQuerySchema.omit({ createdAt: true, updatedAt: true }),
  });

  // create the search query in the database & connect it to the entry:
  const [insertedQueryId] = await db
    .insert(searchQueries)
    .values(generatedQuery.object)
    .$returningId();

  if (!insertedQueryId) {
    throw new Error("Failed to insert or update search query");
  }
  return db.query.entries.findFirst({
    where: eq(entries.inputTerm, term),
    with: {
      searchQuery: true,
    },
  });
}
