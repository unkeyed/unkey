import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { type Identity, and, eq, gt, isNull, sql } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { retry } from "@/pkg/util/retry";
import { schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";
import { keySchema } from "./schema";

const route = createRoute({
  tags: ["apis"],
  operationId: "listKeys",
  method: "get",
  path: "/v1/apis.listKeys",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      apiId: z.string().min(1).openapi({
        description: "The id of the api to fetch",
        example: "api_1234",
      }),
      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of keys to return",
        example: 100,
      }),
      cursor: z.string().optional().openapi({
        description:
          "Use this to fetch the next page of results. A new cursor will be returned in the response if there are more results.",
      }),
      ownerId: z.string().min(1).optional().openapi({
        deprecated: true,
        description: "Deprecated. Use `externalId` instead.",
      }),
      externalId: z.string().min(1).optional().openapi({
        description: "If provided, this will only return keys where the `externalId` matches.",
      }),

      decrypt: z.coerce.boolean().optional().openapi({
        description:
          "Decrypt and display the raw key. Only possible if the key was encrypted when generated.",
      }),
      revalidateKeysCache: z.coerce
        .boolean()
        .default(false)
        .optional()
        .openapi({
          description: `\`EXPERIMENTAL\`

Skip the cache and fetch the keys from the database directly.
When you're creating a key and immediately listing all keys to display them to your user, you might want to skip the cache to ensure the key is displayed immediately.
        `,
        }),
    }),
  },
  responses: {
    200: {
      description: "List of keys for the api",
      content: {
        "application/json": {
          schema: z.object({
            keys: z.array(keySchema),
            cursor: z.string().optional().openapi({
              description:
                "The cursor to use for the next page of results, if no cursor is returned, there are no more results",
              example: "eyJrZXkiOiJrZXlfMTIzNCJ9",
            }),
            total: z.number().int().openapi({
              description: "The total number of keys for this api",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1ApisListKeysResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisListKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId, revalidateKeysCache, limit, cursor, externalId, ownerId, decrypt } =
      c.req.valid("query");
    const { cache, db, rbac, vault } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or, and }) =>
        or(
          "*",
          and(
            or("api.*.read_key", `api.${apiId}.read_key`),
            or("api.*.read_api", `api.${apiId}.read_api`),
          ),
        ),
      ),
    );

    const { val: api, err } = await cache.apiById.swr(apiId, async () => {
      return (
        (await db.readonly.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
          with: {
            keyAuth: true,
          },
        })) ?? null
      );
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to lod api: ${err.message}`,
      });
    }

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `api ${apiId} not found`,
      });
    }

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${apiId} is not setup to handle keys`,
      });
    }

    let identity: Identity | undefined = undefined;
    if (externalId) {
      const { val, err } = await cache.identityByExternalId.swr(externalId, async () => {
        return db.readonly.query.identities.findFirst({
          where: (table, { and, eq }) =>
            and(
              eq(table.externalId, externalId),
              eq(table.workspaceId, auth.authorizedWorkspaceId),
            ),
        });
      });
      if (err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: err.message,
        });
      }
      if (val) {
        identity = val;
      }

      if (!identity) {
        return c.json({ keys: [], cursor: undefined, total: 0 });
      }
    }

    async function loadKeys() {
      const [keys, total] = await Promise.all([
        db.readonly.query.keys.findMany({
          where: and(
            ...[
              eq(schema.keys.keyAuthId, api!.keyAuthId!),
              isNull(schema.keys.deletedAt),
              cursor ? gt(schema.keys.id, cursor) : undefined,
              identity
                ? eq(schema.keys.identityId, identity.id)
                : ownerId
                  ? eq(schema.keys.ownerId, ownerId)
                  : undefined,
            ].filter(Boolean),
          ),
          with: {
            identity: true,
            encrypted: true,
            roles: {
              with: {
                role: {
                  with: {
                    permissions: {
                      with: {
                        permission: true,
                      },
                    },
                  },
                },
              },
            },
            permissions: {
              with: {
                permission: true,
              },
            },
          },
          limit: limit,
          orderBy: schema.keys.id,
        }),

        db.readonly
          .select({ count: sql<string>`count(*)` })
          .from(schema.keys)
          .where(and(eq(schema.keys.keyAuthId, api!.keyAuthId!), isNull(schema.keys.deletedAt))),
      ]);

      /**
       * Createa a unique set of all permissions, whether they're attached directly or connected
       * through a role.
       */

      return {
        keys: keys.map((k) => {
          const permissions = new Set<string>([
            ...k.permissions.map((p) => p.permission.name),
            ...k.roles.flatMap((r) => r.role.permissions.map((p) => p.permission.name)),
          ]);
          return {
            ...k,
            identity: k.identity
              ? {
                  id: k.identity.id,
                  externalId: k.identity.externalId,
                  meta: k.identity.meta ?? {},
                }
              : null,
            permissions: Array.from(permissions.values()),
            roles: k.roles.map((r) => r.role.name),
          };
        }),
        total: Number.parseInt(total.at(0)?.count ?? "0"),
      };
    }

    const cacheKey = [api.keyAuthId, cursor, externalId, ownerId, limit].join("_");

    const data = revalidateKeysCache
      ? await loadKeys().then((res) => {
          c.executionCtx.waitUntil(cache.keysByApiId.set(cacheKey, res));
          return res;
        })
      : await cache.keysByApiId.swr(cacheKey, loadKeys).then((cached) => {
          if (cached.err) {
            throw new UnkeyApiError({
              code: "INTERNAL_SERVER_ERROR",
              message: cached.err.message,
            });
          }
          return cached.val!;
        });

    // keyId->key
    const plaintext: Record<string, string> = {};

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

      await Promise.all(
        data.keys.map(async ({ id, workspaceId, encrypted }) => {
          if (!encrypted) {
            return;
          }

          const decryptedRes = await retry(3, () =>
            vault.decrypt(c, {
              keyring: workspaceId,
              encrypted: encrypted.encrypted,
            }),
          );
          plaintext[id] = decryptedRes.plaintext;
        }),
      );
    }
    return c.json({
      keys: data.keys.map((k) => ({
        id: k.id,
        start: k.start,
        apiId: api.id,
        workspaceId: k.workspaceId,
        name: k.name ?? undefined,
        ownerId: k.ownerId ?? undefined,
        meta: k.meta ? JSON.parse(k.meta) : undefined,
        createdAt: k.createdAt.getTime() ?? undefined,
        updatedAt: k.updatedAtM ?? undefined,
        expires: k.expires?.getTime() ?? undefined,
        ratelimit:
          k.ratelimitAsync !== null && k.ratelimitLimit !== null && k.ratelimitDuration !== null
            ? {
                async: k.ratelimitAsync,
                type: k.ratelimitAsync ? "fast" : ("consistent" as any),
                limit: k.ratelimitLimit,
                duration: k.ratelimitDuration,
                refillRate: k.ratelimitLimit,
                refillInterval: k.ratelimitDuration,
              }
            : undefined,
        remaining: k.remaining ?? undefined,
        refill:
          k.refillInterval && k.refillAmount && k.lastRefillAt
            ? {
                interval: k.refillInterval,
                amount: k.refillAmount,
                refillDay: k.refillInterval === "monthly" && k.refillDay ? k.refillDay : null,
                lastRefillAt: k.lastRefillAt?.getTime(),
              }
            : undefined,
        environment: k.environment ?? undefined,
        plaintext: plaintext[k.id] ?? undefined,
        roles: k.roles,
        permissions: k.permissions,
        identity: k.identity
          ? {
              id: k.identity.id,
              externalId: k.identity.externalId,
              meta: k.identity.meta ?? undefined,
            }
          : undefined,
      })),
      total: data.total,
      cursor: data.keys.at(-1)?.id ?? undefined,
    });
  });
