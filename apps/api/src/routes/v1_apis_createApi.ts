import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["apis"],
  operationId: "createApi",
  method: "post",
  path: "/v1/apis.createApi",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: z.string().min(3).openapi({
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
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1ApisCreateApiResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisCreateApi = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.create_api")),
    );

    const { name } = c.req.valid("json");

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    const keyAuth = {
      id: newId("keyAuth"),
      workspaceId: auth.authorizedWorkspaceId,
      createdAt: new Date(),
      deletedAt: null,
    };
    await db.primary.insert(schema.keyAuth).values(keyAuth);

    /**
     * Set up an api for production
     */
    const apiId = newId("api");

    await db.primary.transaction(async (tx) => {
      await tx.insert(schema.apis).values({
        id: apiId,
        name,
        workspaceId: authorizedWorkspaceId,
        authType: "key",
        keyAuthId: keyAuth.id,
        createdAt: new Date(),
        deletedAt: null,
      });

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: authorizedWorkspaceId,
        event: "api.create",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `Created ${apiId}`,
        resources: [
          {
            type: "api",
            id: apiId,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    return c.json({
      apiId,
      name,
    });
  });
