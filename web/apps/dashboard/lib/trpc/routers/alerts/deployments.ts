import { and, db, eq, gte, lt, schema } from "@/lib/db";
import { shortenId } from "@/lib/shorten-id";
import { workspaceProcedure } from "../../trpc";
import { alertDeploymentsInput } from "./schemas";

export const listAlertDeployments = workspaceProcedure
  .input(alertDeploymentsInput)
  .query(async ({ ctx, input }) => {
    const rows = await db
      .select({
        id: schema.deployments.id,
        createdAt: schema.deployments.createdAt,
        gitSha: schema.deployments.gitCommitSha,
      })
      .from(schema.deployments)
      .where(
        and(
          eq(schema.deployments.workspaceId, ctx.workspace.id),
          eq(schema.deployments.appId, input.appId),
          eq(schema.deployments.environmentId, input.environmentId),
          gte(schema.deployments.createdAt, input.startMs),
          lt(schema.deployments.createdAt, input.endMs),
        ),
      )
      .orderBy(schema.deployments.createdAt);

    return rows.map((deployment) => ({
      ...deployment,
      label: deployment.gitSha?.slice(0, 7) ?? shortenId(deployment.id),
    }));
  });
