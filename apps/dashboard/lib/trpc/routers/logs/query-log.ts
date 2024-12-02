import { getLogs } from "@/lib/clickhouse/logs";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const queryLogs = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      workspaceId: z.string(),
      limit: z.number().int(),
      startTime: z.number().int(),
      endTime: z.number().int(),
      path: z.string().optional().nullable(),
      host: z.string().optional().nullable(),
      requestId: z.string().optional().nullable(),
      method: z.string().optional().nullable(),
      response_status: z.number().int().nullable(),
    })
  )
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          //TODO: change error message later
          message:
            "We are unable opt you in to this beta feature. Please contact support using support@unkey.dev",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Workspace not found, please contact support using support@unkey.dev.",
      });
    }
    const result = await getLogs(input);
    if (result.err) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    return result.val;
  });
