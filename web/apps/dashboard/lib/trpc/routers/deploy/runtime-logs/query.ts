import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq } from "@/lib/db";
import {
  type RuntimeLogsResponseSchema,
  runtimeLogsRequestSchema,
  runtimeLogsResponseSchema,
} from "@/lib/schemas/runtime-logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { apps, deployments } from "@unkey/db/src/schema";
import { transformFilters } from "./utils";

export const queryRuntimeLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(runtimeLogsRequestSchema)
  .output(runtimeLogsResponseSchema)
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, ctx.workspace.id), isNull(table.deletedAtM)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve logs due to an error. If this issue persists, please contact support@unkey.com.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.com.",
      });
    }

    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, workspace.id)),
      columns: { id: true },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found or access denied",
      });
    }

    // Get the live deployment's environmentId from the default app
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.projectId, project.id), eq(apps.workspaceId, workspace.id)),
      columns: { liveDeploymentId: true },
    });

    let activeEnvironmentId: string | null = null;
    if (app?.liveDeploymentId) {
      const liveDeployment = await db.query.deployments.findFirst({
        where: and(
          eq(deployments.id, app.liveDeploymentId),
          eq(deployments.workspaceId, workspace.id),
        ),
        columns: { environmentId: true },
      });
      activeEnvironmentId = liveDeployment?.environmentId ?? null;
    }

    if (!activeEnvironmentId && !input.environmentId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No active deployment found for this project",
      });
    }

    const transformedInputs = transformFilters(input);
    const { logsQuery, totalQuery } = await clickhouse.runtimeLogs.logs({
      ...transformedInputs,
      workspaceId: workspace.id,
      projectId: project.id,
      deploymentId: input.deploymentId ?? null,
      environmentId: input.environmentId ?? activeEnvironmentId!,
    });

    const [countResult, logsResult] = await Promise.all([totalQuery, logsQuery]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = logsResult.val;

    const response: RuntimeLogsResponseSchema = {
      logs,
      hasMore: logs.length === input.limit,
      total: countResult.val[0].total_count,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
