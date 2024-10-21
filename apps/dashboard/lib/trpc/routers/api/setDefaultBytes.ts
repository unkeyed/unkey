import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";

export const setDefaultApiBytes = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      defaultBytes: z
        .number()
        .min(8, "Byte size needs to be at least 8")
        .max(255, "Byte size cannot exceed 255")
        .optional(),
      keyAuthId: z.string(),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const keyAuth = await db.query.keyAuth
      .findFirst({
        where: (table, { eq }) => eq(table.id, input.keyAuthId),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to find the KeyAuth. Please try again or contact support@unkey.dev.",
        });
      });
    if (!keyAuth || keyAuth.workspaceId !== input.workspaceId) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please try again or contact support@unkey.dev",
      });
    }
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keyAuth)
          .set({
            defaultBytes: input.defaultBytes,
          })
          .where(eq(schema.keyAuth.id, input.keyAuthId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update the API default bytes. Please try again or contact support@unkey.dev.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: keyAuth.workspaceId,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          description: `Changed ${keyAuth.workspaceId} default byte size for keys from ${keyAuth.defaultBytes} to ${input.defaultBytes}`,
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the default bytes. Please try again or contact support@unkey.dev.",
        });
      });
  });
