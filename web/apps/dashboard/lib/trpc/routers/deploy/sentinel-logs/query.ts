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
  .output(z.array(sentinelLogsResponseSchema))
  .query(async ({ ctx, input }) => {
    try {
      const project = await db.query.projects.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        columns: { id: true },
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found or access denied",
        });
      }

      const transformedInputs = transformSentinelLogsFilters(input);

      const result = await clickhouse.sentinel.logs({
        workspaceId: ctx.workspace.id,
        ...transformedInputs,
      });

      if (result.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch sentinel logs from ClickHouse.",
        });
      }

      return result.val;
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
        message: "Failed to retrieve sentinel logs. If this persists, contact support@unkey.dev.",
      });
    }
  });
