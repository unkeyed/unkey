import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";

export const createApi = t.procedure
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

    const keyAuthId = newId("keyAuth");
    await db.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: ws.id,
      createdAt: new Date(),
    });

    const apiId = newId("api");
    await db.insert(schema.apis).values({
      id: apiId,
      name: input.name,
      workspaceId: ws.id,
      keyAuthId,
      authType: "key",
      ipWhitelist: null,
      createdAt: new Date(),
    });
    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "api.create",
      description: `Created ${apiId}`,
      resources: [
        {
          type: "api",
          id: apiId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id: apiId,
    };
  });
