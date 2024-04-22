import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { auth, t } from "../../trpc";

export const updateNamespaceName = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(3, "namespace names must contain at least 3 characters"),
      namespaceId: z.string(),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, input.workspaceId), isNull(table.deletedAt)),
      with: {
        ratelimitNamespaces: {
          where: (table, { eq, and, isNull }) =>
            and(isNull(table.deletedAt), eq(schema.ratelimitNamespaces.id, input.namespaceId)),
        },
      },
    });

    if (!ws || ws.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        message: "workspace not found",
        code: "NOT_FOUND",
      });
    }
    const namespace = ws.ratelimitNamespaces.find((ns) => ns.id === input.namespaceId);
    if (!namespace) {
      throw new TRPCError({ message: "namespace not found", code: "NOT_FOUND" });
    }

    await db
      .update(schema.ratelimitNamespaces)
      .set({
        name: input.name,
      })
      .where(eq(schema.ratelimitNamespaces.id, input.namespaceId));
    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "ratelimitNamespace.update",
      description: `Changed ${namespace.id} name from ${namespace.name} to ${input.name}`,
      resources: [
        {
          type: "ratelimitNamespace",
          id: namespace.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
