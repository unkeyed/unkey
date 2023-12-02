import { db, keyService, usageLimiter } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { eq, sql } from "drizzle-orm";

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
            value: z.number().int().nullable().openapi({
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
            remaining: z
              .number()
              .int()
              .nullable()
              .openapi({
                description:
                  "The number of remaining requests for this key after updating it. `null` means unlimited.",
                examples: [1, null],
              }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysUpdateKeyRemainingRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1KeysUpdateKeyRemainingResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysUpdateRemaining = (app: App) =>
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

    const key = await db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, req.keyId),
    });

    if (!key || key.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${req.keyId} not found` });
    }

    switch (req.op) {
      case "increment": {
        if (key.remaining === null) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message:
              "cannot increment a key with unlimited remaining requests, please 'set' a value instead.",
          });
        }
        if (req.value === null) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "cannot increment a key by null.",
          });
        }
        await db
          .update(schema.keys)
          .set({
            remaining: sql`remaining_requests + ${req.value}`,
          })
          .where(eq(schema.keys.id, req.keyId));
      }
      case "decrement": {
        if (key.remaining === null) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message:
              "cannot decrement a key with unlimited remaining requests, please 'set' a value instead.",
          });
        }
        if (req.value === null) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "cannot decrement a key by null.",
          });
        }
        await db
          .update(schema.keys)
          .set({
            remaining: sql`remaining_requests - ${req.value}`,
          })
          .where(eq(schema.keys.id, req.keyId));
      }
      case "set": {
        await db
          .update(schema.keys)
          .set({
            remaining: req.value,
          })
          .where(eq(schema.keys.id, req.keyId));
      }
    }

    await usageLimiter.revalidate({ keyId: key.id });
    const keyAfterUpdate = await db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, req.keyId),
    });
    if (!keyAfterUpdate) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "key not found after update, this should not happen",
      });
    }

    return c.jsonT({
      remaining: keyAfterUpdate.remaining,
    });
  });
