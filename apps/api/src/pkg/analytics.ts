import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { ClickHouse } from "@unkey/clickhouse";
import { newId } from "@unkey/id";
import { auditLogSchemaV1, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { ratelimitSchemaV1 } from "@unkey/schema/src/ratelimit-tinybird";
import { z } from "zod";
import type { MaybeArray } from "./types/maybe";
// const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

/**
 * `t` has the format `2021-01-01 00:00:00`
 *
 * If we transform it as is, we get `1609459200000` which is `2021-01-01 01:00:00` due to fun timezone stuff.
 * So we split the string at the space and take the date part, and then parse that.
 */
const dateToUnixMilli = z.string().transform((t) => new Date(t.split(" ").at(0) ?? t).getTime());

export class Analytics {
  public readonly readClient: Tinybird | NoopTinybird;
  public readonly writeClient: Tinybird | NoopTinybird;
  private clickhouse: ClickHouse;

  constructor(opts: {
    tinybirdToken?: string;
    tinybirdProxy?: {
      url: string;
      token: string;
    };
    clickhouse?: {
      url: string;
    };
  }) {
    this.readClient = opts.tinybirdToken
      ? new Tinybird({ token: opts.tinybirdToken })
      : new NoopTinybird();

    this.writeClient = opts.tinybirdProxy
      ? new Tinybird({ token: opts.tinybirdProxy.token, baseUrl: opts.tinybirdProxy.url })
      : this.readClient;

    this.clickhouse = new ClickHouse({ url: opts.clickhouse?.url });
  }

  public get insertSdkTelemetry() {
    return this.clickhouse.client.insert({
      table: "telemetry.raw_sdks_v1",
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        runtime: z.string(),
        platform: z.string(),
        versions: z.array(z.string()),
      }),
    });
  }
  //tinybird, to be removed
  public get ingestSdkTelemetry() {
    return this.writeClient.buildIngestEndpoint({
      datasource: "sdk_telemetry__v1",
      event: z.object({
        runtime: z.string(),
        platform: z.string(),
        versions: z.array(z.string()),
        requestId: z.string(),
        time: z.number(),
      }),
    });
  }

  //tinybird
  public ingestUnkeyAuditLogsTinybird(logs: MaybeArray<UnkeyAuditLog>) {
    return this.writeClient.buildIngestEndpoint({
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
          meta: l.meta ? JSON.stringify(l.meta) : undefined,
          actor: {
            ...l.actor,
            meta: l.actor.meta ? JSON.stringify(l.actor.meta) : undefined,
          },
          resources: JSON.stringify(l.resources),
        })),
    })(logs);
  }

  //tinybird
  public get ingestGenericAuditLogsTinybird() {
    return this.writeClient.buildIngestEndpoint({
      datasource: "audit_logs__v2",
      event: auditLogSchemaV1.transform((l) => ({
        ...l,
        meta: l.meta ? JSON.stringify(l.meta) : undefined,
        actor: {
          ...l.actor,
          meta: l.actor.meta ? JSON.stringify(l.actor.meta) : undefined,
        },
        resources: JSON.stringify(l.resources),
      })),
    });
  }
  public get insertRatelimit() {
    return this.clickhouse.client.insert({
      table: "ratelimits.raw_ratelimits_v1",
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        workspace_id: z.string(),
        namespace_id: z.string(),
        identifier: z.string(),
        passed: z.boolean(),
      }),
    });
  }

  //tinybird
  public get ingestRatelimit() {
    return this.writeClient.buildIngestEndpoint({
      datasource: "ratelimits__v2",
      event: ratelimitSchemaV1,
    });
  }

  public get insertKeyVerification() {
    return this.clickhouse.client.insert({
      table: "verifications.raw_key_verifications_v1",
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        workspace_id: z.string(),
        key_space_id: z.string(),
        key_id: z.string(),
        region: z.string(),
        outcome: z.enum([
          "VALID",
          "RATE_LIMITED",
          "EXPIRED",
          "DISABLED",
          "FORBIDDEN",
          "USAGE_EXCEEDED",
          "INSUFFICIENT_PERMISSIONS",
        ]),
        identity_id: z.string().optional().default(""),
      }),
    });
  }

  public get insertApiRequest() {
    return this.clickhouse.client.insert({
      table: "metrics.raw_api_requests_v1",
      schema: z.object({
        request_id: z.string(),
        time: z.number().int(),
        workspace_id: z.string(),
        host: z.string(),
        method: z.string(),
        path: z.string(),
        request_headers: z.array(z.string()),
        request_body: z.string(),
        response_status: z.number().int(),
        response_headers: z.array(z.string()),
        response_body: z.string(),
        error: z.string().optional().default(""),
        service_latency: z.number().int(),
        user_agent: z.string(),
        ip_address: z.string(),
      }),
    });
  }
  // replaced by insertKeyVerification
  public get ingestKeyVerification() {
    return this.writeClient.buildIngestEndpoint({
      datasource: "key_verifications__v2",
      event: z.object({
        workspaceId: z.string(),
        apiId: z.string(),
        keyId: z.string(),
        deniedReason: z
          .enum([
            "RATE_LIMITED",
            "USAGE_EXCEEDED",
            "FORBIDDEN",
            "UNAUTHORIZED",
            "DISABLED",
            "INSUFFICIENT_PERMISSIONS",
            "EXPIRED",
          ])
          .optional(),
        time: z.number(),
        ipAddress: z.string().default(""),
        userAgent: z.string().default(""),
        requestedResource: z.string().default(""),
        edgeRegion: z.string().default(""),
        region: z.string(),
        // deprecated, use deniedReason
        ratelimited: z.boolean().default(false),
        // deprecated, use deniedReason
        usageExceeded: z.boolean().default(false),
        ownerId: z.string().optional(),
        keySpaceId: z.string(),
        requestId: z.string().optional(),
        requestBody: z.string().optional(),
        responseBody: z.string().optional(),
      }),
    });
  }

  public get getVerificationsDaily() {
    return this.readClient.buildPipe({
      pipe: "get_verifications_daily__v2",
      parameters: z.object({
        workspaceId: z.string(),
        apiId: z.string(),
        keyId: z.string().optional(),
        start: z.number().optional(),
        end: z.number().optional(),
      }),
      data: z.object({
        time: dateToUnixMilli,
        success: z.number(),
        rateLimited: z.number(),
        usageExceeded: z.number(),
      }),
    });
  }
}

export type UnkeyAuditLog = {
  workspaceId: string;
  event: z.infer<typeof unkeyAuditLogEvents>;
  description: string;
  actor: {
    type: "user" | "key";
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
      | "vercelIntegration"
      | "ratelimitIdentifier"
      | "ratelimitNamespace"
      | "identity"
      | "ratelimit";
    id: string;
    meta?: Record<string, string | number | boolean | null | undefined>;
  }>;
  context: {
    userAgent?: string;
    location: string;
  };
};
