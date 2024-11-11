import { db } from "@/lib/db-marketing/client";
import {
  entries,
  type FirecrawlResponse,
  firecrawlResponses,
  insertSectionContentTypeSchema,
  insertSectionSchema,
  insertSectionsToKeywordsSchema,
  keywords,
  sectionContentTypes,
  sections,
  sectionsToKeywords,
  type SelectKeywords,
  selectKeywordsSchema,
} from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { and, eq, or } from "drizzle-orm";
import { z } from "zod";
import type { CacheStrategy } from "./_generate-glossary-entry";
import { getOrCreateSummary } from "@/lib/firecrawl";
import type { Keyword } from "@/lib/db-marketing/schemas/keywords";
import { performTechnicalEvalTask, performSEOEvalTask, performEditorialEvalTask } from "./evals";

// TODO: this task is a bit flake-y still
// - split up into smaller tasks,  and/or
// - move some of the in-memory storage to db caching, and/or
// - improve the prompts
export const generateOutlineTask = task({
  id: "generate_outline",
  retry: {
    maxAttempts: 5,
  },
  run: async ({
    term,
    onCacheHit = "stale" as CacheStrategy,
  }: { term: string; onCacheHit?: CacheStrategy }) => {
    const existing = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
      with: {
        dynamicSections: {
          with: {
            contentTypes: true,
            sectionsToKeywords: {
              with: {
                keyword: true,
              },
            },
          },
        },
      },
    });

    if (
      existing?.dynamicSections &&
      existing.dynamicSections.length > 0 &&
      onCacheHit === "stale"
    ) {
      return existing;
    }

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${term}`);
    }
    // Fetch top-ranking pages' markdown content
    const organicResults = await db.query.firecrawlResponses.findMany({
      where: eq(firecrawlResponses.inputTerm, term),
      with: {
        serperOrganicResult: {
          columns: { position: true },
        },
      },
    });
    if (organicResults.length === 0) {
      throw new AbortTaskRunError(`No organic results found for term: ${term}`);
    }
    console.info(`Step 1/8 - ORGANIC RESULTS: ${organicResults?.length} results`);

    // Summarize the markdown content to manage token limits
    const summaries = await Promise.all(
      organicResults?.map(async (result) =>
        getOrCreateSummary({
          url: result.sourceUrl,
          connectTo: { term },
          onCacheHit,
        }),
      ),
    );

    const topRankingContent = summaries
      .map((r) => `${r?.sourceUrl}\n${r?.summary}`)
      .join("=========\n\n");
    console.info(`Step 3/8 - SUMMARIES: ${topRankingContent}`);

    const contentKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "headers"), eq(keywords.source, "title")),
        eq(keywords.inputTerm, term),
      ),
    });
    console.info(`Step 3/8 - SUMMARIES: ${topRankingContent}`);

    // Step 4: Generate initial outline
    const initialOutline = await generateInitialOutline({
      term,
      topRankingContent,
      contentKeywords,
    });
    console.info(
      `Step 4/8 - INITIAL OUTLINE RESULT: ${JSON.stringify(initialOutline.object.outline)}`,
    );

    // Step 5: Technical review by domain expert
    const technicalEval = await performTechnicalEvalTask.triggerAndWait({
      input: term,
      content: topRankingContent,
      onCacheHit,
    });
    if (!technicalEval.ok) {
      throw new AbortTaskRunError("Technical evaluation failed");
    }
    console.info(`Step 5/8 - TECHNICAL EVALUATION RESULT: 
        ===
        Ratings: ${JSON.stringify(technicalEval.output.ratings)}
        ===
        Recommendations: ${JSON.stringify(technicalEval.output.recommendations)}
        `);
    const seoKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "related_searches"), eq(keywords.source, "auto_suggest")),
        eq(keywords.inputTerm, term),
      ),
    });

    // Step 6: SEO review
    const seoEval = await performSEOEvalTask.triggerAndWait({
      input: term,
      content: topRankingContent,
      onCacheHit,
    });
    if (!seoEval.ok) {
      throw new AbortTaskRunError("SEO evaluation failed");
    }
    console.info(`Step 6/8 - SEO EVALUATION RESULT: 
        ===
        Ratings: ${JSON.stringify(seoEval.output.ratings)}
        ===
        Recommendations: ${JSON.stringify(seoEval.output.recommendations)}
        `);

    const seoOptimizedOutline = await reviseSEOOutline({
      term,
      outlineToRefine: technicalEval.output.revisedOutline,
      reviewReport: seoEval.output,
      seoKeywordsToAllocate: seoKeywords,
    });
    console.info(
      `Step 7/8 - SEO OPTIMIZED OUTLINE RESULT: ${JSON.stringify(
        seoOptimizedOutline.object.outline,
      )}`,
    );

    // Step 7: Editorial review
    const editorialEval = await performEditorialEvalTask.triggerAndWait({
      input: term,
      content: seoOptimizedOutline.object.outline,
      onCacheHit,
    });
    if (!editorialEval.ok) {
      throw new AbortTaskRunError("Editorial evaluation failed");
    }
    console.info(`Step 8/8 - EDITORIAL EVALUATION RESULT: 
        ===
        Ratings: ${JSON.stringify(editorialEval.output.ratings)}
        ===
        Recommendations: ${JSON.stringify(editorialEval.output.recommendations)}
        `);

    // persist to db as a new entry by with their related entities
    const sectionInsertionPayload = editorialEval.output.outline.map((section) =>
      insertSectionSchema.parse({
        ...section,
        entryId: entry.id,
      }),
    );
    const newSectionIds = await db.insert(sections).values(sectionInsertionPayload).$returningId();

    // associate the keywords with the sections
    const keywordInsertionPayload = [];
    for (let i = 0; i < editorialEval.output.outline.length; i++) {
      // add the newly inserted section id to our outline
      const section = { ...editorialEval.output.outline[i], id: newSectionIds[i].id };
      for (let j = 0; j < section.keywords.length; j++) {
        const keyword = section.keywords[j];
        const keywordId = seoKeywords.find(
          (seoKeyword) => keyword.keyword === seoKeyword.keyword,
        )?.id;
        if (!keywordId) {
          console.warn(`Keyword "${keyword.keyword}" not found in seo keywords`);
          continue;
        }
        const payload = insertSectionsToKeywordsSchema.parse({
          sectionId: section.id,
          keywordId,
        });
        keywordInsertionPayload.push(payload);
      }
    }
    await db.insert(sectionsToKeywords).values(keywordInsertionPayload);

    // associate the content types with the sections
    const contentTypesInsertionPayload = editorialEval.output.outline.flatMap((section, index) =>
      section.contentTypes.map((contentType) =>
        insertSectionContentTypeSchema.parse({
          ...contentType,
          sectionId: newSectionIds[index].id,
        }),
      ),
    );
    await db.insert(sectionContentTypes).values(contentTypesInsertionPayload);

    const newEntry = await db.query.entries.findFirst({
      where: eq(entries.id, entry.id),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
      with: {
        dynamicSections: {
          with: {
            contentTypes: true,
            sectionsToKeywords: {
              with: {
                keyword: true,
              },
            },
          },
        },
      },
    });

    return newEntry;
  },
});

export const reviewSchema = z.object({
  evaluation: z.string(),
  missing: z.string().optional(),
  rating: z.number().min(0).max(10),
});

// Schema for initial outline: array of sections, each with content types and keywords
const finalOutlineSchema = z.object({
  outline: z.array(
    insertSectionSchema.omit({ entryId: true }).extend({
      contentTypes: z.array(insertSectionContentTypeSchema.omit({ sectionId: true })),
      keywords: z.array(selectKeywordsSchema.pick({ keyword: true })),
    }),
  ),
});
// the keywords are associated later
const initialOutlineSchema = finalOutlineSchema.extend({
  outline: z.array(finalOutlineSchema.shape.outline.element.omit({ keywords: true })),
});

async function generateInitialOutline({
  term,
  topRankingContent,
  contentKeywords,
}: {
  term: string;
  topRankingContent: string;
  contentKeywords: Array<SelectKeywords>;
}) {
  const initialOutlineSystem = `You are a **Technical SEO Content Writer** specializing in API development and computer science.
  Your objective is to create a flat, comprehensive outline for a glossary page based on summarized content from top-ranking pages.
  Ensure factual correctness, clarity, and SEO optimization without unnecessary subheadings.`;

  const initialOutlinePrompt = `
  Generate a comprehensive and factually accurate outline for a glossary page dedicated to the term: **${term}**.
  
  **Instructions:**
  - Analyze the summarized content from the top-ranking pages.
  - Create a flat, customized outline with sections that best address the search intent and provide comprehensive coverage of the term.
  - Ensure all sections are factually correct, unique, and tailored to the specific term's context in API development and computer science.
  - Denote the order of the sections
  - Include a short description under each heading that outlines the content to be included, explains its importance, and references sources.
  - Describe recommended content types for each section as per the schema definition called "type" inside the contentTypes array. These represent different type of content forms for SEO pages. Make a recommendation for what to use and keep track of your reasoning.
  - Ensure headers are under 70 characters, descriptive, and maintain clarity and readability.
  
  =====
  TOP RANKING PAGES CONTENT:
  =====
  ${topRankingContent}
  
  =====
  KEYWORDS USED IN HEADERS:
  =====
  FROM PAGE TITLES:
  ${contentKeywords
    .filter((k) => k.source === "title")
    .map((k) => `- ${k.keyword}`)
    .join("\n")}
  FROM HEADERS:
  ${contentKeywords
    .filter((k) => k.source === "headers")
    .map((k) => `- ${k.keyword}`)
    .join("\n")}
  `;

  return await generateObject({
    model: openai("gpt-4o-mini"),
    system: initialOutlineSystem,
    prompt: initialOutlinePrompt,
    schema: initialOutlineSchema,
  });
}

async function reviseSEOOutline({
  term,
  outlineToRefine,
  reviewReport,
  seoKeywordsToAllocate,
}: {
  term: string;
  outlineToRefine: z.infer<typeof initialOutlineSchema>["outline"];
  reviewReport: Awaited<ReturnType<typeof performSEOEvalTask>>["object"];
  seoKeywordsToAllocate: Array<Keyword>;
}) {
  const seoRevisionSystem = `
   You are a **Senior SEO Strategist & Technical Content Specialist** with over 10 years of experience in optimizing content for API development and computer science domains.

   Task:
   - Refine the outline you're given based on the review report and guidelines
   - Allocate the provided keyworeds to the provided outline items

   **Guidelines for Revised Outline:**
   1. Make each header unique and descriptive
   2. Include relevant keywords in headers (use only provided keywords)
   3. Keep headers concise (ideally under 60 characters)
   4. Make headers compelling and engaging
   5. Optimize headers for featured snippets
   6. Avoid keyword stuffing in headers
   7. Use long-tail keywords where appropriate
   8. Ensure headers effectively break up the text
   9. Allocate keywords from the provided list to each section (ie outline item) in the 'keywords' field as an object with the following structure: { keyword: string }
   10. Allocate each keyword only once across all sections
   11. Ensure the keyword allocation makes sense for each section's content
   12. If a keyword doesn't fit any section, leave it unallocated

   **Additional Considerations:**
   - Headers should read technically and logically
   - Headers should explain the content of their respective sections
   - Headers should be distinct from each other
   - Optimize for SEO without sacrificing readability
   - Write for API developers, not general internet users
   - Maintain a technical tone appropriate for the audience

   You have the ability to add, modify, or merge sections in the outline as needed to create the most effective and SEO-optimized structure.
   `;

  const seoRevisionPrompt = `
   Review the following outline for the term "${term}":

   Outline to refine:
   ${JSON.stringify(outlineToRefine)}

   Review report:
   ${JSON.stringify(reviewReport)}

   Provided keywords:
  Related Searches: ${JSON.stringify(
    seoKeywordsToAllocate
      .filter((k) => k.source === "related_searches")
      .map((k) => k.keyword)
      .join(", "),
  )}
  Auto Suggest: ${JSON.stringify(
    seoKeywordsToAllocate
      .filter((k) => k.source === "auto_suggest")
      .map((k) => k.keyword)
      .join(", "),
  )}
   `;

  return await generateObject({
    model: openai("gpt-4o-mini"),
    system: seoRevisionSystem,
    prompt: seoRevisionPrompt,
    schema: finalOutlineSchema,
  });
}
