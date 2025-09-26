import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { DatabaseError } from "@planetscale/database";
import { DrizzleQueryError, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["identities"],
  operationId: "createIdentity",
  summary: "Create identity",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/identities.createIdentity",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            externalId: z
              .string()
              .min(3)
              .openapi({
                description: `The id of this identity in your system.

This usually comes from your authentication provider and could be a userId, organisationId or even an email.
It does not matter what you use, as long as it uniquely identifies something in your application.

\`externalId\`s are unique across your workspace and therefore a \`CONFLICT\` error is returned when you try to create duplicates.
`,
                example: "user_123",
              }),
            meta: z
              .record(z.unknown())
              .optional()
              .openapi({
                description: `Attach metadata to this identity that you need to have access to when verifying a key.

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

When verifying keys, you can specify which limits you want to use and all keys attached to this identity, will share the limits.`,
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
            identityId: z.string().openapi({
              description:
                "The id of the identity. Used internally, you do not need to store this.",
              example: "id_123",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1IdentitiesCreateIdentityRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1IdentitiesCreateIdentityResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1IdentitiesCreateIdentity = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "identity.*.create_identity")),
    );

    const req = c.req.valid("json");

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    const metaLength = req.meta ? JSON.stringify(req.meta).length : 0;
    if (metaLength > 64_000) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",

        message: `metadata is too large, it must be less than 64k characters when json encoded, got: ${metaLength}`,
      });
    }

    const identity = {
      id: newId("identity"),
      externalId: req.externalId,
      workspaceId: auth.authorizedWorkspaceId,
      environment: "default",
      meta: req.meta,
    };
    await db.primary
      .transaction(async (tx) => {
        await tx
          .insert(schema.identities)
          .values(identity)
          .catch((e) => {
            if (
              e instanceof DrizzleQueryError &&
              e.cause instanceof DatabaseError &&
              e.cause.message.includes("Duplicate entry")
            ) {
              throw new UnkeyApiError({
                code: "CONFLICT",
                message: `Identity with externalId "${identity.externalId}" already exists in this workspace`,
              });
            }

            throw e;
          });

        const ratelimits = req.ratelimits
          ? req.ratelimits.map((r) => ({
              id: newId("ratelimit"),
              identityId: identity.id,
              workspaceId: auth.authorizedWorkspaceId,
              name: r.name,
              limit: r.limit,
              duration: r.duration,
            }))
          : [];

        if (ratelimits.length > 0) {
          await tx.insert(schema.ratelimits).values(ratelimits);
        }

        await insertUnkeyAuditLog(c, tx, [
          {
            workspaceId: authorizedWorkspaceId,
            event: "identity.create",
            actor: {
              type: "key",
              id: rootKeyId,
            },
            description: `Created ${identity.id}`,
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
          ...ratelimits.map((r) => ({
            workspaceId: authorizedWorkspaceId,
            event: "ratelimit.create" as const,
            actor: {
              type: "key" as const,
              id: rootKeyId,
            },
            description: `Created ${r.id}`,
            resources: [
              {
                type: "identity" as const,
                id: identity.id,
              },
              {
                type: "ratelimit" as const,
                id: r.id,
              },
            ],

            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          })),
        ]);
      })
      .catch((e) => {
        if (e instanceof UnkeyApiError) {
          throw e;
        }

        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "unable to store identity and ratelimits in the database",
        });
      });
    return c.json({
      identityId: identity.id,
    });
  });
