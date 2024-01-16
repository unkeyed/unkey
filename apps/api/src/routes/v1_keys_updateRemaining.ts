import { db, usageLimiter } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { eq, schema, sql } from "@/pkg/db";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { newId } from "@unkey/id";

const route = createRoute({
  method: "post",
  path: "/v1/keys.updateRemaining",
  security: [{ bearerAuth: [] }],
  request: {
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
            remaining: z.number().int().nullable().openapi({
              description:
                "The number of remaining requests for this key after updating it. `null` means unlimited.",
              example: 100,
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysUpdateRemainingRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1KeysUpdateRemainingResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysUpdateRemaining = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const req = c.req.valid("json");

    const key = await db.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, req.keyId),
    });

    if (!key || key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${req.keyId} not found` });
    }

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    await db.transaction(async (tx) => {
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
          await tx
            .update(schema.keys)
            .set({
              remaining: sql`remaining_requests + ${req.value}`,
            })
            .where(eq(schema.keys.id, req.keyId));
          break;
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
          await tx
            .update(schema.keys)
            .set({
              remaining: sql`remaining_requests - ${req.value}`,
            })
            .where(eq(schema.keys.id, req.keyId));
          break;
        }
        case "set": {
          await tx
            .update(schema.keys)
            .set({
              remaining: req.value,
            })
            .where(eq(schema.keys.id, req.keyId));
          break;
        }
      }

      await tx.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        time: new Date(),
        workspaceId: authorizedWorkspaceId,
        actorType: "key",
        actorId: rootKeyId,
        event: "api.create",
        description: `updated remaining requests for key ${req.keyId}`,
        keyAuthId: key.keyAuthId,
      });
    });

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

    return c.json({
      remaining: keyAfterUpdate.remaining,
    });
  });
