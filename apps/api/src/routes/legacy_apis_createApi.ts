import { db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
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
    const auth = await rootKeyAuth(c);
    const { name } = c.req.valid("json");

    const keyAuth: KeyAuth = {
      id: newId("keyAuth"),
      workspaceId: auth.authorizedWorkspaceId,
      createdAt: new Date(),
      deletedAt: null,
    };
    await db.insert(schema.keyAuth).values(keyAuth);

    /**
     * Set up an api for production
     */
    const apiId = newId("api");
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    await db.transaction(async (tx) => {
      await tx.insert(schema.apis).values({
        id: apiId,
        name,
        workspaceId: authorizedWorkspaceId,
        authType: "key",
        keyAuthId: keyAuth.id,
        createdAt: new Date(),
        deletedAt: null,
      });
      await tx.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        time: new Date(),
        workspaceId: authorizedWorkspaceId,
        actorType: "key",
        actorId: rootKeyId,
        event: "api.create",
        description: `API ${name} created`,
        apiId: apiId,
      });
    });

    return c.json({
      apiId,
      name,
    });
  });
