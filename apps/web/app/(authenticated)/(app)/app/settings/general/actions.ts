"use server";

import { serverAction } from "@/lib/actions";
import { db, eq, schema } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { z } from "zod";

export const updateWorkspaceName = serverAction({
  input: z.object({
    name: z.string().min(3, "workspace names must contain at least 3 characters"),
    workspaceId: z.string(),
  }),
  handler: async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
    });
    if (!ws || ws.tenantId !== ctx.tenantId) {
      throw new Error("workspace not found");
    }

    await db
      .update(schema.workspaces)
      .set({
        name: input.name,
      })
      .where(eq(schema.workspaces.id, input.workspaceId));
    if (ctx.tenantId.startsWith("org_")) {
      await clerkClient.organizations.updateOrganization(ctx.tenantId, { name: input.name });
    }
  },
});
