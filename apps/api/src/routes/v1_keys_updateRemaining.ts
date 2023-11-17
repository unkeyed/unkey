import { db, apiCache, keyService, ApiId } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { withCache } from "@/pkg/cache/with_cache";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { newId } from "@/pkg/id";
import { schema } from "@unkey/db";
import { KeyV1 } from "@/pkg/keys/v1";
import { sha256 } from "@/pkg/hash/sha256";

const route = createRoute({
  method: "post",
  path: "/v1/keys.updateRemaining",
  request: {
    headers: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),

    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().openapi({
              description: "The id of the key you want to modify",
              example: "key_123",
            }),
            op: z.enum(["increment", "decrement", "set"]).openapi({
              description: "The operation you want to perform on the remaining count",
            }),
            value: z.number().int().openapi({
              description: "The value you want to set, add or subtract the remaining count by",
              example: 1,
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
            keyId: z.string().openapi({
              description:
                "The id of the key. This is not a secret and can be stored as a reference if you wish. You need the keyId to update or delete a key later.",
              example: "key_123",
            }),
            key: z.string().openapi({
              description:
                "The newly created api key, do not store this on your own system but pass it along to your user.",
              example: "prefix_xxxxxxxxx",
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
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1KeysCreateKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysCreateKey = (app: App) =>
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

    const req = c.req.valid("json");

    const api = await withCache(c, apiCache, async (id: ApiId) => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq }) => eq(table.id, id),
        })) ?? null
      );
    })(req.apiId);

    if (!api || api.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${req.apiId} not found` });
    }

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${req.apiId} is not setup to handle keys`,
      });
    }

    /**
     * Set up an api for production
     */
    const key = new KeyV1({ byteLength: req.byteLength, prefix: req.prefix }).toString();
    const start = key.slice(0, (req.prefix?.length ?? 0) + 5);
    const keyId = newId("key");
    const hash = await sha256(key.toString());

    await db.insert(schema.keys).values({
      id: keyId,
      keyAuthId: api.keyAuthId,
      name: req.name,
      hash,
      start,
      ownerId: req.ownerId,
      meta: JSON.stringify(req.meta ?? {}),
      workspaceId: rootKey.value.authorizedWorkspaceId,
      forWorkspaceId: null,
      expires: req.expires ? new Date(req.expires) : null,
      createdAt: new Date(),
      ratelimitLimit: req.ratelimit?.limit,
      ratelimitRefillRate: req.ratelimit?.refillRate,
      ratelimitRefillInterval: req.ratelimit?.refillInterval,
      ratelimitType: req.ratelimit?.type,
      remainingRequests: req.remaining,
      totalUses: 0,
      deletedAt: null,
    });
    // TODO: emit event to tinybird
    return c.jsonT({
      keyId,
      key,
    });
  });
