import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, isNull, sql } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["apis"],
  operationId: "deleteKeys",
  method: "post",
  path: "/v1/apis.deleteKeys",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            apiId: z.string().min(1).openapi({
              description: "The id of the api, that the keys belong to.",
              example: "api_1234",
            }),
            permanent: z.boolean().optional().openapi({
              description:
                "If true, the keys will be permanently deleted. If false, the keys will be soft-deleted and can be restored later. ",
              default: false,
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The keys have been deleted",
      content: {
        "application/json": {
          schema: z.object({
            deletedKeys: z.number().int().openapi({
              description: "The number of keys that were deleted",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1ApisDeleteKeysRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1ApisDeleteKeysResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisDeleteKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId, permanent } = c.req.valid("json");
    const { cache, db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.delete_key", `api.${apiId}.delete_key`)),
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
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `api ${apiId} not found`,
      });
    }

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${apiId} is not setup to handle keys`,
      });
    }

    let deletedKeys = 0;
    if (permanent) {
      const where = eq(schema.keys.keyAuthId, api.keyAuthId);
      await db.primary.transaction(async (tx) => {
        const keys = await tx
          .select({ count: sql<string>`count(*)` })
          .from(schema.keys)
          .where(where);
        await tx.delete(schema.keys).where(where).execute();
        deletedKeys = Number.parseInt(keys.at(0)?.count ?? "0");
      });
    } else {
      const where = and(eq(schema.keys.keyAuthId, api.keyAuthId), isNull(schema.keys.deletedAt));
      await db.primary.transaction(async (tx) => {
        const keys = await tx
          .select({ count: sql<string>`count(*)` })
          .from(schema.keys)
          .where(where);
        await tx.update(schema.keys).set({ deletedAt: new Date() }).where(where).execute();
        deletedKeys = Number.parseInt(keys.at(0)?.count ?? "0");
      });
    }

    return c.json({ deletedKeys });
  });
