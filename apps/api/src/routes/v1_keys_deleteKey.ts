import { KeyId, db, keyCache, keyService, verificationCache } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { schema } from "@unkey/db";

import { withCache } from "@/pkg/cache/with_cache";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { eq } from "drizzle-orm";

const route = createRoute({
  method: "post",
  path: "/v1/keys.deleteKey",
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
            keyId: z.string().min(1).openapi({
              description: "The id of the key to revoke",
              example: "key_1234",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The key was successfully revoked, it may take up to 30s for this to take effect in all regions",
      content: {
        "application/json": {
          schema: z.object({}),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysDeleteKeyRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1KeysDeleteKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysDeleteKey = (app: App) =>
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

    const { keyId } = c.req.valid("json");

    console.log("XXXXXXX", keyId);
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

    await db
      .update(schema.keys)
      .set({
        deletedAt: new Date(),
      })
      .where(eq(schema.keys.id, key.id));

    await keyCache.remove(c, keyId);
    await verificationCache.remove(c, key.hash);

    console.log("I GOT THIS FAR");
    return c.jsonT({});
  });
