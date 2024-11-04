import { task } from "@trigger.dev/sdk/v3";
import { keywordResearchTask } from "./keyword-research";
import { generateOutlineTask } from "./generate-outline";
import { draftSectionsTask } from "./draft-sections";
import { seoMetaTagsTask } from "./seo-meta-tags";
import { createPrTask } from "./create-pr";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";
import { createMarkdownContent } from "./create-markdown-content";

export const generateGlossaryEntryTask = task({
  id: "generate_glossary_entry",
  retry: {
    maxAttempts: 0, // Consistent with other tasks in the codebase
  },
  run: async ({ term }: { term: string }) => {
    console.info(`Starting glossary entry generation for term: ${term}`);

    // Step 1: Keyword Research
    console.info("1/6 - Starting keyword research...");
    const keywordResearch = await keywordResearchTask.triggerAndWait({ term });
    if (!keywordResearch.ok) {
      throw new AbortTaskRunError(`Keyword research failed for term: ${term}`);
    }
    console.info(`✓ Keyword research completed with ${keywordResearch.output.keywords.length} keywords`);

    // Step 2: Generate Outline
    console.info("2/6 - Generating outline...");
    const outline = await generateOutlineTask.triggerAndWait({ term });
    if (!outline.ok) {
      throw new AbortTaskRunError(`Outline generation failed for term: ${term}`);
    }
    console.info("✓ Outline generated");

    // Step 3: Draft Sections
    console.info("3/6 - Drafting sections...");
    const draftSections = await draftSectionsTask.triggerAndWait({ term });
    if (!draftSections.ok) {
      throw new AbortTaskRunError(`Section drafting failed for term: ${term}`);
    }
    console.info("✓ All sections drafted");

    // Step 4: Generate SEO Meta Tags
    console.info("4/6 - Generating SEO meta tags...");
    const seoMetaTags = await seoMetaTagsTask.triggerAndWait({ term });
    if (!seoMetaTags.ok) {
      throw new AbortTaskRunError(`SEO meta tags generation failed for term: ${term}`);
    }
    console.info("✓ SEO meta tags generated");

    // Step 5: Compile Markdown
    console.info("5/6 - Compiling markdown...");
    const markdown = await createMarkdownContent.triggerAndWait({ term });
    if (!markdown.ok) {
      throw new AbortTaskRunError(`Markdown compilation failed for term: ${term}`);
    }
    console.info("✓ Markdown compiled");

    // Step 6: Create PR
    console.info("6/6 - Creating PR...");
    const pr = await createPrTask.triggerAndWait({ input: term });
    if (!pr.ok) {
      throw new AbortTaskRunError(`PR creation failed for term: ${term}`);
    }
    console.info(`✓ PR created: ${pr.output.prUrl}`);

    return {
      term,
      prUrl: pr.output.prUrl,
      keywordCount: keywordResearch.output.keywords.length,
      sectionCount: outline?.output?.dynamicSections.length,
      seoMetaTags: seoMetaTags.output,
      message: `Successfully generated glossary entry for ${term}`,
    };
  },
}); 