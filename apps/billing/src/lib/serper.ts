import { db } from "@/lib/db-marketing/client";
import {
  serperOrganicResults,
  serperPeopleAlsoAsk,
  serperRelatedSearches,
  serperSearchResponses,
  serperTopStories,
} from "@/lib/db-marketing/schemas";
import { SerperClient, type serper } from "@agentic/serper";
import { eq, sql } from "drizzle-orm";

export async function persistSerperResults(args: {
  serper: serper.SearchResponse;
  inputTerm: string;
}) {
  return db.transaction(async (tx) => {
    const { serper: searchResults, inputTerm } = args;
    const [{ id: searchResponseId }] = await tx
      .insert(serperSearchResponses)
      .values({
        searchParameters: searchResults.searchParameters,
        answerBox: searchResults.answerBox,
        knowledgeGraph: searchResults.knowledgeGraph,
        inputTerm: inputTerm,
      })
      .$returningId();

    const insertPromises = [
      searchResults.organic &&
        tx.insert(serperOrganicResults).values(
          searchResults.organic.map((result) => ({
            searchResponseId,
            title: result.title,
            link: result.link,
            snippet: result.snippet,
            position: result.position,
            imageUrl: result.imageUrl,
          })),
        ),
      searchResults.topStories &&
        tx.insert(serperTopStories).values(
          searchResults.topStories.map((story) => ({
            searchResponseId,
            title: story.title,
            link: story.link,
            source: story.source,
            date: story.date,
            imageUrl: story.imageUrl,
          })),
        ),
      searchResults.peopleAlsoAsk &&
        tx.insert(serperPeopleAlsoAsk).values(
          searchResults.peopleAlsoAsk.map((item) => ({
            searchResponseId,
            question: item.question,
            snippet: item.snippet,
            title: item.title,
            link: item.link,
          })),
        ),
      searchResults.relatedSearches &&
        tx.insert(serperRelatedSearches).values(
          searchResults.relatedSearches.map((item) => ({
            searchResponseId,
            query: item.query,
          })),
        ),
    ];

    await Promise.all(insertPromises.filter(Boolean));

    const newSearchResponse = await tx.query.serperSearchResponses.findFirst({
      where: eq(serperSearchResponses.id, searchResponseId),
      with: {
        serperOrganicResults: true,
        serperRelatedSearches: true,
        serperPeopleAlsoAsk: true,
      },
    });

    return newSearchResponse;
  });
}

export async function getOrCreateSearchResponse(args: {
  query: string;
  inputTerm: string;
}) {
  // Try to find existing search response
  const existingResponse = await db.query.serperSearchResponses.findFirst({
    // - The findFirst() method is used instead of findMany() to return a single result, which aligns with your request to get the first matching row
    // - The case-insensitive comparison is achieved by using LOWER() on both the extracted JSON value and the search string
    where: sql`LOWER(JSON_UNQUOTE(JSON_EXTRACT(${
      serperSearchResponses.searchParameters
    }, '$.q'))) LIKE ${sql`LOWER(${`%${args.query}%`})`}`,
    with: {
      serperOrganicResults: true,
      serperRelatedSearches: true,
      serperPeopleAlsoAsk: true,
    },
  });

  if (existingResponse && existingResponse.serperOrganicResults.length > 0) {
    return existingResponse;
  }

  // If not found or incomplete, fetch from Serper and persist
  console.info(
    `[search] ℹ️ No complete search response found for '${args.query}', running Serper API call`,
  );
  const serper = new SerperClient({
    apiKey: process.env.SERPER_API_KEY,
    gl: "en",
    num: 10,
    page: 1,
  });

  const searchResults = await serper.search(args.query);

  const inserted = await persistSerperResults({
    serper: searchResults,
    inputTerm: args.inputTerm,
  });

  // Fetch the newly inserted response with relations
  const newResponse = await db.query.serperSearchResponses.findFirst({
    where: eq(serperSearchResponses.id, inserted?.id),
    with: {
      serperOrganicResults: true,
      serperRelatedSearches: true,
      serperPeopleAlsoAsk: true,
    },
  });

  if (!newResponse) {
    throw new Error("Failed to retrieve newly inserted search response");
  }

  return newResponse;
}
