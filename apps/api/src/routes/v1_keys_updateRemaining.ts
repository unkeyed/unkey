import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { eq, schema, sql } from "@/pkg/db";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "updateRemaining",
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
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysUpdateRemainingResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysUpdateRemaining = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const { cache, db, usageLimiter, analytics } = c.get("services");

    const key = await db.readonly.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, req.keyId),
      with: {
        keyAuth: {
          with: {
            api: true,
          },
        },
      },
    });

    if (!key) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${req.keyId} not found` });
    }
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) =>
        or("*", "api.*.update_key", `api.${key.keyAuth.api.id}.update_key`),
      ),
    );
    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${req.keyId} not found` });
    }

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
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
        await db.primary
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
        await db.primary
          .update(schema.keys)
          .set({
            remaining: sql`remaining_requests - ${req.value}`,
          })
          .where(eq(schema.keys.id, req.keyId));
        break;
      }
      case "set": {
        await db.primary
          .update(schema.keys)
          .set({
            remaining: req.value,
          })
          .where(eq(schema.keys.id, req.keyId));
        break;
      }
    }

    await Promise.all([
      usageLimiter.revalidate({ keyId: key.id }),
      cache.keyByHash.remove(key.hash),
      cache.keyById.remove(key.id),
    ]);

    const keyAfterUpdate = await db.readonly.query.keys.findFirst({
      where: (table, { eq }) => eq(table.id, req.keyId),
    });
    if (!keyAfterUpdate) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "key not found after update, this should not happen",
      });
    }
    await analytics.ingestUnkeyAuditLogs({
      actor: {
        type: "key",
        id: rootKeyId,
      },
      event: "key.update",
      workspaceId: authorizedWorkspaceId,
      description: `Changed remaining to ${keyAfterUpdate.remaining}`,
      resources: [
        {
          type: "keyAuth",
          id: key.keyAuthId,
        },
        {
          type: "key",
          id: key.id,
        },
      ],
      context: {
        location: c.get("location"),
        userAgent: c.get("userAgent"),
      },
    });

    return c.json({
      remaining: keyAfterUpdate.remaining,
    });
  });
