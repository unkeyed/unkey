import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import type { UnkeyAuditLog } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { auth, t } from "../../trpc";

import { insertAuditLogs } from "@/lib/audit";
import { upsertPermissions } from "../rbac";

export const createRootKey = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().optional(),
      permissions: z.array(unkeyPermissionValidation).min(1, {
        message: "You need to add at least one permissions.",
      }),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a root key for this workspace. Please try again or contact support@unkey.dev.",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }

    const unkeyApi = await db.query.apis
      .findFirst({
        where: eq(schema.apis.id, env().UNKEY_API_ID),
        with: {
          workspace: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a rootkey for this workspace. Please try again or contact support@unkey.dev.",
        });
      });
    if (!unkeyApi) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `API ${env().UNKEY_API_ID} was not found`,
      });
    }
    if (!unkeyApi.keyAuthId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `API ${env().UNKEY_API_ID} is not setup to handle keys`,
      });
    }

    const keyId = newId("key");

    const { key, hash, start } = await newKey({
      prefix: "unkey",
      byteLength: 16,
    });

    const auditLogs: UnkeyAuditLog[] = [];
    try {
      await db.transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: unkeyApi.keyAuthId!,
          name: input?.name,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: workspace.id,
          expires: null,
          createdAt: new Date(),
          remaining: null,
          refillInterval: null,
          refillAmount: null,
          refillDay: null,
          lastRefillAt: null,
          deletedAt: null,
          enabled: true,
        });

        auditLogs.push({
          workspaceId: workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.create",
          description: `Created ${keyId}`,
          resources: [
            {
              type: "key",
              id: keyId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        let identityId: string | undefined = undefined;
        await tx.query.identities
          .findFirst({
            where: (table, { eq }) => eq(table.externalId, ctx.user.id),
          })
          .then((res) => {
            if (res) {
              identityId = res.id;
            }
          });
        if (!identityId) {
          identityId = newId("identity");
          await tx.insert(schema.identities).values({
            id: identityId,
            workspaceId: workspace.id,
            externalId: ctx.user.id,
          });
          auditLogs.push({
            workspaceId: workspace.id,
            actor: { type: "user", id: ctx.user.id },
            event: "identity.create",
            description: `Created ${identityId}`,
            resources: [
              {
                type: "identity",
                id: identityId,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          });
        }
        await tx.update(schema.keys).set({ identityId }).where(eq(schema.keys.id, keyId));

        const { permissions, auditLogs: createPermissionLogs } = await upsertPermissions(
          ctx,
          env().UNKEY_WORKSPACE_ID,
          input.permissions,
        );
        auditLogs.push(...createPermissionLogs);

        auditLogs.push(
          ...permissions.map((p) => ({
            workspaceId: workspace.id,
            actor: { type: "user" as const, id: ctx.user.id },
            event: "authorization.connect_permission_and_key" as const,
            description: `Connected ${p.id} and ${keyId}`,
            resources: [
              {
                type: "key" as const,
                id: keyId,
              },
              {
                type: "permission" as const,
                id: p.id,
              },
            ],
            context: {
              location: ctx.audit.location,
              userAgent: ctx.audit.userAgent,
            },
          })),
        );

        await tx.insert(schema.keysPermissions).values(
          permissions.map((p) => ({
            keyId,
            permissionId: p.id,
            workspaceId: env().UNKEY_WORKSPACE_ID,
          })),
        );
        await insertAuditLogs(tx, auditLogs);
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to create the rootkey. Please try again or contact support@unkey.dev",
      });
    }

    return { key, keyId };
  });
