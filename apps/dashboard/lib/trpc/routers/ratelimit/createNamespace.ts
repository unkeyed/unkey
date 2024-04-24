import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { DatabaseError } from "@planetscale/database";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";

export const createNamespace = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const namespaceId = newId("ratelimitNamespace");
    try {
      await db.insert(schema.ratelimitNamespaces).values({
        id: namespaceId,
        name: input.name,
        workspaceId: ws.id,

        createdAt: new Date(),
      });
    } catch (e) {
      if (e instanceof DatabaseError && e.body.message.includes("desc = Duplicate entry")) {
        throw new TRPCError({ code: "PRECONDITION_FAILED", message: "Duplicate namespace name" });
      }
      throw e;
    }

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "ratelimitNamespace.create",
      description: `Created ${namespaceId}`,
      resources: [
        {
          type: "ratelimitNamespace",
          id: namespaceId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id: namespaceId,
    };
  });
