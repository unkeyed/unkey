import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";
import { and, eq, isNull } from "drizzle-orm";

const route = createRoute({
  operationId: "deleteApi",
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
    const { cache, db, analytics } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.delete_api", `api.${apiId}.delete_api`)),
    );

    const { val: api, err } = await cache.apiById.swr(apiId, async () => {
      return (
        (await db.readonly.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
          with: {
            keyAuth: true,
          },
        })) ?? null
      );
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load api: ${err.message}`,
      });
    }
    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    await db.primary.transaction(async (tx) => {
      await tx.update(schema.apis).set({ deletedAt: new Date() }).where(eq(schema.apis.id, apiId));

      await analytics.ingestUnkeyAuditLogs({
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
          and(eq(table.keyAuthId, api.keyAuthId!), isNull(table.deletedAt)),
        columns: {
          id: true,
        },
      });
      await tx
        .update(schema.keys)
        .set({ deletedAt: new Date() })
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAt)));

      await analytics.ingestUnkeyAuditLogs(
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
