import { auditLogsQueryPayload } from "@/components/audit-logs-table/schema/audit-logs.schema";
import { fetchUsersByActorIds } from "@/lib/auth/cached-user";
import { clickhouse } from "@/lib/clickhouse";
import type { Limits, Workspace } from "@/lib/db";
import { freeTierLimits } from "@/lib/limits";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { type AuditQueryLogsParams, auditLog } from "./schema";
import { transformFilters } from "./utils";

const LIMIT = 50;
const MAX_LIMIT = 200;

const AuditLogsResponse = z.object({
  auditLogs: z.array(auditLog),
  total: z.number().nullable(),
});

type AuditLogsResponse = z.infer<typeof AuditLogsResponse>;

function nullIfEmpty(s: string): string | null {
  return s.length === 0 ? null : s;
}

function parseJSON(s: string): unknown {
  if (s.length === 0) {
    return null;
  }
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}

export const fetchAuditLog = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(auditLogsQueryPayload)
  .output(AuditLogsResponse)
  .query(async ({ ctx, input }) => {
    const params = transformFilters(input);
    const pageSize = Math.max(1, Math.min(params.limit ?? LIMIT, MAX_LIMIT));
    const page = Math.max(1, params.page ?? 1);
    const offset = (page - 1) * pageSize;

    const queryArgs = buildQueryArgs(params, ctx.workspace, pageSize, offset);
    const logsResult = await clickhouse.auditLogs.logs(queryArgs).getLogsQuery();

    if (logsResult.err) {
      console.error("Clickhouse audit logs query failed", {
        error: logsResult.err.message,
        workspaceId: ctx.workspace.id,
        startTime: queryArgs.startTime,
        endTime: queryArgs.endTime,
      });
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching audit logs from clickhouse.",
        cause: logsResult.err,
      });
    }

    const logs = logsResult.val;
    const uniqueUsers = await fetchUsersByActorIds(
      logs.filter((l) => l.actorType === "user").map((l) => l.actorId),
    );

    const items: AuditLogsResponse["auditLogs"] = logs.map((l) => {
      const user = uniqueUsers[l.actorId];
      return {
        user: user
          ? {
              username: user.email,
              firstName: user.firstName,
              lastName: user.lastName,
              imageUrl: user.avatarUrl,
            }
          : undefined,
        auditLog: {
          id: l.eventId,
          time: l.time,
          actor: {
            id: l.actorId,
            name: nullIfEmpty(l.actorName),
            type: l.actorType,
          },
          location: nullIfEmpty(l.remoteIp),
          description: l.description,
          userAgent: nullIfEmpty(l.userAgent),
          event: l.event,
          workspaceId: ctx.workspace.id,
          targets: l.targets.map(([type, id, name, meta]) => ({
            id,
            type,
            name: nullIfEmpty(name),
            meta: parseJSON(meta),
          })),
        },
      };
    });

    return {
      auditLogs: items,
      total: null,
    };
  });

function buildQueryArgs(
  params: Omit<AuditQueryLogsParams, "workspaceId">,
  workspace: Workspace & { limits: Limits | null },
  limit: number,
  offset: number,
) {
  const events = (params.events ?? []).map((e) => e.value);
  const userValues = (params.users ?? []).map((u) => u.value);
  const rootKeyValues = (params.rootKeys ?? []).map((r) => r.value);
  const actorIds = [...userValues, ...rootKeyValues];

  const retentionDays =
    workspace.limits?.logsAuditRetentionDaysMax || freeTierLimits.logsAuditRetentionDaysMax;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  const startTime = Math.max(
    params.startTime ?? retentionCutoffUnixMilli,
    retentionCutoffUnixMilli,
  );
  const endTime = params.endTime ?? Date.now();

  return {
    workspaceId: workspace.id,
    buckets: [params.bucket],
    limit,
    offset,
    startTime,
    endTime,
    events,
    actorIds,
  };
}
