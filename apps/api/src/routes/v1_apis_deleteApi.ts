import { cache, db, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { z } from "@hono/zod-openapi";

import { UnkeyApiError } from "@/pkg/errors";
import { createAuthenticatedRoute } from "@/pkg/hono/openapi/create-auth-route";
import { schema } from "@unkey/db";
import { eq } from "drizzle-orm";

const route = createAuthenticatedRoute({
  method: "post",
  path: "/v1/apis.deleteApi",
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
    const authorization = c.req.header("authorization")!.replace("Bearer ", "");
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

    const { apiId } = c.req.valid("json");

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq }) => eq(table.id, apiId),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }
    await db.delete(schema.apis).where(eq(schema.apis.id, apiId));
    await cache.remove(c, "apiById", apiId);

    return c.jsonT({});
  });
