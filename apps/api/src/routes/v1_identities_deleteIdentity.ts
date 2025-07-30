import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { eq, inArray, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["identities"],
  operationId: "deleteIdentity",
  summary: "Delete identity",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/identities.deleteIdentity",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            identityId: z.string().min(1).openapi({
              description: "The id of the identity to delete",
              example: "id_1234",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The identity was successfully deleted, it may take up to 30s for this to take effect in all regions",
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

export type V1IdentitiesDeleteIdentityRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1IdentitiesDeleteIdentityResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1IdentitiesDeleteIdentity = (app: App) =>
  app.openapi(route, async (c) => {
    const { identityId } = c.req.valid("json");
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) =>
        or("identity.*.delete_identity", `identity.${identityId}.delete_identity`),
      ),
    );

    const identity = await db.readonly.query.identities.findFirst({
      where: (table, { eq }) => eq(table.id, identityId),
      with: {
        ratelimits: true,
      },
    });

    if (!identity || identity.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `identity ${identityId} not found`,
      });
    }
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    const deleteRatelimitIds = identity.ratelimits.filter((r) => r.keyId === null).map((r) => r.id);

    await db.primary.transaction(async (tx) => {
      if (deleteRatelimitIds.length > 0) {
        await tx.delete(schema.ratelimits).where(inArray(schema.ratelimits.id, deleteRatelimitIds));
      }

      await tx.delete(schema.identities).where(eq(schema.identities.id, identity.id));

      await insertUnkeyAuditLog(c, tx, [
        {
          workspaceId: authorizedWorkspaceId,
          event: "identity.delete",
          actor: {
            type: "key",
            id: rootKeyId,
          },
          description: `Deleted ${identity.id}`,
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

        ...deleteRatelimitIds.map((ratelimitId) => ({
          workspaceId: authorizedWorkspaceId,
          event: "ratelimit.delete" as const,
          actor: {
            type: "key" as const,
            id: rootKeyId,
          },
          description: `Deleted ${ratelimitId}`,
          resources: [
            {
              type: "identity" as const,
              id: identity.id,
            },
            {
              type: "ratelimit" as const,
              id: ratelimitId,
            },
          ],

          context: {
            location: c.get("location"),
            userAgent: c.get("userAgent"),
          },
        })),
      ]);
    });

    return c.json({});
  });
