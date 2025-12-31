import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteRootKeys = workspaceProcedure
  .input(
    z.object({
      keyIds: z
        .union([z.string(), z.array(z.string())])
        .transform((ids) => (Array.isArray(ids) ? ids : [ids])),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    let deletedCount = 0;

    await db.transaction(async (tx) => {
      // Fetch all root keys to validate existence and get details for audit logs
      const rootKeys = await tx.query.keys.findMany({
        where: (table, { eq, inArray, isNull, and }) =>
          and(
            eq(table.workspaceId, env().UNKEY_WORKSPACE_ID),
            eq(table.forWorkspaceId, ctx.workspace.id),
            inArray(table.id, input.keyIds),
            isNull(table.deletedAtM),
          ),
        columns: {
          id: true,
          name: true,
          start: true,
        },
      });

      if (rootKeys.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message:
            "No valid root keys found. They may have already been deleted or you don't have access to them.",
        });
      }

      if (rootKeys.length !== input.keyIds.length) {
        const foundIds = rootKeys.map((k) => k.id);
        const missingIds = input.keyIds.filter((id) => !foundIds.includes(id));
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Root key(s) not found: ${missingIds.join(
            ", ",
          )}. They may have already been deleted or you don't have access to them.`,
        });
      }

      // Set the count before deletion
      deletedCount = rootKeys.length;

      // Soft delete the root keys
      await tx
        .update(schema.keys)
        .set({ deletedAtM: Date.now() })
        .where(
          and(
            inArray(
              schema.keys.id,
              rootKeys.map((k) => k.id),
            ),
            eq(schema.keys.workspaceId, env().UNKEY_WORKSPACE_ID),
            eq(schema.keys.forWorkspaceId, ctx.workspace.id),
          ),
        )
        .catch((err) => {
          console.error("Failed to delete root keys:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the root keys. Please try again or contact support@unkey.dev",
          });
        });

      // Create audit log entry
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "key.delete",
        description: `Deleted ${rootKeys.length} root key(s): ${rootKeys
          .map((k) => k.name || k.start || k.id)
          .join(", ")}`,
        resources: rootKeys.map((key) => ({
          type: "key",
          id: key.id,
          name: key.name || key.start || key.id,
        })),
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      }).catch((err) => {
        console.error("Failed to create audit log:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the root keys. Please try again or contact support@unkey.dev",
        });
      });
    });

    return { deletedCount };
  });
