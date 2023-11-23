import { db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { openApiErrorResponses } from "@/pkg/errors";
import { rootKey } from "@/pkg/hono/middlewares/root-key-middleware";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";

const route = createRoute({
  method: "post",
  path: "/v1/apis.createApi",
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
            name: z.string().min(1).openapi({
              description: "The name for your API. This is not customer facing.",
              example: "my-api",
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
            apiId: z.string().openapi({
              description: "The id of the api",
              example: "api_134",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1ApisCreateApiRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1ApisCreateApiResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisCreateApi = (app: App) => {
  app.use(route.getRoutingPath(), rootKey());
  return app.openapi(route, async (c) => {
    const rootKey = c.get("rootKey");
    const { name } = c.req.valid("json");

    const keyAuth = {
      id: newId("keyAuth"),
      workspaceId: rootKey.authorizedWorkspaceId,
    };
    await db.insert(schema.keyAuth).values(keyAuth);

    /**
     * Set up an api for production
     */
    const apiId = newId("api");
    await db.insert(schema.apis).values({
      id: apiId,
      name,
      workspaceId: rootKey.authorizedWorkspaceId,
      authType: "key",
      keyAuthId: keyAuth.id,
    });

    return c.jsonT({
      apiId,
      name,
    });
  });
};
