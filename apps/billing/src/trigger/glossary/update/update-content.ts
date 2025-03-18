import { tryCatch } from "@/lib/utils/try-catch";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import GithubSlugger from "github-slugger";

/**
 * Task that updates glossary content on the website by creating a GitHub PR
 */
export const updateGlossaryContentTask = task({
  id: "update_glossary_content",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    inputTerm,
    content,
  }: {
    inputTerm: string;
    content: string;
  }) => {
    // Validate inputs
    if (!inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    if (!content) {
      throw new AbortTaskRunError("Content is required");
    }

    // Initialize GitHub client
    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN,
    });

    // Repository details
    const owner = "unkeyed";
    const repo = "unkey";

    // Create a slug from the input term
    const slugger = new GithubSlugger();
    const slug = slugger.slug(inputTerm);

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

    // Extract the file content and SHA using native type checking
    const fileResponse = fileResult.data;

    // Ensure we have a valid response object
    if (typeof fileResponse !== "object" || fileResponse === null) {
      throw new AbortTaskRunError("Invalid file data response");
    }

    // The response can be an array or a single file object
    // We need to ensure we're working with a single file
    if (Array.isArray(fileResponse.data)) {
      throw new AbortTaskRunError("Expected a single file but got a directory listing");
    }

    // Now we know it's a single file object
    const fileData = fileResponse.data;

    // Validate required properties
    if (typeof fileData !== "object" || fileData === null) {
      throw new AbortTaskRunError("File data is missing or invalid");
    }

    // Ensure it's a file type (not a directory, symlink, or submodule)
    if (fileData.type !== "file") {
      throw new AbortTaskRunError(`Expected a file but got ${fileData.type}`);
    }

    // Now TypeScript knows this is a file with content and sha
    if (!fileData.content || typeof fileData.content !== "string") {
      throw new AbortTaskRunError("File content is missing or invalid");
    }

    if (!fileData.sha || typeof fileData.sha !== "string") {
      throw new AbortTaskRunError("File SHA is missing or invalid");
    }

    // Decode the base64 content
    const base64Content = fileData.content;
    const currentContent = Buffer.from(base64Content, "base64").toString("utf-8");
    const fileSha = fileData.sha;

    // Extract frontmatter from the existing file
    const frontmatterMatch = currentContent.match(/^---\n([\s\S]*?)\n---\n/);
    if (!frontmatterMatch || !frontmatterMatch[0]) {
      throw new AbortTaskRunError("Failed to extract frontmatter from file");
    }

    // Preserve the frontmatter and replace the content
    const updatedContent = `${frontmatterMatch[0]}${content}`;

    // Create a unique branch name for this update
    const timestamp = Date.now();
    const branch = `update-glossary-${slug}-${timestamp}`;

    // Get the main branch reference to create our new branch
    // This is needed to get the SHA of the latest commit on main
    // so we can create a new branch from this point
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

    // Extract the SHA from the main branch reference using native type checking
    const mainRefResponse = mainRefResult.data;

    // Ensure we have a valid response object
    if (typeof mainRefResponse !== "object" || mainRefResponse === null) {
      throw new AbortTaskRunError("Invalid main branch reference data");
    }

    const mainRefData = mainRefResponse.data;

    // Validate required properties
    if (typeof mainRefData !== "object" || mainRefData === null) {
      throw new AbortTaskRunError("Main branch reference data is missing or invalid");
    }

    if (typeof mainRefData.object !== "object" || mainRefData.object === null) {
      throw new AbortTaskRunError("Main branch reference object is missing or invalid");
    }

    if (!mainRefData.object.sha || typeof mainRefData.object.sha !== "string") {
      throw new AbortTaskRunError("Main branch SHA is missing or invalid");
    }

    const mainRefSha = mainRefData.object.sha;

    // Create a new branch for this update
    const createBranchResult = await tryCatch(
      octokit.git.createRef({
        owner,
        repo,
        ref: `refs/heads/${branch}`,
        sha: mainRefSha,
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
        message: `feat(glossary): Update ${inputTerm} in glossary`,
        content: Buffer.from(updatedContent).toString("base64"),
        branch,
        sha: fileSha,
      }),
    );

    if (updateFileResult.error) {
      throw new AbortTaskRunError(`Failed to update file: ${updateFileResult.error.message}`);
    }

    // Create a pull request for the changes
    const prResult = await tryCatch(
      octokit.pulls.create({
        owner,
        repo,
        title: `Update ${inputTerm} in Glossary`,
        head: branch,
        base: "main",
        body: `This PR updates the ${inputTerm} entry in the glossary.`,
      }),
    );

    if (prResult.error) {
      throw new AbortTaskRunError(`Failed to create PR: ${prResult.error.message}`);
    }

    // Extract the PR URL using native type checking
    const prResponse = prResult.data;

    // Ensure we have a valid response object
    if (typeof prResponse !== "object" || prResponse === null) {
      throw new AbortTaskRunError("Invalid PR data response");
    }

    const prData = prResponse.data;

    // Validate required properties
    if (typeof prData !== "object" || prData === null) {
      throw new AbortTaskRunError("PR data is missing or invalid");
    }

    if (!prData.html_url || typeof prData.html_url !== "string") {
      throw new AbortTaskRunError("PR URL is missing or invalid");
    }

    const prUrl = prData.html_url;

    // Return the result
    return {
      inputTerm,
      updated: true,
      prUrl,
      branch,
    };
  },
});
