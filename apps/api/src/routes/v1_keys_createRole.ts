import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { Permission, Role, schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  method: "post",
  path: "/v1/keys.createRole",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: z
              .string()
              .min(3)
              .regex(/^[a-zA-Z0-9_\-\.\*]+$/, {
                message:
                  "Must be at least 3 characters long and only contain alphanumeric, periods, dashes and underscores",
              })
              .openapi({
                description: "The unique name of your role. You'll use this to reference it later.",
                example: "domain.dns.manager",
              }),
            description: z.string().optional().openapi({
              description:
                "Explain what this role does. This is just for your team, your users will not see this.",
              example: "domain.dns.manager can read and write dns records for our domains.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            roleId: z.string().openapi({
              description: "The id of the role. This is used internally",
              example: "role_123",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysCreateKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysCreateKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysCreateKey = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.create_key", `api.${req.apiId}.create_key`)),
    );

    const api = await cache.withCache(c, "apiById", req.apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.id, req.apiId), isNull(table.deletedAt)),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `api ${req.apiId} not found`,
      });
    }

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${req.apiId} is not setup to handle keys`,
      });
    }
    if (req.remaining === 0) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "remaining must be greater than 0.",
      });
    }
    if ((req.remaining === null || req.remaining === undefined) && req.refill?.interval) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "remaining must be set if you are using refill.",
      });
    }
    /**
     * Set up an api for production
     */
    const key = new KeyV1({
      byteLength: req.byteLength ?? 16,
      prefix: req.prefix,
    }).toString();
    const start = key.slice(0, (req.prefix?.length ?? 0) + 5);
    const keyId = newId("key");
    const hash = await sha256(key.toString());

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    let roleIds: string[] = [];
    await db.transaction(async (tx) => {
      if (req.roles && req.roles.length > 0) {
        const roles = await tx.query.roles.findMany({
          where: (table, { inArray, and, eq }) =>
            and(eq(table.workspaceId, authorizedWorkspaceId), inArray(table.name, req.roles!)),
        });
        if (roles.length < req.roles.length) {
          const missingRoles = req.roles.filter(
            (name) => !roles.some((role) => role.name === name),
          );
          throw new UnkeyApiError({
            code: "PRECONDITION_FAILED",
            message: `Roles ${JSON.stringify(missingRoles)} are missing, please create them first`,
          });
        }
        roleIds = roles.map((r) => r.id);
      }
      await tx.insert(schema.keys).values({
        id: keyId,
        keyAuthId: api.keyAuthId!,
        name: req.name,
        hash,
        start,
        ownerId: req.ownerId,
        meta: req.meta ? JSON.stringify(req.meta) : null,
        workspaceId: authorizedWorkspaceId,
        forWorkspaceId: null,
        expires: req.expires ? new Date(req.expires) : null,
        createdAt: new Date(),
        ratelimitLimit: req.ratelimit?.limit,
        ratelimitRefillRate: req.ratelimit?.refillRate,
        ratelimitRefillInterval: req.ratelimit?.refillInterval,
        ratelimitType: req.ratelimit?.type,
        remaining: req.remaining,
        refillInterval: req.refill?.interval,
        refillAmount: req.refill?.amount,
        lastRefillAt: req.refill?.interval ? new Date() : null,
        totalUses: 0,
        deletedAt: null,
        enabled: req.enabled,
      });

      await tx.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        time: new Date(),
        workspaceId: authorizedWorkspaceId,
        actorType: "key",
        actorId: rootKeyId,
        event: "key.create",
        description: "Key created",
        apiId: api.id,
        keyAuthId: api.keyAuthId,
      });
      if (roleIds.length > 0) {
        await tx.insert(schema.keysRoles).values(
          roleIds.map((roleId) => ({
            keyId,
            roleId,
            workspaceId: authorizedWorkspaceId,
          })),
        );
      }
    });
    // TODO: emit event to tinybird
    return c.json({
      keyId,
      key,
    });
  });
