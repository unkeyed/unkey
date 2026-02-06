import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const updateKeysEnabled = workspaceProcedure
  .input(
    z.object({
      keyIds: z
        .union([z.string(), z.array(z.string())])
        .transform((ids) => (Array.isArray(ids) ? ids : [ids])),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    // Ensure we have at least one keyId to update
    if (input.keyIds.length === 0) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "At least one keyId must be provided",
      });
    }

    const keys = await db.query.keys
      .findMany({
        where: (table, { eq, and, isNull, inArray }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            inArray(table.id, input.keyIds),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to retrieve the keys. Please try again or contact support@unkey.com",
        });
      });

    if (keys.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find any of the specified keys. Please try again or contact support@unkey.com.",
      });
    }

    // Check if any keys were not found
    const foundKeyIds = keys.map((key) => key.id);
    const missingKeyIds = input.keyIds.filter((id) => !foundKeyIds.includes(id));

    if (missingKeyIds.length > 0) {
      console.warn(`Some keys were not found: ${missingKeyIds.join(", ")}`);
    }

    try {
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            enabled: input.enabled,
          })
          .where(
            and(
              inArray(schema.keys.id, foundKeyIds),
              eq(schema.keys.workspaceId, ctx.workspace.id),
            ),
          )
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update enabled status on these keys. Please try again or contact support@unkey.com",
            });
          });

        const keyIds = keys.map((key) => key.id).join(", ");
        const description = `Updated enabled status of keys [${keyIds}] to ${
          input.enabled ? "enabled" : "disabled"
        }`;

        const resources: UnkeyAuditLog["resources"] = keys.map((key) => ({
          type: "key",
          id: key.id,
          name: key.name || undefined,
          meta: {
            enabled: input.enabled,
            "previous.enabled": key.enabled,
          },
        }));

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description,
          resources,
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    } catch (err) {
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We were unable to update enabled status on these keys. Please try again or contact support@unkey.com",
      });
    }

    return {
      enabled: input.enabled,
      updatedKeyIds: foundKeyIds,
      missingKeyIds: missingKeyIds.length > 0 ? missingKeyIds : undefined,
    };
  });
