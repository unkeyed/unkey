import type { UnkeyAuditLog } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

import { insertAuditLogs } from "@/lib/audit";
import { upsertPermissions } from "../rbac";

export const createRootKey = workspaceProcedure
  .input(
    z.object({
      name: z.string().optional(),
      permissions: z.array(unkeyPermissionValidation).min(1, {
        message: "You need to add at least one permissions.",
      }),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const unkeyApi = await db.query.apis
      .findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.workspaceId, env().UNKEY_WORKSPACE_ID),
            eq(schema.apis.id, env().UNKEY_API_ID),
          ),
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
    const keyAuthId = unkeyApi.keyAuthId;
    if (!keyAuthId) {
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
          keyAuthId,
          name: input?.name,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: ctx.workspace.id,
          expires: null,
          createdAtM: Date.now(),
          remaining: null,
          refillAmount: null,
          refillDay: null,
          lastRefillAt: null,
          enabled: true,
          deletedAtM: null,
        });

        auditLogs.push({
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.create",
          description: `Created ${keyId}`,
          resources: [
            {
              type: "key",
              id: keyId,
              name: input?.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        const { permissions, auditLogs: createPermissionLogs } = await upsertPermissions(
          ctx,
          env().UNKEY_WORKSPACE_ID,
          input.permissions,
        );

        auditLogs.push(...createPermissionLogs);
        auditLogs.push(
          ...permissions.map((p) => ({
            workspaceId: ctx.workspace.id,
            actor: { type: "user" as const, id: ctx.user.id },
            event: "authorization.connect_permission_and_key" as const,
            description: `Connected ${p.id} and ${keyId}`,
            resources: [
              {
                type: "key" as const,
                id: keyId,
                name: input?.name,
              },
              {
                type: "permission" as const,
                id: p.id,
                name: p.name,
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
