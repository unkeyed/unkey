import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth } from "../../trpc";

export const updateKeyName = rateLimitedProcedure(ratelimit.update)
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      name: z.string().nullish(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, isNull, and }) =>
          and(eq(table.id, input.keyId), isNull(table.deletedAt)),
        with: {
          workspace: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the name on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
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
            name: input.name ?? null,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update name on this key. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: key.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `Changed name of ${key.id} to ${input.name}`,
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update name on this key. Please try again or contact support@unkey.dev",
        });
      });

    return true;
  });
