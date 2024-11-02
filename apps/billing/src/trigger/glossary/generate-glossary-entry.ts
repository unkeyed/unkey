import { task } from "@trigger.dev/sdk/v3";
import { keywordResearchTask } from "./keyword-research";
import { generateOutlineTask } from "./generate-outline";
import { draftSectionTask } from "./draft-section";
import { seoMetaTagsTask } from "./seo-meta-tags";
import { compileMarkdownTask } from "./compile-markdown";
import { createPRTask } from "./create-pr";
import { AbortTaskRunError } from "@trigger.dev/sdk/v3";

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
    const draftPromises = outline.output.dynamicSections.map(section =>
      draftSectionTask.triggerAndWait({ 
        sectionId: section.id,
        term 
      })
    );
    const draftResults = await Promise.all(draftPromises);
    
    // Check if any drafts failed
    const failedDrafts = draftResults.filter(result => !result.ok);
    if (failedDrafts.length > 0) {
      throw new AbortTaskRunError(`${failedDrafts.length} section drafts failed for term: ${term}`);
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
    const markdown = await compileMarkdownTask.triggerAndWait({ term });
    if (!markdown.ok) {
      throw new AbortTaskRunError(`Markdown compilation failed for term: ${term}`);
    }
    console.info("✓ Markdown compiled");

    // Step 6: Create PR
    console.info("6/6 - Creating PR...");
    const pr = await createPRTask.triggerAndWait({ input: term });
    if (!pr.ok) {
      throw new AbortTaskRunError(`PR creation failed for term: ${term}`);
    }
    console.info(`✓ PR created: ${pr.output.prUrl}`);

    return {
      term,
      prUrl: pr.output.prUrl,
      keywordCount: keywordResearch.output.keywords.length,
      sectionCount: outline.output.dynamicSections.length,
      seoMetaTags: seoMetaTags.output,
      message: `Successfully generated glossary entry for ${term}`,
    };
  },
}); 