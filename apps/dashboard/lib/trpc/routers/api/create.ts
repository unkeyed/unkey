import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { rateLimitedProcedure } from "../../trpc";
import { CREATE_LIMIT, CREATE_LIMIT_DURATION } from "@/lib/ratelimitValues";

export const createApi = rateLimitedProcedure({ limit: CREATE_LIMIT, duration: CREATE_LIMIT_DURATION })
  .input(
    z.object({
      name: z
        .string()
        .min(1, "workspace names must contain at least 3 characters")
        .max(50, "workspace names must contain at most 50 characters"),
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
        message: "The workspace does not exist.",
      });
    }

    const keyAuthId = newId("keyAuth");
    try {
      await db.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: ws.id,
        createdAt: new Date(),
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We are unable to create an API. Please contact support using support@unkey.dev",
      });
    }

    const apiId = newId("api");

    await db
      .insert(schema.apis)
      .values({
        id: apiId,
        name: input.name,
        workspaceId: ws.id,
        keyAuthId,
        authType: "key",
        ipWhitelist: null,
        createdAt: new Date(),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create the API. Please contact support using support@unkey.dev",
        });
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
