import type { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import type { z } from "zod";

import type { MaybeArray } from "@/lib/types";
import { type Database, type Transaction, schema } from "@unkey/db";
import type { auditLog, auditLogTarget, clickhouseOutbox } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";

export const AUDIT_LOG_BUCKET = "unkey_mutations";

// OUTBOX_VERSION_V1 must match the Go writer's
// pkg/auditlog.OutboxVersionV1 constant — the drainer
// (svc/ctrl/worker/auditlogexport) only consumes payloads whose version
// is in its known list. Bump only when the JSON envelope shape below
// changes in a way an older drainer can't decode.
const OUTBOX_VERSION_V1 = "audit_log.v1";

// EVENT_SOURCE_PLATFORM matches pkg/auditlog.EventSourcePlatform. The
// dashboard only ever emits platform events (Unkey acting on the
// customer's resources). Customer-emitted events would set "customer".
const EVENT_SOURCE_PLATFORM = "platform";

export type UnkeyAuditLog = {
  workspaceId: string;
  event: z.infer<typeof unkeyAuditLogEvents>;
  description: string;
  actor: {
    type: "user" | "key" | "system";
    name?: string;
    id: string;
    meta?: Record<string, string | number | boolean | null>;
  };
  resources: Array<{
    type:
      | "key"
      | "api"
      | "workspace"
      | "role"
      | "permission"
      | "keyAuth"
      | "vercelBinding"
      | "vercelIntegration"
      | "ratelimitNamespace"
      | "ratelimitOverride"
      | "ratelimit"
      | "sentinel"
      | "llmSentinel"
      | "webhook"
      | "reporter"
      | "secret"
      | "project"
      | "identity"
      | "auditLogBucket"
      | "environment"
      | "deployment";

    id: string;
    name?: string;
    meta?: Record<string, string | number | boolean | null | undefined>;
  }>;
  context: {
    userAgent?: string;
    location: string;
  };
  // correlationId groups this event with other audit logs emitted by the
  // same logical user action. Leave undefined to let insertAuditLogs
  // decide: it auto-mints one when the batch contains >1 events.
  // Multi-call procedures (createRole + N permission binds across
  // separate insertAuditLogs calls) should mint via newId("correlation")
  // at the top of the procedure and pass the same value to each call.
  correlationId?: string;
};

// outboxEventEnvelope mirrors pkg/auditlog.Event exactly. The Go drainer
// JSON-decodes into that struct, so field names and shapes must match.
// Additive changes are safe (json.Unmarshal ignores unknown fields);
// renames or type changes need a new OUTBOX_VERSION_V1.
type OutboxEventEnvelope = {
  event_id: string;
  time: number;
  workspace_id: string;
  bucket: string;
  source: string;
  event: string;
  description: string;
  actor: {
    type: string;
    id: string;
    name?: string;
    meta?: Record<string, unknown>;
  };
  remote_ip?: string;
  user_agent?: string;
  meta?: Record<string, unknown>;
  targets?: Array<{
    type: string;
    id: string;
    name?: string;
    meta?: Record<string, unknown>;
  }>;
  correlation_id?: string;
};

export async function insertAuditLogs(
  db: Transaction | Database,
  logOrLogs: MaybeArray<UnkeyAuditLog>,
) {
  const logs = Array.isArray(logOrLogs) ? logOrLogs : [logOrLogs];

  if (logs.length === 0) {
    return Promise.resolve();
  }

  const auditLogs: (typeof auditLog.$inferInsert)[] = [];
  const auditLogTargets: (typeof auditLogTarget.$inferInsert)[] = [];
  // Outbox rows are inserted in the same transaction as the legacy
  // audit_log + audit_log_target rows so the durability story is
  // identical: if the underlying mutation commits, the outbox row
  // commits, and the drainer ships it to ClickHouse on the next tick.
  // The Go API writer dual-writes the same way; this keeps the dashboard
  // and API consistent producers of audit_log.v1 envelopes.
  const outboxRows: (typeof clickhouseOutbox.$inferInsert)[] = [];

  // Auto-mint a shared correlation ID when the batch carries >1 events.
  // Per-event correlationId set on the struct still wins over this
  // shared one (applied per-row below); the shared one fills in for
  // events that didn't bring their own. Matches the Go writer in
  // internal/services/auditlogs/insert.go — auto-mint on len>1 alone,
  // then per-event wins over shared.
  const sharedCorrelationId = logs.length > 1 ? newId("correlation") : undefined;

  for (const log of logs) {
    const auditLogId = newId("auditLog");
    const now = Date.now();

    auditLogs.push({
      id: auditLogId,
      workspaceId: log.workspaceId,
      bucketId: "DEPRECATED",
      bucket: AUDIT_LOG_BUCKET,
      event: log.event,
      time: now,
      display: log.description,
      remoteIp: log.context.location,
      userAgent: log.context.userAgent,
      actorType: log.actor.type,
      actorId: log.actor.id,
      actorName: log.actor.name,
      actorMeta: log.actor.meta,
    });

    if (log.resources.length > 0) {
      auditLogTargets.push(
        ...log.resources.map((r) => ({
          workspaceId: log.workspaceId,
          auditLogId,
          bucketId: "DEPRECATED",
          bucket: AUDIT_LOG_BUCKET,
          displayName: "",
          type: r.type,
          id: r.id,
          name: r.name,
          meta: r.meta,
        })),
      );
    }

    const correlationId = log.correlationId ?? sharedCorrelationId;

    const envelope: OutboxEventEnvelope = {
      event_id: auditLogId,
      time: now,
      workspace_id: log.workspaceId,
      bucket: AUDIT_LOG_BUCKET,
      source: EVENT_SOURCE_PLATFORM,
      event: log.event,
      description: log.description,
      actor: {
        type: log.actor.type,
        id: log.actor.id,
        ...(log.actor.name !== undefined ? { name: log.actor.name } : {}),
        ...(log.actor.meta !== undefined ? { meta: log.actor.meta } : {}),
      },
      ...(log.context.location !== undefined ? { remote_ip: log.context.location } : {}),
      ...(log.context.userAgent !== undefined ? { user_agent: log.context.userAgent } : {}),
      ...(log.resources.length > 0
        ? {
            targets: log.resources.map((r) => ({
              type: r.type,
              id: r.id,
              ...(r.name !== undefined ? { name: r.name } : {}),
              ...(r.meta !== undefined ? { meta: r.meta } : {}),
            })),
          }
        : {}),
      ...(correlationId !== undefined ? { correlation_id: correlationId } : {}),
    };

    outboxRows.push({
      version: OUTBOX_VERSION_V1,
      workspaceId: log.workspaceId,
      eventId: auditLogId,
      payload: envelope,
      createdAt: now,
    });
  }

  await db.insert(schema.auditLog).values(auditLogs);

  if (auditLogTargets.length > 0) {
    await db.insert(schema.auditLogTarget).values(auditLogTargets);
  }

  await db.insert(schema.clickhouseOutbox).values(outboxRows);
}
