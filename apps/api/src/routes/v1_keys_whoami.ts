import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { sha256 } from "@unkey/hash";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "whoami",
  summary: "Get key information",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/keys.whoami",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            key: z.string().min(1).openapi({
              description: "The actual key to fetch",
              example: "sk_123",
            }),
          }),
        },
      },
    },
  },

  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The ID of the key",
              example: "key_123",
            }),
            name: z.string().optional().openapi({
              description: "The name of the key",
              example: "API Key 1",
            }),
            remaining: z.number().int().optional().openapi({
              description: "The remaining number of requests for the key",
              example: 1000,
            }),
            identity: z
              .object({
                id: z.string().openapi({
                  description: "The identity ID associated with the key",
                  example: "id_123",
                }),
                externalId: z.string().openapi({
                  description: "The external identity ID associated with the key",
                  example: "ext123",
                }),
              })
              .optional()
              .openapi({
                description: "The identity object associated with the key",
              }),
            meta: z
              .record(z.unknown())
              .optional()
              .openapi({
                description: "Metadata associated with the key",
                example: { role: "admin", plan: "premium" },
              }),
            createdAt: z.number().int().openapi({
              description: "The timestamp in milliseconds when the key was created",
              example: 1620000000000,
            }),
            enabled: z.boolean().openapi({
              description: "Whether the key is enabled",
              example: true,
            }),
            environment: z.string().optional().openapi({
              description: "The environment the key is associated with",
              example: "production",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysWhoAmIRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysWhoAmIResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysWhoAmI = (app: App) =>
  app.openapi(route, async (c) => {
    const { key: secret } = c.req.valid("json");
    const { cache, db } = c.get("services");
    const hash = await sha256(secret);
    const { val: data, err } = await cache.keyByHash.swr(hash, async () => {
      const dbRes = await db.readonly.query.keys.findFirst({
        where: (table, { eq, and, isNull }) => and(eq(table.hash, hash), isNull(table.deletedAtM)),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
          identity: true,
        },
      });

      if (!dbRes) {
        return null;
      }

      return {
        key: {
          ...dbRes,
        },
        api: dbRes.keyAuth.api,
        identity: dbRes.identity,
      } as any; // this was necessary so that we don't need to return the workspace and other types defined in keyByHash
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load key: ${err.message}`,
      });
    }
    if (!data) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "Key not found",
      });
    }
    const { api, key } = data;
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.read_key", `api.${api.id}.read_key`)),
    );

    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "Key not found",
      });
    }
    let meta = key.meta ? JSON.parse(key.meta) : undefined;
    if (!meta || Object.keys(meta).length === 0) {
      meta = undefined;
    }

    return c.json({
      id: key.id,
      name: key.name ?? undefined,
      remaining: key.remaining ?? undefined,
      identity: data.identity
        ? {
            id: data.identity.id,
            externalId: data.identity.externalId,
          }
        : undefined,
      meta: meta,
      createdAt: key.createdAtM,
      enabled: key.enabled,
      environment: key.environment ?? undefined,
    });
  });
