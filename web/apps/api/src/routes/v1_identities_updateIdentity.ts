import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { type UnkeyAuditLog, insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { type Ratelimit, eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["identities"],
  operationId: "updateIdentity",
  summary: "Update identity",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/identities.updateIdentity",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            identityId: z.string().min(1).optional().openapi({
              description:
                "The id of the identity to update, use either `identityId` or `externalId`, if both are provided, `identityId` takes precedence.",
              example: "id_1234",
            }),
            externalId: z.string().min(1).optional().openapi({
              description:
                "The externalId of the identity to update, use either `identityId` or `externalId`, if both are provided, `identityId` takes precedence.",
              example: "user_1234",
            }),
            environment: z.string().default("default").optional().openapi({
              description: "This is not yet used but here for future compatibility.",
            }),
            meta: z
              .record(z.unknown())
              .optional()
              .openapi({
                description: `Attach metadata to this identity that you need to have access to when verifying a key.

Set to \`{}\` to clear.

This will be returned as part of the \`verifyKey\` response.
`,
              }),
            ratelimits: z
              .array(
                z.object({
                  name: z.string().openapi({
                    description:
                      "The name of this limit. You will need to use this again when verifying a key.",
                    example: "tokens",
                  }),
                  limit: z.number().int().min(1).openapi({
                    description:
                      "How many requests may pass within a given window before requests are rejected.",
                    example: 10,
                  }),
                  duration: z.number().int().min(1000).openapi({
                    description: "The duration for each ratelimit window in milliseconds.",
                    example: 1000,
                  }),
                }),
              )
              .optional()
              .openapi({
                description: `Attach ratelimits to this identity.

This overwrites all existing ratelimits on this identity.
Setting an empty array will delete all existing ratelimits.

When verifying keys, you can specify which limits you want to use and all keys attached to this identity, will share the limits.`,
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The identity after the update.",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of the identity.",
              example: "id_1234",
            }),
            externalId: z.string().openapi({
              description: "The externalId of the identity.",
              example: "user_1234",
            }),
            meta: z.record(z.unknown()).openapi({
              description: "The metadata attached to this identity.",
              example: {
                stripeSubscriptionId: "sub_1234",
              },
            }),
            ratelimits: z.array(
              z.object({
                name: z.string().openapi({
                  description: "The name of this limit.",
                  example: "tokens",
                }),
                limit: z.number().int().openapi({
                  description:
                    "How many requests may pass within a given window before requests are rejected.",
                  example: 10,
                }),
                duration: z.number().int().openapi({
                  description: "The duration for each ratelimit window in milliseconds.",
                  example: 1000,
                }),
              }),
            ),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1IdentitiesUpdateIdentityRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1IdentitiesUpdateIdentityResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1IdentitiesUpdateIdentity = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("identity.*.update_identity")),
    );

    const { db, cache } = c.get("services");

    if (!req.identityId && !req.externalId) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "Provide either identityId or externalId",
      });
    }

    if (req.ratelimits) {
      const uniqueNames = new Set<string>();
      for (const { name } of req.ratelimits) {
        if (uniqueNames.has(name)) {
          throw new UnkeyApiError({
            code: "CONFLICT",
            message: `Ratelimit with name "${name}" is already defined in the request`,
          });
        }
        uniqueNames.add(name);
      }
    }

    const metaLength = req.meta ? JSON.stringify(req.meta).length : 0;
    if (metaLength > 64_000) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",

        message: `metadata is too large, it must be less than 64k characters when json encoded, got: ${metaLength}`,
      });
    }

    const auditLogs: Array<UnkeyAuditLog> = [];

    const identity = await db.primary.transaction(async (tx) => {
      const identity = await tx.query.identities.findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.workspaceId, auth.authorizedWorkspaceId),
            req.identityId
              ? eq(table.id, req.identityId)
              : and(eq(table.externalId, req.externalId!), eq(table.environment, req.environment!)),
          ),
        with: {
          ratelimits: true,
          keys: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
            columns: {
              id: true,
              hash: true,
            },
          },
        },
      });
      if (!identity) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `identity ${req.identityId ?? req.externalId} not found`,
        });
      }

      auditLogs.push({
        workspaceId: auth.authorizedWorkspaceId,
        event: "identity.update",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Updated ${identity.id}`,
        resources: [
          {
            type: "identity",
            id: identity.id,
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      });

      if (typeof req.meta !== "undefined") {
        await tx
          .update(schema.identities)
          .set({
            meta: req.meta,
          })
          .where(eq(schema.identities.id, identity.id));
      }

      if (typeof req.ratelimits !== "undefined") {
        const deleteRatelimits: Ratelimit[] = [];
        const createRatelimits: Required<V1IdentitiesUpdateIdentityRequest["ratelimits"]> = [];
        const updateRatelimits: Ratelimit[] = [];
        for (const rl of identity.ratelimits) {
          const newRl = req.ratelimits.find((r) => r.name === rl.name);
          if (newRl) {
            updateRatelimits.push({
              ...rl,
              limit: newRl.limit,
              duration: newRl.duration,
            });
          } else {
            deleteRatelimits.push(rl);
          }
        }
        for (const newRl of req.ratelimits) {
          if (!identity.ratelimits.find((r) => r.name === newRl.name)) {
            createRatelimits.push(newRl);
          }
        }

        /**
         * Delete undesired ratelimits
         */
        for (const rl of deleteRatelimits) {
          await tx.delete(schema.ratelimits).where(eq(schema.ratelimits.id, rl.id));
          auditLogs.push({
            workspaceId: auth.authorizedWorkspaceId,
            event: "ratelimit.delete" as const,
            actor: {
              type: "key" as const,
              id: auth.key.id,
            },
            description: `Deleted ${rl.id}`,
            resources: [
              {
                type: "identity" as const,
                id: identity.id,
              },
              {
                type: "ratelimit" as const,
                id: rl.id,
                meta: rl,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          });
        }

        /**
         * Update existing
         */

        for (const rl of updateRatelimits) {
          await tx
            .update(schema.ratelimits)
            .set({
              name: rl.name,
              limit: rl.limit,
              duration: rl.duration,
            })
            .where(eq(schema.ratelimits.id, rl.id));
          auditLogs.push({
            workspaceId: auth.authorizedWorkspaceId,
            event: "ratelimit.update" as const,
            actor: {
              type: "key" as const,
              id: auth.key.id,
            },
            description: `Updated ${rl.id}`,
            resources: [
              {
                type: "identity" as const,
                id: identity.id,
              },
              {
                type: "ratelimit" as const,
                id: rl.id,
                meta: rl,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          });
        }

        /**
         * Create new
         */

        for (const rl of createRatelimits) {
          const ratelimitId = newId("ratelimit");
          await tx.insert(schema.ratelimits).values({
            id: ratelimitId,
            workspaceId: identity.workspaceId,
            identityId: identity.id,
            name: rl.name,
            limit: rl.limit,
            duration: rl.duration,
          });
          auditLogs.push({
            workspaceId: auth.authorizedWorkspaceId,
            event: "ratelimit.create" as const,
            actor: {
              type: "key" as const,
              id: auth.key.id,
            },

            description: `Created ${ratelimitId}`,
            resources: [
              {
                type: "identity" as const,
                id: identity.id,
              },
              {
                type: "ratelimit" as const,
                id: ratelimitId,
                meta: rl,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          });
        }
      }
      const identityAfterUpdate = await tx.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, identity.id),
        with: {
          ratelimits: true,
        },
      });

      /**
       * We currently run into "too many subrequests" errors on cloudflare when purging many keys at once
       * so we only purge the keys if there are less than 10 keys to purge and rely on the cache eviction policy
       * to remove the keys from the cache
       */
      if (identity.keys.length < 10) {
        c.executionCtx.waitUntil(
          Promise.all([
            cache.keyById.remove(identity.keys.map(({ id }) => id)),
            cache.keyByHash.remove(identity.keys.map(({ hash }) => hash)),
          ]),
        );
      }
      return identityAfterUpdate!;
    });

    await insertUnkeyAuditLog(c, undefined, auditLogs);

    return c.json({
      id: identity.id,
      externalId: identity.externalId,
      meta: identity.meta ?? {},
      ratelimits: identity.ratelimits.map((rl) => ({
        name: rl.name,
        limit: rl.limit,
        duration: rl.duration,
      })),
    });
  });
