import { db } from "@/lib/db-marketing/client";
import {
  type SelectEntry,
  entries,
  firecrawlResponses,
  type keywords,
  type sectionContentTypes,
  type sections,
  type sectionsToKeywords,
} from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateText } from "ai";
import { eq } from "drizzle-orm";
import type { CacheStrategy } from "./_generate-glossary-entry";

export const draftSectionsTask = task({
  id: "draft_sections",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    term,
    onCacheHit = "stale" as CacheStrategy,
  }: { term: string; onCacheHit?: CacheStrategy }) => {
    const existing = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
      columns: {
        id: true,
        inputTerm: true,
        dynamicSectionsContent: true,
      },
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (existing?.dynamicSectionsContent && onCacheHit === "stale") {
      return existing;
    }

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
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
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${term}`);
    }

    const entryWithMarkdownEnsured = {
      ...entry,
      dynamicSections: entry.dynamicSections.slice(0, 6),
    };

    const draftedContent = await draftSections({ term, entry: entryWithMarkdownEnsured });
    console.info(`Drafted dynamic sections for ${entry.inputTerm}: ${draftedContent}`);

    const reviewedContent = await reviewContent({ term, content: draftedContent });
    console.info(`Reviewed dynamic sections for ${entry.inputTerm}: ${reviewedContent}`);

    const optimizedContent = await seoOptimizeContent({
      term: entry.inputTerm,
      content: reviewedContent,
      keywords: entry.dynamicSections.flatMap((ds) =>
        ds.sectionsToKeywords.map((stk) => stk.keyword.keyword),
      ),
    });
    console.info(`Optimized dynamic sections for ${entry.inputTerm}: ${optimizedContent}`);

    const [inserted] = await db
      .insert(entries)
      .values({
        inputTerm: entry.inputTerm,
        dynamicSectionsContent: optimizedContent,
      })
      .$returningId();
    return db.query.entries.findFirst({
      columns: {
        id: true,
        inputTerm: true,
        dynamicSectionsContent: true,
      },
      where: eq(entries.id, inserted.id),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
  },
});

async function draftSections({
  term,
  entry,
}: {
  term: string;
  entry: typeof entries.$inferSelect & {
    dynamicSections: Array<
      typeof sections.$inferSelect & {
        contentTypes: Array<typeof sectionContentTypes.$inferSelect>;
        sectionsToKeywords: Array<
          typeof sectionsToKeywords.$inferSelect & {
            keyword: typeof keywords.$inferSelect;
          }
        >;
      }
    >;
  };
}) {
  const system = `You are an expert technical writer. You're working on a glossary for API developers. 
  Your task is to draft a section for a glossary entry on "${term}". 
  Focus on providing accurate, concise, and valuable information.`;

  const prompt = `
Draft markdown content for the glossary entry on "${term}" with the following details:

Term: ${entry.inputTerm}
Outline:
- ${entry.dynamicSections.map((ds) => ds.heading).join("\n- ")}

Find some additional information for each section below. Go 

${entry.dynamicSections
  .map(
    (ds) => `
Section: ${ds.heading}
Content Types: ${ds.contentTypes.map((ct) => ct.type).join(", ")}
Keywords: ${ds.sectionsToKeywords.map((stk) => stk.keyword).join(", ")}
`,
  )
  .join("\n")}

Guidelines:
1. Write in markdown format.
2. Start with an introductory paragraph before stating the first section heading
2. Use "##" for the section heading
3. Skip the title of the page, that will be provided separately
4. Ensure the content is factually correct and relevant to API development.
5. Include examples, code snippets or other rich content if appropriate.
6. Keep the content concise but informative, ensure that there are no fluff phrases or statements that don't provide concrete information, context & background to the term.
7. Don't repeat content between sections, ensure that each section adds value
8. Only write the content for the section, do not provide any other context, introductions or statements regarding this task.
`;

  const completion = await generateText({
    model: openai("gpt-4-turbo"),
    system,
    prompt,
  });

  return completion.text;
}

async function reviewContent({ term, content }: { term: string; content: string }) {
  const system = `You are a senior technical reviewer with expertise in API development. Your task is to review the drafted content for factual correctness and relevance to the term "${term}".`;

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
${organicResults.map((r) => `Source URL: ${r.sourceUrl}\nSummary: ${r.summary}`).join("\n")}
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

async function seoOptimizeContent({
  term,
  content,
  keywords,
}: { term: SelectEntry["inputTerm"]; content: string; keywords: string[] }) {
  const system = `You are an SEO expert specializing in technical content. You're working on a glossary for API developers.

  A technical writer has drafted a section for a glossary entry on "${term}" and your task is to optimize the content for SEO with the provided keywords.
  `;

  const prompt = `
  Optimize the following content for SEO:

  ${content}

  Keywords you've researched for this topic: ${keywords.join(", ")}

  Guidelines:
  1. Ensure the content is optimized for the given keywords.
  2. Include the keywords in the content in a natural and engaging manner.
  3. Ensure the content is concise and informative.
  4. Follow best practices for SEO without overly optimizing for keywords
  5. Avoid keyword stuffing, the content should read naturally from an API developer's perspective, who quickly wants to get the information about this term they're looking up.
  `;

  const completion = await generateText({
    model: openai("gpt-4o-mini"),
    system,
    prompt,
  });

  return completion.text;
}
