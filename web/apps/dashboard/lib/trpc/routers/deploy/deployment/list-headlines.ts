import { eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { deployments } from "@unkey/db/src/schema";
import { z } from "zod";
import { queryHeadlineDeployments } from "../headline-deployments";

export const listDeploymentHeadlines = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    const rows = await queryHeadlineDeployments(
      ctx.workspace.id,
      eq(deployments.projectId, input.projectId),
    );
    return rows.map((row) => ({
      appId: row.appId,
      id: row.id,
      status: row.status,
      deployedAt: Number(row.createdAt),
      commitMessage: row.gitCommitMessage ?? null,
      commitSha: row.gitCommitSha ?? null,
      branch: row.gitBranch ?? null,
      prNumber: row.prNumber ?? null,
      forkRepositoryFullName: row.forkRepositoryFullName ?? null,
    }));
  });
