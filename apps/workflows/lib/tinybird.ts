import { Tinybird as Client } from "@chronark/zod-bird";
import { newId } from "@unkey/id";
import { auditLogSchemaV1, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";

type MaybeArray<T> = T | T[];

export class Tinybird {
  private readonly tb: Client;

  constructor(token: string) {
    this.tb = new Client({ token });
  }

  public get activeKeys() {
    return this.tb.buildPipe({
      pipe: "endpoint__active_keys_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),
      data: z.object({
        keys: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
  public get verifications() {
    return this.tb.buildPipe({
      pipe: "endpoint__verifications_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),

      data: z.object({
        success: z.number().int().nullable().default(0),
        ratelimited: z.number().int().nullable().default(0),
        usageExceeded: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }
  public get ratelimits() {
    return this.tb.buildPipe({
      pipe: "endpoint__ratelimits_by_workspace__v1",
      parameters: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),

      data: z.object({
        success: z.number().int().nullable().default(0),
        total: z.number().int().nullable().default(0),
      }),
      opts: {
        cache: "no-store",
      },
    });
  }

  public ingestAuditLogs(
    logs: MaybeArray<{
      workspaceId: string;
      event: z.infer<typeof unkeyAuditLogEvents>;
      description: string;
      actor: {
        type: "user" | "key" | "system";
        name?: string;
        id: string;
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
          | "vercelIntegration";
        id: string;
        meta?: Record<string, string | number | boolean | null>;
      }>;
      context: {
        userAgent?: string;
        location: string;
      };
    }>,
  ) {
    return this.tb.buildIngestEndpoint({
      datasource: "audit_logs__v2",
      event: auditLogSchemaV1
        .merge(
          z.object({
            event: unkeyAuditLogEvents,
            auditLogId: z.string().default(newId("auditLog")),
            bucket: z.string().default("unkey_mutations"),
            time: z.number().default(Date.now()),
          }),
        )
        .transform((l) => ({
          ...l,
          actor: {
            ...l.actor,
            meta: l.actor.meta ? JSON.stringify(l.actor.meta) : undefined,
          },
          resources: JSON.stringify(l.resources),
        })),
    })(logs);
  }
}
