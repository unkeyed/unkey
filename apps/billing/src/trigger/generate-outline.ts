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
import { generateObject, generateText } from "ai";
import { and, desc, eq, or, sql } from "drizzle-orm";
import { z } from "zod";

export const generateOutlineTask = task<"generate_outline", { term: string }>({
  id: "generate_outline",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ term }) => {
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
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
    console.info(`Step 1/8 - ORGANIC RESULTS: ${organicResults?.length} results`);

    const summarizerSystemPrompt = ({ term, position }: { term: string; position: number }) => `You are the **Chief Technology Officer (CTO)** of a leading API Development Tools Company with extensive experience in API development using programming languages such as Go, TypeScript, and Elixir and other backend languages. You have a PhD in computer science from MIT. Your expertise ensures that the content you summarize is technically accurate, relevant, and aligned with best practices in API development and computer science.

**Your Task:**
Accurately and concisely summarize the content from the page that ranks #${position} for the term "${term}". Focus on technical details, including how the content is presented (e.g., text, images, tables). Ensure factual correctness and relevance to API development.

**Instructions:**
- Provide a clear and concise summary of the content.
- Highlight key technical aspects and insights related to API development.
- Mention the types of content included, such as images, tables, code snippets, etc.
- Cite the term the content is ranking for and its position in the SERP.
`;

    // Summarize the markdown content to manage token limits
    const summaryPromises = organicResults?.map(async (result) => {
      if (!result.summary) {
        const system = summarizerSystemPrompt({
          term,
          position: result.serperOrganicResult.position,
        });
        const prompt = `Summarize the following content for the term "${term}" that's ranking #${result.serperOrganicResult.position}:
        =======
        ${result.markdown}
        =======
        `;
        console.info(`Step 2/8 - SUMMARIZING: 
          SYSTEM: ${system}
          ---
          PROMPT: ${prompt}`);
        const summaryCompletion = await generateText({
          model: openai("gpt-4o"),
          system,
          prompt,
          maxTokens: 500,
        });

        // Update the database with the new summary
        await db.update(firecrawlResponses)
          .set({ summary: summaryCompletion.text })
          .where(eq(firecrawlResponses.id, result.id));

        return summaryCompletion.text;
      }
      return result.summary;
    });

    await Promise.all(summaryPromises);

    // now, that we ensure all summaries are in our db, we fetch them again:
    const summariesWithUrls = await db.query.firecrawlResponses.findMany({
      where: eq(firecrawlResponses.inputTerm, term),
      columns: {
        sourceUrl: true,
        summary: true,
      },
    });

    const topRankingContent = summariesWithUrls.map((r) => `${r.sourceUrl}\n${r.summary}`).join("=========\n\n");
    console.info(`Step 3/8 - SUMMARIES: ${topRankingContent}`);

    const contentKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "headers"), eq(keywords.source, "title")),
        eq(keywords.inputTerm, term),
      ),
    });
    console.info(`Step 3/8 - SUMMARIES: ${topRankingContent}`);

    // Step 4: Generate initial outline
    const initialOutline = await generateInitialOutline(term, topRankingContent, contentKeywords);
    console.info(
      `Step 4/8 - INITIAL OUTLINE RESULT: ${JSON.stringify(initialOutline.object.outline)}`,
    );

    // Step 5: Technical review by domain expert
    const technicalReview = await performTechnicalReview({
      term,
      outlineToReview: initialOutline.object.outline,
      authoritativeContent: { markdown: topRankingContent, keywords: contentKeywords },
    });
    console.info(`Step 5/8 - TECHNICAL REVIEW RESULT: 
        ===
        Analysis: ${JSON.stringify(technicalReview.object.analysis)}
        ===
        Recommendations: ${JSON.stringify(technicalReview.object.recommendations)}
        ===
        Revised Outline: ${JSON.stringify(technicalReview.object.revisedOutline)}
        `);
    const seoKeywords = await db.query.keywords.findMany({
      where: and(
        or(eq(keywords.source, "related_searches"), eq(keywords.source, "auto_suggest")),
        eq(keywords.inputTerm, term),
      ),
    });

    // Step 6: SEO review
    const seoReview = await performSEOReview({
      term,
      outlineToReview: technicalReview.object.revisedOutline,
      seoKeywords: seoKeywords,
    });
    console.info(`Step 6/8 - SEO REVIEW RESULT: 
        ===
        Analysis: ${JSON.stringify(seoReview.object.analysis)}
        ===
        Recommendations: ${JSON.stringify(seoReview.object.recommendations)}`);

    const seoOptimizedOutline = await reviseSEOOutline({
      term,
      outlineToRefine: technicalReview.object.revisedOutline,
      reviewReport: seoReview.object,
      seoKeywordsToAllocate: seoKeywords,
    });
    console.info(
      `Step 7/8 - SEO OPTIMIZED OUTLINE RESULT: ${JSON.stringify(
        seoOptimizedOutline.object.outline,
      )}`,
    );

    // Step 7: Editorial review
    const editorialReview = await performEditorialReview({
      term,
      outlineToReview: seoOptimizedOutline.object.outline,
    });
    console.info(`Step 8/8 - EDITORIAL REVIEW RESULT: 
        ===
        Analysis: ${JSON.stringify(editorialReview.object.analysis)}
        ===
        Recommendations: ${JSON.stringify(editorialReview.object.recommendations)}
        ===
        Revised Outline: ${JSON.stringify(editorialReview.object.outline)}
        `);

    // persist to db as a new entry by with their related entities
    const sectionInsertionPayload = editorialReview.object.outline.map((section) =>
      insertSectionSchema.parse({
        ...section,
        entryId: entry.id,
      }),
    );
    const newSectionIds = await db.insert(sections).values(sectionInsertionPayload).$returningId();
    
    // associate the keywords with the sections
    const keywordInsertionPayload = [];
    for (let i = 0; i < editorialReview.object.outline.length; i++) {
      // add the newly inserted section id to our outline
      const section = { ...editorialReview.object.outline[i], id: newSectionIds[i].id };
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
    await db.insert(sectionsToKeywords).values(keywordInsertionPayload)
    
    // associate the content types with the sections
    const contentTypesInsertionPayload = editorialReview.object.outline.flatMap((section, index) =>
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

const reviewSchema = z.object({
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

async function generateInitialOutline(
  term: string,
  topRankingContent: string,
  contentKeywords: Array<SelectKeywords>,
) {
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
    model: openai("gpt-4o"),
    system: initialOutlineSystem,
    prompt: initialOutlinePrompt,
    schema: initialOutlineSchema,
  });
}

async function performTechnicalReview({
  term,
  outlineToReview,
  authoritativeContent,
}: {
  term: string;
  outlineToReview: z.infer<typeof initialOutlineSchema>["outline"];
  authoritativeContent: { markdown: FirecrawlResponse["markdown"]; keywords: Array<Keyword> };
}) {
  const technicalReviewSystem = `
  You are a **Senior Technical Content Writer** with a PhD in computer science from MIT. You are an expert in API development and computer science.

  **Your Task:**
  Review the following outline for the term "${term}" and perform these checks as part of your technical review:
  1. Check if the outline covers all important aspects of the topic
  2. Check if the topic is covered comprehensively (no important details are missed)
  3. Check if the topic is covered concisely (no fluff, no unnecessary details)
  4. Check if the outline doesn't include unrelated topics or terms better suited for a separate entry
  5. Check if the outline is flat and doesn't include subheadings unless absolutely necessary
  
  **Output**
  - Provide an analysis of the outline's topic coverage, comprehensiveness, and conciseness. This includes a rating on a scale of 0-10 for each category.
  - Provide recommendations for improvements or changes based on your technical expertise.
  - Provide a revised outline based on your recommendations.
  `;

  const technicalReviewPrompt = `
  Review the following outline for the term "${term}":

  ${JSON.stringify(outlineToReview)}

  Please provide your analysis and any recommendations for improvement.

  ====
  TOP RANKING PAGES CONTENT:
  ====
  ${authoritativeContent.markdown}

  ====
  KEYWORDS USED IN HEADERS:
  ====
  ${authoritativeContent.keywords.map((k) => `- ${k.keyword}`).join("\n")}
  `;

  return await generateObject({
    model: openai("gpt-4o"),
    system: technicalReviewSystem,
    prompt: technicalReviewPrompt,
    schema: z.object({
      analysis: z.object({
        topicCoverage: reviewSchema,
        comprehensiveness: reviewSchema,
        conciseness: reviewSchema,
      }),
      recommendations: z.array(
        z.object({
          type: z.enum(["addSection", "modifySection", "mergeSection"]),
          description: z.string(),
          suggestedChange: z.string(),
        }),
      ),
      revisedOutline: initialOutlineSchema.shape.outline,
    }),
  });
}

async function performSEOReview({
  term,
  outlineToReview,
  seoKeywords,
}: {
  term: string;
  outlineToReview: z.infer<typeof initialOutlineSchema>["outline"];
  seoKeywords: Array<Keyword>;
}) {
  const relatedSearches = seoKeywords.filter((k) => k.source === "related_searches");
  const autoSuggest = seoKeywords.filter((k) => k.source === "auto_suggest");

  const seoReviewSystem = `
  You are a **Senior SEO Strategist & Technical Content Specialist** with over 10 years of experience in optimizing content for API development and computer science domains.

  **Your Task:**
  Review the outline you're given based on the following guidelines:

  **Guidelines for Review:**
  1. Assess if each header is unique and descriptive
  2. Check if relevant keywords are included in headers (only from the provided keyword list)
  3. Evaluate if headers are concise (ideally under 60 characters)
  4. Analyze if headers are compelling and engaging
  5. Determine if headers are optimized for featured snippets
  6. Look for any instances of keyword stuffing in headers
  7. Check for appropriate use of long-tail keywords
  8. Assess if headers effectively break up the text
  9. Review the allocation of keywords from the provided list to each section
  10. Verify that each keyword is allocated only once across all sections
  11. Evaluate if the keyword allocation makes sense for each section's content
  12. Identify any keywords that don't fit any section and remain unallocated

  **Additional Considerations:**
  - Assess if headers read technically and logically
  - Verify if headers explain the content of their respective sections
  - Check if headers are distinct from each other
  - Evaluate if the outline is optimized for SEO without sacrificing readability
  - Determine if the content is tailored for API developers
  - Assess if the outline maintains a technical tone appropriate for the audience

  **Output:**
  - Provide an analysis of the outline's keyword coverage and allocation, this includes a rating on a scale of 0-10 for each category, a short evaluation and any missing sections (if any).
  - Offer recommendations for improvements based on your SEO expertise
  `;

  const seoReviewPrompt = `
  Review the following technically-reviewed outline for the term "${term}":

  ${JSON.stringify(outlineToReview)}

  Provided keywords:
  Related Searches: ${JSON.stringify(relatedSearches)}
  Auto Suggest: ${JSON.stringify(autoSuggest)}

  Please provide your analysis & recommendations for SEO improvement
  `;

  return await generateObject({
    model: openai("gpt-4o"),
    system: seoReviewSystem,
    prompt: seoReviewPrompt,
    schema: z.object({
      analysis: z.object({
        keywordCoverage: reviewSchema,
        keywordSufficiency: reviewSchema,
      }),
      recommendations: z.array(
        z.object({
          type: z.enum(["addSection", "modifySection", "mergeSection"]),
          description: z.string(),
          suggestedChange: z.string(),
        }),
      ),
    }),
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
  reviewReport: Awaited<ReturnType<typeof performSEOReview>>["object"];
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
    model: openai("gpt-4o"),
    system: seoRevisionSystem,
    prompt: seoRevisionPrompt,
    schema: finalOutlineSchema,
  });
}
async function performEditorialReview({
  term,
  outlineToReview,
}: {
  term: string;
  outlineToReview: z.infer<typeof finalOutlineSchema>["outline"];
}) {
  const editorialReviewSystem = `
  You are a **Senior Technical Content Writer** with a PhD in computer science from MIT. You are an expert in API development and computer science.

  **Your Task:**
  Briefly review the following outline for the term "${term}" for editorial quality.

  **Guidelines:**
  1. Ensure that the reader gets a good understanding of the topic when seeing a table of contents (readers are API developers looking up API development related terms)
  2. Ensure that there's good flow between the headings. A question in one heading should be answered in a later heading.
  3. Ensure that the wording is authoritative and professional
  4. Ensure that headings are curiosity-inducing and engaging
  5. Ensure that headings are not click-baity or cringe-worthy
  6. Ensure that there's content indicated that is share-worthy (e.g., little-known facts, fun facts, etc.)

  **Output**
  - Provide an analysis of the outline's readability, flow, professionalism, engagement, and shareability. This includes a rating on a scale of 0-10 for each category.
  - Provide recommendations for improvements or changes based on your editorial expertise.
  - Provide a revised outline based on your recommendations.
  `;

  const editorialReviewPrompt = `
  Review the following SEO-optimized outline for the term "${term}":

  ${JSON.stringify(outlineToReview)}

  Please provide your analysis and any recommendations for editorial improvement.
  `;
  console.info(`Step 7/8 - EDITORIAL REVIEW PROMPT: ${editorialReviewPrompt}`);

  return await generateObject({
    model: openai("gpt-4o"),
    system: editorialReviewSystem,
    prompt: editorialReviewPrompt,
    schema: z.object({
      analysis: z.object({
        flow: reviewSchema,
        professionalism: reviewSchema,
        engagement: reviewSchema,
      }),
      recommendations: z.array(
        z.object({
          type: z.enum(["modifySection", "mergeSection"]),
          description: z.string(),
          suggestedChange: z.string(),
        }),
      ),
      outline: finalOutlineSchema.shape.outline,
    }),
  });
}
