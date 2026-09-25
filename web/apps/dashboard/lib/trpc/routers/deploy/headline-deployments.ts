import { type SQL, and, db, eq, ne, sql } from "@/lib/db";
import { deployments, environments } from "@unkey/db/src/schema";

// Skipped deployments never ran, so they never headline.
export function queryHeadlineDeployments(workspaceId: string, scope: SQL | undefined) {
  const ranked = db
    .select({
      appId: deployments.appId,
      id: deployments.id,
      rn: sql<number>`ROW_NUMBER() OVER (PARTITION BY ${deployments.appId} ORDER BY (${environments.kind} = 'production') DESC, ${deployments.createdAt} DESC, ${deployments.id} DESC)`.as(
        "rn",
      ),
    })
    .from(deployments)
    .innerJoin(
      environments,
      and(
        eq(environments.id, deployments.environmentId),
        eq(environments.workspaceId, workspaceId),
      ),
    )
    .where(and(eq(deployments.workspaceId, workspaceId), ne(deployments.status, "skipped"), scope))
    .as("ranked_headline_deployments");

  return db
    .select({
      appId: ranked.appId,
      id: deployments.id,
      status: deployments.status,
      createdAt: deployments.createdAt,
      gitCommitMessage: deployments.gitCommitMessage,
      gitCommitSha: deployments.gitCommitSha,
      gitBranch: deployments.gitBranch,
      prNumber: deployments.prNumber,
      forkRepositoryFullName: deployments.forkRepositoryFullName,
    })
    .from(ranked)
    .innerJoin(
      deployments,
      and(eq(deployments.id, ranked.id), eq(deployments.workspaceId, workspaceId)),
    )
    .where(eq(ranked.rn, 1));
}
