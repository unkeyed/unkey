import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const updateKeyEnabled = t.procedure
  .use(auth)
  .input(
    z.object({
      keyId: z.string(),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.keyId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ code: "NOT_FOUND", message: "key not found" });
    }
    await db
      .update(schema.keys)
      .set({
        enabled: input.enabled,
      })
      .where(eq(schema.keys.id, key.id));
    await ingestAuditLogs({
      workspaceId: key.workspace.id,
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
  });
