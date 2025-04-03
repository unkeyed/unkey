import { tryCatch } from "@/lib/utils/try-catch";
import { Octokit } from "@octokit/rest";
import { AbortTaskRunError, metadata, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";

// Schema for cleanup metadata tracking
const CleanupMetadataSchema = z.object({
  prNumber: z.number(),
  branch: z.string(),
  status: z.enum(["running", "completed", "failed"]),
  startedAt: z.string(),
  completedAt: z.string().optional(),
  prClosed: z.boolean().optional(),
  branchDeleted: z.boolean().optional(),
  error: z.string().optional(),
});

type CleanupMetadata = z.infer<typeof CleanupMetadataSchema>;

export const updateTakeawaysCleanupTask = task({
  id: "update_takeaways_cleanup",
  retry: {
    maxAttempts: 3,
  },
  onStart: async (payload: { prUrl: string; branch: string }) => {
    const prNumber = Number.parseInt(payload.prUrl.split("/").pop() || "0", 10);
    if (!prNumber) {
      throw new AbortTaskRunError(`Could not extract PR number from URL: ${payload.prUrl}`);
    }

    const initialMetadata: CleanupMetadata = {
      prNumber,
      branch: payload.branch,
      status: "running",
      startedAt: new Date().toISOString(),
    };

    metadata.replace(initialMetadata);
  },
  onSuccess: async () => {
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
  },
  onFailure: async (_, error) => {
    metadata.set("status", "failed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("error", error instanceof Error ? error.message : String(error));
  },
  run: async (payload: { prUrl: string; branch: string }) => {
    const prNumber = Number.parseInt(payload.prUrl.split("/").pop() || "0", 10);
    if (!prNumber) {
      throw new AbortTaskRunError(`Could not extract PR number from URL: ${payload.prUrl}`);
    }

    const octokit = new Octokit({
      auth: process.env.GITHUB_PERSONAL_ACCESS_TOKEN,
    });

    // Close the PR
    const prResult = await tryCatch(
      octokit.pulls.update({
        owner: "unkeyed",
        repo: "unkey",
        pull_number: prNumber,
        state: "closed",
      }),
    );

    if (prResult.error) {
      throw new AbortTaskRunError(`Failed to close PR: ${prResult.error.message}`);
    }

    metadata.set("prClosed", true);

    // Delete the branch
    const branchResult = await tryCatch(
      octokit.git.deleteRef({
        owner: "unkeyed",
        repo: "unkey",
        ref: `heads/${payload.branch}`,
      }),
    );

    if (branchResult.error) {
      throw new AbortTaskRunError(`Failed to delete branch: ${branchResult.error.message}`);
    }

    metadata.set("branchDeleted", true);

    return {
      success: true,
      prNumber,
      branch: payload.branch,
      prClosed: true,
      branchDeleted: true,
    };
  },
});
