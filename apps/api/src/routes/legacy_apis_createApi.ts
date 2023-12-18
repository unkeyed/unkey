import { db, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { KeyAuth, schema } from "@unkey/db";
import { newId } from "@unkey/id";

const route = createRoute({
  method: "post",
  path: "/v1/apis",
  request: {
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
export type LegacyApisCreateApiRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type LegacyApisCreateApiResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyApisCreateApi = (app: App) =>
  app.openapi(route, async (c) => {
    const authorization = c.req.header("authorization")?.replace("Bearer ", "");
    if (!authorization) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "key required" });
    }
    const rootKey = await keyService.verifyKey(c, { key: authorization });
    if (rootKey.error) {
      throw new UnkeyApiError({ code: "INTERNAL_SERVER_ERROR", message: rootKey.error.message });
    }
    if (!rootKey.value.valid) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "the root key is not valid" });
    }
    if (!rootKey.value.isRootKey) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "root key required" });
    }

    const { name } = c.req.valid("json");

    const keyAuth: KeyAuth = {
      id: newId("keyAuth"),
      workspaceId: rootKey.value.authorizedWorkspaceId,
      createdAt: new Date(),
      deletedAt: null,
    };
    await db.insert(schema.keyAuth).values(keyAuth);

    /**
     * Set up an api for production
     */
    const apiId = newId("api");
    await db.insert(schema.apis).values({
      id: apiId,
      name,
      workspaceId: rootKey.value.authorizedWorkspaceId,
      authType: "key",
      keyAuthId: keyAuth.id,
      createdAt: new Date(),
      deletedAt: null,
    });

    return c.json({
      apiId,
      name,
    });
  });
