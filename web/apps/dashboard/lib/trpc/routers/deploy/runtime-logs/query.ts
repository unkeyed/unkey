import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import {
  type RuntimeLogsResponseSchema,
  runtimeLogsRequestSchema,
  runtimeLogsResponseSchema,
} from "@/lib/schemas/runtime-logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { transformFilters } from "./utils";

export const queryRuntimeLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(runtimeLogsRequestSchema)
  .output(runtimeLogsResponseSchema)
  .query(async ({ ctx, input }) => {
    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { id: true },
      with: {
        environments: {
          columns: { id: true, appId: true },
        },
      },
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found or access denied",
      });
    }

    const environment = input.environmentId
      ? project.environments.find((e) => e.id === input.environmentId)
      : project.environments[0];

    if (!environment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No environment found for this project",
      });
    }

    const transformedInputs = transformFilters(input);
    const { logsQuery, totalQuery } = await clickhouse.runtimeLogs.logs({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      projectId: project.id,
      deploymentId: input.deploymentId ?? null,
      environmentId: environment.id,
      appId: environment.appId,
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
