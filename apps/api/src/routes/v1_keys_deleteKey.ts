import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { schema } from "@unkey/db";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { eq } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "deleteKey",
  method: "post",
  path: "/v1/keys.deleteKey",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().min(1).openapi({
              description: "The id of the key to revoke",
              example: "key_1234",
            }),
            permanent: z.boolean().default(false).optional().openapi({
              description:
                "By default Unkey soft deletes keys, so they may be recovered later. If you want to permanently delete it, set permanent=true. This might be necessary if you run into NOT_UNIQUE errors during key migration.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The key was successfully revoked, it may take up to 30s for this to take effect in all regions",
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
export type V1KeysDeleteKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysDeleteKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysDeleteKey = (app: App) =>
  app.openapi(route, async (c) => {
    const { keyId, permanent } = c.req.valid("json");
    const { cache, db } = c.get("services");

    const data = await cache.keyById.swr(keyId, async () => {
      const dbRes = await db.readonly.query.keys.findFirst({
        where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAt)),
        with: {
          identity: true,
          encrypted: true,
          permissions: {
            with: {
              permission: true,
            },
          },
          roles: {
            with: {
              role: true,
            },
          },
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      });
      if (!dbRes) {
        return null;
      }
      return {
        key: dbRes,
        api: dbRes.keyAuth.api,
        permissions: dbRes.permissions.map((p) => p.permission.name),
        roles: dbRes.roles.map((r) => r.role.name),
        identity: dbRes.identity
          ? {
              id: dbRes.identity.id,
              externalId: dbRes.identity.externalId,
              meta: dbRes.identity.meta,
            }
          : null,
      };
    });

    if (data.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load key: ${data.err.message}`,
      });
    }
    if (!data.val) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${keyId} not found` });
    }
    const { key, api } = data.val;

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.delete_key", `api.${api.id}.delete_key`)),
    );

    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "you are not allowed to do this",
      });
    }

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    await db.primary.transaction(async (tx) => {
      if (permanent) {
        await tx.delete(schema.keys).where(eq(schema.keys.id, key.id));
      } else {
        await tx
          .update(schema.keys)
          .set({ deletedAt: new Date() })
          .where(eq(schema.keys.id, key.id));
      }

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: authorizedWorkspaceId,
        event: "key.delete",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `${permanent ? "Permanently deleted" : "Deleted"} ${key.id}`,
        resources: [
          {
            type: "key",
            id: key.id,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    await Promise.all([cache.keyByHash.remove(key.hash), cache.keyById.remove(key.id)]);

    return c.json({});
  });
