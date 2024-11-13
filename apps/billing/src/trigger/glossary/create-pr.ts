import { db } from "@/lib/db-marketing/client";
import { entries } from "@/lib/db-marketing/schemas";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { eq } from "drizzle-orm";
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
      },
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (existing?.githubPrUrl && onCacheHit === "stale") {
      return {
        entry: existing,
        message: `Found existing PR for ${input}.mdx`,
      };
    }

    // ==== 1. Prepare MDX file ====
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });
    if (!entry?.dynamicSectionsContent) {
      throw new AbortTaskRunError(
        `Unable to create PR: The markdown content for the dynamic sections are not available for the entry to term: ${input}. It's likely that draft-sections.ts didn't run as expected .`,
      );
    }
    // add meta tags to content in .mdx format
    const frontmatter = `---
    title: "${entry.metaTitle}"
    description: "${entry.metaDescription}"
    ---
    `;
    const mdxContent = frontmatter + entry.dynamicSectionsContent;
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

    const owner = "p6l-richard";
    const repo = "unkey";
    const branch = `richard/add-${input.replace(/\s+/g, "-").toLowerCase()}`;
    const path = `apps/www/content/${input.replace(/\s+/g, "-").toLowerCase()}.mdx`;

    // Create a new branch
    const mainRef = await octokit.git.getRef({
      owner,
      repo,
      ref: "heads/main",
    });

    console.info(`2.1 'main' branch found. Should branch off of: ${mainRef.data.object.sha}`);

    console.info("2.2 Handling possible duplicate branches");
    const branchExists = await octokit.git
      .listMatchingRefs({
        owner,
        repo,
        ref: `heads/${branch}`,
      })
      .then((response) => response.data.length > 0);

    if (branchExists) {
      console.info("2.2.1 ⚠️ Duplicate branch found, deleting it");
      await octokit.git.deleteRef({
        owner,
        repo,
        ref: `heads/${branch}`,
      });
      console.info("2.2.2 ⌫ Branch deleted");
    }

    console.info("2.4 🛣️ Creating the new branch");
    await octokit.git.createRef({
      owner,
      repo,
      ref: `refs/heads/${branch}`,
      sha: mainRef.data.object.sha,
    });

    // Commit the MDX file to the new branch
    console.info(`2.5 📦 Committing the MDX file to the new branch "${branch}"`);
    await octokit.repos.createOrUpdateFileContents({
      owner,
      repo,
      path,
      message: `feat(glossary): Add ${input}.mdx to glossary`,
      content: Buffer.from(await file.arrayBuffer()).toString("base64"),
      branch,
    });

    console.info("2.6 📝 Creating the pull request");
    // Create a pull request
    const pr = await octokit.pulls.create({
      owner,
      repo,
      title: `Add ${input} to API documentation`,
      head: branch,
      base: "main",
      body: `This PR adds the ${input}.mdx file to the API documentation.`,
    });

    console.info("2.7 💽 PR created. Storing the URL...");
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
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    return {
      entry: updated,
      message: `feat(glossary): Add ${input}.mdx to glossary`,
    };
  },
});
