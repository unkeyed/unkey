import { identityLogsPayload } from "./query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { z } from "zod";
import { transformIdentityLogsFilters } from "./utils";

// Extended log type that includes key identification
export const identityLog = z.object({
  request_id: z.string(),
  time: z.number().int(),
  region: z.string(),
  outcome: z.enum([
    "VALID",
    "RATE_LIMITED",
    "EXPIRED",
    "DISABLED",
    "FORBIDDEN",
    "USAGE_EXCEEDED",
    "INSUFFICIENT_PERMISSIONS",
    "", // Empty string is a valid outcome in ClickHouse
  ]),
  tags: z.array(z.string()),
  keyId: z.string(),
  keyName: z.string().nullable(),
  apiId: z.string(),
  apiName: z.string(),
});

export type IdentityLog = z.infer<typeof identityLog>;

const identityLogsResponse = z.object({
  logs: z.array(identityLog),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().optional(),
});

type IdentityLogsResponse = z.infer<typeof identityLogsResponse>;

export type { IdentityLogsResponse };

export const queryIdentityLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identityLogsPayload)
  .output(identityLogsResponse)
  .query(async ({ ctx, input }) => {
    // First, validate identity exists and get associated keys
    const identity = await db.query.identities
      .findFirst({
        where: (table, { eq }) => eq(table.id, input.identityId),
        with: {
          workspace: {
            columns: { id: true, orgId: true },
          },
          keys: {
            where: (keysTable, { isNull }) => isNull(keysTable.deletedAtM),
            with: {
              keyAuth: {
                with: {
                  api: {
                    columns: { id: true, name: true },
                  },
                },
              },
            },
            columns: {
              id: true,
              name: true,
              keyAuthId: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve identity details due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!identity) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Identity not found, please contact support using support@unkey.dev.",
      });
    }

    if (identity.workspace.orgId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Identity not found in the specified workspace.",
      });
    }

    if (!identity.keys || identity.keys.length === 0) {
      // Return empty response if identity has no keys
      return {
        logs: [],
        total: 0,
        hasMore: false,
        nextCursor: undefined,
      };
    }

    // Transform filters and add keyIds from identity
    const keyIds = identity.keys.map((key) => key.id);
    const transformedFilters = transformIdentityLogsFilters(input, identity.workspace.id, keyIds);

    // Query ClickHouse for aggregated logs
    const result = await clickhouse.api.identity.logs(transformedFilters);

    if (result.logs.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching identity logs data from ClickHouse.",
      });
    }

    const logs = result.logs.val || [];

    // Enrich logs with key and API information
    const enrichedLogs: IdentityLog[] = logs.map((log) => {
      const key = identity.keys.find((k) => k.id === log.keyId);
      return {
        ...log,
        keyId: log.keyId,
        keyName: key?.name || null,
        apiId: key?.keyAuth.api.id || "",
        apiName: key?.keyAuth.api.name || "",
      };
    });

    const response: IdentityLogsResponse = {
      logs: enrichedLogs,
      total: result.totalCount,
      hasMore: enrichedLogs.length === input.limit,
      nextCursor: enrichedLogs.length === input.limit ? enrichedLogs[enrichedLogs.length - 1]?.time : undefined,
    };

    return response;
  });