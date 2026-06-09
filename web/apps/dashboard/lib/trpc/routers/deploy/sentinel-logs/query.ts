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
  .input(
    sentinelLogsRequestSchema
      .omit({ workspaceId: true })
      .extend({ appId: z.string().nullable().default(null) }),
  )
  .output(
    z.object({
      logs: z.array(sentinelLogsResponseSchema),
      total: z.number().int(),
      hasMore: z.boolean(),
      nextCursor: z.number().int().nullable(),
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
            columns: { id: true, appId: true, slug: true },
          },
        },
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found or access denied",
        });
      }

      if (project.environments.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "No environment found for this project",
        });
      }

      // No environment filter = project-wide: every app and environment.
      // Environments are per-app, so forcing a single "production" env here
      // would silently scope the view to one arbitrary app.
      const transformedInputs = transformSentinelLogsFilters(input);

      // App-scoped view: the table has no app_id column, so scope through the
      // app's environments instead. Default to production, matching the
      // runtime logs router.
      const appId = input.appId || null;
      if (appId && transformedInputs.environmentId.length === 0) {
        const appEnvironments = project.environments.filter((e) => e.appId === appId);
        const scoped = appEnvironments.find((e) => e.slug === "production") ?? appEnvironments[0];
        if (scoped) {
          transformedInputs.environmentId = [scoped.id];
        }
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
        hasMore: logs.length === input.limit,
        nextCursor: logs.length > 0 ? logs[logs.length - 1].time : null,
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
