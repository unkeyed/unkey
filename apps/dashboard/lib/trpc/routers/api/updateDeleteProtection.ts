import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { db, eq, schema } from "@/lib/db";
import { UPDATE_LIMIT, UPDATE_LIMIT_DURATION } from "@/lib/ratelimitValues";
import { ingestAuditLogs } from "@/lib/tinybird";
import { rateLimitedProcedure } from "../../trpc";

export const updateAPIDeleteProtection = rateLimitedProcedure({
  limit: UPDATE_LIMIT,
  duration: UPDATE_LIMIT_DURATION,
})
  .input(
    z.object({
      apiId: z.string(),
      enabled: z.boolean(),
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
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct API. Please contact support using support@unkey.dev.",
      });
    }

    await db
      .update(schema.apis)
      .set({
        deleteProtection: input.enabled,
      })
      .where(eq(schema.apis.id, input.apiId))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the API. Please contact support using support@unkey.dev.",
        });
      });
    await ingestAuditLogs({
      workspaceId: api.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "api.update",
      description: `API ${api.name} delete protection is now ${
        input.enabled ? "enabled" : "disabled"
      }.}`,
      resources: [
        {
          type: "api",
          id: api.id,
          meta: {
            deleteProtection: input.enabled,
          },
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
