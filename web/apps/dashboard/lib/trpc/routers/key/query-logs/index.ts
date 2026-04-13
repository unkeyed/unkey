import { keyDetailsLogsPayload } from "@/components/key-details-logs-table/schema/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { z } from "zod";
import { transformKeyDetailsFilters } from "./utils";

const keyDetailsLogsResponse = z.object({
  logs: z.array(z.custom<KeyDetailsLog>()),
  total: z.number(),
});
type KeyDetailsLogsResponse = z.infer<typeof keyDetailsLogsResponse>;

export const queryKeyDetailsLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keyDetailsLogsPayload)
  .output(keyDetailsLogsResponse)
  .query(async ({ ctx, input }) => {
    // Verify the key belongs to this workspace
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.id, input.keyId),
            eq(table.keyAuthId, input.keyspaceId),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        columns: { id: true },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve key details due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
        });
      });

    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Key not found in the specified workspace.",
      });
    }

    const transformedFilters = transformKeyDetailsFilters(input, ctx.workspace.id);
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
    };

    return response;
  });
