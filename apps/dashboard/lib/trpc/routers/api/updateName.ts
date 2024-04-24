import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const updateApiName = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(3, "api names must contain at least 3 characters"),
      apiId: z.string(),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const api = await db.query.apis.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.id, input.apiId), isNull(table.deletedAt)),
      with: {
        workspace: true,
      },
    });
    if (!api || api.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
    }
    await db
      .update(schema.apis)
      .set({
        name: input.name,
      })
      .where(eq(schema.apis.id, input.apiId));
    await ingestAuditLogs({
      workspaceId: api.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "api.update",
      description: `Changed ${api.id} name from ${api.name} to ${input.name}`,
      resources: [
        {
          type: "api",
          id: api.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
