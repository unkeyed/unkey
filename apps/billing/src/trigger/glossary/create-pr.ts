import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { Octokit } from "@octokit/rest";
import { db } from "@/lib/db-marketing/client";
import { entries } from "@/lib/db-marketing/schemas";
import { eq } from "drizzle-orm";

export const createPrTask = task({
  id: "create_pr",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ input }: { input: string }) => {
    // Initialize Octokit
    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN
    });

    const owner = "p6l-richard";
    const repo = "unkey";
    const branch = `richard/add-${input.replace(/\s+/g, '-').toLowerCase()}`;
    const path = `apps/www/content/${input.replace(/\s+/g, '-').toLowerCase()}.mdx`;

    console.info(`1. Creating PR for "${input}"`);

    console.info("1.1 Attempting to get the ref for main branch");
    // Create a new branch
    const mainRef = await octokit.git.getRef({
      owner,
      repo,
      ref: "heads/main"
    });

    console.info(`1.2 main ref: ${mainRef.data.object.sha}`);

    console.info("1.3 Creating the branch");
    await octokit.git.createRef({
      owner,
      repo,
      ref: `refs/heads/${branch}`,
      sha: mainRef.data.object.sha
    });

    console.info(`1.4 Branch created: ${branch}`);

    console.info("1.5 Adding the file contents to the branch");
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
    });
    if (!entry?.markdown) {
      throw new AbortTaskRunError(`Markdown content not found for term: ${input}`);
    }
    // Convert the string content to a Blob
    const blob = new Blob([entry.markdown], { type: 'text/markdown' });
    
    // Create a File object from the Blob
    const file = new File([blob], `${input.replace(/\s+/g, '-').toLowerCase()}.mdx`, { type: 'text/markdown' });
    // Create or update file contents
    await octokit.repos.createOrUpdateFileContents({
      owner,
      repo,
      path,
      message: `feat(glossary): Add ${input}.mdx to glossary`,
      content: Buffer.from(await file.arrayBuffer()).toString('base64'),
      branch 
    });

    console.info("1.7 Creating the pull request");
    // Create a pull request
    const pr = await octokit.pulls.create({
      owner,
      repo,
      title: `Add ${input} to API documentation`,
      head: branch,
      base: "main",
      body: `This PR adds the ${input}.mdx file to the API documentation.`
    });

    console.info(`1.8 Updating the entry in the database with the PR URL`);
    // Update the entry in the database with the PR URL
    await db.update(entries)
      .set({ prUrl: pr.data.html_url })
      .where(eq(entries.inputTerm, input));

    console.info(`✓ PR created: ${pr.data.html_url}`);

    return {
      prUrl: pr.data.html_url,
      message: `feat(glossary): Add ${input}.mdx to glossary`,
    };
  }
});

