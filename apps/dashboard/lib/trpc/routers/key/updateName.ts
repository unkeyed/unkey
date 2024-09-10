import { db, eq, schema } from "@/lib/db";
import { UPDATE_LIMIT, UPDATE_LIMIT_DURATION } from "@/lib/ratelimitValues";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";
import { rateLimitedProcedure } from "../../trpc";

export const updateKeyName = rateLimitedProcedure({
  limit: UPDATE_LIMIT,
  duration: UPDATE_LIMIT_DURATION,
})
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      name: z.string().nullish(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys.findFirst({
      where: (table, { eq, isNull, and }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please contact support using support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }
    await db
      .update(schema.keys)
      .set({
        name: input.name ?? null,
      })
      .where(eq(schema.keys.id, key.id))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update name on this key. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
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
    return true;
  });
