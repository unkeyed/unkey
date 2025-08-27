import { expirationSchema } from "@/app/(app)/[workspace]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const updateKeyExpiration = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    expirationSchema.extend({
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let expires: Date | null = null;
    if (input.expiration?.enabled && input.expiration.data) {
      expires = input.expiration.data;
    } else {
      expires = null;
    }

    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
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
            "We were unable to update expiration on this key. Please try again or contact support@unkey.dev",
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
            expires,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update expiration on this key. Please try again or contact support@unkey.dev",
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
            input.expiration?.enabled && input.expiration.data
              ? `Changed expiration of ${key.id} to ${input.expiration.data.toUTCString()}`
              : `Disabled expiration for ${key.id}`,
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
            "We were unable to update expiration on this key. Please try again or contact support@unkey.dev",
        });
      });

    return {
      keyId: key.id,
    };
  });
