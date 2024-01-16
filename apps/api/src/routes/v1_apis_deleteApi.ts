import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";

const route = createRoute({
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
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1ApisDeleteApiResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisDeleteApi = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const { apiId } = c.req.valid("json");

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    await db.transaction(async (tx) => {
      await tx.update(schema.apis).set({ deletedAt: new Date() }).where(eq(schema.apis.id, apiId));
      await tx.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        time: new Date(),
        workspaceId: authorizedWorkspaceId,
        actorType: "key",
        actorId: rootKeyId,
        event: "api.delete",
        description: `API ${api.name} deleted`,
        apiId: apiId,
      });
    });
    // TODO: Delete all keys for this api
    await cache.remove(c, "apiById", apiId);

    return c.json({});
  });
