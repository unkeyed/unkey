import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyExpiration = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      enableExpiration: z.boolean(),
      expiration: z.date().nullish(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    let expires: Date | null = null;
    if (input.enableExpiration) {
      if (!input.expiration) {
        throw new TRPCError({
          message: "you must enter a valid date",
          code: "BAD_REQUEST",
        });
      }
      try {
        expires = new Date(input.expiration);
      } catch (e) {
        console.error(e);
        throw new TRPCError({
          message: "you must enter a valid date",
          code: "BAD_REQUEST",
        });
      }
    }

    const key = await db.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
    }
    await db
      .update(schema.keys)
      .set({
        expires,
      })
      .where(eq(schema.keys.id, key.id));
    await ingestAuditLogs({
      workspaceId: key.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "key.update",
      description: `${
        input.expiration
          ? `Changed expiration of ${key.id} to ${input.expiration.toUTCString()}`
          : `Disabled expiration for ${key.id}`
      }`,
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
