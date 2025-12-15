import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { metadataSchema } from "@/lib/schemas/metadata";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const updateKeyMetadata = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    metadataSchema.extend({
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let meta: unknown | null = null;

    if (input.metadata?.enabled && input.metadata.data) {
      try {
        meta = JSON.parse(input.metadata.data);
      } catch (e) {
        throw new TRPCError({
          message: `The metadata is not valid JSON: ${(e as Error).message}. Please try again.`,
          code: "BAD_REQUEST",
        });
      }
    } else {
      meta = null;
    }

    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, isNull, and }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.keyId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update metadata on this key. Please try again or contact support@unkey.dev",
        });
      });

    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            meta: meta ? JSON.stringify(meta) : null,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update metadata on this key. Please try again or contact support@unkey.dev",
            });
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description:
            input.metadata?.enabled && input.metadata.data
              ? `Updated metadata of ${key.id}`
              : `Removed metadata from ${key.id}`,
          resources: [
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
            "We are unable to update metadata on this key. Please try again or contact support@unkey.dev",
        });
      });

    return {
      keyId: key.id,
      success: true,
    };
  });
