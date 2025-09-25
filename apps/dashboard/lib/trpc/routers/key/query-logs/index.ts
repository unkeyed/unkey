import { keyDetailsLogsPayload } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db, isNull } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { z } from "zod";
import { transformKeyDetailsFilters } from "./utils";

const keyDetailsLogsResponse = z.object({
  logs: z.array(z.custom<KeyDetailsLog>()),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().optional(),
});
type KeyDetailsLogsResponse = z.infer<typeof keyDetailsLogsResponse>;

export const queryKeyDetailsLogs = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keyDetailsLogsPayload)
  .output(keyDetailsLogsResponse)
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          keys: {
            where: (keysTable, { eq, and }) =>
              and(
                eq(keysTable.id, input.keyId),
                eq(keysTable.keyAuthId, input.keyspaceId),
                isNull(keysTable.deletedAtM),
              ),
            limit: 1,
            columns: {
              id: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    if (!workspace.keys || workspace.keys.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Key not found in the specified workspace.",
      });
    }

    const transformedFilters = transformKeyDetailsFilters(input, workspace.id);
    const result = await clickhouse.api.key.logs(transformedFilters);

    if (result.logs.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching key details data from ClickHouse.",
      });
    }

    const logs = result.logs.val || [];

    const response: KeyDetailsLogsResponse = {
      logs,
      total: result.totalCount,
      hasMore: logs.length === input.limit,
      nextCursor: logs.length === input.limit ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
