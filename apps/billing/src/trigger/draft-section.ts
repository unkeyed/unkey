import { db } from "@/lib/db-marketing/client";
import {
  type sectionContentTypes,
  firecrawlResponses,
  sections,
} from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateText } from "ai";
import { eq } from "drizzle-orm";

// export const draftSectionsTask = task<"draft_sections", { term: string }>({
//   id: "draft_sections",
//   retry: {
//     maxAttempts: 0,
//   },
//   run: async ({ term }) => {
//     // Fetch the entry and its associated sections
//     let entry = await db.query.entries.findFirst({  
//       where: eq(entries.inputTerm, term),
//       with: {
//         dynamicSections: {
//           with: {
//             contentTypes: true,
//             sectionsToKeywords: {
//               with: {
//                 keyword: true,
//               },
//             },
//           },
//         },
//       },
//     });

//     // If the entry doesn't exist, run the generate-outline task first
//     if (!entry) {
//       const generatedEntry = await generateOutlineTask.triggerAndWait({ term });
//       if (!generatedEntry.ok) {
//         throw new AbortTaskRunError(`Failed to generate outline for term: ${term}`);
//       }
//       entry = generatedEntry.output;
//     }
    
//     // if we still don't have any dynamic sections, we can't draft anything
//     if (!entry?.dynamicSections || entry.dynamicSections.length === 0) {
//       throw new AbortTaskRunError(`No dynamic sections found for term: ${term}`);
//     }

//     const draftedSections = [];

//     for (const section of entry.dynamicSections) {
//       // Draft the section content
//       const draftedContent = await draftSection({term, section});

//       // Review the drafted content for factual correctness
//       const reviewedContent = await reviewContent({term, content: draftedContent});

//       // SEO optimize the content
//       const optimizedContent = await seoOptimizeContent({
//         content: reviewedContent,
//         keywords: section.sectionsToKeywords.map((stk) => stk.keyword.keyword),
//       });
//       console.info(`Drafted section for ${section.heading}: ${optimizedContent}`);

//       draftedSections.push({
//         heading: section.heading,
//         content: optimizedContent,
//       });
//     }

//     return draftedSections;
//   },
// });

export const draftSectionTask = task({
  id: "draft_section",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ sectionId, term }: { sectionId: number; term: string }) => {
    // Fetch the specific section and its associated data
    const section = await db.query.sections.findFirst({
      where: eq(sections.id, sectionId),
      with: {
        contentTypes: true,
        sectionsToKeywords: {
          with: {
            keyword: true,
          },
        },
      },
    });

    if (!section) {
      throw new AbortTaskRunError(`Section not found for id: ${sectionId}`);
    }

    // Draft the section content
    const draftedContent = await draftSection({ term, section });
    console.info(`Drafted section for ${section.heading}: ${draftedContent}`);

    // Review the drafted content for factual correctness
    const reviewedContent = await reviewContent({ term, content: draftedContent });
    console.info(`Reviewed section for ${section.heading}: ${reviewedContent}`);

    // SEO optimize the content
    const optimizedContent = await seoOptimizeContent({
      content: reviewedContent,
      keywords: section.sectionsToKeywords.map((stk) => stk.keyword.keyword),
    });
    console.info(`Optimized section for ${section.heading}: ${optimizedContent}`);

    // persist this content to the section in our db:
    await db.update(sections).set({markdown: optimizedContent}).where(eq(sections.id, sectionId));

    return {
      sectionId: section.id,
      heading: section.heading,
      content: optimizedContent,
    };
  },
});

async function draftSection({
  term,
  section,
}:{
  term: string,
  section: typeof sections.$inferSelect & {
    contentTypes: (typeof sectionContentTypes.$inferSelect)[];
  },
}) {
  const system = `You are an expert technical writer specializing in API development. Your task is to draft a section for a glossary entry on "${term}". Focus on providing accurate, concise, and valuable information for API developers.`;

  const prompt = `
Draft a section for the glossary entry on "${term}" with the following details:

Heading: ${section.heading}
Description: ${section.description}
Content Types: ${section.contentTypes.map((ct) => ct.type).join(", ")}

Guidelines:
1. Write in markdown format.
2. Use "##" for the section heading.
3. Ensure the content is factually correct and relevant to API development.
4. Include examples, code snippets or other rich content if appropriate.
5. Keep the content concise but informative.
6. Only write the content for the section, do not provide any other context, introductions or statements regarding this task.

`;

  const completion = await generateText({
    model: openai("gpt-4-turbo"),
    system,
    prompt,
  });

  return completion.text;
}

async function reviewContent({term, content}:{term: string, content: string}) {
  const system = `You are a senior technical reviewer with expertise in API development. Your task is to review the drafted content for factual correctness and relevance to the term "${term}".`;

  // get the organicResults.summary for the top 3 results for this term & pass it on to the LLM so that it analyzes if there's anything wrong here:
  const organicResults = await db.query.firecrawlResponses.findMany({
    where: eq(firecrawlResponses.inputTerm, term),
    limit: 3,
  });

  const prompt = `
Review the following drafted content for the glossary entry on "${term}":

${content}

Guidelines:
1. Check for any factual errors or inaccuracies by cross-referencing the content with the below summaries from the top 3 organic search results for "${term}".
2. Ensure the content is relevant to API development.
3. Verify that the information is up-to-date and follows best practices.
4. Suggest any necessary corrections or improvements.
5. If the content is accurate and relevant, simply respond with "Content is factually correct and relevant."
6. Verify that the content reads well for developers looking up the term online, there should not be any setences that were introduced by ai agents as part of their response to a message for a task.

Top 3 organic search results for "${term}":
${
    // provide the sourceUrl and the summary from the organicResult
    organicResults.map((r) => `Source URL: ${r.sourceUrl}\nSummary: ${r.summary}`).join("\n")
}
`;

  const completion = await generateText({
    model: openai("gpt-4o-mini"),
    system,
    prompt,
  });

  return completion.text === "Content is factually correct and relevant."
    ? content
    : completion.text;
}

async function seoOptimizeContent({content, keywords}:{content: string, keywords: string[]}) {
  const system = `You are an SEO expert specializing in technical content for
  API developers. Your task is to optimize the following content for SEO.
  `;

  const prompt = `
  Optimize the following content for SEO:

  ${content}

  Keywords: ${keywords.join(", ")}

  Guidelines:
  1. Ensure the content is optimized for the given keywords.
  2. Include the keywords in the content in a natural and engaging manner.
  3. Ensure the content is concise and informative.
  4. Only write the content for the section, do not provide any other context, introductions or statements regarding this task.
  `;

  const completion = await generateText({
    model: openai("gpt-4o-mini"),
    system,
    prompt,
  });

  return completion.text;
}
