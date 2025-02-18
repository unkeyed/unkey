import { db } from "@/lib/db-marketing/client";
import { entries } from "@/lib/db-marketing/schemas";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { eq } from "drizzle-orm";
import GithubSlugger from "github-slugger";
import yaml from "js-yaml"; // install @types/js-yaml?
import type { CacheStrategy } from "./_generate-glossary-entry";

export const createPrTask = task({
  id: "create_pr",
  retry: {
    maxAttempts: 0,
  },
  run: async ({
    input,
    onCacheHit = "stale" as CacheStrategy,
  }: { input: string; onCacheHit?: CacheStrategy }) => {
    // Add check for existing PR URL
    const existing = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      columns: {
        id: true,
        inputTerm: true,
        githubPrUrl: true,
        takeaways: true,
      },
      orderBy: (entries, { asc }) => [asc(entries.createdAt)],
    });

    if (existing?.githubPrUrl && onCacheHit === "stale") {
      return {
        entry: existing,
      };
    }

    // ==== 1. Prepare MDX file ====
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { asc }) => [asc(entries.createdAt)],
    });
    if (!entry?.dynamicSectionsContent) {
      throw new AbortTaskRunError(
        `Unable to create PR: The markdown content for the dynamic sections are not available for the entry to term: ${input}. It's likely that draft-sections.ts didn't run as expected .`,
      );
    }
    if (!entry.takeaways) {
      throw new AbortTaskRunError(
        `Unable to create PR: The takeaways are not available for the entry to term: ${input}. It's likely that content-takeaways.ts didn't run as expected.`,
      );
    }
    const slugger = new GithubSlugger();
    // Convert the object to YAML, ensuring the structure matches our schema
    const yamlString = yaml.dump(
      {
        title: entry.metaTitle,
        description: entry.metaDescription,
        h1: entry.metaH1,
        term: entry.inputTerm,
        categories: entry.categories,
        takeaways: {
          tldr: entry.takeaways.tldr,
          definitionAndStructure: entry.takeaways.definitionAndStructure,
          historicalContext: entry.takeaways.historicalContext,
          usageInAPIs: {
            tags: entry.takeaways.usageInAPIs.tags,
            description: entry.takeaways.usageInAPIs.description,
          },
          bestPractices: entry.takeaways.bestPractices,
          recommendedReading: entry.takeaways.recommendedReading,
          didYouKnow: entry.takeaways.didYouKnow,
        },
        faq: entry.faq,
        updatedAt: entry.updatedAt,
        slug: slugger.slug(entry.inputTerm),
      },
      {
        sortKeys: (a, b) => {
          // Ensure that 'question' always comes first
          if (a === "question" || b === "question") {
            return a === "question" ? -1 : 1;
          }
          return 0;
        },
        lineWidth: -1,
        noRefs: true,
        quotingType: '"',
      },
    );

    // Create frontmatter
    const frontmatter = `---\n${yamlString}---\n`;

    const mdxContent = `${frontmatter}${entry.dynamicSectionsContent}`;
    const blob = new Blob([mdxContent], { type: "text/markdown" });

    // Create a File object from the Blob
    const file = new File([blob], `${input.replace(/\s+/g, "-").toLowerCase()}.mdx`, {
      type: "text/markdown",
    });
    console.info("1. MDX file created");

    // ==== 2. Handle GitHub: create branch, file content & PR ====

    console.info(`2. ⏳ Creating PR for entry to term: "${input}"`);
    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN,
    });

    const owner = "unkeyed";
    const repo = "unkey";
    const branch = `richard/add-${input.replace(/\s+/g, "-").toLowerCase()}`;
    const path = `apps/www/content/glossary/${input.replace(/\s+/g, "-").toLowerCase()}.mdx`;

    const existingPr = await octokit.rest.pulls.list({
      owner,
      repo,
      base: "main",
      head: `${owner}:${branch}`,
      state: "open",
    });

    if (existingPr?.data?.length > 0) {
      console.info("2.1 ⏩︎ Pending (open) PR found. Updating the content of the file directly...");

      console.info(`this is for debugging, check if the file actually exists or not in the github ui: ${existingPr.data[0].head.ref}
        URL: https://github.com/unkeyed/unkey/pull/${existingPr.data[0].number}`);
      // get the blob sha of the file being replaced:
      const existingFile = await octokit.repos.getContent({
        owner,
        repo,
        ref: existingPr.data[0].head.ref,
        path,
      });
      // if an open PR exists, update the content of the file directly
      await octokit.repos.createOrUpdateFileContents({
        owner,
        repo,
        path,
        message: `feat(glossary): Update ${input}.mdx`,
        content: Buffer.from(await file.arrayBuffer()).toString("base64"),
        branch,
        committer: {
          name: "Richard Poelderl",
          email: "richard.poelderl@gmail.com",
        },
        ...("sha" in existingFile.data && { sha: existingFile.data.sha }),
      });

      console.info("2.2 💽 PR updated. Storing the URL...");
      // update the entry in the database with the PR URL
      await db
        .update(entries)
        .set({ githubPrUrl: existingPr.data[0].html_url })
        .where(eq(entries.inputTerm, input));

      const updated = await db.query.entries.findFirst({
        columns: {
          id: true,
          inputTerm: true,
          githubPrUrl: true,
        },
        where: eq(entries.inputTerm, input),
        orderBy: (entries, { desc }) => [desc(entries.createdAt)],
      });

      console.info("2.3 🎉 PR updated. Returning the entry...");

      return {
        entry: updated,
      };
    }

    // if there's no open PR, we have to handle the merged PR case:
    const existingMergedPr = await octokit.rest.pulls.list({
      owner,
      repo,
      base: "main",
      head: `${owner}:${branch}`,
      state: "closed",
    });

    if (existingMergedPr?.data?.length > 0) {
      console.info("2.1 ⚠️ Merged PR found. Deleting the stale branch...");
      // if a merged PR exists, we can delete the branch to create a new one & commit the file
      await octokit.git.deleteRef({
        owner,
        repo,
        ref: `heads/${branch}`,
      });
    }

    console.info("2.2 🛣️ Creating the new branch");
    // create a new branch off of main
    const mainRef = await octokit.git.getRef({
      owner,
      repo,
      ref: "heads/main",
    });
    // create a new branch off of main
    await octokit.git.createRef({
      owner,
      repo,
      ref: `refs/heads/${branch}`,
      sha: mainRef.data.object.sha,
    });

    // Commit the MDX file to the new branch
    console.info(`2.3 📦 Committing the MDX file to the new branch "${branch}"`);
    // get the existing file's sha (if exists):
    const existingFile = await octokit.repos.getContent({
      owner,
      repo,
      path,
    });
    await octokit.repos.createOrUpdateFileContents({
      owner,
      repo,
      path,
      message: `feat(glossary): Add ${input}.mdx to glossary`,
      content: Buffer.from(await file.arrayBuffer()).toString("base64"),
      branch,
      ...("sha" in existingFile.data && { sha: existingFile.data.sha }),
    });

    console.info("2.4 📝 Creating the pull request");
    // Create a pull request
    const pr = await octokit.pulls.create({
      owner,
      repo,
      title: `Add ${input} to Glossary`,
      head: branch,
      base: "main",
      body: `This PR adds the ${input}.mdx file to the API documentation.`,
    });

    console.info("2.5 💽 PR created. Storing the URL...");
    // Update the entry in the database with the PR URL
    await db
      .update(entries)
      .set({ githubPrUrl: pr.data.html_url })
      .where(eq(entries.inputTerm, input));

    const updated = await db.query.entries.findFirst({
      columns: {
        id: true,
        inputTerm: true,
        githubPrUrl: true,
      },
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { asc }) => [asc(entries.createdAt)],
    });

    console.info("2.6 🎉 PR created. Returning the entry...");

    return {
      entry: updated,
    };
  },
});
