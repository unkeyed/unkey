import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import type { UnkeyAuditLog } from "@/pkg/analytics";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { eq, inArray, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["identities"],
  operationId: "updateIdentity",
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
                  limit: z.number().min(1).openapi({
                    description:
                      "How many requests may pass within a given window before requests are rejected.",
                    example: 10,
                  }),
                  duration: z.number().min(1000).openapi({
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
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the permission. This is used internally",
                example: "perm_123",
              }),
              name: z.string().openapi({
                description: "The name of the permission",
                example: "dns.record.create",
              }),
            }),
          ),
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

    const { db, analytics, cache } = c.get("services");

    if (!req.identityId && (!req.externalId || !req.environment)) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "Provide either identityId or externalId",
      });
    }

    if (req.ratelimits && req.ratelimits.length >= 2) {
      const ratelimitNames = new Set<string>();
      for (const rl of req.ratelimits) {
        ratelimitNames.add(rl.name);
      }
      if (ratelimitNames.size !== req.ratelimits.length) {
        throw new UnkeyApiError({
          code: "PRECONDITION_FAILED",
          message: "ratelimit names must be unique",
        });
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

      if (typeof req.meta !== "undefined") {
        await tx
          .update(schema.identities)
          .set({
            meta: req.meta,
          })
          .where(eq(schema.identities.id, identity.id));
      }

      if (typeof req.ratelimits !== "undefined") {
        if (identity.ratelimits.length > 0) {
          await tx.delete(schema.ratelimits).where(
            inArray(
              schema.ratelimits.id,
              identity.ratelimits.map((r) => r.id),
            ),
          );
        }

        for (const rl of identity.ratelimits) {
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
              },
            ],

            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          });
        }

        if (req.ratelimits.length > 0) {
          const newRatelimits = req.ratelimits.map((r) => ({
            id: newId("ratelimit"),
            workspaceId: auth.authorizedWorkspaceId,
            identityId: identity.id,
            name: r.name,
            limit: r.limit,
            duration: r.duration,
          }));
          await tx.insert(schema.ratelimits).values(newRatelimits);
          for (const rl of newRatelimits) {
            auditLogs.push({
              workspaceId: auth.authorizedWorkspaceId,
              event: "ratelimit.create" as const,
              actor: {
                type: "key" as const,
                id: auth.key.id,
              },
              description: `Created ${rl.id}`,
              resources: [
                {
                  type: "identity" as const,
                  id: identity.id,
                },
                {
                  type: "ratelimit" as const,
                  id: rl.id,
                },
              ],

              context: {
                location: c.get("location"),
                userAgent: c.get("userAgent"),
              },
            });
          }
        }
      }

      const identityAfterUpdate = await tx.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, identity.id),
        with: {
          ratelimits: true,
        },
      });

      c.executionCtx.waitUntil(
        Promise.all(
          identity.keys.flatMap(({ id, hash }) => [
            cache.keyById.remove(id),
            cache.keyByHash.remove(hash),
          ]),
        ),
      );
      return identityAfterUpdate!;
    });

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs([
        {
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
        },
        ...auditLogs,
      ]),
    );

    return c.json({
      id: identity.id,
      externalId: identity.externalId,
      meta: identity.meta,
      ratelimits: identity.ratelimits.map((rl) => ({
        name: rl.name,
        limit: rl.limit,
        duration: rl.duration,
      })),
    });
  });
