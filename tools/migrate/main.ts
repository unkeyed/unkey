import { schema } from "@unkey/db";

import { Tinybird } from "@chronark/zod-bird";
import { connect } from "@planetscale/database";
import { newId } from "@unkey/id";
import { drizzle } from "drizzle-orm/planetscale-serverless";
import { z } from "zod";

const ingestAuditLog = new Tinybird({ token: process.env.TINYBIRD_TOKEN! }).buildIngestEndpoint({
  datasource: "audit_logs__v2",
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
            "keyAuthId",
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
        userAgent: z.string().optional(),
        ipAddress: z.string().ip().optional(),
      })
      .optional(),
  }),
});

async function main() {
  console.log("RUNNING");
  const _db = drizzle(
    connect({
      host: process.env.DATABASE_HOST,
      username: process.env.DATABASE_USERNAME,
      password: process.env.DATABASE_PASSWORD,
    }),
    {
      schema,
    },
  );

  for (const l of existingLogs) {
    console.log(l.description);

    const resources: {
      type:
        | "key"
        | "api"
        | "workspace"
        | "role"
        | "permission"
        | "keyAuthId"
        | "vercelBinding"
        | "vercelIntegration";
      id: string;
    }[] = [];
    if (l.apiId) {
      resources.push({
        type: "api",
        id: l.apiId,
      });
    }
    if (l.keyAuthId) {
      resources.push({
        type: "keyAuthId",
        id: l.keyAuthId,
      });
    }
    if (l.keyId) {
      resources.push({
        type: "key",
        id: l.keyId,
      });
    }

    if (l.vercelBindingId) {
      resources.push({
        type: "vercelBinding",
        id: l.vercelBindingId,
      });
    }

    if (l.vercelIntegrationId) {
      resources.push({
        type: "vercelIntegration",
        id: l.vercelIntegrationId,
      });
    }

    await ingestAuditLog({
      actor: {
        type: l.actorType,
        id: l.actorId,
      },

      event: l.event,
      resources,
      workspaceId: l.workspaceId,
      time: l.time.getTime(),
    });
  }
}

main();
