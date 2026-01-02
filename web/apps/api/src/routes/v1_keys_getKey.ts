import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { retry } from "@/pkg/util/retry";
import { buildUnkeyQuery } from "@unkey/rbac";
import { keySchema } from "./schema";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "getKey",
  summary: "Get API key",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/keys.getKey",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      keyId: z.string().min(1).openapi({
        description: "The id of the key to fetch",
        example: "key_1234",
      }),
      decrypt: z.coerce.boolean().optional().openapi({
        description:
          "Decrypt and display the raw key. Only possible if the key was encrypted when generated.",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema: keySchema,
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysGetKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1KeysGetKey = (app: App) =>
  app.openapi(route, async (c) => {
    const { keyId, decrypt } = c.req.valid("query");
    const { cache, db, vault, rbac } = c.get("services");

    const { val: data, err } = await cache.keyById.swr(keyId, async () => {
      const dbRes = await db.readonly.query.keys.findFirst({
        where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAtM)),
        with: {
          encrypted: true,
          permissions: { with: { permission: true } },
          roles: { with: { role: true } },
          keyAuth: {
            with: {
              api: true,
            },
          },
          identity: true,
          ratelimits: true,
        },
      });
      if (!dbRes) {
        return null;
      }
      return {
        key: dbRes,
        api: dbRes.keyAuth.api,
        permissions: dbRes.permissions.map((p) => p.permission.name),
        roles: dbRes.roles.map((p) => p.role.name),
        ratelimits: dbRes.ratelimits,
        identity: dbRes.identity
          ? {
              id: dbRes.identity.id,
              externalId: dbRes.identity.externalId,
              meta: dbRes.identity.meta ?? {},
            }
          : null,
      };
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load key: ${err.message}`,
      });
    }
    if (!data) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }
    const { api, key } = data;
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.read_key", `api.${api.id}.read_key`)),
    );

    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }
    let meta = key.meta ? JSON.parse(key.meta) : undefined;
    if (!meta || Object.keys(meta).length === 0) {
      meta = undefined;
    }

    let plaintext: string | undefined = undefined;
    if (decrypt) {
      const { val, err } = rbac.evaluatePermissions(
        buildUnkeyQuery(({ or }) => or("*", "api.*.decrypt_key", `api.${api.id}.decrypt_key`)),
        auth.permissions,
      );
      if (err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "unable to evaluate permission",
        });
      }
      if (!val.valid) {
        throw new UnkeyApiError({
          code: "UNAUTHORIZED",
          message: "you're not allowed to decrypt this key",
        });
      }
      if (key.encrypted) {
        const decryptRes = await retry(3, () =>
          vault.decrypt(c, {
            keyring: key.workspaceId,
            encrypted: key.encrypted!.encrypted,
          }),
        );
        plaintext = decryptRes.plaintext;
      }
    }

    const ratelimit = key.ratelimits.find((rl) => rl.name === "default");

    return c.json({
      id: key.id,
      start: key.start,
      apiId: api.id,
      workspaceId: key.workspaceId,
      name: key.name ?? undefined,
      ownerId: key.ownerId ?? undefined,
      meta: key.meta ? JSON.parse(key.meta) : undefined,
      createdAt: key.createdAtM,
      updatedAt: key.updatedAtM ?? undefined,
      expires: key.expires?.getTime() ?? undefined,
      remaining: key.remaining ?? undefined,
      refill: key.refillAmount
        ? {
            interval: key.refillDay ? ("monthly" as const) : ("daily" as const),
            amount: key.refillAmount,
            refillDay: key.refillDay,
            lastRefillAt: key.lastRefillAt?.getTime(),
          }
        : undefined,
      ratelimit: ratelimit
        ? {
            async: false,
            limit: ratelimit.limit,
            duration: ratelimit.duration,
            refillRate: ratelimit.limit,
            refillInterval: ratelimit.duration,
          }
        : undefined,
      roles: data.roles,
      permissions: data.permissions,
      enabled: key.enabled,
      plaintext,
      identity: data.identity
        ? {
            id: data.identity.id,
            externalId: data.identity.externalId,
            meta: data.identity.meta ?? undefined,
          }
        : undefined,
    });
  });
