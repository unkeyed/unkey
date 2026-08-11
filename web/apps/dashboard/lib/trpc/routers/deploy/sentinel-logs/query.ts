import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  sentinelLogsRequestSchema,
  sentinelLogsResponseSchema,
} from "@unkey/clickhouse/src/sentinel";
import { z } from "zod";
import { transformSentinelLogsFilters } from "./utils";

export const querySentinelLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(sentinelLogsRequestSchema.omit({ workspaceId: true }))
  .output(
    z.object({
      logs: z.array(sentinelLogsResponseSchema),
      total: z.number().int(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
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

      // If no app filter and no environment filter apply, the query reads all
      // apps and all environments of the project.
      const transformedInputs = transformSentinelLogsFilters(input);

      // `environment_id` comes before `time` in the sort key. If only `app_id`
      // applies, ClickHouse cannot use the time bound to skip granules. The
      // environments of an app contain all rows of that app, so this filter
      // keeps the same rows. If the apps have no environments, the array stays
      // empty and no environment filter applies.
      if (transformedInputs.appId.length > 0 && transformedInputs.environmentId.length === 0) {
        const selectedApps = new Set(transformedInputs.appId);
        transformedInputs.environmentId = project.environments
          .filter((environment) => selectedApps.has(environment.appId))
          .map((environment) => environment.id);
      }

      const { logsQuery, totalQuery } = await clickhouse.sentinel.logs({
        workspaceId: ctx.workspace.id,
        ...transformedInputs,
      });

      const [logsResult, totalResult] = await Promise.all([logsQuery, totalQuery]);

      if (logsResult.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch requests from ClickHouse.",
        });
      }

      if (totalResult.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch total count from ClickHouse.",
        });
      }

      const logs = logsResult.val;
      const total = totalResult.val[0]?.total_count ?? 0;

      return {
        logs,
        total,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      console.error({
        message: "Failed to fetch sentinel logs",
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        error: error instanceof Error ? error.message : String(error),
      });

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve requests. If this persists, contact support@unkey.com.",
      });
    }
  });
