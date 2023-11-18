import { KeyId, db, keyCache, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { withCache } from "@/pkg/cache/with_cache";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { keySchema } from "./schema";

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
          schema: keySchema,
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysGetKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;
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
