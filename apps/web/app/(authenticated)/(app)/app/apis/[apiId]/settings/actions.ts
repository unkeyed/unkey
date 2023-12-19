"use server";

import { serverAction } from "@/lib/actions";
import { db, eq, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { revalidatePath } from "next/cache";
import { z } from "zod";

export const updateApiName = serverAction({
  input: z.object({
    name: z.string().min(3, "api names must contain at least 3 characters"),
    apiId: z.string(),
    workspaceId: z.string(),
  }),
  handler: async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
      with: {
        apis: {
          where: eq(schema.apis.id, input.apiId),
        },
      },
    });
    if (!ws || ws.tenantId !== ctx.tenantId) {
      throw new Error("workspace not found");
    }
    const api = ws.apis.find((api) => api.id === input.apiId);
    if (!api) {
      throw new Error("api not found");
    }

    await db
      .update(schema.apis)
      .set({
        name: input.name,
      })
      .where(eq(schema.apis.id, input.apiId));

    revalidatePath(`/apps/api/${input.apiId}`);
  },
});

export const updateIpWhitelist = serverAction({
  input: z.object({
    ips: z.string().transform((s, ctx) => {
      const ips = s.split(/,|\n/).map((ip) => ip.trim());
      const parsedIps = z.array(z.string().ip()).safeParse(ips);
      if (!parsedIps.success) {
        ctx.addIssue(parsedIps.error.issues[0]);
        return z.NEVER;
      }
      return parsedIps.data;
    }),
    apiId: z.string(),
    workspaceId: z.string(),
  }),
  handler: async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(schema.workspaces.id, input.workspaceId), isNull(table.deletedAt)),
      with: {
        apis: {
          where: eq(schema.apis.id, input.apiId),
        },
      },
    });
    if (!ws || ws.tenantId !== ctx.tenantId) {
      throw new Error("workspace not found");
    }
    const api = ws.apis.find((api) => api.id === input.apiId);
    if (!api) {
      throw new Error("api not found");
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.apis)
        .set({
          ipWhitelist: input.ips.join(","),
        })
        .where(eq(schema.apis.id, input.apiId));
      await db.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        workspaceId: ws.id,
        apiId: api.id,
        event: "api.update",
        description: "IP whitelist updated",
        time: new Date(),
        actorType: "user",
        actorId: ctx.userId,
      });
    });

    revalidatePath(`/apps/api/${input.apiId}`);
  },
});
