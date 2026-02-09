import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { metadataSchema } from "@/lib/schemas/metadata";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const updateIdentityMetadata = workspaceProcedure
  .input(
    metadataSchema.extend({
      identityId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let meta: Record<string, unknown> | null = null;

    if (input.metadata?.enabled && input.metadata.data) {
      try {
        meta = JSON.parse(input.metadata.data);
      } catch (e) {
        throw new TRPCError({
          message: `The metadata is not valid JSON: ${(e as Error).message}. Please try again.`,
          code: "BAD_REQUEST",
        });
      }

      // Check 1MB size limit (1048576 bytes)
      const metadataString = JSON.stringify(meta);
      const sizeInBytes = new TextEncoder().encode(metadataString).length;
      if (sizeInBytes > 1048576) {
        throw new TRPCError({
          message: `Metadata size (${Math.round(sizeInBytes / 1024)}KB) exceeds the 1MB limit.`,
          code: "BAD_REQUEST",
        });
      }
    } else {
      meta = null;
    }

    const identity = await db.query.identities
      .findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.identityId)),
      })
      .catch((err) => {
        console.error("Failed to fetch identity:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to load this identity. Please try again or contact support@unkey.com",
        });
      });

    if (!identity) {
      throw new TRPCError({
        message:
          "We are unable to find the correct identity. Please try again or contact support@unkey.com.",
        code: "NOT_FOUND",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.identities)
          .set({
            meta,
          })
          .where(eq(schema.identities.id, identity.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update metadata on this identity. Please try again or contact support@unkey.com",
            });
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "identity.update",
          description:
            input.metadata?.enabled && input.metadata.data
              ? `Updated metadata of ${identity.id}`
              : `Removed metadata from ${identity.id}`,
          resources: [
            {
              type: "identity",
              id: identity.id,
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
            "We are unable to update metadata on this identity. Please try again or contact support@unkey.com",
        });
      });

    return {
      identityId: identity.id,
      success: true,
    };
  });
