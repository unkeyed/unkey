import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { rootKey } from "@/pkg/hono/middlewares/root-key-middleware";
import { schema } from "@unkey/db";
import { eq } from "drizzle-orm";

const route = createRoute({
  method: "post",
  path: "/v1/apis.deleteApi",
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

export const registerV1ApisDeleteApi = (app: App) => {
  app.use(route.getRoutingPath(), rootKey());
  return app.openapi(route, async (c) => {
    const rootKey = c.get("rootKey");

    const { apiId } = c.req.valid("json");

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq }) => eq(table.id, apiId),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== rootKey.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }
    await db.delete(schema.apis).where(eq(schema.apis.id, apiId));
    await cache.remove(c, "apiById", apiId);

    return c.jsonT({});
  });
};
