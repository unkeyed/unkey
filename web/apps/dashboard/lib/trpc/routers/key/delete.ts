import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deleteKeys = workspaceProcedure
  .input(
    z.object({
      keyIds: z.array(z.string()).min(1, "At least one key ID must be provided"),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const { keyIds } = input;
    const userId = ctx.user.id;
    const tenantId = ctx.tenant.id;

    if (keyIds.length === 0) {
      console.warn({
        message: "No key IDs provided for deletion",
        userId,
        tenantId,
      });
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "No keys were provided for deletion",
      });
    }

    try {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, tenantId), isNull(table.deletedAtM)),
        with: {
          keys: {
            where: (table, { and, inArray, isNull }) =>
              and(isNull(table.deletedAtM), inArray(table.id, keyIds)),
            columns: {
              id: true,
            },
          },
        },
      });

      if (!workspace) {
        console.error({
          message: "Workspace not found",
          userId,
          tenantId,
        });
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found. Please verify your workspace selection and try again.",
        });
      }

      // Check if keys were found in the workspace
      if (workspace.keys.length === 0) {
        console.warn({
          message: "No keys found for deletion",
          userId,
          tenantId,
          keyIds,
        });
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "None of the specified keys could be found in your workspace.",
        });
      }

      try {
        await db.transaction(async (tx) => {
          await tx
            .update(schema.keys)
            .set({ deletedAtM: Date.now() })
            .where(
              and(
                eq(schema.keys.workspaceId, workspace.id),
                inArray(
                  schema.keys.id,
                  workspace.keys.map((k) => k.id),
                ),
              ),
            );

          await insertAuditLogs(
            tx,
            workspace.keys.map((key) => ({
              workspaceId: workspace.id,
              actor: { type: "user", id: userId },
              event: "key.delete",
              description: `Deleted key ${key.id}`,
              resources: [
                {
                  type: "key",
                  id: key.id,
                },
              ],
              context: {
                location: ctx.audit.location,
                userAgent: ctx.audit.userAgent,
              },
            })),
          );
        });
      } catch (txErr) {
        console.error({
          message: "Transaction failed during key deletion",
          userId,
          tenantId,
          workspaceId: workspace.id,
          error: txErr instanceof Error ? txErr.message : String(txErr),
          stack: txErr instanceof Error ? txErr.stack : undefined,
        });

        // Check for specific error types
        const errorMessage = String(txErr);
        if (errorMessage.includes("Forbidden")) {
          throw new TRPCError({
            code: "FORBIDDEN",
            message:
              "Permission denied. The system doesn't have sufficient access to complete this operation.",
          });
        }
        if (errorMessage.includes("auditLog")) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Unable to create audit logs. Please contact support@unkey.com.",
          });
        }
        throw txErr; // Re-throw to be caught by outer catch
      }

      return {
        deletedKeyIds: workspace.keys.map((k) => k.id),
        totalDeleted: workspace.keys.length,
      };
    } catch (err) {
      if (err instanceof TRPCError) {
        // Re-throw if it's already a TRPC error
        throw err;
      }

      console.error({
        message: "Error during key deletion",
        userId,
        tenantId,
        keyIds,
        error: err instanceof Error ? err.message : String(err),
        stack: err instanceof Error ? err.stack : undefined,
      });

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete keys. Our team has been notified of this issue.",
      });
    }
  });
