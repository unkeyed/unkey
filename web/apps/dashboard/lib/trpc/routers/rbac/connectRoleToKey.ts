import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const connectRoleToKey = workspaceProcedure
  .input(
    z.object({
      roleId: z.string(),
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
          keys: {
            where: (table, { eq }) => eq(table.id, input.keyId),
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to connect the role to the key. Please try again or contact support@unkey.com",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.com.",
      });
    }
    const role = workspace.roles.at(0);
    if (!role) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct role. Please try again or contact support@unkey.com.",
      });
    }
    const key = workspace.keys.at(0);
    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.com.",
      });
    }

    const tuple = {
      workspaceId: workspace.id,
      keyId: key.id,
      roleId: role.id,
    };
    await db
      .transaction(async (tx) => {
        await tx
          .insert(schema.keysRoles)
          .values({ ...tuple, createdAtM: Date.now() })
          .onDuplicateKeyUpdate({
            set: { ...tuple, updatedAtM: Date.now() },
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to connect the role and key. Please try again or contact support@unkey.com.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "authorization.connect_role_and_key",
          description: `Connect role ${role.id} to ${key.id}`,
          resources: [
            {
              type: "role",
              id: role.id,
              name: role.name,
            },
            {
              type: "key",
              id: key.id,
              name: key.name || undefined,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to connect the role to the key. Please try again or contact support@unkey.com",
        });
      });
  });
