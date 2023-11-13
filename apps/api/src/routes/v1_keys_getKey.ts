import { KeyId, db, keyCache, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { withCache } from "@/pkg/cache/with_cache";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";

const route = createRoute({
  method: "get",
  path: "/v1/keys.getKey",
  request: {
    header: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),
    query: z.object({
      keyId: z.string().min(1).openapi({
        description: "The id of the key to fetch",
        example: "key_1234",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of the key",
              example: "key_1234",
            }),
            start: z.string().openapi({
              description: "The first few characters of the key to visually identify it",
              example: "sk_5j1",
            }),
            workspaceId: z.string().openapi({
              description: "The id of the workspace that owns the key",
              example: "ws_1234",
            }),
            apiId: z.string().optional().openapi({
              description: "The id of the api that this key is for",
              example: "api_1234",
            }),
            name: z.string().optional().openapi({
              description:
                "The name of the key, give keys a name to easily identifiy their purpose",
              example: "Customer X",
            }),
            ownerId: z.string().optional().openapi({
              description:
                "The id of the tenant associated with this key. Use whatever reference you have in your system to identify the tenant. When verifying the key, we will send this field back to you, so you know who is accessing your API.",
              example: "user_123",
            }),
            meta: z
              .record(z.unknown())
              .optional()
              .openapi({
                description: "Any additional metadata you want to store with the key",
                example: {
                  roles: ["admin", "user"],
                  stripeCustomerId: "cus_1234",
                },
              }),
            createdAt: z.number().openapi({
              description: "The unix timestamp in milliseconds when the key was created",
              example: Date.now(),
            }),
            deletedAt: z.number().optional().openapi({
              description:
                "The unix timestamp in milliseconds when the key was deleted. We don't delete the key outright, you can restore it later.",
              example: Date.now(),
            }),
            expiresAt: z.number().optional().openapi({
              description:
                "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
              example: Date.now(),
            }),
            remaining: z.number().optional().openapi({
              description:
                "The number of requests that can be made with this key before it becomes invalid. If this field is null or undefined, the key has no request limit.",
              example: 1000,
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export const registerV1KeysGetKey = (app: App) =>
  app.openapi(route, async (c) => {
    const authorization = c.req.header("authorization")!.replace("Bearer ", "");
    const rootKey = await keyService.verifyKey(c, { key: authorization });
    if (rootKey.error) {
      throw new UnkeyApiError({ code: "INTERNAL_SERVER_ERROR", message: rootKey.error.message });
    }
    if (!rootKey.value.valid) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "the root key is not valid" });
    }
    if (!rootKey.value.isRootKey) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "root key required" });
    }

    const { keyId } = c.req.query();

    const key = await withCache(c, keyCache, async (kid: KeyId) => {
      return (
        (await db.query.keys.findFirst({
          where: (table, { eq }) => eq(table.id, kid),
        })) ?? null
      );
    })(keyId);

    if (!key || key.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${keyId} not found` });
    }
    if (key.deletedAt !== null) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${keyId} not found` });
    }

    return c.jsonT({
      id: key.id,
      start: key.start,
      workspaceId: key.workspaceId,
      name: key.name ?? undefined,
      ownerId: key.ownerId ?? undefined,
      meta: key.meta ?? undefined,
      createdAt: key.createdAt.getTime() ?? undefined,
      expiresAt: key.expires?.getTime() ?? undefined,
      remaining: key.remainingRequests ?? undefined,
    });
  });
