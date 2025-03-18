import { tryCatch } from "@/lib/utils/try-catch";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";

/**
 * Task that cleans up GitHub PRs and branches created by the update-content task
 * This is used to clean up after tests to avoid leaving open PRs and branches
 */
export const cleanupGlossaryUpdateTask = task({
  id: "cleanup_glossary_update",
  retry: {
    maxAttempts: 3,
  },
  run: async ({
    prNumber,
    branch,
  }: {
    prNumber?: string | number;
    branch?: string;
  }) => {
    // Validate that we have at least one of prNumber or branch
    if (!prNumber && !branch) {
      throw new AbortTaskRunError("Either prNumber or branch must be provided");
    }

    // Initialize GitHub client
    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN,
    });

    // Repository details
    const owner = "unkeyed";
    const repo = "unkey";

    const results = {
      prClosed: false,
      branchDeleted: false,
      prNumber,
      branch,
    };

    // If PR number is provided, close the PR
    if (prNumber) {
      // Extract PR number from URL if a full URL was provided
      const extractedPrNumber =
        typeof prNumber === "string" && prNumber.includes("github.com")
          ? prNumber.split("/").pop()
          : prNumber;

      if (!extractedPrNumber) {
        throw new AbortTaskRunError("Could not extract PR number from URL");
      }

      console.info(`Closing PR #${extractedPrNumber}`);

      const closePrResult = await tryCatch(
        octokit.pulls.update({
          owner,
          repo,
          pull_number: Number(extractedPrNumber),
          state: "closed",
        }),
      );

      if (closePrResult.error) {
        console.error(`Failed to close PR: ${closePrResult.error.message}`);
      } else {
        console.info(`Successfully closed PR #${extractedPrNumber}`);
        results.prClosed = true;
      }
    }

    // If branch name is provided, delete the branch
    if (branch) {
      console.info(`Deleting branch: ${branch}`);

      const deleteBranchResult = await tryCatch(
        octokit.git.deleteRef({
          owner,
          repo,
          ref: `heads/${branch}`,
        }),
      );

      if (deleteBranchResult.error) {
        console.error(`Failed to delete branch: ${deleteBranchResult.error.message}`);
      } else {
        console.info(`Successfully deleted branch: ${branch}`);
        results.branchDeleted = true;
      }
    }

    return results;
  },
});
