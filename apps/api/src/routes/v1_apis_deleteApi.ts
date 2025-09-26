import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, errorSchemaFactory, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { and, eq } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["apis"],
  operationId: "deleteApi",
  summary: "Delete API namespace",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/apis.deleteApi",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            apiId: z.string().min(1).openapi({
              description: "The id of the api to delete",
              example: "api_1234",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The api was successfully deleted, it may take up to 30s for this to take effect in all regions",
      content: {
        "application/json": {
          schema: z.object({}),
        },
      },
    },
    ...openApiErrorResponses,
    429: {
      description: "The api is protected from deletions",
      content: {
        "application/json": {
          schema: errorSchemaFactory(z.enum(["DELETE_PROTECTED"])).openapi("ErrDeleteProtected"),
        },
      },
    },
  },
});
export type Route = typeof route;

export type V1ApisDeleteApiRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1ApisDeleteApiResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisDeleteApi = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId } = c.req.valid("json");
    const { cache, db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.delete_api", `api.${apiId}.delete_api`)),
    );

    /**
     * We do not want to cache this. Deleting the api is a very infrequent operation and
     * it's absolutely critical that we don't read a stale value from the cache when checking
     * for delete protection.
     */
    const api = await db.readonly.query.apis.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAtM)),
      with: {
        keyAuth: true,
      },
    });

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }
    if (api.deleteProtection) {
      throw new UnkeyApiError({
        code: "DELETE_PROTECTED",
        message: `api ${apiId} is protected from deletions`,
      });
    }

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    await db.primary.transaction(async (tx) => {
      await tx.update(schema.apis).set({ deletedAtM: Date.now() }).where(eq(schema.apis.id, apiId));

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: authorizedWorkspaceId,
        event: "api.delete",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `Deleted ${apiId}`,
        resources: [
          {
            type: "api",
            id: apiId,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });

      const keyIds = await tx.query.keys.findMany({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.keyAuthId, api.keyAuthId!), isNull(table.deletedAtM)),
        columns: {
          id: true,
        },
      });
      await tx
        .update(schema.keys)
        .set({ deletedAtM: Date.now() })
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!)));

      await insertUnkeyAuditLog(
        c,
        tx,
        keyIds.map((key) => ({
          workspaceId: authorizedWorkspaceId,
          event: "key.delete",
          actor: {
            type: "key",
            id: rootKeyId,
          },
          description: `Deleted ${key.id} as part of ${api.id} deletion`,
          resources: [
            {
              type: "keyAuth",
              id: api.keyAuthId!,
            },
            {
              type: "key",
              id: key.id,
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        })),
      );
    });

    await cache.apiById.remove(apiId);

    return c.json({});
  });
