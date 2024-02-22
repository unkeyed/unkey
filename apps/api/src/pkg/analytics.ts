import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { newId } from "@unkey/id";
import { z } from "zod";

// const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

/**
 * `t` has the format `2021-01-01 00:00:00`
 *
 * If we transform it as is, we get `1609459200000` which is `2021-01-01 01:00:00` due to fun timezone stuff.
 * So we split the string at the space and take the date part, and then parse that.
 */
const dateToUnixMilli = z.string().transform((t) => new Date(t.split(" ").at(0) ?? t).getTime());

export class Analytics {
  public readonly client: Tinybird | NoopTinybird;

  constructor(token?: string) {
    this.client = token ? new Tinybird({ token }) : new NoopTinybird();
  }

  public get ingestSdkTelemetry() {
    return this.client.buildIngestEndpoint({
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

  public get ingestAuditLogs() {
    return this.client.buildIngestEndpoint({
      datasource: "audit_logs__v1",
      event: z.object({
        workspaceId: z.string(),
        auditLogId: z.string().default(() => newId("auditLog")),
        event: z.enum([
          "workspace.create",
          "workspace.update",
          "workspace.delete",
          "api.create",
          "api.update",
          "api.delete",
          "key.create",
          "key.update",
          "key.delete",
          "vercelIntegration.create",
          "vercelIntegration.update",
          "vercelIntegration.delete",
          "vercelBinding.create",
          "vercelBinding.update",
          "vercelBinding.delete",
          "role.create",
          "role.update",
          "role.delete",
          "permission.create",
          "permission.update",
          "permission.delete",
          "authorization.connect_role_and_permission",
          "authorization.disconnect_role_and_permissions",
          "authorization.connect_role_and_key",
          "authorization.disconnect_role_and_key",
          "authorization.connect_permission_and_key",
          "authorization.disconnect_permission_and_key",
        ]),
        time: z.number().default(() => Date.now()),
        actor: z.object({
          type: z.enum(["user", "key"]),
          id: z.string(),
        }),
        resources: z.array(
          z
            .object({
              type: z.enum([
                "key",
                "api",
                "workspace",
                "role",
                "permission",
                "keyAuth",
                "vercelBinding",
                "vercelIntegration",
              ]),
              id: z.string(),
              meta: z.record(z.union([z.string(), z.number(), z.boolean()])).optional(),
            })
            .transform((r) => JSON.stringify(r)),
        ),
        context: z
          .object({
            userAgent: z.string().nullable(),
            ipAddress: z.string().ip().nullable(),
          })
          .optional(),
      }),
    });
  }

  public get ingestKeyVerification() {
    return this.client.buildIngestEndpoint({
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
      }),
    });
  }

  public get getVerificationsDaily() {
    return this.client.buildPipe({
      pipe: "get_verifications_daily__v1",
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
