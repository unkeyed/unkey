import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";
import { sha256 } from "@unkey/hash";

const route = createRoute({
  tags: ["keys"],
  operationId: "whoAmI",
  method: "get",
  path: "/v1/keys.whoAmI",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      key: z.string().min(1).openapi({
        description: "The actual key to fetch",
        example: "actual_key_value_here",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema:  z.object({
            id: z.string().openapi({
              description: "The ID of the key",
              example: "abc123",
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
                  example: "identity123",
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
              .object({})
              .optional()
              .openapi({
                description: "Metadata associated with the key",
                example: { role: "admin", plan: "premium" },
              }),
            createdAt: z.number().int().openapi({
              description: "The timestamp when the key was created",
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
export type V1KeysGetKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysWhoAmi = (app: App) =>
    app.openapi(route, async (c) => {
        const { key: secret } = c.req.valid("query");
        const { cache, db } = c.get("services");
        const hash = await sha256(secret);
        const { val: data, err } = await cache.keyByHash.swr(hash, async () => {
          const dbRes = await db.readonly.query.keys.findFirst({
            where: (table, { eq, and, isNull }) => and(eq(table.hash, hash), isNull(table.deletedAt)),
            with: {
              encrypted: true,
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
              encrypted: dbRes.encrypted ?? null, 
            },
            api: dbRes.keyAuth.api,
            identity: dbRes.identity
              ? {
                  id: dbRes.identity.id,
                  externalId: dbRes.identity.externalId,
                  meta: dbRes.identity.meta ?? {},
                }
              : null,
          } as any; // this was neccessary so that we don't need to return the workspace and other types defined in keyByHash 
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
            message: `key ${secret} not found`,
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
            message: `key ${secret} not found`,
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
            meta: key.meta ? JSON.parse(key.meta) : undefined,
            createdAt: key.createdAt.getTime(),
            enabled: key.enabled,
            environment: key.environment ?? undefined,
        });
      });