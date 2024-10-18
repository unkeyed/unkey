import { db } from "@/lib/db-marketing/client";
import {
  type sectionContentTypes,
  entries,
  firecrawlResponses,
  type keywords,
  type sections,
  type sectionsToKeywords,
  type SelectEntry,
} from "@/lib/db-marketing/schemas";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateText } from "ai";
import { and, eq, isNotNull } from "drizzle-orm";
import { UTApi } from "uploadthing/server";
import { seoMetaTagsTask } from "./seo-meta-tags";

export const draftSectionsTask = task({
  id: "draft_sections",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ term }: { term: string }) => {
    // Fetch the specific section and its associated data
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
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${term}`);
    }

    const entryWithMarkdownEnsured = {
      ...entry,
      dynamicSections: entry.dynamicSections
      .slice(0,6)
    };

    // Draft the markdown content
    const draftedContent = await draftSections({ term, entry: entryWithMarkdownEnsured });
    console.info(`Drafted dynamic sections for ${entry.inputTerm}: ${draftedContent}`);

    // Review the drafted content for factual correctness
    const reviewedContent = await reviewContent({ term, content: draftedContent });
    console.info(`Reviewed dynamic sections for ${entry.inputTerm}: ${reviewedContent}`);

    // SEO optimize the content
    const optimizedContent = await seoOptimizeContent({
      term: entry.inputTerm,
      content: reviewedContent,
      keywords: entry.dynamicSections.flatMap((ds) =>
        ds.sectionsToKeywords.map((stk) => stk.keyword.keyword),
      ),
    });
    console.info(`Optimized dynamic sections for ${entry.inputTerm}: ${optimizedContent}`);

    // given that optimizedContent is markdown, but we want to have our .mdx files, we need to add the frontmatter, e.g.:
    // ---
    // title: "MIME Types"
    // description: "MIME types, also known as Media Types, are essential in web and API development for defining the content type of transmitted data."
    // categories: ["API Development", "Software Architecture"]
    // ---
    // we'll call our `seoMetaTagsTask` here to get the title and description, and then add the categories from the dynamic sections
    const seoMetaTags = await seoMetaTagsTask.triggerAndWait({ term: entry.inputTerm });
    if (!seoMetaTags.ok) {
      throw new AbortTaskRunError("Failed to get SEO meta tags");
    }

    const withFrontmatter = [
      '---',
      `title: "${seoMetaTags.output.title}"`,
      `description: "${seoMetaTags.output.description}"`,
      '---',
      optimizedContent
    ].join('\n');

    // Convert the string content to a Blob
    const blob = new Blob([withFrontmatter], { type: "text/markdown" });

    // Create a File object from the Blob
    const file = new File([blob], `${entry.inputTerm}.mdx`, { type: "text/markdown" });

    const utapi = new UTApi({ token: process.env.UPLOADTHING_TOKEN });
    const [response] = await utapi.uploadFiles([file]);

    if (response.error) {
      throw new AbortTaskRunError(response.error.message);
    }
    await db
      .update(entries)
      .set({ utKey: response.data.key, utUrl: response.data.url })
      .where(eq(entries.id, entry.id));

    return response.data.url;
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
  organicResults
    .map((r) => `Source URL: ${r.sourceUrl}\nSummary: ${r.summary}`)
    .join("\n")
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
