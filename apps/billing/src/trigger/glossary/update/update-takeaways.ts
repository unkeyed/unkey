import { tryCatch } from "@/lib/utils/try-catch";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, metadata, task } from "@trigger.dev/sdk/v3";
import GithubSlugger from "github-slugger";
import yaml from "js-yaml";
import type { FieldSelection } from "../generate/takeaways/generate-takeaways";

/**
 * Task that updates takeaways content in the glossary by creating a GitHub PR
 */
export const updateTakeawaysTask = task({
  id: "update_takeaways",
  retry: {
    maxAttempts: 3,
  },
  onStart: async (payload: {
    term: string;
    takeaways: Record<string, any>;
    fields?: FieldSelection;
  }) => {
    metadata.replace({
      term: payload.term,
      status: "running",
      startedAt: new Date().toISOString(),
      fields: payload.fields || "all",
      progress: 0,
    });
  },
  onSuccess: async () => {
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("progress", 1);
  },
  run: async (payload: {
    term: string;
    takeaways: Record<string, any>;
    fields?: FieldSelection;
  }) => {
    const { term, takeaways, fields } = payload;

    if (!term) {
      throw new AbortTaskRunError("Term is required");
    }

    if (!takeaways) {
      throw new AbortTaskRunError("Takeaways are required");
    }

    metadata.set("progress", 0.2);

    // Initialize GitHub client
    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN,
    });

    // Repository details
    const owner = "unkeyed";
    const repo = "unkey";

    // Create a slug from the input term
    const slugger = new GithubSlugger();
    const slug = slugger.slug(term);

    // File path for the glossary entry
    const filePath = `apps/www/content/glossary/${slug}.mdx`;

    // Check if the file exists in the repository
    const fileResult = await tryCatch(
      octokit.repos.getContent({
        owner,
        repo,
        path: filePath,
        ref: "main",
      }),
    );

    // If the file doesn't exist, abort the task
    if (fileResult.error) {
      throw new AbortTaskRunError(`File not found: ${filePath}. Cannot update non-existent file.`);
    }

    // Extract the file content and SHA
    const fileResponse = fileResult.data.data;
    if (!("content" in fileResponse) || !("sha" in fileResponse)) {
      throw new AbortTaskRunError("Invalid file data response");
    }

    // Decode the base64 content
    const base64Content = fileResponse.content as string;
    const currentContent = Buffer.from(base64Content, "base64").toString("utf-8");
    const fileSha = fileResponse.sha as string;

    // Extract frontmatter
    const frontmatterMatch = currentContent.match(/^---([\s\S]*?)\n---\n([\s\S]*)/);
    if (!frontmatterMatch) {
      throw new AbortTaskRunError("Failed to extract frontmatter from file");
    }

    const [, frontmatterContent, content] = frontmatterMatch;

    // Parse the YAML frontmatter
    let parsedFrontmatter: Record<string, any>;
    try {
      parsedFrontmatter = yaml.load(frontmatterContent) as Record<string, any>;
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      throw new AbortTaskRunError(`Failed to parse frontmatter: ${errorMessage}`);
    }

    // Update takeaways in the frontmatter
    const updatedFields: Record<string, any> = {};
    if (fields) {
      // Only update specified fields
      parsedFrontmatter.takeaways = parsedFrontmatter.takeaways || {};
      Object.entries(fields).forEach(([field, value]) => {
        if (value && takeaways[field]) {
          parsedFrontmatter.takeaways[field] = takeaways[field];
          updatedFields[field] = takeaways[field];
        }
      });
    } else {
      // Update all fields
      parsedFrontmatter.takeaways = takeaways;
      Object.keys(takeaways).forEach((field) => {
        updatedFields[field] = takeaways[field];
      });
    }

    // Convert the updated frontmatter back to YAML
    const updatedFrontmatter = yaml.dump(parsedFrontmatter, {
      lineWidth: -1,
      noRefs: true,
      quotingType: '"',
    });

    // Combine the updated frontmatter with the original content
    const updatedContent = `---\n${updatedFrontmatter}---\n${content}`;

    // Create a unique branch name for this update
    const timestamp = Date.now();
    const branch = `update-glossary-${slug}-${timestamp}`;

    // Get the main branch reference
    const mainRefResult = await tryCatch(
      octokit.git.getRef({
        owner,
        repo,
        ref: "heads/main",
      }),
    );

    if (mainRefResult.error) {
      throw new AbortTaskRunError(
        `Failed to get main branch reference: ${mainRefResult.error.message}`,
      );
    }

    // Create a new branch
    const createBranchResult = await tryCatch(
      octokit.git.createRef({
        owner,
        repo,
        ref: `refs/heads/${branch}`,
        sha: mainRefResult.data.data.object.sha,
      }),
    );

    if (createBranchResult.error) {
      throw new AbortTaskRunError(`Failed to create branch: ${createBranchResult.error.message}`);
    }

    // Update the file in the new branch
    const updateFileResult = await tryCatch(
      octokit.repos.createOrUpdateFileContents({
        owner,
        repo,
        path: filePath,
        message: `feat(glossary): Update ${term} takeaways`,
        content: Buffer.from(updatedContent).toString("base64"),
        branch,
        sha: fileSha,
      }),
    );

    if (updateFileResult.error) {
      throw new AbortTaskRunError(`Failed to update file: ${updateFileResult.error.message}`);
    }

    // Create a pull request
    const prResult = await tryCatch(
      octokit.pulls.create({
        owner,
        repo,
        title: `Update ${term} takeaways`,
        head: branch,
        base: "main",
        body: `This PR updates the following takeaways for ${term}:\n\n${Object.keys(updatedFields)
          .map((field) => `- ${field}`)
          .join("\n")}`,
      }),
    );

    if (prResult.error) {
      throw new AbortTaskRunError(`Failed to create PR: ${prResult.error.message}`);
    }

    // Get the diff information
    const diffResult = await tryCatch(
      octokit.pulls.get({
        owner,
        repo,
        pull_number: prResult.data.data.number,
        mediaType: {
          format: "diff",
        },
      }),
    );

    // Return the result with PR information and updated fields
    return {
      inputTerm: term,
      updated: true,
      prUrl: prResult.data.data.html_url,
      branch,
      updatedFields,
      diff: diffResult.data,
    };
  },
});
