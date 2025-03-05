import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { transformKeysFilters } from "./utils";
import { keysOverviewLogs as keysLogs } from "@unkey/clickhouse/src/keys/keys";

export const keysQueryOverviewLogsPayload = z.object({
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  apiId: z.string(),
  since: z.string(),
  cursor: z
    .object({
      requestId: z.string().nullable(),
      time: z.number().nullable(),
    })
    .optional()
    .nullable(),
  outcomes: z
    .array(
      z.object({
        value: z.enum([
          "VALID",
          "INSUFFICIENT_PERMISSIONS",
          "RATE_LIMITED",
          "FORBIDDEN",
          "DISABLED",
          "EXPIRED",
          "USAGE_EXCEEDED",
          "",
        ]),
        operator: z.literal("is"),
      })
    )
    .optional()
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      })
    )
    .optional()
    .nullable(),
  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "valid_count", "error_count"]),
        direction: z.enum(["asc", "desc"]),
      })
    )
    .optional()
    .nullable(),
});

export type KeysQueryOverviewLogsPayload = z.infer<
  typeof keysQueryOverviewLogsPayload
>;

const KeysOverviewLogsResponse = z.object({
  keysOverviewLogs: z.array(keysLogs),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type KeysOverviewLogsResponse = z.infer<typeof KeysOverviewLogsResponse>;

export const queryKeysOverviewLogs = rateLimitedProcedure(ratelimit.read)
  .input(keysQueryOverviewLogsPayload)
  .output(KeysOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    // First, verify the API exists and belongs to the current workspace
    const apiResult = await db.query.apis
      .findFirst({
        where: (api, { and, eq, isNull }) =>
          and(
            eq(api.id, input.apiId),
            eq(api.workspaceId, ctx.workspace.id),
            isNull(api.deletedAtM)
          ),
        with: {
          keyAuth: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve API information. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!apiResult || !apiResult?.keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or does not have key authentication enabled",
      });
    }

    // Prepare query conditions for keys
    // const keyIdValues = input.keyIds?.map((k) => k.value) || [];

    // Pre-fetch all keys for this API that match our filters
    const keysResult = await db.query.keys
      .findMany({
        where: (key, { and, eq, isNull }) => {
          const conditions = [
            eq(key.keyAuthId, apiResult.keyAuth?.id ?? ""),
            isNull(key.deletedAtM),
          ];
          //
          // // Add key ID filter if provided
          // if (keyIdValues.length > 0) {
          //   conditions.push(
          //     or(
          //       ...(keyIdValues ?? []).map((keyId) => {
          //         const matchingFilter = input.keyIds?.find(
          //           (k) => k.value === keyId
          //         );
          //         if (matchingFilter?.operator === "contains") {
          //           return (expr) => expr`${key.id} LIKE ${`%${keyId}%`}`;
          //         }
          //         return eq(key.id, keyId);
          //       })
          //     )
          //   );
          // }

          return and(...conditions);
        },
        columns: {
          id: true,
          keyAuthId: true,
          name: true,
          ownerId: true,
          identityId: true,
          meta: true,
          enabled: true,
          remaining: true,
          ratelimitAsync: true,
          ratelimitLimit: true,
          ratelimitDuration: true,
          environment: true,
          refillDay: true,
          refillAmount: true,
          lastRefillAt: true,
          expires: true,
          workspaceId: true,
        },
      })
      .catch((err) => {
        console.error("Error fetching keys from database:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve keys data. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    const keyDetailsMap = new Map();
    for (const key of keysResult) {
      const keyDetails = {
        id: key.id,
        key_auth_id: key.keyAuthId,
        name: key.name,
        owner_id: key.ownerId,
        identity_id: key.identityId,
        meta: key.meta,
        enabled: key.enabled,
        remaining_requests: key.remaining,
        ratelimit_async: key.ratelimitAsync,
        ratelimit_limit: key.ratelimitLimit,
        ratelimit_duration: key.ratelimitDuration,
        environment: key.environment,
        refill_day: key.refillDay,
        refill_amount: key.refillAmount,
        last_refill_at: key.lastRefillAt,
        expires: key.expires,
        workspace_id: key.workspaceId,
      };

      keyDetailsMap.set(key.id, keyDetails);
    }

    const transformedInputs = transformKeysFilters(input);

    const result = await clickhouse.api.keys.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: ctx.workspace.id,
      keyspaceId: apiResult.keyAuth.id,
    });

    if (!result || result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = result.val || [];

    if (logs.length === 0) {
      return {
        keysOverviewLogs: [],
        hasMore: false,
      };
    }

    // Join the keys data with the logs
    const keysOverviewLogs = logs.map((log) => ({
      ...log,
      key_details: keyDetailsMap.get(log.key_id) || null,
    }));

    const response: KeysOverviewLogsResponse = {
      keysOverviewLogs,
      hasMore: logs.length === input.limit,
      nextCursor:
        logs.length === input.limit
          ? {
              time: logs[logs.length - 1].time,
              requestId: logs[logs.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });
