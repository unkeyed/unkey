import { insertAuditLogs } from "@/lib/audit";
import { db, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const updateKeysEnabled = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      keyIds: z.array(z.string()),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
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
            "We were unable to retrieve the keys. Please try again or contact support@unkey.dev",
        });
      });

    if (keys.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find any of the specified keys. Please try again or contact support@unkey.dev.",
      });
    }

    // Check if any keys were not found
    const foundKeyIds = keys.map((key) => key.id);
    const missingKeyIds = input.keyIds.filter((id) => !foundKeyIds.includes(id));

    await db
      .transaction(async (tx) => {
        // Update all keys in a single query
        await tx
          .update(schema.keys)
          .set({
            enabled: input.enabled,
          })
          .where(inArray(schema.keys.id, foundKeyIds))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update enabled status on these keys. Please try again or contact support@unkey.dev",
            });
          });

        // Insert audit logs for each key
        for (const key of keys) {
          await insertAuditLogs(tx, {
            workspaceId: key.workspaceId,
            actor: {
              type: "user",
              id: ctx.user.id,
            },
            event: "key.update",
            description: `${input.enabled ? "Enabled" : "Disabled"} ${key.id}`,
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
          });
        }
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update enabled status on these keys. Please try again or contact support@unkey.dev",
        });
      });

    return {
      enabled: input.enabled,
      updatedKeyIds: foundKeyIds,
      missingKeyIds: missingKeyIds.length > 0 ? missingKeyIds : undefined,
    };
  });
