import { keyBytesSchema } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const setDefaultApiBytes = workspaceProcedure
  .input(
    z.object({
      defaultBytes: keyBytesSchema,
      keyAuthId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const keyAuth = await db.query.keyAuth
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.keyAuthId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the key auth. Please try again or contact support@unkey.com",
        });
      });

    if (!keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct key auth. Please try again or contact support@unkey.com.",
      });
    }

    try {
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keyAuth)
          .set({
            defaultBytes: input.defaultBytes,
          })
          .where(eq(schema.keyAuth.id, keyAuth.id));

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          description: `Changed ${keyAuth.id} default byte size for keys from ${keyAuth.defaultBytes} to ${input.defaultBytes}`,
          resources: [
            {
              type: "keyAuth",
              id: keyAuth.id,
            },
          ],
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
          "We were unable to update the default bytes. Please try again or contact support@unkey.com.",
      });
    }
  });
